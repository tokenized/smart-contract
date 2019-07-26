package handlers

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"

	"github.com/tokenized/smart-contract/internal/asset"
	"github.com/tokenized/smart-contract/internal/contract"
	"github.com/tokenized/smart-contract/internal/holdings"
	"github.com/tokenized/smart-contract/internal/platform/db"
	"github.com/tokenized/smart-contract/internal/platform/node"
	"github.com/tokenized/smart-contract/internal/platform/state"
	"github.com/tokenized/smart-contract/internal/transactions"
	"github.com/tokenized/smart-contract/internal/vote"
	"github.com/tokenized/smart-contract/pkg/bitcoin"
	"github.com/tokenized/smart-contract/pkg/inspector"
	"github.com/tokenized/smart-contract/pkg/wallet"
	"github.com/tokenized/specification/dist/golang/protocol"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/pkg/errors"
	"go.opencensus.io/trace"
)

type Asset struct {
	MasterDB *db.DB
	Config   *node.Config
}

// DefinitionRequest handles an incoming Asset Definition and prepares a Creation response
func (a *Asset) DefinitionRequest(ctx context.Context, w *node.ResponseWriter, itx *inspector.Transaction, rk *wallet.Key) error {
	ctx, span := trace.StartSpan(ctx, "handlers.Asset.Definition")
	defer span.End()

	msg, ok := itx.MsgProto.(*protocol.AssetDefinition)
	if !ok {
		return errors.New("Could not assert as *protocol.AssetDefinition")
	}

	v := ctx.Value(node.KeyValues).(*node.Values)

	// Validate all fields have valid values.
	if itx.RejectCode != 0 {
		node.LogWarn(ctx, "Asset definition invalid")
		return node.RespondReject(ctx, w, itx, rk, itx.RejectCode)
	}

	// Locate Contract
	contractPKH := protocol.PublicKeyHashFromBytes(bitcoin.Hash160(rk.Key.PublicKey().Bytes()))
	ct, err := contract.Retrieve(ctx, a.MasterDB, contractPKH)
	if err != nil {
		return errors.Wrap(err, "Failed to retrieve contract")
	}

	if !ct.MovedTo.IsZero() {
		node.LogWarn(ctx, "Contract address changed : %s", ct.MovedTo.String())
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectContractMoved)
	}

	if ct.FreezePeriod.Nano() > v.Now.Nano() {
		node.LogWarn(ctx, "Contract frozen : %s", ct.FreezePeriod.String())
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectContractFrozen)
	}

	if ct.ContractExpiration.Nano() != 0 && ct.ContractExpiration.Nano() < v.Now.Nano() {
		node.LogWarn(ctx, "Contract expired : %s", ct.ContractExpiration.String())
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectContractExpired)
	}

	if _, err = protocol.ReadAuthFlags(msg.AssetAuthFlags, asset.FieldCount, len(ct.VotingSystems)); err != nil {
		node.LogWarn(ctx, "Invalid asset auth flags : %s", err)
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectMsgMalformed)
	}

	// Verify administration is sender of tx.
	addressPKH, ok := bitcoin.PKH(itx.Inputs[0].Address)
	if !ok || !bytes.Equal(addressPKH, ct.AdministrationPKH.Bytes()) {
		if ok {
			node.LogWarn(ctx, "Only administration can create assets: %x", addressPKH)
		} else {
			node.LogWarn(ctx, "Only administration can create assets")
		}
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectNotAdministration)
	}

	// Generate Asset ID
	assetCode := protocol.AssetCodeFromContract(contractPKH.Bytes(), uint64(len(ct.AssetCodes)))

	// Locate Asset
	_, err = asset.Retrieve(ctx, a.MasterDB, contractPKH, assetCode)
	if err != asset.ErrNotFound {
		if err == nil {
			node.LogWarn(ctx, "Asset already exists : %s", assetCode.String())
			return node.RespondReject(ctx, w, itx, rk, protocol.RejectAssetCodeExists)
		} else {
			return errors.Wrap(err, "Failed to retrieve asset")
		}
	}

	// Allowed to have more assets
	if !contract.CanHaveMoreAssets(ctx, ct) {
		node.LogWarn(ctx, "Number of assets exceeds contract Qty: %s %s", contractPKH.String(), assetCode.String())
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectContractFixedQuantity)
	}

	// Validate payload
	payload := protocol.AssetTypeMapping(msg.AssetType)
	if payload == nil {
		node.LogWarn(ctx, "Asset payload type unknown : %s", msg.AssetType)
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectMsgMalformed)
	}

	_, err = payload.Write(msg.AssetPayload)
	if err != nil {
		node.LogWarn(ctx, "Failed to parse asset payload : %s", err)
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectMsgMalformed)
	}

	if err := payload.Validate(); err != nil {
		node.LogWarn(ctx, "Asset %s payload is invalid : %s", msg.AssetType, err)
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectMsgMalformed)
	}

	node.Log(ctx, "Accepting asset creation request : %s %s", contractPKH.String(), assetCode.String())

	// Asset Creation <- Asset Definition
	ac := protocol.AssetCreation{}

	err = node.Convert(ctx, &msg, &ac)
	if err != nil {
		return err
	}

	ac.Timestamp = v.Now
	ac.AssetCode = *assetCode
	ac.AssetIndex = uint64(len(ct.AssetCodes))

	// Convert to bitcoin.ScriptTemplate
	contractAddress, err := bitcoin.NewScriptTemplatePKH(contractPKH.Bytes())
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
func (a *Asset) ModificationRequest(ctx context.Context, w *node.ResponseWriter, itx *inspector.Transaction, rk *wallet.Key) error {
	ctx, span := trace.StartSpan(ctx, "handlers.Asset.Definition")
	defer span.End()

	msg, ok := itx.MsgProto.(*protocol.AssetModification)
	if !ok {
		return errors.New("Could not assert as *protocol.AssetModification")
	}

	v := ctx.Value(node.KeyValues).(*node.Values)

	// Validate all fields have valid values.
	if itx.RejectCode != 0 {
		node.LogWarn(ctx, "Asset modification request invalid")
		return node.RespondReject(ctx, w, itx, rk, itx.RejectCode)
	}

	// Locate Asset
	contractPKH := protocol.PublicKeyHashFromBytes(bitcoin.Hash160(rk.Key.PublicKey().Bytes()))
	ct, err := contract.Retrieve(ctx, a.MasterDB, contractPKH)
	if err != nil {
		return errors.Wrap(err, "Failed to retrieve contract")
	}

	if !ct.MovedTo.IsZero() {
		node.LogWarn(ctx, "Contract address changed : %s", ct.MovedTo.String())
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectContractMoved)
	}

	requestorAddressPKH, ok := bitcoin.PKH(itx.Inputs[0].Address)
	if !ok {
		node.LogVerbose(ctx, "Requestor is not PKH : %s", msg.AssetCode.String())
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectNotOperator)
	}
	requestorPKH := protocol.PublicKeyHashFromBytes(requestorAddressPKH)
	if !contract.IsOperator(ctx, ct, requestorPKH) {
		node.LogVerbose(ctx, "Requestor is not operator : %s", msg.AssetCode.String())
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectNotOperator)
	}

	as, err := asset.Retrieve(ctx, a.MasterDB, contractPKH, &msg.AssetCode)
	if err != nil {
		return errors.Wrap(err, "Failed to retrieve asset")
	}

	// Asset could not be found
	if as == nil {
		node.LogVerbose(ctx, "Asset ID not found: %s", msg.AssetCode.String())
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectAssetNotFound)
	}

	// Revision mismatch
	if as.Revision != msg.AssetRevision {
		node.LogVerbose(ctx, "Asset Revision does not match current: %s", msg.AssetCode.String())
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
		voteResultTx, err := transactions.GetTx(ctx, a.MasterDB, refTxId, a.Config.IsTest)
		if err != nil {
			node.LogWarn(ctx, "Vote Result tx not found for amendment")
			return node.RespondReject(ctx, w, itx, rk, protocol.RejectMsgMalformed)
		}

		voteResult, ok := voteResultTx.MsgProto.(*protocol.Result)
		if !ok {
			node.LogWarn(ctx, "Vote Result invalid for amendment")
			return node.RespondReject(ctx, w, itx, rk, protocol.RejectMsgMalformed)
		}

		// Retrieve the vote
		vt, err := vote.Retrieve(ctx, a.MasterDB, contractPKH, &voteResult.VoteTxId)
		if err == vote.ErrNotFound {
			node.LogWarn(ctx, "Vote not found : %s", voteResult.VoteTxId.String())
			return node.RespondReject(ctx, w, itx, rk, protocol.RejectVoteNotFound)
		} else if err != nil {
			node.LogWarn(ctx, "Failed to retrieve vote : %s : %s", voteResult.VoteTxId.String(), err)
			return errors.Wrap(err, "Failed to retrieve vote")
		}

		if vt.CompletedAt.Nano() == 0 {
			node.LogWarn(ctx, "Vote not complete yet")
			return node.RespondReject(ctx, w, itx, rk, protocol.RejectMsgMalformed)
		}

		if vt.Result != "A" {
			node.LogWarn(ctx, "Vote result not A(Accept)")
			return node.RespondReject(ctx, w, itx, rk, protocol.RejectMsgMalformed)
		}

		if !vt.Specific {
			node.LogWarn(ctx, "Vote was not for specific amendments")
			return node.RespondReject(ctx, w, itx, rk, protocol.RejectMsgMalformed)
		}

		if !vt.AssetSpecificVote || !bytes.Equal(msg.AssetCode.Bytes(), vt.AssetCode.Bytes()) {
			node.LogWarn(ctx, "Vote was not for this asset code")
			return node.RespondReject(ctx, w, itx, rk, protocol.RejectMsgMalformed)
		}

		// Verify proposal amendments match these amendments.
		if len(voteResult.ProposedAmendments) != len(msg.Amendments) {
			node.LogWarn(ctx, "%s : Proposal has different count of amendments : %d != %d",
				v.TraceID, len(voteResult.ProposedAmendments), len(msg.Amendments))
			return node.RespondReject(ctx, w, itx, rk, protocol.RejectMsgMalformed)
		}

		for i, amendment := range voteResult.ProposedAmendments {
			if !amendment.Equal(msg.Amendments[i]) {
				node.LogWarn(ctx, "Proposal amendment %d doesn't match", i)
				return node.RespondReject(ctx, w, itx, rk, protocol.RejectMsgMalformed)
			}
		}

		proposalInitiator = vt.Initiator
		votingSystem = vt.VoteSystem
	}

	if err := checkAssetAmendmentsPermissions(as, ct.VotingSystems, msg.Amendments, proposed, proposalInitiator, votingSystem); err != nil {
		node.LogWarn(ctx, "Asset amendments not permitted : %s", err)
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectAssetAuthFlags)
	}

	// Asset Creation <- Asset Modification
	ac := protocol.AssetCreation{}

	err = node.Convert(ctx, as, &ac)
	if err != nil {
		return errors.Wrap(err, "Failed to convert state asset to asset creation")
	}

	ac.AssetRevision = as.Revision + 1
	ac.Timestamp = v.Now
	ac.AssetCode = msg.AssetCode // Asset code not in state data

	node.Log(ctx, "Amending asset : %s", msg.AssetCode.String())

	if err := applyAssetAmendments(&ac, ct.VotingSystems, msg.Amendments); err != nil {
		node.LogWarn(ctx, "Asset amendments failed : %s", err)
		return node.RespondReject(ctx, w, itx, rk, protocol.RejectMsgMalformed)
	}

	var h state.Holding
	updateHoldings := false
	if ac.TokenQty != as.TokenQty {
		updateHoldings = true

		// Check administration balance for token quantity reductions. Administration has to hold any tokens being "burned".
		h, err = holdings.GetHolding(ctx, a.MasterDB, contractPKH, &msg.AssetCode, &ct.AdministrationPKH, v.Now)
		if err != nil {
			return errors.Wrap(err, "Failed to get admin holding")
		}

		txid := protocol.TxIdFromBytes(itx.Hash[:])

		if ac.TokenQty < as.TokenQty {
			if err := holdings.AddDebit(&h, txid, as.TokenQty-ac.TokenQty, v.Now); err != nil {
				node.LogWarn(ctx, "%s : Failed to reduce administration holdings : %s", v.TraceID, err)
				if err == holdings.ErrInsufficientHoldings {
					return node.RespondReject(ctx, w, itx, rk, protocol.RejectInsufficientQuantity)
				} else {
					return errors.Wrap(err, "Failed to reduce holdings")
				}
			}
		} else {
			if err := holdings.AddDeposit(&h, txid, ac.TokenQty-as.TokenQty, v.Now); err != nil {
				node.LogWarn(ctx, "%s : Failed to increase administration holdings : %s", v.TraceID, err)
				return errors.Wrap(err, "Failed to increase holdings")
			}
		}
	}

	// Convert to bitcoin.ScriptTemplate
	contractAddress, err := bitcoin.NewScriptTemplatePKH(contractPKH.Bytes())
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
	if err := node.RespondSuccess(ctx, w, itx, rk, &ac); err != nil {
		return errors.Wrap(err, "Failed to respond")
	}

	if updateHoldings {
		if err := holdings.Save(ctx, a.MasterDB, contractPKH, &msg.AssetCode, &h); err != nil {
			return errors.Wrap(err, "Failed to save holdings")
		}
	}

	return nil
}

// CreationResponse handles an outgoing Asset Creation and writes it to the state
func (a *Asset) CreationResponse(ctx context.Context, w *node.ResponseWriter, itx *inspector.Transaction, rk *wallet.Key) error {
	ctx, span := trace.StartSpan(ctx, "handlers.Asset.Definition")
	defer span.End()

	msg, ok := itx.MsgProto.(*protocol.AssetCreation)
	if !ok {
		return errors.New("Could not assert as *protocol.AssetCreation")
	}

	v := ctx.Value(node.KeyValues).(*node.Values)

	if itx.RejectCode != 0 {
		return errors.New("Asset creation response invalid")
	}

	// Locate Asset
	contractPKH := protocol.PublicKeyHashFromBytes(bitcoin.Hash160(rk.Key.PublicKey().Bytes()))
	if !itx.Inputs[0].Address.Equal(rk.Address) {
		return fmt.Errorf("Asset Creation not from contract : %x",
			itx.Inputs[0].Address.Bytes())
	}

	ct, err := contract.Retrieve(ctx, a.MasterDB, contractPKH)
	if err != nil {
		return errors.Wrap(err, "Failed to retrieve contract")
	}

	if !ct.MovedTo.IsZero() {
		return fmt.Errorf("Contract address changed : %s", ct.MovedTo.String())
	}

	as, err := asset.Retrieve(ctx, a.MasterDB, contractPKH, &msg.AssetCode)
	if err != nil && err != asset.ErrNotFound {
		return errors.Wrap(err, "Failed to retrieve asset")
	}

	// Get request tx
	request, err := transactions.GetTx(ctx, a.MasterDB, &itx.Inputs[0].UTXO.Hash, a.Config.IsTest)
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
			voteResultTx, err := transactions.GetTx(ctx, a.MasterDB, refTxId, a.Config.IsTest)
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

		na.AdministrationPKH = ct.AdministrationPKH

		if err := contract.AddAssetCode(ctx, a.MasterDB, contractPKH, &msg.AssetCode, v.Now); err != nil {
			return err
		}

		if err := asset.Create(ctx, a.MasterDB, contractPKH, &msg.AssetCode, &na, v.Now); err != nil {
			return errors.Wrap(err, "Failed to create asset")
		}
		node.Log(ctx, "Created asset : %s", msg.AssetCode.String())

		// Update administration balance
		h, err := holdings.GetHolding(ctx, a.MasterDB, contractPKH, &msg.AssetCode,
			&ct.AdministrationPKH, v.Now)
		if err != nil {
			return errors.Wrap(err, "Failed to get admin holding")
		}
		txid := protocol.TxIdFromBytes(itx.Hash[:])
		holdings.AddDeposit(&h, txid, msg.TokenQty, msg.Timestamp)
		holdings.FinalizeTx(&h, txid, msg.Timestamp)
		if err := holdings.Save(ctx, a.MasterDB, contractPKH, &msg.AssetCode, &h); err != nil {
			return errors.Wrap(err, "Failed to save holdings")
		}
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
			node.Log(ctx, "Updating asset type (%s) : %s", msg.AssetCode.String(), *ua.AssetType)
		}
		if !bytes.Equal(as.AssetAuthFlags[:], msg.AssetAuthFlags[:]) {
			ua.AssetAuthFlags = &msg.AssetAuthFlags
			node.Log(ctx, "Updating asset auth flags (%s) : %x", msg.AssetCode.String(), *ua.AssetAuthFlags)
		}
		if as.TransfersPermitted != msg.TransfersPermitted {
			ua.TransfersPermitted = &msg.TransfersPermitted
			node.Log(ctx, "Updating asset transfers permitted (%s) : %t", msg.AssetCode.String(), *ua.TransfersPermitted)
		}
		if as.EnforcementOrdersPermitted != msg.EnforcementOrdersPermitted {
			ua.EnforcementOrdersPermitted = &msg.EnforcementOrdersPermitted
			node.Log(ctx, "Updating asset enforcement orders permitted (%s) : %t", msg.AssetCode.String(), *ua.EnforcementOrdersPermitted)
		}
		if as.VoteMultiplier != msg.VoteMultiplier {
			ua.VoteMultiplier = &msg.VoteMultiplier
			node.Log(ctx, "Updating asset vote multiplier (%s) : %02x", msg.AssetCode.String(), *ua.VoteMultiplier)
		}
		if as.AdministrationProposal != msg.AdministrationProposal {
			ua.AdministrationProposal = &msg.AdministrationProposal
			node.Log(ctx, "Updating asset administration proposal (%s) : %t", msg.AssetCode.String(), *ua.AdministrationProposal)
		}
		if as.HolderProposal != msg.HolderProposal {
			ua.HolderProposal = &msg.HolderProposal
			node.Log(ctx, "Updating asset holder proposal (%s) : %t", msg.AssetCode.String(), *ua.HolderProposal)
		}
		if as.AssetModificationGovernance != msg.AssetModificationGovernance {
			ua.AssetModificationGovernance = &msg.AssetModificationGovernance
			node.Log(ctx, "Updating asset modification governance (%s) : %d", msg.AssetCode.String(), *ua.AssetModificationGovernance)
		}

		var h state.Holding
		updateHoldings := false
		if as.TokenQty != msg.TokenQty {
			ua.TokenQty = &msg.TokenQty
			node.Log(ctx, "Updating asset token quantity %d : %s", *ua.TokenQty, msg.AssetCode.String())

			h, err = holdings.GetHolding(ctx, a.MasterDB, contractPKH, &msg.AssetCode,
				&ct.AdministrationPKH, v.Now)
			if err != nil {
				return errors.Wrap(err, "Failed to get admin holding")
			}

			txid := protocol.TxIdFromBytes(itx.Hash[:])

			holdings.FinalizeTx(&h, txid, msg.Timestamp)
			updateHoldings = true

			if msg.TokenQty > as.TokenQty {
				node.Log(ctx, "Increasing token quantity by %d to %d : %s", msg.TokenQty-as.TokenQty, *ua.TokenQty, msg.AssetCode.String())
			} else {
				node.Log(ctx, "Decreasing token quantity by %d to %d : %s", as.TokenQty-msg.TokenQty, *ua.TokenQty, msg.AssetCode.String())
			}
			if err != nil {
				node.LogWarn(ctx, "Failed to update administration holding : %s", msg.AssetCode.String())
				return err
			}
		}
		if !bytes.Equal(as.AssetPayload, msg.AssetPayload) {
			ua.AssetPayload = &msg.AssetPayload
			node.Log(ctx, "Updating asset payload (%s) : %s", msg.AssetCode.String(), *ua.AssetPayload)
		}

		// Check if trade restrictions are different
		different := len(as.TradeRestrictions) != len(msg.TradeRestrictions)
		if !different {
			for i, tradeRestriction := range as.TradeRestrictions {
				if !bytes.Equal(tradeRestriction[:], msg.TradeRestrictions[i][:]) {
					different = true
					break
				}
			}
		}

		if different {
			ua.TradeRestrictions = &msg.TradeRestrictions
		}

		if updateHoldings {
			if err := holdings.Save(ctx, a.MasterDB, contractPKH, &msg.AssetCode, &h); err != nil {
				return errors.Wrap(err, "Failed to save holdings")
			}
		}
		if err := asset.Update(ctx, a.MasterDB, contractPKH, &msg.AssetCode, &ua, v.Now); err != nil {
			node.LogWarn(ctx, "Failed to update asset : %s", msg.AssetCode.String())
			return err
		}
		node.Log(ctx, "Updated asset : %s", msg.AssetCode.String())

		// Mark vote as "applied" if this amendment was a result of a vote.
		if vt != nil {
			node.Log(ctx, "Marking vote as applied : %s", vt.VoteTxId.String())
			if err := vote.MarkApplied(ctx, a.MasterDB, contractPKH, &vt.VoteTxId, protocol.TxIdFromBytes(request.Hash[:]), v.Now); err != nil {
				return errors.Wrap(err, "Failed to mark vote applied")
			}
		}
	}

	return nil
}

// checkAssetAmendmentsPermissions verifies that the amendments are permitted based on the auth flags.
func checkAssetAmendmentsPermissions(as *state.Asset, votingSystems []protocol.VotingSystem,
	amendments []protocol.Amendment, proposed bool, proposalInitiator, votingSystem uint8) error {

	permissions, err := protocol.ReadAuthFlags(as.AssetAuthFlags, asset.FieldCount, len(votingSystems))
	if err != nil {
		return fmt.Errorf("Invalid asset auth flags : %s", err)
	}

	for _, amendment := range amendments {
		if int(amendment.FieldIndex) >= len(permissions) {
			return fmt.Errorf("Amendment field index out of range : %d", amendment.FieldIndex)
		}
		if proposed {
			switch proposalInitiator {
			case 0: // Administration
				if !permissions[amendment.FieldIndex].AdministrationProposal {
					return fmt.Errorf("Field %d amendment not permitted by administration proposal", amendment.FieldIndex)
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

		case 1: // AssetAuthFlags
			ac.AssetAuthFlags = amendment.Data
			authFieldsUpdated = true

		case 2: // TransfersPermitted
			if len(amendment.Data) != 1 {
				return fmt.Errorf("TransfersPermitted amendment value is wrong size : %d", len(amendment.Data))
			}
			buf := bytes.NewBuffer(amendment.Data)
			if err := binary.Read(buf, protocol.DefaultEndian, &ac.TransfersPermitted); err != nil {
				return fmt.Errorf("TransfersPermitted amendment value failed to deserialize : %s", err)
			}

		case 3: // TradeRestrictions
			switch amendment.Operation {
			case 0: // Modify
				if int(amendment.Element) >= len(ac.TradeRestrictions) {
					return fmt.Errorf("Asset amendment element out of range for TradeRestrictions : %d",
						amendment.Element)
				}

				if len(amendment.Data) != 3 {
					return fmt.Errorf("TradeRestrictions amendment value is wrong size : %d", len(amendment.Data))
				}
				buf := bytes.NewBuffer(amendment.Data)
				if err := binary.Read(buf, protocol.DefaultEndian, &ac.TradeRestrictions[amendment.Element]); err != nil {
					return fmt.Errorf("TradeRestrictions amendment value failed to deserialize : %s", err)
				}

			case 1: // Add element
				var newPolity [3]byte
				if len(amendment.Data) != 3 {
					return fmt.Errorf("TradeRestrictions amendment value is wrong size : %d", len(amendment.Data))
				}
				buf := bytes.NewBuffer(amendment.Data)
				if err := binary.Read(buf, protocol.DefaultEndian, &newPolity); err != nil {
					return fmt.Errorf("Contract amendment addition to TradeRestrictions failed to deserialize : %s", err)
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

		case 4: // EnforcementOrdersPermitted
			if len(amendment.Data) != 1 {
				return fmt.Errorf("EnforcementOrdersPermitted amendment value is wrong size : %d", len(amendment.Data))
			}
			buf := bytes.NewBuffer(amendment.Data)
			if err := binary.Read(buf, protocol.DefaultEndian, &ac.EnforcementOrdersPermitted); err != nil {
				return fmt.Errorf("EnforcementOrdersPermitted amendment value failed to deserialize : %s", err)
			}

		case 5: // VotingRights
			if len(amendment.Data) != 1 {
				return fmt.Errorf("VotingRights amendment value is wrong size : %d", len(amendment.Data))
			}
			buf := bytes.NewBuffer(amendment.Data)
			if err := binary.Read(buf, protocol.DefaultEndian, &ac.VotingRights); err != nil {
				return fmt.Errorf("VotingRights amendment value failed to deserialize : %s", err)
			}

		case 6: // VoteMultiplier
			if len(amendment.Data) != 1 {
				return fmt.Errorf("VoteMultiplier amendment value is wrong size : %d", len(amendment.Data))
			}
			buf := bytes.NewBuffer(amendment.Data)
			if err := binary.Read(buf, protocol.DefaultEndian, &ac.VoteMultiplier); err != nil {
				return fmt.Errorf("VoteMultiplier amendment value failed to deserialize : %s", err)
			}

		case 7: // AdministrationProposal
			if len(amendment.Data) != 1 {
				return fmt.Errorf("AdministrationProposal amendment value is wrong size : %d", len(amendment.Data))
			}
			buf := bytes.NewBuffer(amendment.Data)
			if err := binary.Read(buf, protocol.DefaultEndian, &ac.AdministrationProposal); err != nil {
				return fmt.Errorf("AdministrationProposal amendment value failed to deserialize : %s", err)
			}

		case 8: // HolderProposal
			if len(amendment.Data) != 1 {
				return fmt.Errorf("HolderProposal amendment value is wrong size : %d", len(amendment.Data))
			}
			buf := bytes.NewBuffer(amendment.Data)
			if err := binary.Read(buf, protocol.DefaultEndian, &ac.HolderProposal); err != nil {
				return fmt.Errorf("HolderProposal amendment value failed to deserialize : %s", err)
			}

		case 9: // AssetModificationGovernance
			if len(amendment.Data) != 1 {
				return fmt.Errorf("AssetModificationGovernance amendment value is wrong size : %d", len(amendment.Data))
			}
			buf := bytes.NewBuffer(amendment.Data)
			if err := binary.Read(buf, protocol.DefaultEndian, &ac.AssetModificationGovernance); err != nil {
				return fmt.Errorf("AssetModificationGovernance amendment value failed to deserialize : %s", err)
			}

		case 10: // TokenQty
			if len(amendment.Data) != 8 {
				return fmt.Errorf("TokenQty amendment value is wrong size : %d", len(amendment.Data))
			}
			buf := bytes.NewBuffer(amendment.Data)
			if err := binary.Read(buf, protocol.DefaultEndian, &ac.TokenQty); err != nil {
				return fmt.Errorf("TokenQty amendment value failed to deserialize : %s", err)
			}

		case 11: // AssetPayload
			// Validate payload
			payload := protocol.AssetTypeMapping(ac.AssetType)
			if payload == nil {
				return fmt.Errorf("Asset payload type unknown : %s", ac.AssetType)
			}

			_, err := payload.Write(amendment.Data)
			if err != nil {
				return fmt.Errorf("AssetPayload amendment value failed to deserialize : %s", err)
			}

			ac.AssetPayload = amendment.Data

		default:
			return fmt.Errorf("Asset amendment field offset out of range : %d", amendment.FieldIndex)
		}
	}

	if authFieldsUpdated {
		if _, err := protocol.ReadAuthFlags(ac.AssetAuthFlags, asset.FieldCount, len(votingSystems)); err != nil {
			return fmt.Errorf("Invalid asset auth flags : %s", err)
		}
	}

	// Check validity of updated asset data
	if err := ac.Validate(); err != nil {
		return fmt.Errorf("Asset data invalid after amendments : %s", err)
	}

	return nil
}
