package handlers

import (
	"bytes"
	"context"
	"errors"

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

	"go.opencensus.io/trace"
)

type Governance struct {
	MasterDB *db.DB
	Config   *node.Config
}

// ProposalRequest handles an incoming proposal request and prepares a Vote response
func (g *Governance) ProposalRequest(ctx context.Context, w *node.ResponseWriter, itx *inspector.Transaction, rk *wallet.RootKey) error {
	ctx, span := trace.StartSpan(ctx, "handlers.Governance.InitiativeRequest")
	defer span.End()

	msg, ok := itx.MsgProto.(*protocol.Proposal)
	if !ok {
		return errors.New("Could not assert as *protocol.Initiative")
	}

	dbConn := g.MasterDB

	v := ctx.Value(node.KeyValues).(*node.Values)

	// Locate Contract
	contractAddr := protocol.PublicKeyHashFromBytes(rk.Address.ScriptAddress())
	ct, err := contract.Retrieve(ctx, dbConn, contractAddr)
	if err != nil {
		return err
	}

	// Contract could not be found
	if ct == nil {
		logger.Warn(ctx, "%s : Contract not found: %s", v.TraceID, contractAddr.String())
		return node.ErrNoResponse
	}

	// Contract does not allow voting
	if !contract.IsVotingPermitted(ctx, ct) {
		logger.Warn(ctx, "%s : Contract does not allow voting: %s", v.TraceID, contractAddr.String())
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectionCodeContractAuthFlags)
	}

	var senderPKH *protocol.PublicKeyHash

	// Check if sender is allowed to make proposal
	if msg.Initiator == 0 { // Issuer Proposal
		if !bytes.Equal(itx.Inputs[0].Address.ScriptAddress(), ct.IssuerPKH.Bytes()) &&
			!bytes.Equal(itx.Inputs[0].Address.ScriptAddress(), ct.OperatorPKH.Bytes()) {
			logger.Warn(ctx, "%s : Initiator PKH is not issuer or operator : %s", v.TraceID, contractAddr.String())
			return node.RespondReject(ctx, w, itx, rk, protocol.RejectionCodeIssuerAddress)
		}

		if !ct.IssuerProposal {
			logger.Warn(ctx, "%s : Contract does not allow issuer initiated proposals : %s", v.TraceID, contractAddr.String())
			return node.RespondReject(ctx, w, itx, rk, protocol.RejectionCodeContractAuthFlags)
		}
	} else if msg.Initiator == 1 { // Holder Proposal
		if !ct.HolderProposal {
			logger.Warn(ctx, "%s : Contract does not allow holder initiated proposals : %s", v.TraceID, contractAddr.String())
			return node.RespondReject(ctx, w, itx, rk, protocol.RejectionCodeContractAuthFlags)
		}

		// Sender must hold balance of at least one asset
		senderPKH = protocol.PublicKeyHashFromBytes(itx.Inputs[0].Address.ScriptAddress())
		if !contract.HasAnyBalance(ctx, dbConn, ct, senderPKH) {
			logger.Warn(ctx, "%s : Sender holds no assets : %s %s", v.TraceID, contractAddr.String(), senderPKH.String())
			return node.RespondReject(ctx, w, itx, rk, protocol.RejectionCodeInsufficientAssets)
		}
	} else {
		logger.Warn(ctx, "%s : Invalid Initiator value : %02x", v.TraceID, msg.Initiator)
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectionCodeMalformed)
	}

	var as *state.Asset
	if msg.AssetSpecificVote {
		as, err = asset.Retrieve(ctx, dbConn, contractAddr, &msg.AssetCode)
		if err != nil {
			logger.Warn(ctx, "%s : Asset not found : %s %s", v.TraceID, contractAddr.String(), msg.AssetCode.String())
			return node.RespondReject(ctx, w, itx, rk, protocol.RejectionCodeAssetNotFound)
		}

		// Asset does not allow voting
		if !asset.IsVotingPermitted(ctx, as) {
			logger.Warn(ctx, "%s : Asset does not allow voting: %s %s", v.TraceID, contractAddr.String(), msg.AssetCode.String())
			return node.RespondReject(ctx, w, itx, rk, protocol.RejectionCodeContractAuthFlags)
		}

		// Sender does not have any balance of the asset
		if asset.GetBalance(ctx, as, senderPKH) < 1 {
			logger.Warn(ctx, "%s : Insufficient funds: %s %s", v.TraceID, contractAddr.String(), msg.AssetCode.String())
			return node.RespondReject(ctx, w, itx, rk, protocol.RejectionCodeInsufficientAssets)
		}
	}

	if int(msg.VoteSystem) >= len(ct.VotingSystems) {
		logger.Warn(ctx, "%s : Proposal vote system invalid : %s", v.TraceID, contractAddr, senderPKH)
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectionCodeMalformed)
	}

	// Validate messages vote related values
	if !vote.ValidateProposal(msg) {
		logger.Warn(ctx, "%s : Proposal validation failed : %s %s", v.TraceID, contractAddr, senderPKH)
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectionCodeMalformed)
	}

	// TODO Validate auth flags for proposed amendments.
	if msg.Specific {
		if msg.AssetSpecificVote {
			permissions, err := protocol.ReadAuthFlags(as.AssetAuthFlags, asset.FieldCount, len(ct.VotingSystems))

		} else { // Contract amendments vote
			permissions, err := protocol.ReadAuthFlags(ct.ContractAuthFlags, contract.FieldCount, len(ct.VotingSystems))

		}
	}

	// logger.Info(ctx, "%s : Initiative Request : %s %s", v.TraceID, contractAddr, msg.AssetCode)
	logger.Warn(ctx, "%s : Initiative Request Not Implemented : %s %s", v.TraceID, contractAddr, msg.AssetCode)
	return errors.New("Initiative Request Not Implemented")

	// TODO(srg) auth flags

	// Vote <- Initiative
	// vote := protocol.NewVote()
	// vote.AssetType = msg.AssetType
	// vote.AssetCode = msg.AssetCode
	// vote.VoteType = msg.VoteType
	// vote.VoteOptions = msg.VoteOptions
	// vote.VoteMax = msg.VoteMax
	// vote.VoteLogic = msg.VoteLogic
	// vote.ProposalDescription = msg.ProposalDescription
	// vote.ProposalDocumentHash = msg.ProposalDocumentHash
	// vote.VoteCutOffTimestamp = msg.VoteCutOffTimestamp
	// vote.Timestamp = uint64(v.Now.Unix())

	// // Build outputs
	// // 1 - Contract Address
	// // 2 - Issuer Address (Change)
	// // 3 - Fee
	// w.AddOutput(ctx, contractAddr, 0)
	// w.AddChangeOutput(ctx, issuerAddress)

	// // Add fee output
	// w.AddFee(ctx)

	// // Respond specifically using the first UTXO
	// itxUtxos := itx.UTXOs()
	// utxos := inspector.UTXOs{itxUtxos[0]}
	// w.SetUTXOs(utxos)

	// // Respond with a vote action
	// return node.RespondSuccess(ctx, w, itx, rk, &vote)
}

// VoteResponse handles an incoming Vote request and prepares a BallotCounted response
func (g *Governance) VoteResponse(ctx context.Context, w *node.ResponseWriter, itx *inspector.Transaction, rk *wallet.RootKey) error {
	ctx, span := trace.StartSpan(ctx, "handlers.Governance.VoteResponse")
	defer span.End()

	// msg, ok := itx.MsgProto.(*protocol.Vote)
	// if !ok {
	// 	return errors.New("Could not assert as *protocol.Vote")
	// }

	// NB(srg): Voting has changed quite a bit in the next protocol verison
	// so this is left out

	return nil
}

// BallotCastRequest handles an incoming BallotCast request and prepares a BallotCounted response
func (g *Governance) BallotCastRequest(ctx context.Context, w *node.ResponseWriter, itx *inspector.Transaction, rk *wallet.RootKey) error {
	return nil
}

// BallotCountedResponse handles an outgoing BallotCounted action and writes it to the state
func (g *Governance) BallotCountedResponse(ctx context.Context, w *node.ResponseWriter, itx *inspector.Transaction, rk *wallet.RootKey) error {
	return nil
}

// ResultResponse handles an outgoing Result action and writes it to the state
func (g *Governance) ResultResponse(ctx context.Context, w *node.ResponseWriter, itx *inspector.Transaction, rk *wallet.RootKey) error {
	return nil
}
