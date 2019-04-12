package handlers

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"

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

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcutil"
	"github.com/pkg/errors"
	"go.opencensus.io/trace"
)

type Asset struct {
	MasterDB *db.DB
	Config   *node.Config
}

// DefinitionRequest handles an incoming Asset Definition and prepares a Creation response
func (a *Asset) DefinitionRequest(ctx context.Context, w *node.ResponseWriter, itx *inspector.Transaction, rk *wallet.RootKey) error {
	ctx, span := trace.StartSpan(ctx, "handlers.Asset.Definition")
	defer span.End()

	msg, ok := itx.MsgProto.(*protocol.AssetDefinition)
	if !ok {
		return errors.New("Could not assert as *protocol.AssetDefinition")
	}

	v := ctx.Value(node.KeyValues).(*node.Values)

	// Validate all fields have valid values.
	if err := msg.Validate(); err != nil {
		logger.Warn(ctx, "%s : Asset definition invalid : %s", v.TraceID, err)
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectMsgMalformed)
	}

	// Locate Contract
	contractPKH := protocol.PublicKeyHashFromBytes(rk.Address.ScriptAddress())
	ct, err := contract.Retrieve(ctx, a.MasterDB, contractPKH)
	if err != nil {
		return errors.Wrap(err, "Failed to retrieve contract")
	}

	// Contract could not be found
	if ct == nil {
		logger.Warn(ctx, "%s : Contract not found: %s", v.TraceID, contractPKH.String())
		return node.ErrNoResponse
	}

	// Verify issuer is sender of tx.
	if !bytes.Equal(itx.Inputs[0].Address.ScriptAddress(), ct.IssuerPKH.Bytes()) {
		logger.Warn(ctx, "%s : Only issuer can create assets: %s %s", v.TraceID, contractPKH.String(), msg.AssetCode.String())
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectNotIssuer)
	}

	// Locate Asset
	_, err = asset.Retrieve(ctx, a.MasterDB, contractPKH, &msg.AssetCode)
	if err != asset.ErrNotFound {
		if err == nil {
			logger.Warn(ctx, "%s : Asset already exists : %s", v.TraceID, msg.AssetCode.String())
			return node.RespondReject(ctx, w, itx, rk, protocol.RejectAssetCodeExists)
		} else {
			return errors.Wrap(err, "Failed to retrieve asset")
		}
	}

	// Allowed to have more assets
	if !contract.CanHaveMoreAssets(ctx, ct) {
		logger.Warn(ctx, "%s : Number of assets exceeds contract Qty: %s %s", v.TraceID, contractPKH.String(), msg.AssetCode.String())
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectContractFixedQuantity)
	}

	// Validate payload
	payload := protocol.AssetTypeMapping(msg.AssetType)
	if payload == nil {
		logger.Warn(ctx, "%s : Asset payload type unknown : %s", v.TraceID, msg.AssetType)
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectMsgMalformed)
	}

	_, err = payload.Write(msg.AssetPayload)
	if err != nil {
		logger.Warn(ctx, "%s : Failed to parse asset payload : %s", v.TraceID, err)
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectMsgMalformed)
	}

	if err := payload.Validate(); err != nil {
		logger.Warn(ctx, "%s : Asset %s payload is invalid : %s", v.TraceID, msg.AssetType, err)
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectMsgMalformed)
	}

	logger.Info(ctx, "%s : Accepting asset creation request : %s %s", v.TraceID, contractPKH.String(), msg.AssetCode.String())

	// Asset Creation <- Asset Definition
	ac := protocol.AssetCreation{}

	err = node.Convert(ctx, &msg, &ac)
	if err != nil {
		return err
	}

	ac.Timestamp = v.Now

	// Convert to btcutil.Address
	contractAddress, err := btcutil.NewAddressPubKeyHash(contractPKH.Bytes(), &a.Config.ChainParams)
	if err != nil {
		return err
	}

	// Build outputs
	// 1 - Contract Address
	// 2 - Contract Fee (change)
	w.AddOutput(ctx, contractAddress, 0)
	w.AddContractFee(ctx, ct.ContractFee)

	// Save Tx.
	if err := transactions.AddTx(ctx, a.MasterDB, itx); err != nil {
		return errors.Wrap(err, "Failed to save tx")
	}

	// Respond with a formation
	return node.RespondSuccess(ctx, w, itx, rk, &ac)
}

// ModificationRequest handles an incoming Asset Modification and prepares a Creation response
func (a *Asset) ModificationRequest(ctx context.Context, w *node.ResponseWriter, itx *inspector.Transaction, rk *wallet.RootKey) error {
	ctx, span := trace.StartSpan(ctx, "handlers.Asset.Definition")
	defer span.End()

	msg, ok := itx.MsgProto.(*protocol.AssetModification)
	if !ok {
		return errors.New("Could not assert as *protocol.AssetModification")
	}

	v := ctx.Value(node.KeyValues).(*node.Values)

	// Validate all fields have valid values.
	if err := msg.Validate(); err != nil {
		logger.Warn(ctx, "%s : Asset modification request invalid : %s", v.TraceID, err)
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectMsgMalformed)
	}

	// Locate Asset
	contractPKH := protocol.PublicKeyHashFromBytes(rk.Address.ScriptAddress())
	ct, err := contract.Retrieve(ctx, a.MasterDB, contractPKH)
	if err != nil {
		return errors.Wrap(err, "Failed to retrieve contract")
	}

	requestorPKH := protocol.PublicKeyHashFromBytes(itx.Inputs[0].Address.ScriptAddress())
	if !contract.IsOperator(ctx, ct, requestorPKH) {
		logger.Verbose(ctx, "%s : Requestor is not operator : %s %s", v.TraceID, contractPKH.String(), msg.AssetCode.String())
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectNotOperator)
	}

	as, err := asset.Retrieve(ctx, a.MasterDB, contractPKH, &msg.AssetCode)
	if err != nil {
		return err
	}

	// Asset could not be found
	if as == nil {
		logger.Verbose(ctx, "%s : Asset ID not found: %s %s", v.TraceID, contractPKH.String(), msg.AssetCode.String())
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectAssetNotFound)
	}

	// Revision mismatch
	if as.Revision != msg.AssetRevision {
		logger.Verbose(ctx, "%s : Asset Revision does not match current: %s %s", v.TraceID, contractPKH.String(), msg.AssetCode.String())
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectAssetRevision)
	}

	// Check proposal if there was one
	proposed := false
	proposalInitiator := uint8(0)
	votingSystem := uint8(0)

	if !msg.RefTxID.IsZero() { // Vote Result Action allowing these amendments
		proposed = true

		refTxId, err := chainhash.NewHash(msg.RefTxID.Bytes())
		if err != nil {
			return errors.Wrap(err, "Failed to convert protocol.TxId to chainhash")
		}

		// Retrieve Vote Result
		voteResultTx, err := transactions.GetTx(ctx, a.MasterDB, refTxId, &a.Config.ChainParams)
		if err != nil {
			logger.Warn(ctx, "%s : Vote Result tx not found for amendment", v.TraceID)
			return node.RespondReject(ctx, w, itx, rk, protocol.RejectMsgMalformed)
		}

		voteResult, ok := voteResultTx.MsgProto.(*protocol.Result)
		if !ok {
			logger.Warn(ctx, "%s : Vote Result invalid for amendment", v.TraceID)
			return node.RespondReject(ctx, w, itx, rk, protocol.RejectMsgMalformed)
		}

		// Retrieve the vote
		vt, err := vote.Retrieve(ctx, a.MasterDB, contractPKH, &voteResult.VoteTxId)
		if err == vote.ErrNotFound {
			logger.Warn(ctx, "%s : Vote not found : %s", v.TraceID, voteResult.VoteTxId.String())
			return node.RespondReject(ctx, w, itx, rk, protocol.RejectVoteNotFound)
		} else if err != nil {
			logger.Warn(ctx, "%s : Failed to retrieve vote : %s : %s", v.TraceID, voteResult.VoteTxId.String(), err)
			return errors.Wrap(err, "Failed to retrieve vote")
		}

		if vt.CompletedAt.Nano() == 0 {
			logger.Warn(ctx, "%s : Vote not complete yet", v.TraceID)
			return node.RespondReject(ctx, w, itx, rk, protocol.RejectMsgMalformed)
		}

		if vt.Result != "A" {
			logger.Warn(ctx, "%s : Vote result not A(Accept)", v.TraceID)
			return node.RespondReject(ctx, w, itx, rk, protocol.RejectMsgMalformed)
		}

		if !vt.Specific {
			logger.Warn(ctx, "%s : Vote was not for specific amendments", v.TraceID)
			return node.RespondReject(ctx, w, itx, rk, protocol.RejectMsgMalformed)
		}

		if !vt.AssetSpecificVote || !bytes.Equal(msg.AssetCode.Bytes(), vt.AssetCode.Bytes()) {
			logger.Warn(ctx, "%s : Vote was not for this asset code", v.TraceID)
			return node.RespondReject(ctx, w, itx, rk, protocol.RejectMsgMalformed)
		}

		// Verify proposal amendments match these amendments.
		if len(voteResult.ProposedAmendments) != len(msg.Amendments) {
			logger.Warn(ctx, "%s : Proposal has different count of amendments : %d != %d",
				v.TraceID, len(voteResult.ProposedAmendments), len(msg.Amendments))
			return node.RespondReject(ctx, w, itx, rk, protocol.RejectMsgMalformed)
		}

		for i, amendment := range voteResult.ProposedAmendments {
			if !amendment.Equal(msg.Amendments[i]) {
				logger.Warn(ctx, "%s : Proposal amendment %d doesn't match", v.TraceID, i)
				return node.RespondReject(ctx, w, itx, rk, protocol.RejectMsgMalformed)
			}
		}

		proposalInitiator = vt.Initiator
		votingSystem = vt.VoteSystem
	}

	if err := checkAssetAmendmentsPermissions(as, ct.VotingSystems, msg.Amendments, proposed, proposalInitiator, votingSystem); err != nil {
		logger.Warn(ctx, "%s : Asset amendments not permitted : %s", v.TraceID, err)
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectAssetAuthFlags)
	}

	// Asset Creation <- Asset Modification
	ac := protocol.AssetCreation{}

	err = node.Convert(ctx, &as, &ac)
	if err != nil {
		return errors.Wrap(err, "Failed to convert state asset to asset creation")
	}

	ac.AssetRevision = as.Revision + 1
	ac.Timestamp = v.Now

	if err := applyAssetAmendments(&ac, ct.VotingSystems, msg.Amendments); err != nil {
		logger.Warn(ctx, "%s : Asset amendments failed : %s", v.TraceID, err)
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectMsgMalformed)
	}

	// Check issuer balance for token quantity reductions. Issuer has to hold any tokens being "burned".
	issuerBalance := asset.GetBalance(ctx, as, &ct.IssuerPKH)
	if ac.TokenQty < as.TokenQty && issuerBalance < as.TokenQty-ac.TokenQty {
		logger.Warn(ctx, "%s : Issuer doesn't hold required amount for token quantity reduction : %d < %d",
			v.TraceID, issuerBalance, as.TokenQty-ac.TokenQty)
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectMsgMalformed)
	}

	// Convert to btcutil.Address
	contractAddress, err := btcutil.NewAddressPubKeyHash(contractPKH.Bytes(), &a.Config.ChainParams)
	if err != nil {
		return err
	}

	// Build outputs
	// 1 - Contract Address
	// 2 - Contract Fee (change)
	w.AddOutput(ctx, contractAddress, 0)
	w.AddContractFee(ctx, ct.ContractFee)

	// Save Tx.
	if err := transactions.AddTx(ctx, a.MasterDB, itx); err != nil {
		return errors.Wrap(err, "Failed to save tx")
	}

	// Respond with a formation
	return node.RespondSuccess(ctx, w, itx, rk, &ac)
}

// CreationResponse handles an outgoing Asset Creation and writes it to the state
func (a *Asset) CreationResponse(ctx context.Context, w *node.ResponseWriter, itx *inspector.Transaction, rk *wallet.RootKey) error {
	ctx, span := trace.StartSpan(ctx, "handlers.Asset.Definition")
	defer span.End()

	msg, ok := itx.MsgProto.(*protocol.AssetCreation)
	if !ok {
		return errors.New("Could not assert as *protocol.AssetCreation")
	}

	v := ctx.Value(node.KeyValues).(*node.Values)

	// Locate Asset
	contractPKH := protocol.PublicKeyHashFromBytes(rk.Address.ScriptAddress())
	if !bytes.Equal(itx.Inputs[0].Address.ScriptAddress(), contractPKH.Bytes()) {
		return fmt.Errorf("Asset Creation not from contract : %x", itx.Inputs[0].Address.ScriptAddress())
	}

	ct, err := contract.Retrieve(ctx, a.MasterDB, contractPKH)
	if err != nil {
		return errors.Wrap(err, "Failed to retrieve contract")
	}

	as, err := asset.Retrieve(ctx, a.MasterDB, contractPKH, &msg.AssetCode)
	if err != nil && err != asset.ErrNotFound {
		return errors.Wrap(err, "Failed to retrieve asset")
	}

	// Get request tx
	request, err := transactions.GetTx(ctx, a.MasterDB, &itx.Inputs[0].UTXO.Hash, &a.Config.ChainParams)
	if err != nil {
		return errors.Wrap(err, "Failed to retrieve request tx")
	}
	var vt *state.Vote
	var modification *protocol.AssetModification
	if request != nil {
		var ok bool
		modification, ok = request.MsgProto.(*protocol.AssetModification)

		if ok && !modification.RefTxID.IsZero() {
			refTxId, err := chainhash.NewHash(modification.RefTxID.Bytes())
			if err != nil {
				return errors.Wrap(err, "Failed to convert protocol.TxId to chainhash")
			}

			// Retrieve Vote Result
			voteResultTx, err := transactions.GetTx(ctx, a.MasterDB, refTxId, &a.Config.ChainParams)
			if err != nil {
				return errors.Wrap(err, "Failed to retrieve vote result tx")
			}

			voteResult, ok := voteResultTx.MsgProto.(*protocol.Result)
			if !ok {
				return errors.New("Vote Result invalid for modification")
			}

			// Retrieve the vote
			vt, err = vote.Retrieve(ctx, a.MasterDB, contractPKH, &voteResult.VoteTxId)
			if err == vote.ErrNotFound {
				return errors.New("Vote not found for modification")
			} else if err != nil {
				return errors.New("Failed to retrieve vote for modification")
			}
		}
	}

	// Create or update Asset
	if as == nil {
		// Prepare creation object
		na := asset.NewAsset{}

		if err = node.Convert(ctx, &msg, &na); err != nil {
			return err
		}

		if err := contract.AddAssetCode(ctx, a.MasterDB, contractPKH, &msg.AssetCode, v.Now); err != nil {
			return err
		}

		if err := asset.Create(ctx, a.MasterDB, contractPKH, &msg.AssetCode, &na, v.Now); err != nil {
			return errors.Wrap(err, "Failed to create asset")
		}
		logger.Info(ctx, "%s : Created asset : %s %s", v.TraceID, contractPKH.String(), msg.AssetCode.String())
	} else {
		// Required pointers
		stringPointer := func(s string) *string { return &s }

		// Prepare update object
		ua := asset.UpdateAsset{
			Revision:  &msg.AssetRevision,
			Timestamp: &msg.Timestamp,
		}

		if as.AssetType != string(msg.AssetType) {
			ua.AssetType = stringPointer(string(msg.AssetType))
			logger.Info(ctx, "%s : Updating asset type (%s) : %s", v.TraceID, msg.AssetCode.String(), *ua.AssetType)
		}
		if !bytes.Equal(as.AssetAuthFlags[:], msg.AssetAuthFlags[:]) {
			ua.AssetAuthFlags = &msg.AssetAuthFlags
			logger.Info(ctx, "%s : Updating asset auth flags (%s) : %s", v.TraceID, msg.AssetCode.String(), *ua.AssetAuthFlags)
		}
		if as.TransfersPermitted != msg.TransfersPermitted {
			ua.TransfersPermitted = &msg.TransfersPermitted
			logger.Info(ctx, "%s : Updating asset transfers permitted (%s) : %t", v.TraceID, msg.AssetCode.String(), *ua.TransfersPermitted)
		}
		if as.EnforcementOrdersPermitted != msg.EnforcementOrdersPermitted {
			ua.EnforcementOrdersPermitted = &msg.EnforcementOrdersPermitted
			logger.Info(ctx, "%s : Updating asset enforcement orders permitted (%s) : %t", v.TraceID, msg.AssetCode.String(), *ua.EnforcementOrdersPermitted)
		}
		if as.VoteMultiplier != msg.VoteMultiplier {
			ua.VoteMultiplier = &msg.VoteMultiplier
			logger.Info(ctx, "%s : Updating asset vote multiplier (%s) : %02x", v.TraceID, msg.AssetCode.String(), *ua.VoteMultiplier)
		}
		if as.IssuerProposal != msg.IssuerProposal {
			ua.IssuerProposal = &msg.IssuerProposal
			logger.Info(ctx, "%s : Updating asset issuer proposal (%s) : %t", v.TraceID, msg.AssetCode.String(), *ua.IssuerProposal)
		}
		if as.HolderProposal != msg.HolderProposal {
			ua.HolderProposal = &msg.HolderProposal
			logger.Info(ctx, "%s : Updating asset holder proposal (%s) : %t", v.TraceID, msg.AssetCode.String(), *ua.HolderProposal)
		}
		if as.AssetModificationGovernance != msg.AssetModificationGovernance {
			ua.AssetModificationGovernance = &msg.AssetModificationGovernance
			logger.Info(ctx, "%s : Updating asset modification governance (%s) : %t", v.TraceID, msg.AssetCode.String(), *ua.AssetModificationGovernance)
		}
		if as.TokenQty != msg.TokenQty {
			ua.TokenQty = &msg.TokenQty
			logger.Info(ctx, "%s : Updating asset token quantity (%s) : %d", v.TraceID, msg.AssetCode.String(), *ua.TokenQty)

			issuerBalance := asset.GetBalance(ctx, as, &ct.IssuerPKH)
			if msg.TokenQty > as.TokenQty {
				// Increasing token quantity. Give tokens to issuer.
				issuerBalance += msg.TokenQty - as.TokenQty
			} else {
				// Increasing token quantity. Take tokens from issuer.
				issuerBalance -= as.TokenQty - msg.TokenQty
			}
			ua.NewBalances = make(map[protocol.PublicKeyHash]uint64)
			ua.NewBalances[ct.IssuerPKH] = issuerBalance
		}
		if !bytes.Equal(as.AssetPayload, msg.AssetPayload) {
			ua.AssetPayload = &msg.AssetPayload
			logger.Info(ctx, "%s : Updating asset payload (%s) : %s", v.TraceID, msg.AssetCode.String(), *ua.AssetPayload)
		}

		// Check if trade restrictions are different
		different := len(as.TradeRestrictions) != len(msg.TradeRestrictions)
		if !different {
			for i, tradeRestriction := range as.TradeRestrictions {
				if !bytes.Equal(tradeRestriction.Bytes(), msg.TradeRestrictions[i].Bytes()) {
					different = true
					break
				}
			}
		}

		if different {
			ua.TradeRestrictions = &msg.TradeRestrictions
		}

		if err := asset.Update(ctx, a.MasterDB, contractPKH, &msg.AssetCode, &ua, v.Now); err != nil {
			logger.Warn(ctx, "%s : Failed to update asset : %s %s", v.TraceID, contractPKH.String(), msg.AssetCode.String())
			return err
		}
		logger.Info(ctx, "%s : Updated asset : %s %s", v.TraceID, contractPKH, msg.AssetCode.String())

		// Mark vote as "applied" if this amendment was a result of a vote.
		if vt != nil {
			uv := vote.UpdateVote{AppliedTxId: protocol.TxIdFromBytes(request.Hash[:])}
			if err := vote.Update(ctx, a.MasterDB, contractPKH, &vt.VoteTxId, &uv, v.Now); err != nil {
				return errors.Wrap(err, "Failed to update vote")
			}
		}
	}

	return nil
}

// checkAssetAmendmentsPermissions verifies that the amendments are permitted based on the auth flags.
func checkAssetAmendmentsPermissions(as *state.Asset, votingSystems []protocol.VotingSystem,
	amendments []protocol.Amendment, proposed bool, proposalInitiator, votingSystem uint8) error {

	permissions, err := protocol.ReadAuthFlags(as.AssetAuthFlags, contract.FieldCount, len(votingSystems))
	if err != nil {
		return fmt.Errorf("Invalid asset auth flags : %s", err)
	}

	for _, amendment := range amendments {
		if int(amendment.FieldIndex) >= len(permissions) {
			return fmt.Errorf("Amendment field index out of range : %d", amendment.FieldIndex)
		}
		if proposed {
			switch proposalInitiator {
			case 0: // Issuer
				if !permissions[amendment.FieldIndex].IssuerProposal {
					return fmt.Errorf("Field %d amendment not permitted by issuer proposal", amendment.FieldIndex)
				}
			case 1: // Holder
				if !permissions[amendment.FieldIndex].HolderProposal {
					return fmt.Errorf("Field %d amendment not permitted by holder proposal", amendment.FieldIndex)
				}
			default:
				return fmt.Errorf("Invalid proposal initiator type : %d", proposalInitiator)
			}

			if int(votingSystem) >= len(permissions[amendment.FieldIndex].VotingSystemsAllowed) {
				return fmt.Errorf("Field %d amendment voting system out of range : %d", amendment.FieldIndex, votingSystem)
			}
			if !permissions[amendment.FieldIndex].VotingSystemsAllowed[votingSystem] {
				return fmt.Errorf("Field %d amendment not allowed using voting system %d", amendment.FieldIndex, votingSystem)
			}
		} else if !permissions[amendment.FieldIndex].Permitted {
			return fmt.Errorf("Field %d amendment not permitted without proposal", amendment.FieldIndex)
		}
	}

	return nil
}

func applyAssetAmendments(ac *protocol.AssetCreation, votingSystems []protocol.VotingSystem, amendments []protocol.Amendment) error {
	authFieldsUpdated := false
	for _, amendment := range amendments {
		switch amendment.FieldIndex {
		case 0: // AssetType
			return fmt.Errorf("Asset amendment attempting to update asset type")

		case 1: // AssetCode
			return fmt.Errorf("Asset amendment attempting to update asset code")

		case 2: // AssetAuthFlags
			ac.AssetAuthFlags = amendment.Data
			authFieldsUpdated = true

		case 3: // TransfersPermitted
			if len(amendment.Data) != 1 {
				return fmt.Errorf("TransfersPermitted amendment value is wrong size : %d", len(amendment.Data))
			}
			buf := bytes.NewBuffer(amendment.Data)
			if err := binary.Read(buf, protocol.DefaultEndian, &ac.TransfersPermitted); err != nil {
				return fmt.Errorf("TransfersPermitted amendment value failed to deserialize : %s", err)
			}

		case 4: // TradeRestrictions
			switch amendment.Operation {
			case 0: // Modify
				if int(amendment.Element) >= len(ac.TradeRestrictions) {
					return fmt.Errorf("Asset amendment element out of range for TradeRestrictions : %d",
						amendment.Element)
				}

				buf := bytes.NewBuffer(amendment.Data)
				if err := ac.TradeRestrictions[amendment.Element].Write(buf); err != nil {
					return fmt.Errorf("Contract amendment addition to TradeRestrictions failed to deserialize : %s",
						err)
				}

			case 1: // Add element
				newPolity := protocol.Polity{}
				buf := bytes.NewBuffer(amendment.Data)
				if err := newPolity.Write(buf); err != nil {
					return fmt.Errorf("Contract amendment addition to TradeRestrictions failed to deserialize : %s",
						err)
				}

				ac.TradeRestrictions = append(ac.TradeRestrictions, newPolity)

			case 2: // Delete element
				if int(amendment.Element) >= len(ac.TradeRestrictions) {
					return fmt.Errorf("Asset amendment element out of range for TradeRestrictions : %d",
						amendment.Element)
				}
				ac.TradeRestrictions = append(ac.TradeRestrictions[:amendment.Element],
					ac.TradeRestrictions[amendment.Element+1:]...)

			default:
				return fmt.Errorf("Invalid asset amendment operation for TradeRestrictions : %d", amendment.Operation)
			}

		case 5: // EnforcementOrdersPermitted
			if len(amendment.Data) != 1 {
				return fmt.Errorf("EnforcementOrdersPermitted amendment value is wrong size : %d", len(amendment.Data))
			}
			buf := bytes.NewBuffer(amendment.Data)
			if err := binary.Read(buf, protocol.DefaultEndian, &ac.EnforcementOrdersPermitted); err != nil {
				return fmt.Errorf("EnforcementOrdersPermitted amendment value failed to deserialize : %s", err)
			}

		case 6: // VotingRights
			if len(amendment.Data) != 1 {
				return fmt.Errorf("VotingRights amendment value is wrong size : %d", len(amendment.Data))
			}
			buf := bytes.NewBuffer(amendment.Data)
			if err := binary.Read(buf, protocol.DefaultEndian, &ac.VotingRights); err != nil {
				return fmt.Errorf("VotingRights amendment value failed to deserialize : %s", err)
			}

		case 7: // VoteMultiplier
			if len(amendment.Data) != 1 {
				return fmt.Errorf("VoteMultiplier amendment value is wrong size : %d", len(amendment.Data))
			}
			buf := bytes.NewBuffer(amendment.Data)
			if err := binary.Read(buf, protocol.DefaultEndian, &ac.VoteMultiplier); err != nil {
				return fmt.Errorf("VoteMultiplier amendment value failed to deserialize : %s", err)
			}

		case 8: // IssuerProposal
			if len(amendment.Data) != 1 {
				return fmt.Errorf("IssuerProposal amendment value is wrong size : %d", len(amendment.Data))
			}
			buf := bytes.NewBuffer(amendment.Data)
			if err := binary.Read(buf, protocol.DefaultEndian, &ac.IssuerProposal); err != nil {
				return fmt.Errorf("IssuerProposal amendment value failed to deserialize : %s", err)
			}

		case 9: // HolderProposal
			if len(amendment.Data) != 1 {
				return fmt.Errorf("HolderProposal amendment value is wrong size : %d", len(amendment.Data))
			}
			buf := bytes.NewBuffer(amendment.Data)
			if err := binary.Read(buf, protocol.DefaultEndian, &ac.HolderProposal); err != nil {
				return fmt.Errorf("HolderProposal amendment value failed to deserialize : %s", err)
			}

		case 10: // AssetModificationGovernance
			if len(amendment.Data) != 1 {
				return fmt.Errorf("AssetModificationGovernance amendment value is wrong size : %d", len(amendment.Data))
			}
			buf := bytes.NewBuffer(amendment.Data)
			if err := binary.Read(buf, protocol.DefaultEndian, &ac.AssetModificationGovernance); err != nil {
				return fmt.Errorf("AssetModificationGovernance amendment value failed to deserialize : %s", err)
			}

		case 11: // TokenQty
			if len(amendment.Data) != 8 {
				return fmt.Errorf("TokenQty amendment value is wrong size : %d", len(amendment.Data))
			}
			buf := bytes.NewBuffer(amendment.Data)
			if err := binary.Read(buf, protocol.DefaultEndian, &ac.TokenQty); err != nil {
				return fmt.Errorf("TokenQty amendment value failed to deserialize : %s", err)
			}

		case 12: // AssetPayload
			// Validate payload
			payload := protocol.AssetTypeMapping(ac.AssetType)
			if payload == nil {
				return fmt.Errorf("Asset payload type unknown : %s", ac.AssetType)
			}

			_, err := payload.Write(amendment.Data)
			if err != nil {
				return fmt.Errorf("AssetPayload amendment value failed to deserialize : %s", err)
			}

		default:
			return fmt.Errorf("Asset amendment field offset out of range : %d", amendment.FieldIndex)
		}
	}

	if authFieldsUpdated {
		if _, err := protocol.ReadAuthFlags(ac.AssetAuthFlags, contract.FieldCount, len(votingSystems)); err != nil {
			return fmt.Errorf("Invalid asset auth flags : %s", err)
		}
	}

	// Check validity of updated asset data
	if err := ac.Validate(); err != nil {
		return fmt.Errorf("Asset data invalid after amendments : %s", err)
	}

	return nil
}
