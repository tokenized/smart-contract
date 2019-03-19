package handlers

import (
	"context"
	"errors"

	"github.com/tokenized/smart-contract/internal/asset"
	"github.com/tokenized/smart-contract/internal/contract"
	"github.com/tokenized/smart-contract/internal/platform/db"
	"github.com/tokenized/smart-contract/internal/platform/node"
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

// InitiativeRequest handles an incoming Initiative request and prepares a BallotCounted response
func (g *Governance) InitiativeRequest(ctx context.Context, w *node.ResponseWriter, itx *inspector.Transaction, rk *wallet.RootKey) error {
	ctx, span := trace.StartSpan(ctx, "handlers.Governance.InitiativeRequest")
	defer span.End()

	msg, ok := itx.MsgProto.(*protocol.Initiative)
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
		logger.Warn(ctx, "%s : Contract not found: %s", v.TraceID, contractAddr)
		return node.ErrNoResponse
	}

	// Contract does not allow voting
	if !contract.IsVotingPermitted(ctx, ct) {
		logger.Warn(ctx, "%s : Contract does not allow voting: %s", v.TraceID, contractAddr)
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectionCodeContractAuthFlags)
	}

	// Validate issuer address
	// issuerAddress, err := btcutil.DecodeAddress(ct.IssuerAddress, &chaincfg.MainNetParams)
	// if err != nil {
	// logger.Warn(ctx, "%s : Invalid issuer address: %s %s", v.TraceID, contractAddr, ct.IssuerAddress)
	// return err
	// }

	// Sender must hold balance of at least one asset
	senderAddr := protocol.PublicKeyHashFromBytes(itx.Inputs[0].Address.ScriptAddress())
	if !contract.HasAnyBalance(ctx, ct, senderAddr) {
		logger.Warn(ctx, "%s : Sender holds no assets: %s %s", v.TraceID, contractAddr, senderAddr)
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectionCodeInsufficientAssets)
	}

	// TODO Validate messages values
	if !vote.ValidateInitiative(msg) {
		logger.Warn(ctx, "%s : Initiative validation failed: %s %s", v.TraceID, contractAddr, senderAddr)
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectionCodeInvalidInitiative)
	}

	// If an asset is specified
	if !msg.AssetCode.IsZero() {
		// Locate asset
		as, err := asset.Retrieve(ctx, dbConn, contractAddr, msg.AssetCode)
		if err != nil {
			return err
		}

		// Asset could not be found
		if as == nil {
			logger.Warn(ctx, "%s : Asset ID not found: %s %s", v.TraceID, contractAddr, msg.AssetCode)
			return node.RespondReject(ctx, w, itx, rk, protocol.RejectionCodeAssetNotFound)
		}

		// Asset does not allow voting
		// if !asset.IsVotingPermitted(ctx, as) {
		// logger.Warn(ctx, "%s : Asset does not allow voting: %s %s", v.TraceID, contractAddr, msg.AssetCode)
		// return node.RespondReject(ctx, w, itx, rk, protocol.RejectionCodeContractAuthFlags)
		// }

		// Sender does not have any balance of the asset
		if asset.GetBalance(ctx, as, senderAddr) < 1 {
			logger.Warn(ctx, "%s : Insufficient funds: %s %s", v.TraceID, contractAddr, msg.AssetCode)
			return node.RespondReject(ctx, w, itx, rk, protocol.RejectionCodeInsufficientAssets)
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
	// outs := []node.Output{{
	// Address: contractAddr,
	// Value:   g.Config.DustLimit,
	// }, {
	// Address: issuerAddress,
	// Value:   g.Config.DustLimit,
	// Change:  true,
	// }}

	// // Add fee output
	// if fee := node.OutputFee(ctx, g.Config); fee != nil {
	// outs = append(outs, *fee)
	// }

	// // Respond specifically using the first UTXO
	// itxUtxos := itx.UTXOs()
	// utxos := inspector.UTXOs{itxUtxos[0]}

	// // Respond with a vote action
	// return node.RespondUTXO(ctx, w, itx, rk, &vote, outs, utxos)
}

// ReferendumRequest handles an incoming Referendum request and prepares a BallotCounted response
func (g *Governance) ReferendumRequest(ctx context.Context, w *node.ResponseWriter, itx *inspector.Transaction, rk *wallet.RootKey) error {
	ctx, span := trace.StartSpan(ctx, "handlers.Governance.ReferendumRequest")
	defer span.End()

	msg, ok := itx.MsgProto.(*protocol.Referendum)
	if !ok {
		return errors.New("Could not assert as *protocol.Referendum")
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
		logger.Warn(ctx, "%s : Contract not found: %s", v.TraceID, contractAddr)
		return node.ErrNoResponse
	}

	// Contract does not allow voting
	if !contract.IsVotingPermitted(ctx, ct) {
		logger.Warn(ctx, "%s : Contract does not allow voting: %s", v.TraceID, contractAddr)
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectionCodeContractAuthFlags)
	}

	// Validate issuer address
	// issuerAddress, err := btcutil.DecodeAddress(string(ct.IssuerAddress), &chaincfg.MainNetParams)
	// if err != nil {
	// logger.Warn(ctx, "%s : Invalid issuer address: %s %s", v.TraceID, contractAddr, ct.IssuerAddress)
	// return node.RespondReject(ctx, w, itx, rk, protocol.RejectionCodeUnknownAddress)
	// }

	// Sender must be a contract operator
	senderAddr := protocol.PublicKeyHashFromBytes(itx.Inputs[0].Address.ScriptAddress())
	if !contract.IsOperator(ctx, ct, senderAddr) {
		logger.Warn(ctx, "%s : Sender is not an operator: %s %s", v.TraceID, contractAddr, senderAddr)
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectionCodeUnknownAddress)
	}

	// Validate messages values
	// if !vote.ValidateReferendum(msg) {
	// logger.Warn(ctx, "%s : Initiative validation failed: %s %s", v.TraceID, contractAddr, senderAddr)
	// return node.RespondReject(ctx, w, itx, rk, protocol.RejectionCodeInvalidValue)
	// }

	// If an asset is specified
	if !msg.AssetCode.IsZero() {
		// Locate asset
		as, err := asset.Retrieve(ctx, dbConn, contractAddr, msg.AssetCode)
		if err != nil {
			return err
		}

		// Asset could not be found
		if as == nil {
			logger.Warn(ctx, "%s : Asset ID not found: %s %s", v.TraceID, contractAddr, msg.AssetCode)
			return node.RespondReject(ctx, w, itx, rk, protocol.RejectionCodeAssetNotFound)
		}

		// Asset does not allow voting
		// if !asset.IsVotingPermitted(ctx, as) {
		// logger.Warn(ctx, "%s : Asset does not allow voting: %s %s", v.TraceID, contractAddr, msg.AssetCode)
		// return node.RespondReject(ctx, w, itx, rk, protocol.RejectionCodeContractAuthFlags)
		// }
	}

	// logger.Info(ctx, "%s : Referendum Request : %s %s", v.TraceID, contractAddr, msg.AssetCode)
	logger.Warn(ctx, "%s : Referendum Request Not Implemented : %s %s", v.TraceID, contractAddr, msg.AssetCode)
	return errors.New("Referendum Request Not Implemented")

	// TODO(srg) auth flags

	// Vote <- Referendum
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
	// outs := []node.Output{{
	// Address: contractAddr,
	// Value:   g.Config.DustLimit,
	// }, {
	// Address: issuerAddress,
	// Value:   g.Config.DustLimit,
	// Change:  true,
	// }}

	// // Add fee output
	// if fee := node.OutputFee(ctx, g.Config); fee != nil {
	// outs = append(outs, *fee)
	// }

	// // Respond specifically using the first UTXO
	// itxUtxos := itx.UTXOs()
	// utxos := inspector.UTXOs{itxUtxos[0]}

	// // Respond with a vote action
	// return node.RespondUTXO(ctx, w, itx, rk, &vote, outs, utxos)
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
