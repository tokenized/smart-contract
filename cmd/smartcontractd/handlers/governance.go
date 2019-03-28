package handlers

import (
	"bytes"
	"context"

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

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcutil"
	"github.com/pkg/errors"
	"go.opencensus.io/trace"
)

type Governance struct {
	MasterDB *db.DB
	Config   *node.Config
	TxCache  InspectorTxCache
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

	// Locate Contract
	contractPKH := protocol.PublicKeyHashFromBytes(rk.Address.ScriptAddress())
	ct, err := contract.Retrieve(ctx, g.MasterDB, contractPKH)
	if err != nil {
		return err
	}

	var senderPKH *protocol.PublicKeyHash

	// Check if sender is allowed to make proposal
	if msg.Initiator == 0 { // Issuer Proposal
		if !bytes.Equal(itx.Inputs[0].Address.ScriptAddress(), ct.IssuerPKH.Bytes()) &&
			!bytes.Equal(itx.Inputs[0].Address.ScriptAddress(), ct.OperatorPKH.Bytes()) {
			logger.Warn(ctx, "%s : Initiator PKH is not issuer or operator : %s", v.TraceID, contractPKH.String())
			return node.RespondReject(ctx, w, itx, rk, protocol.RejectionCodeIssuerAddress)
		}
	} else if msg.Initiator == 1 { // Holder Proposal
		// Sender must hold balance of at least one asset
		senderPKH = protocol.PublicKeyHashFromBytes(itx.Inputs[0].Address.ScriptAddress())
		if !contract.HasAnyBalance(ctx, g.MasterDB, ct, senderPKH) {
			logger.Warn(ctx, "%s : Sender holds no assets : %s %s", v.TraceID, contractPKH.String(), senderPKH.String())
			return node.RespondReject(ctx, w, itx, rk, protocol.RejectionCodeInsufficientAssets)
		}
	} else {
		logger.Warn(ctx, "%s : Invalid Initiator value : %02x", v.TraceID, msg.Initiator)
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectionCodeMalformed)
	}

	if int(msg.VoteSystem) >= len(ct.VotingSystems) {
		logger.Warn(ctx, "%s : Proposal vote system invalid : %s", v.TraceID, contractPKH.String(), senderPKH.String())
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectionCodeMalformed)
	}

	// Validate messages vote related values
	if !vote.ValidateProposal(msg, v.Now) {
		logger.Warn(ctx, "%s : Proposal validation failed : %s %s", v.TraceID, contractPKH, senderPKH.String())
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectionCodeMalformed)
	}

	if msg.AssetSpecificVote {
		as, err := asset.Retrieve(ctx, g.MasterDB, contractPKH, &msg.AssetCode)
		if err != nil {
			logger.Warn(ctx, "%s : Asset not found : %s %s", v.TraceID, contractPKH.String(), msg.AssetCode.String())
			return node.RespondReject(ctx, w, itx, rk, protocol.RejectionCodeAssetNotFound)
		}

		// Asset does not allow voting
		if err := asset.ValidateVoting(ctx, as, msg.Initiator, &ct.VotingSystems[msg.VoteSystem]); err != nil {
			logger.Warn(ctx, "%s : Asset does not allow voting: %s %s : %s", v.TraceID, contractPKH.String(), msg.AssetCode.String(), err)
			return node.RespondReject(ctx, w, itx, rk, protocol.RejectionCodeContractAuthFlags)
		}

		// Sender does not have any balance of the asset
		if msg.Initiator > 0 && asset.GetVotingBalance(ctx, as, senderPKH, ct.VotingSystems[msg.VoteSystem].VoteMultiplierPermitted, v.Now) == 0 {
			logger.Warn(ctx, "%s : Insufficient funds: %s %s", v.TraceID, contractPKH.String(), msg.AssetCode.String())
			return node.RespondReject(ctx, w, itx, rk, protocol.RejectionCodeInsufficientAssets)
		}

		if msg.Specific {
			// Asset amendments vote. Validate permissions for fields being amended.
			if err := asset.ValidatePermissions(ctx, as, len(ct.VotingSystems), msg.ProposedAmendments, true, msg.Initiator, msg.VoteSystem); err != nil {
				logger.Warn(ctx, "%s : Asset amendment not authorized : %s %s", v.TraceID, contractPKH.String(), msg.AssetCode.String())
				return node.RespondReject(ctx, w, itx, rk, protocol.RejectionCodeAssetAuthFlags)
			}
		}
	} else {
		// Contract does not allow voting
		if err := contract.ValidateVoting(ctx, ct, msg.Initiator); err != nil {
			logger.Warn(ctx, "%s : Contract does not allow voting: %s : %s", v.TraceID, contractPKH.String(), err)
			return node.RespondReject(ctx, w, itx, rk, protocol.RejectionCodeContractAuthFlags)
		}

		if msg.Specific {
			// Contract amendments vote. Validate permissions for fields being amended.
			if err := contract.ValidatePermissions(ctx, ct, msg.ProposedAmendments, true, msg.Initiator, msg.VoteSystem); err != nil {
				logger.Warn(ctx, "%s : Contract amendment not authorized : %s", v.TraceID, contractPKH.String())
				return node.RespondReject(ctx, w, itx, rk, protocol.RejectionCodeContractAuthFlags)
			}
		}

		// Sender does not have any balance of the asset
		if msg.Initiator > 0 && contract.GetVotingBalance(ctx, g.MasterDB, ct, senderPKH, ct.VotingSystems[msg.VoteSystem].VoteMultiplierPermitted, v.Now) == 0 {
			logger.Warn(ctx, "%s : Insufficient funds: %s %s", v.TraceID, contractPKH.String(), msg.AssetCode.String())
			return node.RespondReject(ctx, w, itx, rk, protocol.RejectionCodeInsufficientAssets)
		}
	}

	// Build Response
	vote := protocol.Vote{Timestamp: v.Now}

	// Convert to btcutil.Address
	contractAddress, err := btcutil.NewAddressPubKeyHash(contractPKH.Bytes(), &g.Config.ChainParams)
	if err != nil {
		return err
	}

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
		logger.Warn(ctx, "%s : Response not from contract : %s", v.TraceID, contractPKH.String())
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectionCodeContractAddress)
	}

	// Retrieve Proposal
	proposalTx := g.TxCache.GetTx(ctx, &itx.Inputs[0].UTXO.Hash)
	if proposalTx == nil {
		logger.Warn(ctx, "%s : Proposal not found for vote : %s", v.TraceID, contractPKH.String())
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectionCodeMalformed)
	}

	proposal, ok := proposalTx.MsgProto.(*protocol.Proposal)
	if !ok {
		logger.Warn(ctx, "%s : Proposal invalid for vote : %s", v.TraceID, contractPKH.String())
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectionCodeMalformed)
	}

	voteTxId := protocol.TxIdFromBytes(itx.Hash[:])

	_, err = vote.Retrieve(ctx, g.MasterDB, contractPKH, voteTxId)
	if err != vote.ErrNotFound {
		if err != nil {
			logger.Warn(ctx, "%s : Failed to retreive vote : %s %s : %s", v.TraceID, contractPKH.String(), voteTxId.String(), err)
		} else {
			logger.Warn(ctx, "%s : Vote already exists : %s %s", v.TraceID, contractPKH.String(), voteTxId.String())
		}
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectionCodeMalformed)
	}

	nv := vote.NewVote{}
	err = node.Convert(ctx, msg, &nv)
	if err != nil {
		return errors.Wrap(err, "Failed to convert vote message to new vote")
	}

	nv.VoteTxId = *voteTxId
	nv.ProposalTxId = *protocol.TxIdFromBytes(proposalTx.Hash[:])
	nv.Expires = proposal.VoteCutOffTimestamp

	if proposal.AssetSpecificVote {
		as, err := asset.Retrieve(ctx, g.MasterDB, contractPKH, &proposal.AssetCode)
		if err != nil {
			logger.Warn(ctx, "%s : Asset not found : %s %s", v.TraceID, contractPKH.String(), proposal.AssetCode.String())
			return node.RespondReject(ctx, w, itx, rk, protocol.RejectionCodeAssetNotFound)
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

	contractPKH := protocol.PublicKeyHashFromBytes(rk.Address.ScriptAddress())

	ct, err := contract.Retrieve(ctx, g.MasterDB, contractPKH)
	if err != nil {
		return err
	}

	vt, err := vote.Retrieve(ctx, g.MasterDB, contractPKH, &msg.VoteTxId)
	if err == vote.ErrNotFound {
		logger.Warn(ctx, "%s : Vote not found : %s %s", v.TraceID, contractPKH.String(), msg.VoteTxId.String())
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectionCodeVoteNotFound)
	} else if err != nil {
		logger.Warn(ctx, "%s : Failed to retreive vote : %s %s : %s", v.TraceID, contractPKH.String(), msg.VoteTxId.String(), err)
		return errors.Wrap(err, "Failed to retreive vote")
	}

	if vt.Expires.Nano() <= v.Now.Nano() {
		logger.Warn(ctx, "%s : Vote expired : %s %s", v.TraceID, contractPKH.String(), msg.VoteTxId.String())
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectionCodeVoteClosed)
	}

	// Get Proposal
	hash, err := chainhash.NewHash(vt.ProposalTxId.Bytes())
	proposalTx := g.TxCache.GetTx(ctx, hash)
	if proposalTx == nil {
		logger.Warn(ctx, "%s : Proposal not found for vote : %s", v.TraceID, contractPKH.String())
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectionCodeMalformed)
	}

	proposal, ok := proposalTx.MsgProto.(*protocol.Proposal)
	if !ok {
		logger.Warn(ctx, "%s : Proposal invalid for vote : %s", v.TraceID, contractPKH.String())
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectionCodeMalformed)
	}

	// Validate vote
	if len(msg.Vote) > int(proposal.VoteMax) {
		logger.Warn(ctx, "%s : Ballot voted on too many options : %s", v.TraceID, contractPKH.String())
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectionCodeMalformed)
	}

	if len(msg.Vote) == 0 {
		logger.Warn(ctx, "%s : Ballot did not vote any options : %s", v.TraceID, contractPKH.String())
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectionCodeMalformed)
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
			return node.RespondReject(ctx, w, itx, rk, protocol.RejectionCodeMalformed)
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
			return node.RespondReject(ctx, w, itx, rk, protocol.RejectionCodeAssetNotFound)
		}

		quantity = asset.GetVotingBalance(ctx, as, holderPKH, ct.VotingSystems[proposal.VoteSystem].VoteMultiplierPermitted, v.Now)
	} else {
		quantity = contract.GetVotingBalance(ctx, g.MasterDB, ct, holderPKH, ct.VotingSystems[proposal.VoteSystem].VoteMultiplierPermitted, v.Now)
	}

	if quantity == 0 {
		logger.Warn(ctx, "%s : User PKH doesn't hold any voting tokens : %s %s", v.TraceID, contractPKH.String(), holderPKH.String())
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectionCodeInsufficientAssets)
	}

	// TODO Check issue where two ballots are sent simultaneously and the second received before the first response is processed.
	if err := vote.CheckBallot(ctx, vt, holderPKH); err != nil {
		logger.Warn(ctx, "%s : Failed to check ballot : %s", v.TraceID, contractPKH.String(), err)
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectionCodeBallotCounted)
	}

	// Build Response
	ballotCounted := protocol.BallotCounted{
		Quantity:  quantity,
		Timestamp: v.Now,
	}

	// Convert to btcutil.Address
	contractAddress, err := btcutil.NewAddressPubKeyHash(contractPKH.Bytes(), &g.Config.ChainParams)
	if err != nil {
		return err
	}

	// Build outputs
	// 1 - Contract Address (change)
	// 2 - Contract/Proposal Fee
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
		logger.Warn(ctx, "%s : Response not from contract : %s", v.TraceID, contractPKH.String())
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectionCodeContractAddress)
	}

	castTx := g.TxCache.GetTx(ctx, &itx.Inputs[0].UTXO.Hash)
	if castTx == nil {
		logger.Warn(ctx, "%s : Ballot cast not found for ballot counted msg : %s", v.TraceID, contractPKH.String())
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectionCodeMalformed)
	}

	cast, ok := castTx.MsgProto.(*protocol.BallotCast)
	if !ok {
		logger.Warn(ctx, "%s : Ballot cast invalid for ballot counted : %s", v.TraceID, contractPKH.String())
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectionCodeMalformed)
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
		logger.Warn(ctx, "%s : Response not from contract : %s", v.TraceID, contractPKH.String())
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectionCodeContractAddress)
	}

	uv := vote.UpdateVote{}
	err := node.Convert(ctx, msg, &uv)
	if err != nil {
		return errors.Wrap(err, "Failed to convert result message to update vote")
	}

	uv.CompletedAt = &msg.Timestamp

	if err := vote.Update(ctx, g.MasterDB, contractPKH, &msg.VoteTxID, &uv, v.Now); err != nil {
		return errors.Wrap(err, "Failed to update vote")
	}
	return nil
}
