package handlers

import (
	"bytes"
	"context"
	"fmt"

	"github.com/tokenized/smart-contract/cmd/smartcontractd/listeners"
	"github.com/tokenized/smart-contract/internal/asset"
	"github.com/tokenized/smart-contract/internal/contract"
	"github.com/tokenized/smart-contract/internal/holdings"
	"github.com/tokenized/smart-contract/internal/platform/db"
	"github.com/tokenized/smart-contract/internal/platform/node"
	"github.com/tokenized/smart-contract/internal/platform/protomux"
	"github.com/tokenized/smart-contract/internal/platform/state"
	"github.com/tokenized/smart-contract/internal/transactions"
	"github.com/tokenized/smart-contract/internal/vote"
	"github.com/tokenized/smart-contract/pkg/bitcoin"
	"github.com/tokenized/smart-contract/pkg/inspector"
	"github.com/tokenized/smart-contract/pkg/scheduler"
	"github.com/tokenized/smart-contract/pkg/wallet"
	"github.com/tokenized/smart-contract/pkg/wire"
	"github.com/tokenized/specification/dist/golang/actions"
	"github.com/tokenized/specification/dist/golang/protocol"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/pkg/errors"
	"go.opencensus.io/trace"
)

type Governance struct {
	handler   protomux.Handler
	MasterDB  *db.DB
	Config    *node.Config
	Scheduler *scheduler.Scheduler
}

// ProposalRequest handles an incoming proposal request and prepares a Vote response
func (g *Governance) ProposalRequest(ctx context.Context, w *node.ResponseWriter, itx *inspector.Transaction, rk *wallet.Key) error {
	ctx, span := trace.StartSpan(ctx, "handlers.Governance.ProposalRequest")
	defer span.End()

	msg, ok := itx.MsgProto.(*actions.Proposal)
	if !ok {
		return errors.New("Could not assert as *actions.Initiative")
	}

	v := ctx.Value(node.KeyValues).(*node.Values)

	// Validate all fields have valid values.
	if itx.RejectCode != 0 {
		node.LogWarn(ctx, "Proposal request invalid")
		return node.RespondReject(ctx, w, itx, rk, itx.RejectCode)
	}

	// Locate Contract
	ct, err := contract.Retrieve(ctx, g.MasterDB, rk.Address)
	if err != nil {
		return errors.Wrap(err, "Failed to retrieve contract")
	}

	if ct.MovedTo != nil {
		address := bitcoin.NewAddressFromRawAddress(ct.MovedTo,
			wire.BitcoinNet(w.Config.ChainParams.Net))
		node.LogWarn(ctx, "Contract address changed : %s", address.String())
		return node.RespondReject(ctx, w, itx, rk, actions.RejectContractMoved)
	}

	if ct.FreezePeriod.Nano() > v.Now.Nano() {
		node.LogWarn(ctx, "Contract frozen")
		return node.RespondReject(ctx, w, itx, rk, actions.RejectContractFrozen)
	}

	if ct.ContractExpiration.Nano() != 0 && ct.ContractExpiration.Nano() < v.Now.Nano() {
		node.LogWarn(ctx, "Contract expired : %s", ct.ContractExpiration.String())
		return node.RespondReject(ctx, w, itx, rk, actions.RejectContractExpired)
	}

	// Verify first two outputs are to contract
	if len(itx.Outputs) < 2 || !itx.Outputs[0].Address.Equal(rk.Address) ||
		!itx.Outputs[1].Address.Equal(rk.Address) {
		node.LogWarn(ctx, "Proposal failed to fund vote and result txs")
		return node.RespondReject(ctx, w, itx, rk, actions.RejectInsufficientTxFeeFunding)
	}

	// Check if sender is allowed to make proposal
	if msg.Initiator == 0 { // Administration Proposal
		if !contract.IsOperator(ctx, ct, itx.Inputs[0].Address) {
			node.LogWarn(ctx, "Initiator is not administration or operator")
			return node.RespondReject(ctx, w, itx, rk, actions.RejectNotOperator)
		}
	} else if msg.Initiator == 1 { // Holder Proposal
		// Sender must hold balance of at least one asset
		if !contract.HasAnyBalance(ctx, g.MasterDB, ct, itx.Inputs[0].Address) {
			address := bitcoin.NewAddressFromRawAddress(itx.Inputs[0].Address,
				wire.BitcoinNet(w.Config.ChainParams.Net))
			node.LogWarn(ctx, "Sender holds no assets : %s", address.String())
			return node.RespondReject(ctx, w, itx, rk, actions.RejectInsufficientQuantity)
		}
	} else {
		node.LogWarn(ctx, "Invalid Initiator value : %02x", msg.Initiator)
		return node.RespondReject(ctx, w, itx, rk, actions.RejectMsgMalformed)
	}

	if int(msg.VoteSystem) >= len(ct.VotingSystems) {
		node.LogWarn(ctx, "Proposal vote system invalid")
		return node.RespondReject(ctx, w, itx, rk, actions.RejectMsgMalformed)
	}

	if msg.Specific && ct.VotingSystems[msg.VoteSystem].VoteType == "P" {
		node.LogWarn(ctx, "Plurality votes not allowed for specific votes")
		return node.RespondReject(ctx, w, itx, rk, actions.RejectVoteSystemNotPermitted)
	}

	// Validate messages vote related values
	if err := vote.ValidateProposal(msg, v.Now); err != nil {
		node.LogWarn(ctx, "Proposal validation failed : %s", err)
		return node.RespondReject(ctx, w, itx, rk, actions.RejectMsgMalformed)
	}

	if msg.AssetSpecificVote {
		as, err := asset.Retrieve(ctx, g.MasterDB, rk.Address,
			protocol.AssetCodeFromBytes(msg.AssetCode))
		if err != nil {
			node.LogWarn(ctx, "Asset not found : %x", msg.AssetCode)
			return node.RespondReject(ctx, w, itx, rk, actions.RejectAssetNotFound)
		}

		if as.FreezePeriod.Nano() > v.Now.Nano() {
			node.LogWarn(ctx, "Proposal failed. Asset frozen : %x", msg.AssetCode)
			return node.RespondReject(ctx, w, itx, rk, actions.RejectAssetFrozen)
		}

		// Asset does not allow voting
		if err := asset.ValidateVoting(ctx, as, msg.Initiator, ct.VotingSystems[msg.VoteSystem]); err != nil {
			node.LogWarn(ctx, "Asset does not allow voting: %x : %s", msg.AssetCode, err)
			return node.RespondReject(ctx, w, itx, rk, actions.RejectAssetAuthFlags)
		}

		// Check sender balance
		h, err := holdings.GetHolding(ctx, g.MasterDB, rk.Address,
			protocol.AssetCodeFromBytes(msg.AssetCode), itx.Inputs[0].Address, v.Now)
		if err != nil {
			return errors.Wrap(err, "Failed to get requestor holding")
		}

		if msg.Initiator > 0 && holdings.VotingBalance(as, h,
			ct.VotingSystems[msg.VoteSystem].VoteMultiplierPermitted, v.Now) == 0 {
			address := bitcoin.NewAddressFromRawAddress(itx.Inputs[0].Address,
				wire.BitcoinNet(w.Config.ChainParams.Net))
			node.LogWarn(ctx, "Requestor is not a holder : %x %s", msg.AssetCode, address.String())
			return node.RespondReject(ctx, w, itx, rk, actions.RejectInsufficientQuantity)
		}

		if msg.Specific {
			// Asset amendments vote. Validate permissions for fields being amended.
			if err := checkAssetAmendmentsPermissions(as, ct.VotingSystems, msg.ProposedAmendments,
				true, msg.Initiator, msg.VoteSystem); err != nil {
				node.LogWarn(ctx, "Asset amendments not permitted : %s", err)
				return node.RespondReject(ctx, w, itx, rk, actions.RejectAssetAuthFlags)
			}

			if msg.VoteOptions != "AB" || msg.VoteMax != 1 {
				node.LogWarn(ctx, "Single option AB votes are required for specific amendments")
				return node.RespondReject(ctx, w, itx, rk, actions.RejectMsgMalformed)
			}

			// Validate proposed amendments.
			ac := actions.AssetCreation{}

			err = node.Convert(ctx, &as, &ac)
			if err != nil {
				return errors.Wrap(err, "Failed to convert state asset to asset creation")
			}

			ac.AssetRevision = as.Revision + 1
			ac.Timestamp = v.Now.Nano()

			if err := applyAssetAmendments(&ac, ct.VotingSystems, msg.ProposedAmendments); err != nil {
				node.LogWarn(ctx, "Asset amendments failed : %s", err)
				return node.RespondReject(ctx, w, itx, rk, actions.RejectMsgMalformed)
			}
		}
	} else {
		// Contract does not allow voting
		if err := contract.ValidateVoting(ctx, ct, msg.Initiator); err != nil {
			node.LogWarn(ctx, "Contract does not allow voting : %s", err)
			return node.RespondReject(ctx, w, itx, rk, actions.RejectContractAuthFlags)
		}

		if msg.Specific {
			// Contract amendments vote. Validate permissions for fields being amended.
			if err := checkContractAmendmentsPermissions(ct, msg.ProposedAmendments, true, msg.Initiator, msg.VoteSystem); err != nil {
				node.LogWarn(ctx, "Asset amendments not permitted : %s", err)
				return node.RespondReject(ctx, w, itx, rk, actions.RejectContractAuthFlags)
			}

			if msg.VoteOptions != "AB" || msg.VoteMax != 1 {
				node.LogWarn(ctx, "Single option AB votes are required for specific amendments")
				return node.RespondReject(ctx, w, itx, rk, actions.RejectMsgMalformed)
			}

			// Validate proposed amendments.
			cf := actions.ContractFormation{}

			// Get current state
			err = node.Convert(ctx, &ct, &cf)
			if err != nil {
				return errors.Wrap(err, "Failed to convert state contract to contract formation")
			}

			// Apply modifications
			cf.ContractRevision = ct.Revision + 1 // Bump the revision
			cf.Timestamp = v.Now.Nano()

			if err := applyContractAmendments(&cf, msg.ProposedAmendments); err != nil {
				node.LogWarn(ctx, "Contract amendments failed : %s", err)
				return node.RespondReject(ctx, w, itx, rk, actions.RejectMsgMalformed)
			}
		}

		// Sender does not have any balance of the asset
		if msg.Initiator > 0 && contract.GetVotingBalance(ctx, g.MasterDB, ct, itx.Inputs[0].Address,
			ct.VotingSystems[msg.VoteSystem].VoteMultiplierPermitted, v.Now) == 0 {
			node.LogWarn(ctx, "Requestor is not a holder : %x", msg.AssetCode)
			return node.RespondReject(ctx, w, itx, rk, actions.RejectInsufficientQuantity)
		}
	}

	if msg.Specific {
		// Check existing votes that have not been applied yet for conflicting fields.
		votes, err := vote.List(ctx, g.MasterDB, rk.Address)
		if err != nil {
			return errors.Wrap(err, "Failed to list votes")
		}
		for _, vt := range votes {
			if !vt.AppliedTxId.IsZero() || !vt.Specific {
				continue // Already applied or doesn't contain specific amendments
			}

			if msg.AssetSpecificVote {
				if !vt.AssetSpecificVote || msg.AssetType != vt.AssetType ||
					!bytes.Equal(msg.AssetCode, vt.AssetCode.Bytes()) {
					continue // Not an asset amendment
				}
			} else {
				if vt.AssetSpecificVote {
					continue // Not a contract amendment
				}
			}

			// Determine if any fields conflict
			for _, field := range msg.ProposedAmendments {
				for _, otherField := range vt.ProposedAmendments {
					if field.FieldIndex == otherField.FieldIndex {
						// Reject because of conflicting field amendment on unapplied vote.
						node.LogWarn(ctx, "Proposed amendment conflicts with unapplied vote")
						return node.RespondReject(ctx, w, itx, rk, actions.RejectProposalConflicts)
					}
				}
			}
		}
	}

	// Build Response
	vote := actions.Vote{Timestamp: v.Now.Nano()}

	// Fund with first output of proposal tx. Second is reserved for vote result tx.
	w.SetUTXOs(ctx, []inspector.UTXO{itx.Outputs[0].UTXO})

	// Build outputs
	// 1 - Contract Address
	// 2 - Contract/Proposal Fee (change)
	w.AddOutput(ctx, rk.Address, 0)

	feeAmount := ct.ContractFee
	if msg.Initiator > 0 {
		feeAmount += ct.VotingSystems[msg.VoteSystem].HolderProposalFee
	}
	w.AddContractFee(ctx, feeAmount)

	// Save Tx.
	if err := transactions.AddTx(ctx, g.MasterDB, itx); err != nil {
		return errors.Wrap(err, "Failed to save tx")
	}

	// Respond with a vote
	node.LogVerbose(ctx, "Accepting proposal")
	return node.RespondSuccess(ctx, w, itx, rk, &vote)
}

// VoteResponse handles an incoming Vote response
func (g *Governance) VoteResponse(ctx context.Context, w *node.ResponseWriter, itx *inspector.Transaction, rk *wallet.Key) error {
	ctx, span := trace.StartSpan(ctx, "handlers.Governance.VoteResponse")
	defer span.End()

	msg, ok := itx.MsgProto.(*actions.Vote)
	if !ok {
		return errors.New("Could not assert as *actions.Vote")
	}

	if itx.RejectCode != 0 {
		return errors.New("Vote response invalid")
	}

	v := ctx.Value(node.KeyValues).(*node.Values)

	ct, err := contract.Retrieve(ctx, g.MasterDB, rk.Address)
	if err != nil {
		return err
	}

	if ct.MovedTo != nil {
		address := bitcoin.NewAddressFromRawAddress(ct.MovedTo,
			wire.BitcoinNet(w.Config.ChainParams.Net))
		return fmt.Errorf("Contract address changed : %s", address.String())
	}

	// Verify input is from contract
	if !itx.Inputs[0].Address.Equal(rk.Address) {
		return errors.New("Response not from contract")
	}

	// Retrieve Proposal
	proposalTx, err := transactions.GetTx(ctx, g.MasterDB, &itx.Inputs[0].UTXO.Hash, g.Config.IsTest)
	if err != nil {
		return errors.New("Proposal not found for vote")
	}

	proposal, ok := proposalTx.MsgProto.(*actions.Proposal)
	if !ok {
		return errors.New("Proposal invalid for vote")
	}

	voteTxId := protocol.TxIdFromBytes(itx.Hash[:])

	_, err = vote.Retrieve(ctx, g.MasterDB, rk.Address, voteTxId)
	if err != vote.ErrNotFound {
		if err != nil {
			return fmt.Errorf("Failed to retrieve vote : %s : %s", voteTxId.String(), err)
		} else {
			return fmt.Errorf("Vote already exists : %s", voteTxId.String())
		}
	}

	nv := vote.NewVote{}
	err = node.Convert(ctx, proposal, &nv)
	if err != nil {
		return errors.Wrap(err, "Failed to convert vote message to new vote")
	}

	nv.VoteTxId = *voteTxId
	nv.ProposalTxId = *protocol.TxIdFromBytes(proposalTx.Hash[:])
	nv.Expires = protocol.NewTimestamp(proposal.VoteCutOffTimestamp)
	nv.Timestamp = protocol.NewTimestamp(msg.Timestamp)

	if proposal.AssetSpecificVote {
		as, err := asset.Retrieve(ctx, g.MasterDB, rk.Address,
			protocol.AssetCodeFromBytes(proposal.AssetCode))
		if err != nil {
			return fmt.Errorf("Asset not found : %x", proposal.AssetCode)
		}

		if as.AssetModificationGovernance == 1 { // Contract wide asset governance
			nv.ContractWideVote = true
			nv.TokenQty = contract.GetTokenQty(ctx, g.MasterDB, ct,
				ct.VotingSystems[proposal.VoteSystem].VoteMultiplierPermitted)
		} else if ct.VotingSystems[proposal.VoteSystem].VoteMultiplierPermitted {
			nv.TokenQty = as.TokenQty * uint64(as.VoteMultiplier)
		} else {
			nv.TokenQty = as.TokenQty
		}
	} else {
		nv.TokenQty = contract.GetTokenQty(ctx, g.MasterDB, ct,
			ct.VotingSystems[proposal.VoteSystem].VoteMultiplierPermitted)
	}

	if err := vote.Create(ctx, g.MasterDB, rk.Address, voteTxId, &nv, v.Now); err != nil {
		return errors.Wrap(err, "Failed to save vote")
	}

	if err := g.Scheduler.ScheduleJob(ctx, listeners.NewVoteFinalizer(g.handler, itx,
		protocol.NewTimestamp(proposal.VoteCutOffTimestamp))); err != nil {
		return errors.Wrap(err, "Failed to schedule vote finalizer")
	}

	node.LogVerbose(ctx, "Creating vote : %s", itx.Hash.String())
	return nil
}

// BallotCastRequest handles an incoming BallotCast request and prepares a BallotCounted response
func (g *Governance) BallotCastRequest(ctx context.Context, w *node.ResponseWriter,
	itx *inspector.Transaction, rk *wallet.Key) error {

	ctx, span := trace.StartSpan(ctx, "handlers.Governance.BallotCastRequest")
	defer span.End()

	msg, ok := itx.MsgProto.(*actions.BallotCast)
	if !ok {
		return errors.New("Could not assert as *actions.BallotCast")
	}

	v := ctx.Value(node.KeyValues).(*node.Values)

	// Validate all fields have valid values.
	if itx.RejectCode != 0 {
		node.LogWarn(ctx, "Ballot cast request invalid")
		return node.RespondReject(ctx, w, itx, rk, itx.RejectCode)
	}

	ct, err := contract.Retrieve(ctx, g.MasterDB, rk.Address)
	if err != nil {
		return err
	}

	if ct.MovedTo != nil {
		address := bitcoin.NewAddressFromRawAddress(ct.MovedTo,
			wire.BitcoinNet(w.Config.ChainParams.Net))
		node.LogWarn(ctx, "Contract address changed : %s", address.String())
		return node.RespondReject(ctx, w, itx, rk, actions.RejectContractMoved)
	}

	voteTxId := protocol.TxIdFromBytes(msg.VoteTxId)
	vt, err := vote.Retrieve(ctx, g.MasterDB, rk.Address, voteTxId)
	if err == vote.ErrNotFound {
		node.LogWarn(ctx, "Vote not found : %s", voteTxId.String())
		return node.RespondReject(ctx, w, itx, rk, actions.RejectVoteNotFound)
	} else if err != nil {
		node.LogWarn(ctx, "Failed to retrieve vote : %s : %s", voteTxId.String(), err)
		return errors.Wrap(err, "Failed to retrieve vote")
	}

	if vt.Expires.Nano() <= v.Now.Nano() {
		node.LogWarn(ctx, "Vote expired : %s", voteTxId.String())
		return node.RespondReject(ctx, w, itx, rk, actions.RejectVoteClosed)
	}

	// Get Proposal
	hash, err := chainhash.NewHash(vt.ProposalTxId.Bytes())
	proposalTx, err := transactions.GetTx(ctx, g.MasterDB, hash, g.Config.IsTest)
	if err != nil {
		node.LogWarn(ctx, "Proposal not found for vote")
		return node.RespondReject(ctx, w, itx, rk, actions.RejectMsgMalformed)
	}

	proposal, ok := proposalTx.MsgProto.(*actions.Proposal)
	if !ok {
		node.LogWarn(ctx, "Proposal invalid for vote")
		return node.RespondReject(ctx, w, itx, rk, actions.RejectMsgMalformed)
	}

	// Validate vote
	if len(msg.Vote) > int(proposal.VoteMax) {
		node.LogWarn(ctx, "Ballot voted on too many options")
		return node.RespondReject(ctx, w, itx, rk, actions.RejectMsgMalformed)
	}

	if len(msg.Vote) == 0 {
		node.LogWarn(ctx, "Ballot did not vote any options")
		return node.RespondReject(ctx, w, itx, rk, actions.RejectMsgMalformed)
	}

	// Validate all chosen options are valid.
	for _, choice := range msg.Vote {
		found := false
		for _, option := range proposal.VoteOptions {
			if option == choice {
				found = true
				break
			}
		}
		if !found {
			node.LogWarn(ctx, "Ballot chose an invalid option : %s", msg.Vote)
			return node.RespondReject(ctx, w, itx, rk, actions.RejectMsgMalformed)
		}
	}

	// TODO Handle transfers during vote time to ensure they don't vote the same tokens more than once.

	quantity := uint64(0)

	// Add applicable holdings
	if proposal.AssetSpecificVote && !vt.ContractWideVote {
		as, err := asset.Retrieve(ctx, g.MasterDB, rk.Address,
			protocol.AssetCodeFromBytes(proposal.AssetCode))
		if err != nil {
			node.LogWarn(ctx, "Asset not found : %s", proposal.String())
			return node.RespondReject(ctx, w, itx, rk, actions.RejectAssetNotFound)
		}

		h, err := holdings.GetHolding(ctx, g.MasterDB, rk.Address,
			protocol.AssetCodeFromBytes(proposal.AssetCode), itx.Inputs[0].Address, v.Now)
		if err != nil {
			return errors.Wrap(err, "Failed to get requestor holding")
		}

		quantity = holdings.VotingBalance(as, h,
			ct.VotingSystems[proposal.VoteSystem].VoteMultiplierPermitted, v.Now)
	} else {
		quantity = contract.GetVotingBalance(ctx, g.MasterDB, ct, itx.Inputs[0].Address,
			ct.VotingSystems[proposal.VoteSystem].VoteMultiplierPermitted, v.Now)
	}

	if quantity == 0 {
		address := bitcoin.NewAddressFromRawAddress(itx.Inputs[0].Address,
			wire.BitcoinNet(w.Config.ChainParams.Net))
		node.LogWarn(ctx, "User PKH doesn't hold any voting tokens : %s", address.String())
		return node.RespondReject(ctx, w, itx, rk, actions.RejectInsufficientQuantity)
	}

	// TODO Check issue where two ballots are sent simultaneously and the second received before the first response is processed.
	if err := vote.CheckBallot(ctx, vt, itx.Inputs[0].Address); err != nil {
		node.LogWarn(ctx, "Failed to check ballot : %s", err)
		return node.RespondReject(ctx, w, itx, rk, actions.RejectBallotAlreadyCounted)
	}

	// Build Response
	ballotCounted := actions.BallotCounted{}
	err = node.Convert(ctx, msg, &ballotCounted)
	if err != nil {
		return errors.Wrap(err, "Failed to convert ballot cast to counted")
	}
	ballotCounted.Quantity = quantity
	ballotCounted.Timestamp = v.Now.Nano()

	// Build outputs
	// 1 - Contract Address
	// 2 - Contract Fee (change)
	w.AddOutput(ctx, rk.Address, 0)
	w.AddContractFee(ctx, ct.ContractFee)

	// Save Tx for response.
	if err := transactions.AddTx(ctx, g.MasterDB, itx); err != nil {
		return errors.Wrap(err, "Failed to add tx")
	}

	// Respond with a vote
	address := bitcoin.NewAddressFromRawAddress(itx.Inputs[0].Address,
		wire.BitcoinNet(w.Config.ChainParams.Net))
	node.LogWarn(ctx, "Accepting ballot for %d from %s", quantity, address.String())
	return node.RespondSuccess(ctx, w, itx, rk, &ballotCounted)
}

// BallotCountedResponse handles an outgoing BallotCounted action and writes it to the state
func (g *Governance) BallotCountedResponse(ctx context.Context, w *node.ResponseWriter, itx *inspector.Transaction, rk *wallet.Key) error {
	ctx, span := trace.StartSpan(ctx, "handlers.Governance.BallotCountedResponse")
	defer span.End()

	msg, ok := itx.MsgProto.(*actions.BallotCounted)
	if !ok {
		return errors.New("Could not assert as *actions.BallotCounted")
	}

	v := ctx.Value(node.KeyValues).(*node.Values)
	if itx.RejectCode != 0 {
		return errors.New("Ballot counted response invalid")
	}

	if !itx.Inputs[0].Address.Equal(rk.Address) {
		address := bitcoin.NewAddressFromRawAddress(itx.Inputs[0].Address,
			wire.BitcoinNet(w.Config.ChainParams.Net))
		return fmt.Errorf("Ballot counted not from contract : %s", address.String())
	}

	ct, err := contract.Retrieve(ctx, g.MasterDB, rk.Address)
	if err != nil {
		return errors.Wrap(err, "Failed to retrieve contract")
	}

	if ct.MovedTo != nil {
		address := bitcoin.NewAddressFromRawAddress(ct.MovedTo,
			wire.BitcoinNet(w.Config.ChainParams.Net))
		return fmt.Errorf("Contract address changed : %s", address.String())
	}

	castTx, err := transactions.GetTx(ctx, g.MasterDB, &itx.Inputs[0].UTXO.Hash, g.Config.IsTest)
	if err != nil {
		return fmt.Errorf("Ballot cast not found for ballot counted msg")
	}

	cast, ok := castTx.MsgProto.(*actions.BallotCast)
	if !ok {
		return fmt.Errorf("Ballot cast invalid for ballot counted")
	}

	voteTxId := protocol.TxIdFromBytes(cast.VoteTxId)
	vt, err := vote.Retrieve(ctx, g.MasterDB, rk.Address, voteTxId)
	if err != nil {
		return errors.Wrap(err, "Failed to retrieve vote for ballot cast")
	}

	ballot := state.Ballot{
		Address:   bitcoin.NewJSONRawAddress(castTx.Inputs[0].Address),
		Vote:      cast.Vote,
		Timestamp: protocol.NewTimestamp(msg.Timestamp),
		Quantity:  msg.Quantity,
	}

	// Add to vote results
	if err := vote.AddBallot(ctx, g.MasterDB, rk.Address, vt, &ballot, v.Now); err != nil {
		return errors.Wrap(err, "Failed to add ballot")
	}

	return nil
}

// FinalizeVote is called when a vote expires and sends the result response.
func (g *Governance) FinalizeVote(ctx context.Context, w *node.ResponseWriter, itx *inspector.Transaction, rk *wallet.Key) error {
	ctx, span := trace.StartSpan(ctx, "handlers.Governance.FinalizeVote")
	defer span.End()

	_, ok := itx.MsgProto.(*actions.Vote)
	if !ok {
		return errors.New("Could not assert as *actions.Vote")
	}

	v := ctx.Value(node.KeyValues).(*node.Values)

	node.LogVerbose(ctx, "Finalizing vote : %s", itx.Hash.String())

	// Retrieve contract
	ct, err := contract.Retrieve(ctx, g.MasterDB, rk.Address)
	if err != nil {
		return err
	}

	if ct.MovedTo != nil {
		address := bitcoin.NewAddressFromRawAddress(ct.MovedTo,
			wire.BitcoinNet(w.Config.ChainParams.Net))
		return fmt.Errorf("Contract address changed : %s", address.String())
	}

	// Retrieve vote
	voteTxId := protocol.TxIdFromBytes(itx.Hash[:])
	vt, err := vote.Retrieve(ctx, g.MasterDB, rk.Address, voteTxId)
	if err != nil {
		return errors.Wrap(err, "Failed to retrieve vote for ballot cast")
	}

	// Get Proposal
	hash, err := chainhash.NewHash(vt.ProposalTxId.Bytes())
	proposalTx, err := transactions.GetTx(ctx, g.MasterDB, hash, g.Config.IsTest)
	if err != nil {
		return fmt.Errorf("Proposal not found for vote")
	}

	proposal, ok := proposalTx.MsgProto.(*actions.Proposal)
	if !ok {
		return fmt.Errorf("Proposal invalid for vote")
	}

	// Build Response
	voteResult := actions.Result{}
	err = node.Convert(ctx, proposal, &voteResult)
	if err != nil {
		return errors.Wrap(err, "Failed to convert vote proposal to result")
	}

	voteResult.VoteTxId = voteTxId.Bytes()
	voteResult.Timestamp = v.Now.Nano()

	// Calculate Results
	voteResult.OptionTally, voteResult.Result, err = vote.CalculateResults(ctx, vt, proposal,
		ct.VotingSystems[proposal.VoteSystem])
	if err != nil {
		return errors.Wrap(err, "Failed to calculate vote results")
	}

	// Fund with second output of proposal tx.
	w.SetUTXOs(ctx, []inspector.UTXO{proposalTx.Outputs[1].UTXO})

	// Build outputs
	// 1 - Contract Address
	// 2 - Contract Fee (change)
	w.AddOutput(ctx, rk.Address, 0)
	w.AddContractFee(ctx, ct.ContractFee)

	// Save Tx for response.
	if err := transactions.AddTx(ctx, g.MasterDB, itx); err != nil {
		return errors.Wrap(err, "Failed to save tx")
	}

	// Respond with a vote
	return node.RespondSuccess(ctx, w, itx, rk, &voteResult)
}

// ResultResponse handles an outgoing Result action and writes it to the state
func (g *Governance) ResultResponse(ctx context.Context, w *node.ResponseWriter, itx *inspector.Transaction, rk *wallet.Key) error {
	ctx, span := trace.StartSpan(ctx, "handlers.Governance.ResultResponse")
	defer span.End()

	msg, ok := itx.MsgProto.(*actions.Result)
	if !ok {
		return errors.New("Could not assert as *actions.Result")
	}

	v := ctx.Value(node.KeyValues).(*node.Values)

	if itx.RejectCode != 0 {
		return errors.New("Result reponse invalid")
	}

	if !itx.Inputs[0].Address.Equal(rk.Address) {
		address := bitcoin.NewAddressFromRawAddress(itx.Inputs[0].Address,
			wire.BitcoinNet(w.Config.ChainParams.Net))
		return fmt.Errorf("Vote result not from contract : %x", address.String())
	}

	ct, err := contract.Retrieve(ctx, g.MasterDB, rk.Address)
	if err != nil {
		return errors.Wrap(err, "Failed to retrieve contract")
	}

	if ct.MovedTo != nil {
		address := bitcoin.NewAddressFromRawAddress(ct.MovedTo,
			wire.BitcoinNet(w.Config.ChainParams.Net))
		return fmt.Errorf("Contract address changed : %s", address.String())
	}

	uv := vote.UpdateVote{}
	err = node.Convert(ctx, msg, &uv)
	if err != nil {
		return errors.Wrap(err, "Failed to convert result message to update vote")
	}

	ts := protocol.NewTimestamp(msg.Timestamp)
	uv.CompletedAt = &ts

	voteTxId := protocol.TxIdFromBytes(msg.VoteTxId)
	if err := vote.Update(ctx, g.MasterDB, rk.Address, voteTxId, &uv, v.Now); err != nil {
		return errors.Wrap(err, "Failed to update vote")
	}

	if msg.Specific {
		// Save result for amendment action
		if err := transactions.AddTx(ctx, g.MasterDB, itx); err != nil {
			return errors.Wrap(err, "Failed to save tx")
		}
	}

	return nil
}
