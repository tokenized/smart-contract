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
	"github.com/tokenized/specification/dist/golang/actions"
	"github.com/tokenized/specification/dist/golang/assets"
	"github.com/tokenized/specification/dist/golang/protocol"

	"github.com/pkg/errors"
	"go.opencensus.io/trace"
)

type Asset struct {
	MasterDB        *db.DB
	Config          *node.Config
	HoldingsChannel *holdings.CacheChannel
}

// DefinitionRequest handles an incoming Asset Definition and prepares a Creation response
func (a *Asset) DefinitionRequest(ctx context.Context, w *node.ResponseWriter, itx *inspector.Transaction, rk *wallet.Key) error {
	ctx, span := trace.StartSpan(ctx, "handlers.Asset.Definition")
	defer span.End()

	msg, ok := itx.MsgProto.(*actions.AssetDefinition)
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
	ct, err := contract.Retrieve(ctx, a.MasterDB, rk.Address)
	if err != nil {
		return errors.Wrap(err, "Failed to retrieve contract")
	}

	if !ct.MovedTo.IsEmpty() {
		address := bitcoin.NewAddressFromRawAddress(ct.MovedTo, w.Config.Net)
		node.LogWarn(ctx, "Contract address changed : %s", address.String())
		return node.RespondReject(ctx, w, itx, rk, actions.RejectionsContractMoved)
	}

	if ct.FreezePeriod.Nano() > v.Now.Nano() {
		node.LogWarn(ctx, "Contract frozen : %s", ct.FreezePeriod.String())
		return node.RespondReject(ctx, w, itx, rk, actions.RejectionsContractFrozen)
	}

	if ct.ContractExpiration.Nano() != 0 && ct.ContractExpiration.Nano() < v.Now.Nano() {
		node.LogWarn(ctx, "Contract expired : %s", ct.ContractExpiration.String())
		return node.RespondReject(ctx, w, itx, rk, actions.RejectionsContractExpired)
	}

	if _, err = protocol.ReadAuthFlags(msg.AssetAuthFlags, actions.AssetFieldCount,
		len(ct.VotingSystems)); err != nil {
		node.LogWarn(ctx, "Invalid asset auth flags : %s", err)
		return node.RespondReject(ctx, w, itx, rk, actions.RejectionsMsgMalformed)
	}

	// Verify administration is sender of tx.
	if !itx.Inputs[0].Address.Equal(ct.AdministrationAddress) {
		address := bitcoin.NewAddressFromRawAddress(itx.Inputs[0].Address,
			w.Config.Net)
		node.LogWarn(ctx, "Only administration can create assets: %s", address)
		return node.RespondReject(ctx, w, itx, rk, actions.RejectionsNotAdministration)
	}

	// Generate Asset ID
	assetCode := protocol.AssetCodeFromContract(rk.Address, uint64(len(ct.AssetCodes)))

	// Locate Asset
	_, err = asset.Retrieve(ctx, a.MasterDB, rk.Address, assetCode)
	if err != asset.ErrNotFound {
		if err == nil {
			node.LogWarn(ctx, "Asset already exists : %s", assetCode.String())
			return node.RespondReject(ctx, w, itx, rk, actions.RejectionsAssetCodeExists)
		} else {
			return errors.Wrap(err, "Failed to retrieve asset")
		}
	}

	// Allowed to have more assets
	if !contract.CanHaveMoreAssets(ctx, ct) {
		address := bitcoin.NewAddressFromRawAddress(rk.Address, w.Config.Net)
		node.LogWarn(ctx, "Number of assets exceeds contract Qty: %s %s", address.String(), assetCode.String())
		return node.RespondReject(ctx, w, itx, rk, actions.RejectionsContractFixedQuantity)
	}

	// Validate payload
	payload := assets.NewAssetFromCode(msg.AssetType)
	if payload == nil {
		node.LogWarn(ctx, "Asset payload type unknown : %s", msg.AssetType)
		return node.RespondReject(ctx, w, itx, rk, actions.RejectionsMsgMalformed)
	}

	asset, err := assets.Deserialize([]byte(msg.AssetType), msg.AssetPayload)
	if err != nil {
		node.LogWarn(ctx, "Failed to parse asset payload : %s", err)
		return node.RespondReject(ctx, w, itx, rk, actions.RejectionsMsgMalformed)
	}

	if err := asset.Validate(); err != nil {
		node.LogWarn(ctx, "Asset %s payload is invalid : %s", msg.AssetType, err)
		return node.RespondReject(ctx, w, itx, rk, actions.RejectionsMsgMalformed)
	}

	address := bitcoin.NewAddressFromRawAddress(rk.Address, w.Config.Net)
	node.Log(ctx, "Accepting asset creation request : %s %s", address.String(), assetCode.String())

	// Asset Creation <- Asset Definition
	ac := actions.AssetCreation{}

	err = node.Convert(ctx, &msg, &ac)
	if err != nil {
		return err
	}

	ac.Timestamp = v.Now.Nano()
	ac.AssetCode = assetCode.Bytes()
	ac.AssetIndex = uint64(len(ct.AssetCodes))

	// Build outputs
	// 1 - Contract Address
	// 2 - Contract Fee (change)
	w.AddOutput(ctx, rk.Address, 0)
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

	msg, ok := itx.MsgProto.(*actions.AssetModification)
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
	ct, err := contract.Retrieve(ctx, a.MasterDB, rk.Address)
	if err != nil {
		return errors.Wrap(err, "Failed to retrieve contract")
	}

	if !ct.MovedTo.IsEmpty() {
		address := bitcoin.NewAddressFromRawAddress(ct.MovedTo, w.Config.Net)
		node.LogWarn(ctx, "Contract address changed : %s", address.String())
		return node.RespondReject(ctx, w, itx, rk, actions.RejectionsContractMoved)
	}

	if !contract.IsOperator(ctx, ct, itx.Inputs[0].Address) {
		address := bitcoin.NewAddressFromRawAddress(itx.Inputs[0].Address, w.Config.Net)
		node.LogVerbose(ctx, "Requestor is not operator : %x %s", msg.AssetCode, address.String())
		return node.RespondReject(ctx, w, itx, rk, actions.RejectionsNotOperator)
	}

	assetCode := protocol.AssetCodeFromBytes(msg.AssetCode)
	as, err := asset.Retrieve(ctx, a.MasterDB, rk.Address, assetCode)
	if err != nil {
		return errors.Wrap(err, "Failed to retrieve asset")
	}

	// Asset could not be found
	if as == nil {
		node.LogVerbose(ctx, "Asset ID not found: %x", msg.AssetCode)
		return node.RespondReject(ctx, w, itx, rk, actions.RejectionsAssetNotFound)
	}

	// Revision mismatch
	if as.Revision != msg.AssetRevision {
		node.LogVerbose(ctx, "Asset Revision does not match current: %x", msg.AssetCode)
		return node.RespondReject(ctx, w, itx, rk, actions.RejectionsAssetRevision)
	}

	// Check proposal if there was one
	proposed := false
	proposalInitiator := uint32(0)
	votingSystem := uint32(0)

	if len(msg.RefTxID) != 0 { // Vote Result Action allowing these amendments
		proposed = true

		refTxId, err := bitcoin.NewHash32(msg.RefTxID)
		if err != nil {
			return errors.Wrap(err, "Failed to convert protocol.TxId to Hash32")
		}

		// Retrieve Vote Result
		voteResultTx, err := transactions.GetTx(ctx, a.MasterDB, refTxId, a.Config.IsTest)
		if err != nil {
			node.LogWarn(ctx, "Vote Result tx not found for amendment")
			return node.RespondReject(ctx, w, itx, rk, actions.RejectionsMsgMalformed)
		}

		voteResult, ok := voteResultTx.MsgProto.(*actions.Result)
		if !ok {
			node.LogWarn(ctx, "Vote Result invalid for amendment")
			return node.RespondReject(ctx, w, itx, rk, actions.RejectionsMsgMalformed)
		}

		// Retrieve the vote
		voteTxId := protocol.TxIdFromBytes(voteResult.VoteTxId)
		vt, err := vote.Retrieve(ctx, a.MasterDB, rk.Address, voteTxId)
		if err == vote.ErrNotFound {
			node.LogWarn(ctx, "Vote not found : %x", voteResult.VoteTxId)
			return node.RespondReject(ctx, w, itx, rk, actions.RejectionsVoteNotFound)
		} else if err != nil {
			node.LogWarn(ctx, "Failed to retrieve vote : %x : %s", voteResult.VoteTxId, err)
			return errors.Wrap(err, "Failed to retrieve vote")
		}

		if vt.CompletedAt.Nano() == 0 {
			node.LogWarn(ctx, "Vote not complete yet")
			return node.RespondReject(ctx, w, itx, rk, actions.RejectionsMsgMalformed)
		}

		if vt.Result != "A" {
			node.LogWarn(ctx, "Vote result not A(Accept)")
			return node.RespondReject(ctx, w, itx, rk, actions.RejectionsMsgMalformed)
		}

		if !vt.Specific {
			node.LogWarn(ctx, "Vote was not for specific amendments")
			return node.RespondReject(ctx, w, itx, rk, actions.RejectionsMsgMalformed)
		}

		if !vt.AssetSpecificVote || !bytes.Equal(msg.AssetCode, vt.AssetCode.Bytes()) {
			node.LogWarn(ctx, "Vote was not for this asset code")
			return node.RespondReject(ctx, w, itx, rk, actions.RejectionsMsgMalformed)
		}

		// Verify proposal amendments match these amendments.
		if len(voteResult.ProposedAmendments) != len(msg.Amendments) {
			node.LogWarn(ctx, "%s : Proposal has different count of amendments : %d != %d",
				v.TraceID, len(voteResult.ProposedAmendments), len(msg.Amendments))
			return node.RespondReject(ctx, w, itx, rk, actions.RejectionsMsgMalformed)
		}

		for i, amendment := range voteResult.ProposedAmendments {
			if !amendment.Equal(msg.Amendments[i]) {
				node.LogWarn(ctx, "Proposal amendment %d doesn't match", i)
				return node.RespondReject(ctx, w, itx, rk, actions.RejectionsMsgMalformed)
			}
		}

		proposalInitiator = vt.Initiator
		votingSystem = vt.VoteSystem
	}

	if err := checkAssetAmendmentsPermissions(as, ct.VotingSystems, msg.Amendments, proposed,
		proposalInitiator, votingSystem); err != nil {
		node.LogWarn(ctx, "Asset amendments not permitted : %s", err)
		return node.RespondReject(ctx, w, itx, rk, actions.RejectionsAssetAuthFlags)
	}

	// Asset Creation <- Asset Modification
	ac := actions.AssetCreation{}

	err = node.Convert(ctx, as, &ac)
	if err != nil {
		return errors.Wrap(err, "Failed to convert state asset to asset creation")
	}

	ac.AssetRevision = as.Revision + 1
	ac.Timestamp = v.Now.Nano()
	ac.AssetCode = msg.AssetCode // Asset code not in state data

	node.Log(ctx, "Amending asset : %x", msg.AssetCode)

	if err := applyAssetAmendments(&ac, ct.VotingSystems, msg.Amendments); err != nil {
		node.LogWarn(ctx, "Asset amendments failed : %s", err)
		return node.RespondReject(ctx, w, itx, rk, actions.RejectionsMsgMalformed)
	}

	var h *state.Holding
	updateHoldings := false
	if ac.TokenQty != as.TokenQty {
		updateHoldings = true

		// Check administration balance for token quantity reductions. Administration has to hold any tokens being "burned".
		h, err = holdings.GetHolding(ctx, a.MasterDB, rk.Address, assetCode, ct.AdministrationAddress, v.Now)
		if err != nil {
			return errors.Wrap(err, "Failed to get admin holding")
		}

		txid := protocol.TxIdFromBytes(itx.Hash[:])

		if ac.TokenQty < as.TokenQty {
			if err := holdings.AddDebit(h, txid, as.TokenQty-ac.TokenQty, v.Now); err != nil {
				node.LogWarn(ctx, "%s : Failed to reduce administration holdings : %s", v.TraceID, err)
				if err == holdings.ErrInsufficientHoldings {
					return node.RespondReject(ctx, w, itx, rk, actions.RejectionsInsufficientQuantity)
				} else {
					return errors.Wrap(err, "Failed to reduce holdings")
				}
			}
		} else {
			if err := holdings.AddDeposit(h, txid, ac.TokenQty-as.TokenQty, v.Now); err != nil {
				node.LogWarn(ctx, "%s : Failed to increase administration holdings : %s", v.TraceID, err)
				return errors.Wrap(err, "Failed to increase holdings")
			}
		}
	}

	// Build outputs
	// 1 - Contract Address
	// 2 - Contract Fee (change)
	w.AddOutput(ctx, rk.Address, 0)
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
		cacheItem, err := holdings.Save(ctx, a.MasterDB, rk.Address, assetCode, h)
		if err != nil {
			return errors.Wrap(err, "Failed to save holdings")
		}
		a.HoldingsChannel.Add(cacheItem)
	}

	return nil
}

// CreationResponse handles an outgoing Asset Creation and writes it to the state
func (a *Asset) CreationResponse(ctx context.Context, w *node.ResponseWriter,
	itx *inspector.Transaction, rk *wallet.Key) error {

	ctx, span := trace.StartSpan(ctx, "handlers.Asset.Definition")
	defer span.End()

	msg, ok := itx.MsgProto.(*actions.AssetCreation)
	if !ok {
		return errors.New("Could not assert as *actions.AssetCreation")
	}

	v := ctx.Value(node.KeyValues).(*node.Values)

	if itx.RejectCode != 0 {
		return errors.New("Asset creation response invalid")
	}

	// Locate Asset
	if !itx.Inputs[0].Address.Equal(rk.Address) {
		return fmt.Errorf("Asset Creation not from contract : %x",
			itx.Inputs[0].Address.Bytes())
	}

	ct, err := contract.Retrieve(ctx, a.MasterDB, rk.Address)
	if err != nil {
		return errors.Wrap(err, "Failed to retrieve contract")
	}

	if !ct.MovedTo.IsEmpty() {
		address := bitcoin.NewAddressFromRawAddress(ct.MovedTo,
			w.Config.Net)
		return fmt.Errorf("Contract address changed : %s", address.String())
	}

	assetCode := protocol.AssetCodeFromBytes(msg.AssetCode)
	as, err := asset.Retrieve(ctx, a.MasterDB, rk.Address, assetCode)
	if err != nil && err != asset.ErrNotFound {
		return errors.Wrap(err, "Failed to retrieve asset")
	}

	// Get request tx
	request, err := transactions.GetTx(ctx, a.MasterDB, itx.Inputs[0].UTXO.Hash, a.Config.IsTest)
	if err != nil {
		return errors.Wrap(err, "Failed to retrieve request tx")
	}
	var vt *state.Vote
	var modification *actions.AssetModification
	if request != nil {
		var ok bool
		modification, ok = request.MsgProto.(*actions.AssetModification)

		if ok && len(modification.RefTxID) != 0 {
			refTxId, err := bitcoin.NewHash32(modification.RefTxID)
			if err != nil {
				return errors.Wrap(err, "Failed to convert protocol.TxId to Hash32")
			}

			// Retrieve Vote Result
			voteResultTx, err := transactions.GetTx(ctx, a.MasterDB, refTxId, a.Config.IsTest)
			if err != nil {
				return errors.Wrap(err, "Failed to retrieve vote result tx")
			}

			voteResult, ok := voteResultTx.MsgProto.(*actions.Result)
			if !ok {
				return errors.New("Vote Result invalid for modification")
			}

			// Retrieve the vote
			voteTxId := protocol.TxIdFromBytes(voteResult.VoteTxId)
			vt, err = vote.Retrieve(ctx, a.MasterDB, rk.Address, voteTxId)
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

		na.AdministrationAddress = ct.AdministrationAddress

		if err := contract.AddAssetCode(ctx, a.MasterDB, rk.Address, assetCode, v.Now); err != nil {
			return err
		}

		if err := asset.Create(ctx, a.MasterDB, rk.Address, assetCode, &na, v.Now); err != nil {
			return errors.Wrap(err, "Failed to create asset")
		}
		node.Log(ctx, "Created asset : %x", msg.AssetCode)

		// Update administration balance
		h, err := holdings.GetHolding(ctx, a.MasterDB, rk.Address, assetCode,
			ct.AdministrationAddress, v.Now)
		if err != nil {
			return errors.Wrap(err, "Failed to get admin holding")
		}
		txid := protocol.TxIdFromBytes(itx.Hash[:])
		holdings.AddDeposit(h, txid, msg.TokenQty, protocol.NewTimestamp(msg.Timestamp))
		holdings.FinalizeTx(h, txid, protocol.NewTimestamp(msg.Timestamp))
		cacheItem, err := holdings.Save(ctx, a.MasterDB, rk.Address, assetCode, h)
		if err != nil {
			return errors.Wrap(err, "Failed to save holdings")
		}
		a.HoldingsChannel.Add(cacheItem)
	} else {
		// Required pointers
		stringPointer := func(s string) *string { return &s }

		// Prepare update object
		ts := protocol.NewTimestamp(msg.Timestamp)
		ua := asset.UpdateAsset{
			Revision:  &msg.AssetRevision,
			Timestamp: &ts,
		}

		if as.AssetType != msg.AssetType {
			ua.AssetType = stringPointer(string(msg.AssetType))
			node.Log(ctx, "Updating asset type (%x) : %s", msg.AssetCode, *ua.AssetType)
		}
		if !bytes.Equal(as.AssetAuthFlags[:], msg.AssetAuthFlags[:]) {
			ua.AssetAuthFlags = &msg.AssetAuthFlags
			node.Log(ctx, "Updating asset auth flags (%x) : %x", msg.AssetCode, *ua.AssetAuthFlags)
		}
		if as.TransfersPermitted != msg.TransfersPermitted {
			ua.TransfersPermitted = &msg.TransfersPermitted
			node.Log(ctx, "Updating asset transfers permitted (%x) : %t", msg.AssetCode, *ua.TransfersPermitted)
		}
		if as.EnforcementOrdersPermitted != msg.EnforcementOrdersPermitted {
			ua.EnforcementOrdersPermitted = &msg.EnforcementOrdersPermitted
			node.Log(ctx, "Updating asset enforcement orders permitted (%x) : %t", msg.AssetCode, *ua.EnforcementOrdersPermitted)
		}
		if as.VoteMultiplier != msg.VoteMultiplier {
			ua.VoteMultiplier = &msg.VoteMultiplier
			node.Log(ctx, "Updating asset vote multiplier (%x) : %02x", msg.AssetCode, *ua.VoteMultiplier)
		}
		if as.AdministrationProposal != msg.AdministrationProposal {
			ua.AdministrationProposal = &msg.AdministrationProposal
			node.Log(ctx, "Updating asset administration proposal (%x) : %t", msg.AssetCode, *ua.AdministrationProposal)
		}
		if as.HolderProposal != msg.HolderProposal {
			ua.HolderProposal = &msg.HolderProposal
			node.Log(ctx, "Updating asset holder proposal (%x) : %t", msg.AssetCode, *ua.HolderProposal)
		}
		if as.AssetModificationGovernance != msg.AssetModificationGovernance {
			ua.AssetModificationGovernance = &msg.AssetModificationGovernance
			node.Log(ctx, "Updating asset modification governance (%x) : %d", msg.AssetCode, *ua.AssetModificationGovernance)
		}

		var h *state.Holding
		updateHoldings := false
		if as.TokenQty != msg.TokenQty {
			ua.TokenQty = &msg.TokenQty
			node.Log(ctx, "Updating asset token quantity %d : %x", *ua.TokenQty, msg.AssetCode)

			h, err = holdings.GetHolding(ctx, a.MasterDB, rk.Address, assetCode,
				ct.AdministrationAddress, v.Now)
			if err != nil {
				return errors.Wrap(err, "Failed to get admin holding")
			}

			txid := protocol.TxIdFromBytes(itx.Hash[:])

			holdings.FinalizeTx(h, txid, protocol.NewTimestamp(msg.Timestamp))
			updateHoldings = true

			if msg.TokenQty > as.TokenQty {
				node.Log(ctx, "Increasing token quantity by %d to %d : %x", msg.TokenQty-as.TokenQty, *ua.TokenQty, msg.AssetCode)
			} else {
				node.Log(ctx, "Decreasing token quantity by %d to %d : %x", as.TokenQty-msg.TokenQty, *ua.TokenQty, msg.AssetCode)
			}
			if err != nil {
				node.LogWarn(ctx, "Failed to update administration holding : %x", msg.AssetCode)
				return err
			}
		}
		if !bytes.Equal(as.AssetPayload, msg.AssetPayload) {
			ua.AssetPayload = &msg.AssetPayload
			node.Log(ctx, "Updating asset payload (%x) : %s", msg.AssetCode, *ua.AssetPayload)
		}

		// Check if trade restrictions are different
		different := len(as.TradeRestrictions) != len(msg.TradeRestrictions)
		if !different {
			for i, tradeRestriction := range as.TradeRestrictions {
				if tradeRestriction != msg.TradeRestrictions[i] {
					different = true
					break
				}
			}
		}

		if different {
			ua.TradeRestrictions = &msg.TradeRestrictions
		}

		if updateHoldings {
			cacheItem, err := holdings.Save(ctx, a.MasterDB, rk.Address, assetCode, h)
			if err != nil {
				return errors.Wrap(err, "Failed to save holdings")
			}
			a.HoldingsChannel.Add(cacheItem)
		}
		if err := asset.Update(ctx, a.MasterDB, rk.Address, assetCode, &ua, v.Now); err != nil {
			node.LogWarn(ctx, "Failed to update asset : %x", msg.AssetCode)
			return err
		}
		node.Log(ctx, "Updated asset : %x", msg.AssetCode)

		// Mark vote as "applied" if this amendment was a result of a vote.
		if vt != nil {
			node.Log(ctx, "Marking vote as applied : %s", vt.VoteTxId.String())
			if err := vote.MarkApplied(ctx, a.MasterDB, rk.Address, vt.VoteTxId, protocol.TxIdFromBytes(request.Hash[:]), v.Now); err != nil {
				return errors.Wrap(err, "Failed to mark vote applied")
			}
		}
	}

	return nil
}

// checkAssetAmendmentsPermissions verifies that the amendments are permitted based on the auth flags.
func checkAssetAmendmentsPermissions(as *state.Asset, votingSystems []*actions.VotingSystemField,
	amendments []*actions.AmendmentField, proposed bool, proposalInitiator, votingSystem uint32) error {

	permissions, err := protocol.ReadAuthFlags(as.AssetAuthFlags, actions.AssetFieldCount, len(votingSystems))
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

func applyAssetAmendments(ac *actions.AssetCreation, votingSystems []*actions.VotingSystemField,
	amendments []*actions.AmendmentField) error {
	authFieldsUpdated := false
	for _, amendment := range amendments {
		switch amendment.FieldIndex {
		case actions.AssetFieldAssetType:
			return fmt.Errorf("Asset amendment attempting to update asset type")

		case actions.AssetFieldAssetAuthFlags:
			ac.AssetAuthFlags = amendment.Data
			authFieldsUpdated = true

		case actions.AssetFieldTransfersPermitted:
			if len(amendment.Data) != 1 {
				return fmt.Errorf("TransfersPermitted amendment value is wrong size : %d", len(amendment.Data))
			}
			buf := bytes.NewBuffer(amendment.Data)
			if err := binary.Read(buf, protocol.DefaultEndian, &ac.TransfersPermitted); err != nil {
				return fmt.Errorf("TransfersPermitted amendment value failed to deserialize : %s", err)
			}

		case actions.AssetFieldTradeRestrictions:
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

				ac.TradeRestrictions = append(ac.TradeRestrictions, string(newPolity[:]))

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

		case actions.AssetFieldEnforcementOrdersPermitted:
			if len(amendment.Data) != 1 {
				return fmt.Errorf("EnforcementOrdersPermitted amendment value is wrong size : %d", len(amendment.Data))
			}
			buf := bytes.NewBuffer(amendment.Data)
			if err := binary.Read(buf, protocol.DefaultEndian, &ac.EnforcementOrdersPermitted); err != nil {
				return fmt.Errorf("EnforcementOrdersPermitted amendment value failed to deserialize : %s", err)
			}

		case actions.AssetFieldVotingRights:
			if len(amendment.Data) != 1 {
				return fmt.Errorf("VotingRights amendment value is wrong size : %d", len(amendment.Data))
			}
			buf := bytes.NewBuffer(amendment.Data)
			if err := binary.Read(buf, protocol.DefaultEndian, &ac.VotingRights); err != nil {
				return fmt.Errorf("VotingRights amendment value failed to deserialize : %s", err)
			}

		case actions.AssetFieldVoteMultiplier:
			if len(amendment.Data) != 1 {
				return fmt.Errorf("VoteMultiplier amendment value is wrong size : %d", len(amendment.Data))
			}
			buf := bytes.NewBuffer(amendment.Data)
			if err := binary.Read(buf, protocol.DefaultEndian, &ac.VoteMultiplier); err != nil {
				return fmt.Errorf("VoteMultiplier amendment value failed to deserialize : %s", err)
			}

		case actions.AssetFieldAdministrationProposal:
			if len(amendment.Data) != 1 {
				return fmt.Errorf("AdministrationProposal amendment value is wrong size : %d", len(amendment.Data))
			}
			buf := bytes.NewBuffer(amendment.Data)
			if err := binary.Read(buf, protocol.DefaultEndian, &ac.AdministrationProposal); err != nil {
				return fmt.Errorf("AdministrationProposal amendment value failed to deserialize : %s", err)
			}

		case actions.AssetFieldHolderProposal:
			if len(amendment.Data) != 1 {
				return fmt.Errorf("HolderProposal amendment value is wrong size : %d", len(amendment.Data))
			}
			buf := bytes.NewBuffer(amendment.Data)
			if err := binary.Read(buf, protocol.DefaultEndian, &ac.HolderProposal); err != nil {
				return fmt.Errorf("HolderProposal amendment value failed to deserialize : %s", err)
			}

		case actions.AssetFieldAssetModificationGovernance:
			if len(amendment.Data) != 1 {
				return fmt.Errorf("AssetModificationGovernance amendment value is wrong size : %d", len(amendment.Data))
			}
			buf := bytes.NewBuffer(amendment.Data)
			if err := binary.Read(buf, protocol.DefaultEndian, &ac.AssetModificationGovernance); err != nil {
				return fmt.Errorf("AssetModificationGovernance amendment value failed to deserialize : %s", err)
			}

		case actions.AssetFieldTokenQty:
			if len(amendment.Data) != 8 {
				return fmt.Errorf("TokenQty amendment value is wrong size : %d", len(amendment.Data))
			}
			buf := bytes.NewBuffer(amendment.Data)
			if err := binary.Read(buf, protocol.DefaultEndian, &ac.TokenQty); err != nil {
				return fmt.Errorf("TokenQty amendment value failed to deserialize : %s", err)
			}

		case actions.AssetFieldAssetPayload:
			// Validate payload
			_, err := assets.Deserialize([]byte(ac.AssetType), ac.AssetPayload)
			if err != nil {
				return fmt.Errorf("Asset payload deserialize failed : %s %s", ac.AssetType, err)
			}

			ac.AssetPayload = amendment.Data

		default:
			return fmt.Errorf("Asset amendment field offset out of range : %d", amendment.FieldIndex)
		}
	}

	if authFieldsUpdated {
		if _, err := protocol.ReadAuthFlags(ac.AssetAuthFlags, actions.AssetFieldCount,
			len(votingSystems)); err != nil {
			return fmt.Errorf("Invalid asset auth flags : %s", err)
		}
	}

	// Check validity of updated asset data
	if err := ac.Validate(); err != nil {
		return fmt.Errorf("Asset data invalid after amendments : %s", err)
	}

	return nil
}
