package handlers

import (
	"bytes"
	"context"
	"fmt"

	"github.com/tokenized/smart-contract/internal/asset"
	"github.com/tokenized/smart-contract/internal/contract"
	"github.com/tokenized/smart-contract/internal/platform/db"
	"github.com/tokenized/smart-contract/internal/platform/node"
	"github.com/tokenized/smart-contract/internal/platform/state"
	"github.com/tokenized/smart-contract/internal/platform/wallet"
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
	TxCache   InspectorTxCache
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
		return errors.Wrap(err, "Failed to retreive contract")
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
		logger.Warn(ctx, "%s : Proposal vote system invalid : %s", v.TraceID, contractPKH.String(), senderPKH.String())
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectMsgMalformed)
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
			if err := asset.ValidatePermissions(ctx, as, len(ct.VotingSystems), msg.ProposedAmendments, true, msg.Initiator, msg.VoteSystem); err != nil {
				logger.Warn(ctx, "%s : Asset amendment not authorized : %s %s", v.TraceID, contractPKH.String(), msg.AssetCode.String())
				return node.RespondReject(ctx, w, itx, rk, protocol.RejectAssetAuthFlags)
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
			if err := contract.ValidatePermissions(ctx, ct, msg.ProposedAmendments, true, msg.Initiator, msg.VoteSystem); err != nil {
				logger.Warn(ctx, "%s : Contract amendment not authorized : %s", v.TraceID, contractPKH.String())
				return node.RespondReject(ctx, w, itx, rk, protocol.RejectContractAuthFlags)
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
						logger.Warn(ctx, "%s : Proposed amendment conflicts with unapplied vote : %s %s", v.TraceID, contractPKH.String())
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
	// 1 - Contract Address (change)
	// 2 - Contract/Proposal Fee
	w.AddChangeOutput(ctx, contractAddress)

	feeAmount := ct.ContractFee
	if msg.Initiator > 0 {
		feeAmount += ct.VotingSystems[msg.VoteSystem].HolderProposalFee
	}
	w.AddContractFee(ctx, feeAmount)

	// Save Tx.
	g.TxCache.SaveTx(ctx, itx)

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
	proposalTx := g.TxCache.GetTx(ctx, &itx.Inputs[0].UTXO.Hash)
	if proposalTx == nil {
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
			logger.Warn(ctx, "%s : Failed to retreive vote : %s %s : %s", v.TraceID, contractPKH.String(), voteTxId.String(), err)
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

		if ct.VotingSystems[proposal.VoteSystem].VoteMultiplierPermitted {
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

	if err := g.Scheduler.ScheduleJob(ctx, NewVoteFinalizer(g.handler, itx, proposal.VoteCutOffTimestamp)); err != nil {
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
		logger.Warn(ctx, "%s : Failed to retreive vote : %s %s : %s", v.TraceID, contractPKH.String(), msg.VoteTxId.String(), err)
		return errors.Wrap(err, "Failed to retreive vote")
	}

	if vt.Expires.Nano() <= v.Now.Nano() {
		logger.Warn(ctx, "%s : Vote expired : %s %s", v.TraceID, contractPKH.String(), msg.VoteTxId.String())
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectVoteClosed)
	}

	// Get Proposal
	hash, err := chainhash.NewHash(vt.ProposalTxId.Bytes())
	proposalTx := g.TxCache.GetTx(ctx, hash)
	if proposalTx == nil {
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
	if proposal.AssetSpecificVote {
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
		logger.Warn(ctx, "%s : Failed to check ballot : %s", v.TraceID, contractPKH.String(), err)
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
	// 1 - Contract Address (change)
	// 2 - Contract Fee
	w.AddChangeOutput(ctx, contractAddress)
	w.AddContractFee(ctx, ct.ContractFee)

	// Save Tx for response.
	g.TxCache.SaveTx(ctx, itx)

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

	castTx := g.TxCache.GetTx(ctx, &itx.Inputs[0].UTXO.Hash)
	if castTx == nil {
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
	proposalTx := g.TxCache.GetTx(ctx, hash)
	if proposalTx == nil {
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
	// 1 - Contract Address (change)
	// 2 - Contract Fee
	w.AddChangeOutput(ctx, contractAddress)
	w.AddContractFee(ctx, ct.ContractFee)

	// Save Tx for response.
	g.TxCache.SaveTx(ctx, itx)

	// Respond with a vote
	return node.RespondSuccess(ctx, w, itx, rk, &voteResult)

	return errors.New("FinalizeVote not implemented")
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

	vt, err := vote.Retrieve(ctx, g.MasterDB, contractPKH, &msg.VoteTxId)
	if err != nil {
		return errors.Wrap(err, "Failed to retrieve vote for ballot cast")
	}

	uv := vote.UpdateVote{}
	err = node.Convert(ctx, msg, &uv)
	if err != nil {
		return errors.Wrap(err, "Failed to convert result message to update vote")
	}

	uv.CompletedAt = &msg.Timestamp

	if err := vote.Update(ctx, g.MasterDB, contractPKH, &msg.VoteTxId, &uv, v.Now); err != nil {
		return errors.Wrap(err, "Failed to update vote")
	}

	// Remove cached txs
	hash, err := chainhash.NewHash(vt.ProposalTxId.Bytes())
	g.TxCache.RemoveTx(ctx, hash)
	hash, err = chainhash.NewHash(msg.VoteTxId.Bytes())
	g.TxCache.RemoveTx(ctx, hash)
	return nil
}
