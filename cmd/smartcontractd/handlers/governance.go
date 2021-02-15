package handlers

import (
	"bytes"
	"context"
	"fmt"

	"github.com/tokenized/pkg/bitcoin"
	"github.com/tokenized/pkg/scheduler"
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
	"github.com/tokenized/smart-contract/pkg/inspector"
	"github.com/tokenized/smart-contract/pkg/wallet"
	"github.com/tokenized/specification/dist/golang/actions"
	"github.com/tokenized/specification/dist/golang/protocol"

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
func (g *Governance) ProposalRequest(ctx context.Context, w *node.ResponseWriter,
	itx *inspector.Transaction, rk *wallet.Key) error {
	ctx, span := trace.StartSpan(ctx, "handlers.Governance.ProposalRequest")
	defer span.End()

	msg, ok := itx.MsgProto.(*actions.Proposal)
	if !ok {
		return errors.New("Could not assert as *actions.Initiative")
	}

	v := ctx.Value(node.KeyValues).(*node.Values)

	// Validate all fields have valid values.
	if itx.RejectCode != 0 {
		node.LogWarn(ctx, "Proposal invalid : %d %s", itx.RejectCode, itx.RejectText)
		return node.RespondRejectText(ctx, w, itx, rk, itx.RejectCode, itx.RejectText)
	}

	// Locate Contract
	ct, err := contract.Retrieve(ctx, g.MasterDB, rk.Address, g.Config.IsTest)
	if err != nil {
		return errors.Wrap(err, "Failed to retrieve contract")
	}

	if !ct.MovedTo.IsEmpty() {
		address := bitcoin.NewAddressFromRawAddress(ct.MovedTo,
			w.Config.Net)
		node.LogWarn(ctx, "Contract address changed : %s", address.String())
		return node.RespondReject(ctx, w, itx, rk, actions.RejectionsContractMoved)
	}

	if ct.FreezePeriod.Nano() > v.Now.Nano() {
		node.LogWarn(ctx, "Contract frozen")
		return node.RespondReject(ctx, w, itx, rk, actions.RejectionsContractFrozen)
	}

	if ct.ContractExpiration.Nano() != 0 && ct.ContractExpiration.Nano() < v.Now.Nano() {
		node.LogWarn(ctx, "Contract expired : %s", ct.ContractExpiration.String())
		return node.RespondReject(ctx, w, itx, rk, actions.RejectionsContractExpired)
	}

	// Verify first two outputs are to contract
	if len(itx.Outputs) < 2 || !itx.Outputs[0].Address.Equal(rk.Address) ||
		!itx.Outputs[1].Address.Equal(rk.Address) {
		node.LogWarn(ctx, "Proposal failed to fund vote and result txs")
		return node.RespondReject(ctx, w, itx, rk, actions.RejectionsInsufficientTxFeeFunding)
	}

	// Check if sender is allowed to make proposal
	if msg.Type == 0 { // Administration Proposal
		if !contract.IsOperator(ctx, ct, itx.Inputs[0].Address) {
			node.LogWarn(ctx, "Initiator is not administration or operator")
			return node.RespondReject(ctx, w, itx, rk, actions.RejectionsNotOperator)
		}
	} else if msg.Type == 1 { // Holder Proposal
		// Sender must hold balance of at least one asset
		if !contract.HasAnyBalance(ctx, g.MasterDB, ct, itx.Inputs[0].Address) {
			address := bitcoin.NewAddressFromRawAddress(itx.Inputs[0].Address,
				w.Config.Net)
			node.LogWarn(ctx, "Sender holds no assets : %s", address.String())
			return node.RespondReject(ctx, w, itx, rk, actions.RejectionsInsufficientQuantity)
		}
	} else if msg.Type == 2 { // Administrative Matter
		if !contract.IsOperator(ctx, ct, itx.Inputs[0].Address) {
			node.LogWarn(ctx, "Initiator is not administration or operator")
			return node.RespondReject(ctx, w, itx, rk, actions.RejectionsNotOperator)
		}
	} else {
		node.LogWarn(ctx, "Invalid Initiator value : %02x", msg.Type)
		return node.RespondReject(ctx, w, itx, rk, actions.RejectionsMsgMalformed)
	}

	if int(msg.VoteSystem) >= len(ct.VotingSystems) {
		node.LogWarn(ctx, "Proposal vote system invalid")
		return node.RespondReject(ctx, w, itx, rk, actions.RejectionsMsgMalformed)
	}

	if len(msg.ProposedAmendments) > 0 && ct.VotingSystems[msg.VoteSystem].VoteType == "P" {
		node.LogWarn(ctx, "Plurality votes not allowed for specific votes")
		return node.RespondReject(ctx, w, itx, rk, actions.RejectionsVoteSystemNotPermitted)
	}

	// Validate messages vote related values
	if err := vote.ValidateProposal(msg, v.Now); err != nil {
		node.LogWarn(ctx, "Proposal validation failed : %s", err)
		return node.RespondReject(ctx, w, itx, rk, actions.RejectionsMsgMalformed)
	}

	if len(msg.AssetCode) > 0 {
		assetCode, err := bitcoin.NewHash20(msg.AssetCode)
		if err != nil {
			node.LogVerbose(ctx, "Invalid asset code : 0x%x", msg.AssetCode)
			return node.RespondReject(ctx, w, itx, rk, actions.RejectionsMsgMalformed)
		}

		as, err := asset.Retrieve(ctx, g.MasterDB, rk.Address, assetCode)
		if err != nil {
			node.LogWarn(ctx, "Asset not found : %s", assetCode)
			return node.RespondReject(ctx, w, itx, rk, actions.RejectionsAssetNotFound)
		}

		if as.FreezePeriod.Nano() > v.Now.Nano() {
			node.LogWarn(ctx, "Proposal failed. Asset frozen : %s", assetCode)
			return node.RespondReject(ctx, w, itx, rk, actions.RejectionsAssetFrozen)
		}

		// Asset does not allow voting
		if err := asset.ValidateVoting(ctx, as, msg.Type, ct.VotingSystems[msg.VoteSystem]); err != nil {
			node.LogWarn(ctx, "Asset does not allow voting: %s : %s", assetCode, err)
			return node.RespondReject(ctx, w, itx, rk, actions.RejectionsAssetPermissions)
		}

		// Check sender balance
		h, err := holdings.GetHolding(ctx, g.MasterDB, rk.Address, assetCode, itx.Inputs[0].Address,
			v.Now)
		if err != nil {
			return errors.Wrap(err, "Failed to get requestor holding")
		}

		if msg.Type == 1 && holdings.VotingBalance(as, h,
			ct.VotingSystems[msg.VoteSystem].VoteMultiplierPermitted, v.Now) == 0 {
			address := bitcoin.NewAddressFromRawAddress(itx.Inputs[0].Address,
				w.Config.Net)
			node.LogWarn(ctx, "Requestor is not a holder : %s %s", assetCode, address)
			return node.RespondReject(ctx, w, itx, rk, actions.RejectionsInsufficientQuantity)
		}

		if len(msg.ProposedAmendments) > 0 {
			if msg.VoteOptions != "AB" || msg.VoteMax != 1 {
				node.LogWarn(ctx, "Single option AB votes are required for specific amendments")
				return node.RespondReject(ctx, w, itx, rk, actions.RejectionsMsgMalformed)
			}

			// Validate proposed amendments.
			ac := actions.AssetCreation{}

			err = node.Convert(ctx, &as, &ac)
			if err != nil {
				return errors.Wrap(err, "Failed to convert state asset to asset creation")
			}

			ac.AssetRevision = as.Revision + 1
			ac.Timestamp = v.Now.Nano()

			// Verify that included amendments are valid and have necessary permission.
			if err := applyAssetAmendments(&ac, ct.VotingSystems, msg.ProposedAmendments, true,
				msg.Type, msg.VoteSystem); err != nil {
				node.LogWarn(ctx, "Asset amendments failed : %s", err)
				code, ok := node.ErrorCode(err)
				if ok {
					return node.RespondReject(ctx, w, itx, rk, code)
				}
				return node.RespondReject(ctx, w, itx, rk, actions.RejectionsMsgMalformed)
			}
		}
	} else {
		// Contract does not allow voting
		if err := contract.ValidateVoting(ctx, ct, msg.Type); err != nil {
			node.LogWarn(ctx, "Contract does not allow voting : %s", err)
			return node.RespondReject(ctx, w, itx, rk, actions.RejectionsContractPermissions)
		}

		if len(msg.ProposedAmendments) > 0 {
			if msg.VoteOptions != "AB" || msg.VoteMax != 1 {
				node.LogWarn(ctx, "Single option AB votes are required for specific amendments")
				return node.RespondReject(ctx, w, itx, rk, actions.RejectionsMsgMalformed)
			}

			// Get current state
			cf, err := contract.FetchContractFormation(ctx, g.MasterDB, rk.Address, g.Config.IsTest)
			if err != nil {
				return errors.Wrap(err, "Failed to convert state contract to contract formation")
			}

			// Apply modifications
			cf.ContractRevision = ct.Revision + 1 // Bump the revision
			cf.Timestamp = v.Now.Nano()

			// Verify that included amendments are valid and have necessary permission.
			if err := applyContractAmendments(cf, msg.ProposedAmendments, true, msg.Type,
				msg.VoteSystem); err != nil {
				node.LogWarn(ctx, "Contract amendments failed : %s", err)
				code, ok := node.ErrorCode(err)
				if ok {
					return node.RespondReject(ctx, w, itx, rk, code)
				}
				return node.RespondReject(ctx, w, itx, rk, actions.RejectionsMsgMalformed)
			}
		}

		// Sender does not have any balance of the asset
		if msg.Type == 1 && contract.GetVotingBalance(ctx, g.MasterDB, ct, itx.Inputs[0].Address,
			ct.VotingSystems[msg.VoteSystem].VoteMultiplierPermitted, v.Now) == 0 {
			node.LogWarn(ctx, "Requestor is not a holder")
			return node.RespondReject(ctx, w, itx, rk, actions.RejectionsInsufficientQuantity)
		}
	}

	if len(msg.ProposedAmendments) > 0 {
		// Check existing votes that have not been applied yet for conflicting fields.
		votes, err := vote.List(ctx, g.MasterDB, rk.Address)
		if err != nil {
			return errors.Wrap(err, "Failed to list votes")
		}
		for _, vt := range votes {
			if !vt.AppliedTxId.IsZero() || len(vt.ProposedAmendments) == 0 {
				continue // Already applied or doesn't contain specific amendments
			}

			if len(msg.AssetCode) > 0 {
				if vt.AssetCode.IsZero() || msg.AssetType != vt.AssetType ||
					!bytes.Equal(msg.AssetCode, vt.AssetCode.Bytes()) {
					continue // Not an asset amendment
				}
			} else {
				if !vt.AssetCode.IsZero() {
					continue // Not a contract amendment
				}
			}

			// Determine if any fields conflict
			for _, field := range msg.ProposedAmendments {
				for _, otherField := range vt.ProposedAmendments {
					if bytes.Equal(field.FieldIndexPath, otherField.FieldIndexPath) {
						// Reject because of conflicting field amendment on unapplied vote.
						node.LogWarn(ctx, "Proposed amendment conflicts with unapplied vote")
						return node.RespondReject(ctx, w, itx, rk, actions.RejectionsProposalConflicts)
					}
				}
			}
		}
	}

	// Build Response
	vote := actions.Vote{Timestamp: v.Now.Nano()}

	// Fund with first output of proposal tx. Second is reserved for vote result tx.
	w.SetUTXOs(ctx, []bitcoin.UTXO{itx.Outputs[0].UTXO})

	// Build outputs
	// 1 - Contract Address
	// 2 - Contract/Proposal Fee (change)
	w.AddOutput(ctx, rk.Address, 0)

	feeAmount := ct.ContractFee
	if msg.Type == 1 {
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

	v := ctx.Value(node.KeyValues).(*node.Values)

	ct, err := contract.Retrieve(ctx, g.MasterDB, rk.Address, g.Config.IsTest)
	if err != nil {
		return err
	}

	if !ct.MovedTo.IsEmpty() {
		address := bitcoin.NewAddressFromRawAddress(ct.MovedTo,
			w.Config.Net)
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

	_, err = vote.Retrieve(ctx, g.MasterDB, rk.Address, itx.Hash)
	if err != vote.ErrNotFound {
		if err != nil {
			return fmt.Errorf("Failed to retrieve vote : %s : %s", itx.Hash, err)
		} else {
			return fmt.Errorf("Vote already exists : %s", itx.Hash)
		}
	}

	nv := vote.NewVote{}
	err = node.Convert(ctx, proposal, &nv)
	if err != nil {
		return errors.Wrap(err, "Failed to convert vote message to new vote")
	}

	nv.VoteTxId = *itx.Hash
	nv.ProposalTxId = *proposalTx.Hash
	nv.Expires = protocol.NewTimestamp(proposal.VoteCutOffTimestamp)
	nv.Timestamp = protocol.NewTimestamp(msg.Timestamp)
	nv.Ballots = make(map[bitcoin.Hash20]state.Ballot)

	if len(proposal.AssetCode) > 0 {
		assetCode, err := bitcoin.NewHash20(proposal.AssetCode)
		if err != nil {
			node.LogWarn(ctx, "Invalid asset code : %s", assetCode)
			return node.RespondReject(ctx, w, itx, rk, actions.RejectionsMsgMalformed)
		}

		as, err := asset.Retrieve(ctx, g.MasterDB, rk.Address, assetCode)
		if err != nil {
			return fmt.Errorf("Asset not found : %s", assetCode)
		}

		if as.AssetModificationGovernance == 1 { // Contract wide asset governance
			nv.ContractWideVote = true
			nv.TokenQty = contract.GetTokenQty(ctx, g.MasterDB, ct,
				ct.VotingSystems[proposal.VoteSystem].VoteMultiplierPermitted)
		} else if ct.VotingSystems[proposal.VoteSystem].VoteMultiplierPermitted {
			nv.TokenQty = as.AuthorizedTokenQty * uint64(as.VoteMultiplier)
		} else {
			nv.TokenQty = as.AuthorizedTokenQty
		}
	} else {
		nv.TokenQty = contract.GetTokenQty(ctx, g.MasterDB, ct,
			ct.VotingSystems[proposal.VoteSystem].VoteMultiplierPermitted)
	}

	// Populate nv.Ballots with current holdings that apply to the vote
	if proposal.Type == 2 { // Administrative Token holders only
		if ct.AdminMemberAsset.IsZero() {
			node.LogWarn(ctx, "Admin Member Asset not defined")
			return node.RespondReject(ctx, w, itx, rk, actions.RejectionsAssetNotFound)
		}

		as, err := asset.Retrieve(ctx, g.MasterDB, rk.Address, &ct.AdminMemberAsset)
		if err != nil {
			node.LogWarn(ctx, "Admin Member Asset not found : %s", ct.AdminMemberAsset)
			return node.RespondReject(ctx, w, itx, rk, actions.RejectionsAssetNotFound)
		}

		err = holdings.AppendBallots(ctx, g.MasterDB, rk.Address, as, &nv.Ballots,
			ct.VotingSystems[proposal.VoteSystem].VoteMultiplierPermitted, v.Now)
		if err != nil {
			return errors.Wrap(err, "append ballots")
		}
	} else if len(proposal.AssetCode) > 0 && !nv.ContractWideVote {
		assetCode, err := bitcoin.NewHash20(proposal.AssetCode)
		if err != nil {
			node.LogWarn(ctx, "Invalid asset code : %s", assetCode)
			return node.RespondReject(ctx, w, itx, rk, actions.RejectionsMsgMalformed)
		}

		as, err := asset.Retrieve(ctx, g.MasterDB, rk.Address, assetCode)
		if err != nil {
			node.LogWarn(ctx, "Asset not found : %s", assetCode)
			return node.RespondReject(ctx, w, itx, rk, actions.RejectionsAssetNotFound)
		}

		err = holdings.AppendBallots(ctx, g.MasterDB, rk.Address, as, &nv.Ballots,
			ct.VotingSystems[proposal.VoteSystem].VoteMultiplierPermitted, v.Now)
		if err != nil {
			return errors.Wrap(err, "append ballots")
		}
	} else { // Contract Vote
		for _, a := range ct.AssetCodes {
			if a.Equal(&ct.AdminMemberAsset) {
				continue // Administrative tokens don't count for holder votes.
			}
			as, err := asset.Retrieve(ctx, g.MasterDB, ct.Address, a)
			if err != nil {
				continue
			}

			err = holdings.AppendBallots(ctx, g.MasterDB, rk.Address, as, &nv.Ballots,
				ct.VotingSystems[proposal.VoteSystem].VoteMultiplierPermitted, v.Now)
			if err != nil {
				return errors.Wrap(err, "append ballots")
			}
		}
	}

	if err := vote.Create(ctx, g.MasterDB, rk.Address, itx.Hash, &nv, v.Now); err != nil {
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
		node.LogWarn(ctx, "Ballot cast invalid : %d %s", itx.RejectCode, itx.RejectText)
		return node.RespondRejectText(ctx, w, itx, rk, itx.RejectCode, itx.RejectText)
	}

	ct, err := contract.Retrieve(ctx, g.MasterDB, rk.Address, g.Config.IsTest)
	if err != nil {
		return err
	}

	if !ct.MovedTo.IsEmpty() {
		address := bitcoin.NewAddressFromRawAddress(ct.MovedTo, w.Config.Net)
		node.LogWarn(ctx, "Contract address changed : %s", address)
		return node.RespondReject(ctx, w, itx, rk, actions.RejectionsContractMoved)
	}

	voteTxId, err := bitcoin.NewHash32(msg.VoteTxId)
	if err != nil {
		node.LogWarn(ctx, "Invalid vote txid : 0x%x", msg.VoteTxId)
		return node.RespondReject(ctx, w, itx, rk, actions.RejectionsMsgMalformed)
	}

	vt, err := vote.Retrieve(ctx, g.MasterDB, rk.Address, voteTxId)
	if err == vote.ErrNotFound {
		node.LogWarn(ctx, "Vote not found : %s", voteTxId)
		return node.RespondReject(ctx, w, itx, rk, actions.RejectionsVoteNotFound)
	} else if err != nil {
		node.LogWarn(ctx, "Failed to retrieve vote : %s : %s", voteTxId, err)
		return errors.Wrap(err, "Failed to retrieve vote")
	}

	if vt.Expires.Nano() <= v.Now.Nano() {
		node.LogWarn(ctx, "Vote expired : %s", voteTxId)
		return node.RespondReject(ctx, w, itx, rk, actions.RejectionsVoteClosed)
	}

	// Get Proposal
	hash, err := bitcoin.NewHash32(vt.ProposalTxId.Bytes())
	proposalTx, err := transactions.GetTx(ctx, g.MasterDB, hash, g.Config.IsTest)
	if err != nil {
		node.LogWarn(ctx, "Proposal not found for vote")
		return node.RespondReject(ctx, w, itx, rk, actions.RejectionsMsgMalformed)
	}

	proposal, ok := proposalTx.MsgProto.(*actions.Proposal)
	if !ok {
		node.LogWarn(ctx, "Proposal invalid for vote")
		return node.RespondReject(ctx, w, itx, rk, actions.RejectionsMsgMalformed)
	}

	// Validate vote
	if len(msg.Vote) > int(proposal.VoteMax) {
		node.LogWarn(ctx, "Ballot voted on too many options")
		return node.RespondReject(ctx, w, itx, rk, actions.RejectionsMsgMalformed)
	}

	if len(msg.Vote) == 0 {
		node.LogWarn(ctx, "Ballot did not vote any options")
		return node.RespondReject(ctx, w, itx, rk, actions.RejectionsMsgMalformed)
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
			return node.RespondReject(ctx, w, itx, rk, actions.RejectionsMsgMalformed)
		}
	}

	addressHash, err := itx.Inputs[0].Address.Hash()
	if err != nil {
		node.LogWarn(ctx, "Ballot address not valid : %s",
			bitcoin.NewAddressFromRawAddress(itx.Inputs[0].Address, g.Config.Net).String())
		return node.RespondReject(ctx, w, itx, rk, actions.RejectionsMsgMalformed)
	}

	ballot, exists := vt.Ballots[*addressHash]
	if !exists {
		node.LogWarn(ctx, "Ballot address not permitted to vote : %s",
			bitcoin.NewAddressFromRawAddress(itx.Inputs[0].Address, g.Config.Net).String())
		return node.RespondReject(ctx, w, itx, rk, actions.RejectionsUnauthorizedAddress)
	}

	if ballot.Timestamp.Nano() != 0 {
		node.LogWarn(ctx, "Ballot address already voted : %s",
			bitcoin.NewAddressFromRawAddress(itx.Inputs[0].Address, g.Config.Net).String())
		return node.RespondReject(ctx, w, itx, rk, actions.RejectionsBallotAlreadyCounted)
	}

	// Build Response
	ballotCounted := actions.BallotCounted{}
	err = node.Convert(ctx, msg, &ballotCounted)
	if err != nil {
		return errors.Wrap(err, "Failed to convert ballot cast to counted")
	}
	ballotCounted.Quantity = ballot.Quantity
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
		w.Config.Net)
	node.LogWarn(ctx, "Accepting ballot for %d from %s", ballot.Quantity, address.String())
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

	if !itx.Inputs[0].Address.Equal(rk.Address) {
		address := bitcoin.NewAddressFromRawAddress(itx.Inputs[0].Address,
			w.Config.Net)
		return fmt.Errorf("Ballot counted not from contract : %s", address.String())
	}

	ct, err := contract.Retrieve(ctx, g.MasterDB, rk.Address, g.Config.IsTest)
	if err != nil {
		return errors.Wrap(err, "Failed to retrieve contract")
	}

	if !ct.MovedTo.IsEmpty() {
		address := bitcoin.NewAddressFromRawAddress(ct.MovedTo,
			w.Config.Net)
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

	voteTxId, err := bitcoin.NewHash32(cast.VoteTxId)
	if err != nil {
		return errors.Wrap(err, "invalid vote txid")
	}

	vt, err := vote.Retrieve(ctx, g.MasterDB, rk.Address, voteTxId)
	if err != nil {
		return errors.Wrap(err, "Failed to retrieve vote for ballot cast")
	}

	hash, err := castTx.Inputs[0].Address.Hash()
	if err != nil {
		return fmt.Errorf("Ballot address not valid : %s",
			bitcoin.NewAddressFromRawAddress(castTx.Inputs[0].Address, g.Config.Net).String())
	}

	ballot, exists := vt.Ballots[*hash]
	if !exists {
		return fmt.Errorf("Ballot address not permitted to vote : %s",
			bitcoin.NewAddressFromRawAddress(castTx.Inputs[0].Address, g.Config.Net).String())
	}

	ballot.Vote = cast.Vote
	ballot.Timestamp = protocol.NewTimestamp(msg.Timestamp)

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
	ct, err := contract.Retrieve(ctx, g.MasterDB, rk.Address, g.Config.IsTest)
	if err != nil {
		return err
	}

	if !ct.MovedTo.IsEmpty() {
		address := bitcoin.NewAddressFromRawAddress(ct.MovedTo,
			w.Config.Net)
		return fmt.Errorf("Contract address changed : %s", address.String())
	}

	// Retrieve vote
	vt, err := vote.Retrieve(ctx, g.MasterDB, rk.Address, itx.Hash)
	if err != nil {
		return errors.Wrap(err, "Failed to retrieve vote for ballot cast")
	}

	// Get Proposal
	hash, err := bitcoin.NewHash32(vt.ProposalTxId.Bytes())
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

	voteResult.VoteTxId = itx.Hash.Bytes()
	voteResult.Timestamp = v.Now.Nano()

	// Calculate Results
	voteResult.OptionTally, voteResult.Result, err = vote.CalculateResults(ctx, vt, proposal,
		ct.VotingSystems[proposal.VoteSystem])
	if err != nil {
		return errors.Wrap(err, "Failed to calculate vote results")
	}

	// Fund with second output of proposal tx.
	w.SetUTXOs(ctx, []bitcoin.UTXO{proposalTx.Outputs[1].UTXO})

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
func (g *Governance) ResultResponse(ctx context.Context, w *node.ResponseWriter,
	itx *inspector.Transaction, rk *wallet.Key) error {
	ctx, span := trace.StartSpan(ctx, "handlers.Governance.ResultResponse")
	defer span.End()

	msg, ok := itx.MsgProto.(*actions.Result)
	if !ok {
		return errors.New("Could not assert as *actions.Result")
	}

	v := ctx.Value(node.KeyValues).(*node.Values)

	if !itx.Inputs[0].Address.Equal(rk.Address) {
		address := bitcoin.NewAddressFromRawAddress(itx.Inputs[0].Address, w.Config.Net)
		return fmt.Errorf("Vote result not from contract : %s", address)
	}

	ct, err := contract.Retrieve(ctx, g.MasterDB, rk.Address, g.Config.IsTest)
	if err != nil {
		return errors.Wrap(err, "Failed to retrieve contract")
	}

	if !ct.MovedTo.IsEmpty() {
		address := bitcoin.NewAddressFromRawAddress(ct.MovedTo,
			w.Config.Net)
		return fmt.Errorf("Contract address changed : %s", address.String())
	}

	uv := vote.UpdateVote{}
	err = node.Convert(ctx, msg, &uv)
	if err != nil {
		return errors.Wrap(err, "Failed to convert result message to update vote")
	}

	ts := protocol.NewTimestamp(msg.Timestamp)
	uv.CompletedAt = &ts

	voteTxId, err := bitcoin.NewHash32(msg.VoteTxId)
	if err != nil {
		return errors.Wrap(err, "invalid vote txid")
	}

	if err := vote.Update(ctx, g.MasterDB, rk.Address, voteTxId, &uv, v.Now); err != nil {
		return errors.Wrap(err, "Failed to update vote")
	}

	if len(msg.AssetCode) > 0 {
		// Save result for amendment action
		if err := transactions.AddTx(ctx, g.MasterDB, itx); err != nil {
			return errors.Wrap(err, "Failed to save tx")
		}
	}

	return nil
}
