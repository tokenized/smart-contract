package handlers

import (
	"bytes"
	"context"
	"fmt"

	"github.com/tokenized/smart-contract/cmd/smartcontractd/listeners"
	"github.com/tokenized/smart-contract/internal/asset"
	"github.com/tokenized/smart-contract/internal/contract"
	"github.com/tokenized/smart-contract/internal/platform/db"
	"github.com/tokenized/smart-contract/internal/platform/node"
	"github.com/tokenized/smart-contract/internal/platform/state"
	"github.com/tokenized/smart-contract/internal/platform/wallet"
	"github.com/tokenized/smart-contract/internal/transactions"
	"github.com/tokenized/smart-contract/internal/vote"
	"github.com/tokenized/smart-contract/pkg/inspector"
	"github.com/tokenized/smart-contract/pkg/logger"
	"github.com/tokenized/smart-contract/pkg/protocol"
	"github.com/tokenized/smart-contract/pkg/scheduler"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcutil"
	"github.com/pkg/errors"
	"go.opencensus.io/trace"
)

type Governance struct {
	handler   *node.App
	MasterDB  *db.DB
	Config    *node.Config
	Scheduler *scheduler.Scheduler
}

// ProposalRequest handles an incoming proposal request and prepares a Vote response
func (g *Governance) ProposalRequest(ctx context.Context, w *node.ResponseWriter, itx *inspector.Transaction, rk *wallet.RootKey) error {
	ctx, span := trace.StartSpan(ctx, "handlers.Governance.ProposalRequest")
	defer span.End()

	msg, ok := itx.MsgProto.(*protocol.Proposal)
	if !ok {
		return errors.New("Could not assert as *protocol.Initiative")
	}

	v := ctx.Value(node.KeyValues).(*node.Values)

	// Validate all fields have valid values.
	if err := msg.Validate(); err != nil {
		logger.Warn(ctx, "%s : Proposal request invalid : %s", v.TraceID, err)
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectMsgMalformed)
	}

	// Locate Contract
	contractPKH := protocol.PublicKeyHashFromBytes(rk.Address.ScriptAddress())
	ct, err := contract.Retrieve(ctx, g.MasterDB, contractPKH)
	if err != nil {
		return errors.Wrap(err, "Failed to retrieve contract")
	}

	if ct.FreezePeriod.Nano() > v.Now.Nano() {
		logger.Warn(ctx, "%s : Proposal failed. Contract frozen : %s", v.TraceID, contractPKH.String())
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectContractFrozen)
	}

	// Verify first two outputs are to contract
	if len(itx.Outputs) < 2 || !bytes.Equal(itx.Outputs[0].Address.ScriptAddress(), contractPKH.Bytes()) ||
		!bytes.Equal(itx.Outputs[1].Address.ScriptAddress(), contractPKH.Bytes()) {
		logger.Warn(ctx, "%s : Proposal failed to fund vote and result txs : %s", v.TraceID, contractPKH.String())
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectInsufficientTxFeeFunding)
	}

	senderPKH := protocol.PublicKeyHashFromBytes(itx.Inputs[0].Address.ScriptAddress())

	// Check if sender is allowed to make proposal
	if msg.Initiator == 0 { // Issuer Proposal
		if !contract.IsOperator(ctx, ct, senderPKH) {
			logger.Warn(ctx, "%s : Initiator PKH is not issuer or operator : %s", v.TraceID, contractPKH.String())
			return node.RespondReject(ctx, w, itx, rk, protocol.RejectNotOperator)
		}
	} else if msg.Initiator == 1 { // Holder Proposal
		// Sender must hold balance of at least one asset
		if !contract.HasAnyBalance(ctx, g.MasterDB, ct, senderPKH) {
			logger.Warn(ctx, "%s : Sender holds no assets : %s %s", v.TraceID, contractPKH.String(), senderPKH.String())
			return node.RespondReject(ctx, w, itx, rk, protocol.RejectInsufficientQuantity)
		}
	} else {
		logger.Warn(ctx, "%s : Invalid Initiator value : %02x", v.TraceID, msg.Initiator)
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectMsgMalformed)
	}

	if int(msg.VoteSystem) >= len(ct.VotingSystems) {
		logger.Warn(ctx, "%s : Proposal vote system invalid", v.TraceID)
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectMsgMalformed)
	}

	if msg.Specific && ct.VotingSystems[msg.VoteSystem].VoteType == byte('P') {
		logger.Warn(ctx, "%s : Plurality votes not allowed for specific votes", v.TraceID)
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectVoteSystemNotPermitted)
	}

	// Validate messages vote related values
	if !vote.ValidateProposal(msg, v.Now) {
		logger.Warn(ctx, "%s : Proposal validation failed : %s %s", v.TraceID, contractPKH, senderPKH.String())
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectMsgMalformed)
	}

	if msg.AssetSpecificVote {
		as, err := asset.Retrieve(ctx, g.MasterDB, contractPKH, &msg.AssetCode)
		if err != nil {
			logger.Warn(ctx, "%s : Asset not found : %s %s", v.TraceID, contractPKH.String(), msg.AssetCode.String())
			return node.RespondReject(ctx, w, itx, rk, protocol.RejectAssetNotFound)
		}

		if as.FreezePeriod.Nano() > v.Now.Nano() {
			logger.Warn(ctx, "%s : Proposal failed. Asset frozen : %s %s", v.TraceID, contractPKH.String(), msg.AssetCode.String())
			return node.RespondReject(ctx, w, itx, rk, protocol.RejectAssetFrozen)
		}

		// Asset does not allow voting
		if err := asset.ValidateVoting(ctx, as, msg.Initiator, &ct.VotingSystems[msg.VoteSystem]); err != nil {
			logger.Warn(ctx, "%s : Asset does not allow voting: %s %s : %s", v.TraceID, contractPKH.String(), msg.AssetCode.String(), err)
			return node.RespondReject(ctx, w, itx, rk, protocol.RejectAssetAuthFlags)
		}

		// Sender does not have any balance of the asset
		if msg.Initiator > 0 && asset.GetVotingBalance(ctx, as, senderPKH, ct.VotingSystems[msg.VoteSystem].VoteMultiplierPermitted, v.Now) == 0 {
			logger.Warn(ctx, "%s : Insufficient funds: %s %s", v.TraceID, contractPKH.String(), msg.AssetCode.String())
			return node.RespondReject(ctx, w, itx, rk, protocol.RejectInsufficientQuantity)
		}

		if msg.Specific {
			// Asset amendments vote. Validate permissions for fields being amended.
			if err := checkAssetAmendmentsPermissions(as, ct.VotingSystems, msg.ProposedAmendments, true, msg.Initiator, msg.VoteSystem); err != nil {
				logger.Warn(ctx, "%s : Asset amendments not permitted : %s", v.TraceID, err)
				return node.RespondReject(ctx, w, itx, rk, protocol.RejectAssetAuthFlags)
			}

			if msg.VoteOptions != "AB" || msg.VoteMax != 1 {
				logger.Warn(ctx, "%s : Single option AB votes are required for specific amendments", v.TraceID)
				return node.RespondReject(ctx, w, itx, rk, protocol.RejectMsgMalformed)
			}

			// Validate proposed amendments.
			ac := protocol.AssetCreation{}

			err = node.Convert(ctx, &as, &ac)
			if err != nil {
				return errors.Wrap(err, "Failed to convert state asset to asset creation")
			}

			ac.AssetRevision = as.Revision + 1
			ac.Timestamp = v.Now

			if err := applyAssetAmendments(&ac, ct.VotingSystems, msg.ProposedAmendments); err != nil {
				logger.Warn(ctx, "%s : Asset amendments failed : %s", v.TraceID, err)
				return node.RespondReject(ctx, w, itx, rk, protocol.RejectMsgMalformed)
			}
		}
	} else {
		// Contract does not allow voting
		if err := contract.ValidateVoting(ctx, ct, msg.Initiator); err != nil {
			logger.Warn(ctx, "%s : Contract does not allow voting: %s : %s", v.TraceID, contractPKH.String(), err)
			return node.RespondReject(ctx, w, itx, rk, protocol.RejectContractAuthFlags)
		}

		if msg.Specific {
			// Contract amendments vote. Validate permissions for fields being amended.
			if err := checkContractAmendmentsPermissions(ct, msg.ProposedAmendments, true, msg.Initiator, msg.VoteSystem); err != nil {
				logger.Warn(ctx, "%s : Asset amendments not permitted : %s", v.TraceID, err)
				return node.RespondReject(ctx, w, itx, rk, protocol.RejectContractAuthFlags)
			}

			if msg.VoteOptions != "AB" || msg.VoteMax != 1 {
				logger.Warn(ctx, "%s : Single option AB votes are required for specific amendments", v.TraceID)
				return node.RespondReject(ctx, w, itx, rk, protocol.RejectMsgMalformed)
			}

			// Validate proposed amendments.
			cf := protocol.ContractFormation{}

			// Get current state
			err = node.Convert(ctx, &ct, &cf)
			if err != nil {
				return errors.Wrap(err, "Failed to convert state contract to contract formation")
			}

			// Apply modifications
			cf.ContractRevision = ct.Revision + 1 // Bump the revision
			cf.Timestamp = v.Now

			if err := applyContractAmendments(&cf, msg.ProposedAmendments); err != nil {
				logger.Warn(ctx, "%s : Contract amendments failed : %s", v.TraceID, err)
				return node.RespondReject(ctx, w, itx, rk, protocol.RejectMsgMalformed)
			}
		}

		// Sender does not have any balance of the asset
		if msg.Initiator > 0 && contract.GetVotingBalance(ctx, g.MasterDB, ct, senderPKH, ct.VotingSystems[msg.VoteSystem].VoteMultiplierPermitted, v.Now) == 0 {
			logger.Warn(ctx, "%s : Insufficient funds: %s %s", v.TraceID, contractPKH.String(), msg.AssetCode.String())
			return node.RespondReject(ctx, w, itx, rk, protocol.RejectInsufficientQuantity)
		}
	}

	if msg.Specific {
		// Check existing votes that have not been applied yet for conflicting fields.
		votes, err := vote.List(ctx, g.MasterDB, contractPKH)
		if err != nil {
			return errors.Wrap(err, "Failed to list votes")
		}
		for _, vt := range votes {
			if !vt.AppliedTxId.IsZero() || !vt.Specific {
				continue // Already applied or doesn't contain specific amendments
			}

			if msg.AssetSpecificVote {
				if !vt.AssetSpecificVote || msg.AssetType != vt.AssetType || !bytes.Equal(msg.AssetCode.Bytes(), vt.AssetCode.Bytes()) {
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
						logger.Warn(ctx, "%s : Proposed amendment conflicts with unapplied vote : %s", v.TraceID, contractPKH.String())
						return node.RespondReject(ctx, w, itx, rk, protocol.RejectProposalConflicts)
					}
				}
			}
		}
	}

	// Build Response
	vote := protocol.Vote{Timestamp: v.Now}

	// Convert to btcutil.Address
	contractAddress, err := btcutil.NewAddressPubKeyHash(contractPKH.Bytes(), &g.Config.ChainParams)
	if err != nil {
		return err
	}

	// Fund with first output of proposal tx. Second is reserved for vote result tx.
	w.SetUTXOs(ctx, []inspector.UTXO{itx.Outputs[0].UTXO})

	// Build outputs
	// 1 - Contract Address
	// 2 - Contract/Proposal Fee (change)
	w.AddOutput(ctx, contractAddress, 0)

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
	return node.RespondSuccess(ctx, w, itx, rk, &vote)
}

// VoteResponse handles an incoming Vote response
func (g *Governance) VoteResponse(ctx context.Context, w *node.ResponseWriter, itx *inspector.Transaction, rk *wallet.RootKey) error {
	ctx, span := trace.StartSpan(ctx, "handlers.Governance.VoteResponse")
	defer span.End()

	msg, ok := itx.MsgProto.(*protocol.Vote)
	if !ok {
		return errors.New("Could not assert as *protocol.Vote")
	}

	v := ctx.Value(node.KeyValues).(*node.Values)

	contractPKH := protocol.PublicKeyHashFromBytes(rk.Address.ScriptAddress())

	ct, err := contract.Retrieve(ctx, g.MasterDB, contractPKH)
	if err != nil {
		return err
	}

	// Verify input is from contract
	if !bytes.Equal(itx.Inputs[0].Address.ScriptAddress(), contractPKH.Bytes()) {
		return fmt.Errorf("Response not from contract")
	}

	// Retrieve Proposal
	proposalTx, err := transactions.GetTx(ctx, g.MasterDB, &itx.Inputs[0].UTXO.Hash, &g.Config.ChainParams)
	if err != nil {
		logger.Warn(ctx, "%s : Proposal not found for vote : %s", v.TraceID, contractPKH.String())
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectMsgMalformed)
	}

	proposal, ok := proposalTx.MsgProto.(*protocol.Proposal)
	if !ok {
		logger.Warn(ctx, "%s : Proposal invalid for vote : %s", v.TraceID, contractPKH.String())
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectMsgMalformed)
	}

	voteTxId := protocol.TxIdFromBytes(itx.Hash[:])

	_, err = vote.Retrieve(ctx, g.MasterDB, contractPKH, voteTxId)
	if err != vote.ErrNotFound {
		if err != nil {
			logger.Warn(ctx, "%s : Failed to retrieve vote : %s %s : %s", v.TraceID, contractPKH.String(), voteTxId.String(), err)
		} else {
			logger.Warn(ctx, "%s : Vote already exists : %s %s", v.TraceID, contractPKH.String(), voteTxId.String())
		}
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectMsgMalformed)
	}

	nv := vote.NewVote{}
	err = node.Convert(ctx, proposal, &nv)
	if err != nil {
		return errors.Wrap(err, "Failed to convert vote message to new vote")
	}

	nv.VoteTxId = *voteTxId
	nv.ProposalTxId = *protocol.TxIdFromBytes(proposalTx.Hash[:])
	nv.Expires = proposal.VoteCutOffTimestamp
	nv.Timestamp = msg.Timestamp

	if proposal.AssetSpecificVote {
		as, err := asset.Retrieve(ctx, g.MasterDB, contractPKH, &proposal.AssetCode)
		if err != nil {
			logger.Warn(ctx, "%s : Asset not found : %s %s", v.TraceID, contractPKH.String(), proposal.AssetCode.String())
			return node.RespondReject(ctx, w, itx, rk, protocol.RejectAssetNotFound)
		}

		if as.AssetModificationGovernance == 1 { // Contract wide asset governance
			nv.ContractWideVote = true
			nv.TokenQty = contract.GetTokenQty(ctx, g.MasterDB, ct, ct.VotingSystems[proposal.VoteSystem].VoteMultiplierPermitted)
		} else if ct.VotingSystems[proposal.VoteSystem].VoteMultiplierPermitted {
			nv.TokenQty = as.TokenQty * uint64(as.VoteMultiplier)
		} else {
			nv.TokenQty = as.TokenQty
		}
	} else {
		nv.TokenQty = contract.GetTokenQty(ctx, g.MasterDB, ct, ct.VotingSystems[proposal.VoteSystem].VoteMultiplierPermitted)
	}

	if err := vote.Create(ctx, g.MasterDB, contractPKH, voteTxId, &nv, v.Now); err != nil {
		return errors.Wrap(err, "Failed to save vote")
	}

	if err := g.Scheduler.ScheduleJob(ctx, listeners.NewVoteFinalizer(g.handler, itx, proposal.VoteCutOffTimestamp)); err != nil {
		return errors.Wrap(err, "Failed to schedule vote finalizer")
	}

	return nil
}

// BallotCastRequest handles an incoming BallotCast request and prepares a BallotCounted response
func (g *Governance) BallotCastRequest(ctx context.Context, w *node.ResponseWriter, itx *inspector.Transaction, rk *wallet.RootKey) error {
	ctx, span := trace.StartSpan(ctx, "handlers.Governance.BallotCastRequest")
	defer span.End()

	msg, ok := itx.MsgProto.(*protocol.BallotCast)
	if !ok {
		return errors.New("Could not assert as *protocol.BallotCast")
	}

	v := ctx.Value(node.KeyValues).(*node.Values)

	// Validate all fields have valid values.
	if err := msg.Validate(); err != nil {
		logger.Warn(ctx, "%s : Ballot cast request invalid : %s", v.TraceID, err)
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectMsgMalformed)
	}

	contractPKH := protocol.PublicKeyHashFromBytes(rk.Address.ScriptAddress())

	ct, err := contract.Retrieve(ctx, g.MasterDB, contractPKH)
	if err != nil {
		return err
	}

	vt, err := vote.Retrieve(ctx, g.MasterDB, contractPKH, &msg.VoteTxId)
	if err == vote.ErrNotFound {
		logger.Warn(ctx, "%s : Vote not found : %s %s", v.TraceID, contractPKH.String(), msg.VoteTxId.String())
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectVoteNotFound)
	} else if err != nil {
		logger.Warn(ctx, "%s : Failed to retrieve vote : %s %s : %s", v.TraceID, contractPKH.String(), msg.VoteTxId.String(), err)
		return errors.Wrap(err, "Failed to retrieve vote")
	}

	if vt.Expires.Nano() <= v.Now.Nano() {
		logger.Warn(ctx, "%s : Vote expired : %s %s", v.TraceID, contractPKH.String(), msg.VoteTxId.String())
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectVoteClosed)
	}

	// Get Proposal
	hash, err := chainhash.NewHash(vt.ProposalTxId.Bytes())
	proposalTx, err := transactions.GetTx(ctx, g.MasterDB, hash, &g.Config.ChainParams)
	if err != nil {
		logger.Warn(ctx, "%s : Proposal not found for vote : %s", v.TraceID, contractPKH.String())
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectMsgMalformed)
	}

	proposal, ok := proposalTx.MsgProto.(*protocol.Proposal)
	if !ok {
		logger.Warn(ctx, "%s : Proposal invalid for vote : %s", v.TraceID, contractPKH.String())
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectMsgMalformed)
	}

	// Validate vote
	if len(msg.Vote) > int(proposal.VoteMax) {
		logger.Warn(ctx, "%s : Ballot voted on too many options : %s", v.TraceID, contractPKH.String())
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectMsgMalformed)
	}

	if len(msg.Vote) == 0 {
		logger.Warn(ctx, "%s : Ballot did not vote any options : %s", v.TraceID, contractPKH.String())
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectMsgMalformed)
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
			logger.Warn(ctx, "%s : Ballot chose an invalid option : %s", v.TraceID, contractPKH.String())
			return node.RespondReject(ctx, w, itx, rk, protocol.RejectMsgMalformed)
		}
	}

	// TODO Handle transfers during vote time to ensure they don't vote the same tokens more than once.

	holderPKH := protocol.PublicKeyHashFromBytes(itx.Inputs[0].Address.ScriptAddress())
	quantity := uint64(0)

	// Add applicable holdings
	if proposal.AssetSpecificVote && !vt.ContractWideVote {
		as, err := asset.Retrieve(ctx, g.MasterDB, contractPKH, &proposal.AssetCode)
		if err != nil {
			logger.Warn(ctx, "%s : Asset not found : %s %s", v.TraceID, contractPKH.String(), proposal.String())
			return node.RespondReject(ctx, w, itx, rk, protocol.RejectAssetNotFound)
		}

		quantity = asset.GetVotingBalance(ctx, as, holderPKH, ct.VotingSystems[proposal.VoteSystem].VoteMultiplierPermitted, v.Now)
	} else {
		quantity = contract.GetVotingBalance(ctx, g.MasterDB, ct, holderPKH, ct.VotingSystems[proposal.VoteSystem].VoteMultiplierPermitted, v.Now)
	}

	if quantity == 0 {
		logger.Warn(ctx, "%s : User PKH doesn't hold any voting tokens : %s %s", v.TraceID, contractPKH.String(), holderPKH.String())
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectInsufficientQuantity)
	}

	// TODO Check issue where two ballots are sent simultaneously and the second received before the first response is processed.
	if err := vote.CheckBallot(ctx, vt, holderPKH); err != nil {
		logger.Warn(ctx, "%s : Failed to check ballot : %s", v.TraceID, err)
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectBallotAlreadyCounted)
	}

	// Build Response
	ballotCounted := protocol.BallotCounted{}
	err = node.Convert(ctx, msg, &ballotCounted)
	if err != nil {
		return errors.Wrap(err, "Failed to convert ballot cast to counted")
	}
	ballotCounted.Quantity = quantity
	ballotCounted.Timestamp = v.Now

	// Convert to btcutil.Address
	contractAddress, err := btcutil.NewAddressPubKeyHash(contractPKH.Bytes(), &g.Config.ChainParams)
	if err != nil {
		return err
	}

	// Build outputs
	// 1 - Contract Address
	// 2 - Contract Fee (change)
	w.AddOutput(ctx, contractAddress, 0)
	w.AddContractFee(ctx, ct.ContractFee)

	// Save Tx for response.
	if err := transactions.AddTx(ctx, g.MasterDB, itx); err != nil {
		return errors.Wrap(err, "Failed to add tx")
	}

	// Respond with a vote
	return node.RespondSuccess(ctx, w, itx, rk, &ballotCounted)
}

// BallotCountedResponse handles an outgoing BallotCounted action and writes it to the state
func (g *Governance) BallotCountedResponse(ctx context.Context, w *node.ResponseWriter, itx *inspector.Transaction, rk *wallet.RootKey) error {
	ctx, span := trace.StartSpan(ctx, "handlers.Governance.BallotCountedResponse")
	defer span.End()

	msg, ok := itx.MsgProto.(*protocol.BallotCounted)
	if !ok {
		return errors.New("Could not assert as *protocol.BallotCounted")
	}

	v := ctx.Value(node.KeyValues).(*node.Values)

	contractPKH := protocol.PublicKeyHashFromBytes(rk.Address.ScriptAddress())
	if !bytes.Equal(itx.Inputs[0].Address.ScriptAddress(), contractPKH.Bytes()) {
		return fmt.Errorf("Ballot counted not from contract : %x", itx.Inputs[0].Address.ScriptAddress())
	}

	castTx, err := transactions.GetTx(ctx, g.MasterDB, &itx.Inputs[0].UTXO.Hash, &g.Config.ChainParams)
	if err != nil {
		logger.Warn(ctx, "%s : Ballot cast not found for ballot counted msg : %s", v.TraceID, contractPKH.String())
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectMsgMalformed)
	}

	cast, ok := castTx.MsgProto.(*protocol.BallotCast)
	if !ok {
		logger.Warn(ctx, "%s : Ballot cast invalid for ballot counted : %s", v.TraceID, contractPKH.String())
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectMsgMalformed)
	}

	vt, err := vote.Retrieve(ctx, g.MasterDB, contractPKH, &cast.VoteTxId)
	if err != nil {
		return errors.Wrap(err, "Failed to retrieve vote for ballot cast")
	}

	holderPKH := protocol.PublicKeyHashFromBytes(castTx.Inputs[0].Address.ScriptAddress())

	ballot := state.Ballot{
		PKH:       *holderPKH,
		Vote:      cast.Vote,
		Timestamp: msg.Timestamp,
		Quantity:  msg.Quantity,
	}

	// Add to vote results
	if err := vote.AddBallot(ctx, g.MasterDB, contractPKH, vt, &ballot, v.Now); err != nil {
		return errors.Wrap(err, "Failed to add ballot")
	}

	return nil
}

// FinalizeVote is called when a vote expires and sends the result response.
func (g *Governance) FinalizeVote(ctx context.Context, w *node.ResponseWriter, itx *inspector.Transaction, rk *wallet.RootKey) error {
	ctx, span := trace.StartSpan(ctx, "handlers.Governance.FinalizeVote")
	defer span.End()

	_, ok := itx.MsgProto.(*protocol.Vote)
	if !ok {
		return errors.New("Could not assert as *protocol.Vote")
	}

	v := ctx.Value(node.KeyValues).(*node.Values)

	contractPKH := protocol.PublicKeyHashFromBytes(rk.Address.ScriptAddress())

	// Retrieve contract
	ct, err := contract.Retrieve(ctx, g.MasterDB, contractPKH)
	if err != nil {
		return err
	}

	// Retrieve vote
	voteTxId := protocol.TxIdFromBytes(itx.Hash[:])
	vt, err := vote.Retrieve(ctx, g.MasterDB, contractPKH, voteTxId)
	if err != nil {
		return errors.Wrap(err, "Failed to retrieve vote for ballot cast")
	}

	// Get Proposal
	hash, err := chainhash.NewHash(vt.ProposalTxId.Bytes())
	proposalTx, err := transactions.GetTx(ctx, g.MasterDB, hash, &g.Config.ChainParams)
	if err != nil {
		logger.Warn(ctx, "%s : Proposal not found for vote : %s", v.TraceID, contractPKH.String())
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectMsgMalformed)
	}

	proposal, ok := proposalTx.MsgProto.(*protocol.Proposal)
	if !ok {
		logger.Warn(ctx, "%s : Proposal invalid for vote : %s", v.TraceID, contractPKH.String())
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectMsgMalformed)
	}

	// Build Response
	voteResult := protocol.Result{}
	err = node.Convert(ctx, proposal, &voteResult)
	if err != nil {
		return errors.Wrap(err, "Failed to convert vote proposal to result")
	}

	voteResult.VoteTxId = *voteTxId
	voteResult.Timestamp = v.Now

	// Calculate Results
	voteResult.OptionTally, voteResult.Result, err = vote.CalculateResults(ctx, vt, proposal, &ct.VotingSystems[proposal.VoteSystem])
	if err != nil {
		return errors.Wrap(err, "Failed to calculate vote results")
	}

	// Convert to btcutil.Address
	contractAddress, err := btcutil.NewAddressPubKeyHash(contractPKH.Bytes(), &g.Config.ChainParams)
	if err != nil {
		return err
	}

	// Fund with second output of proposal tx.
	w.SetUTXOs(ctx, []inspector.UTXO{proposalTx.Outputs[1].UTXO})

	// Build outputs
	// 1 - Contract Address
	// 2 - Contract Fee (change)
	w.AddOutput(ctx, contractAddress, 0)
	w.AddContractFee(ctx, ct.ContractFee)

	// Save Tx for response.
	if err := transactions.AddTx(ctx, g.MasterDB, itx); err != nil {
		return errors.Wrap(err, "Failed to save tx")
	}

	// Respond with a vote
	return node.RespondSuccess(ctx, w, itx, rk, &voteResult)
}

// ResultResponse handles an outgoing Result action and writes it to the state
func (g *Governance) ResultResponse(ctx context.Context, w *node.ResponseWriter, itx *inspector.Transaction, rk *wallet.RootKey) error {
	ctx, span := trace.StartSpan(ctx, "handlers.Governance.ResultResponse")
	defer span.End()

	msg, ok := itx.MsgProto.(*protocol.Result)
	if !ok {
		return errors.New("Could not assert as *protocol.Result")
	}

	v := ctx.Value(node.KeyValues).(*node.Values)

	contractPKH := protocol.PublicKeyHashFromBytes(rk.Address.ScriptAddress())
	if !bytes.Equal(itx.Inputs[0].Address.ScriptAddress(), contractPKH.Bytes()) {
		return fmt.Errorf("Vote result not from contract : %x", itx.Inputs[0].Address.ScriptAddress())
	}

	uv := vote.UpdateVote{}
	err := node.Convert(ctx, msg, &uv)
	if err != nil {
		return errors.Wrap(err, "Failed to convert result message to update vote")
	}

	uv.CompletedAt = &msg.Timestamp

	if err := vote.Update(ctx, g.MasterDB, contractPKH, &msg.VoteTxId, &uv, v.Now); err != nil {
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
