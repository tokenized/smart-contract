package handlers

import (
	"bytes"
	"context"
	"fmt"

	"github.com/tokenized/pkg/bitcoin"
	"github.com/tokenized/smart-contract/internal/asset"
	"github.com/tokenized/smart-contract/internal/contract"
	"github.com/tokenized/smart-contract/internal/holdings"
	"github.com/tokenized/smart-contract/internal/platform/db"
	"github.com/tokenized/smart-contract/internal/platform/node"
	"github.com/tokenized/smart-contract/internal/platform/state"
	"github.com/tokenized/smart-contract/internal/transactions"
	"github.com/tokenized/smart-contract/internal/vote"
	"github.com/tokenized/smart-contract/pkg/inspector"
	"github.com/tokenized/smart-contract/pkg/wallet"
	"github.com/tokenized/specification/dist/golang/actions"
	"github.com/tokenized/specification/dist/golang/assets"
	"github.com/tokenized/specification/dist/golang/permissions"
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
func (a *Asset) DefinitionRequest(ctx context.Context, w *node.ResponseWriter,
	itx *inspector.Transaction, rk *wallet.Key) error {
	ctx, span := trace.StartSpan(ctx, "handlers.Asset.Definition")
	defer span.End()

	msg, ok := itx.MsgProto.(*actions.AssetDefinition)
	if !ok {
		return errors.New("Could not assert as *actions.AssetDefinition")
	}

	v := ctx.Value(node.KeyValues).(*node.Values)

	// Validate all fields have valid values.
	if itx.RejectCode != 0 {
		node.LogWarn(ctx, "Asset definition invalid : %d %s", itx.RejectCode, itx.RejectText)
		return node.RespondRejectText(ctx, w, itx, rk, itx.RejectCode, itx.RejectText)
	}

	// Locate Contract
	ct, err := contract.Retrieve(ctx, a.MasterDB, rk.Address, a.Config.IsTest)
	if err != nil {
		return errors.Wrap(err, "retrieve contract")
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

	if _, err = permissions.PermissionsFromBytes(msg.AssetPermissions,
		len(ct.VotingSystems)); err != nil {
		node.LogWarn(ctx, "Invalid asset permissions : %s", err)
		return node.RespondReject(ctx, w, itx, rk, actions.RejectionsMsgMalformed)
	}

	// Verify administration is sender of tx.
	if !itx.Inputs[0].Address.Equal(ct.AdminAddress) {
		address := bitcoin.NewAddressFromRawAddress(itx.Inputs[0].Address, w.Config.Net)
		node.LogWarn(ctx, "Only administration can create assets: %s", address)
		return node.RespondReject(ctx, w, itx, rk, actions.RejectionsNotAdministration)
	}

	// Generate Asset ID
	assetCode := protocol.AssetCodeFromContract(rk.Address, uint64(len(ct.AssetCodes)))

	// Locate Asset
	_, err = asset.Retrieve(ctx, a.MasterDB, rk.Address, &assetCode)
	if err != asset.ErrNotFound {
		if err == nil {
			node.LogWarn(ctx, "Asset already exists : %s", assetCode.String())
			return node.RespondReject(ctx, w, itx, rk, actions.RejectionsAssetCodeExists)
		} else {
			return errors.Wrap(err, "retrieve asset")
		}
	}

	// Allowed to have more assets
	if !contract.CanHaveMoreAssets(ctx, ct) {
		address := bitcoin.NewAddressFromRawAddress(rk.Address, w.Config.Net)
		node.LogWarn(ctx, "Number of assets exceeds contract Qty: %s %s", address.String(),
			assetCode.String())
		return node.RespondReject(ctx, w, itx, rk, actions.RejectionsContractFixedQuantity)
	}

	// Validate payload
	assetPayload, err := assets.Deserialize([]byte(msg.AssetType), msg.AssetPayload)
	if err != nil {
		node.LogWarn(ctx, "Failed to parse asset payload : %s", err)
		return node.RespondReject(ctx, w, itx, rk, actions.RejectionsMsgMalformed)
	}

	if err := assetPayload.Validate(); err != nil {
		node.LogWarn(ctx, "Asset %s payload is invalid : %s", msg.AssetType, err)
		return node.RespondRejectText(ctx, w, itx, rk, actions.RejectionsMsgMalformed, err.Error())
	}

	// Only one Owner/Administrator Membership asset allowed
	if msg.AssetType == assets.CodeMembership &&
		(!ct.AdminMemberAsset.IsZero() || !ct.OwnerMemberAsset.IsZero()) {
		membership, ok := assetPayload.(*assets.Membership)
		if !ok {
			node.LogWarn(ctx, "Membership payload is wrong type")
			return node.RespondReject(ctx, w, itx, rk, actions.RejectionsMsgMalformed)
		}
		if membership.MembershipClass == "Owner" && !ct.OwnerMemberAsset.IsZero() {
			node.LogWarn(ctx, "Only one Owner Membership asset allowed")
			return node.RespondReject(ctx, w, itx, rk, actions.RejectionsContractNotPermitted)
		}
		if membership.MembershipClass == "Administrator" && !ct.AdminMemberAsset.IsZero() {
			node.LogWarn(ctx, "Only one Administrator Membership asset allowed")
			return node.RespondReject(ctx, w, itx, rk, actions.RejectionsContractNotPermitted)
		}
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
	if err := node.RespondSuccess(ctx, w, itx, rk, &ac); err != nil {
		return err
	}

	// Add the asset code now rather than when the asset creation is processed in case another asset
	//   definition is received before then.
	if err := contract.AddAssetCode(ctx, a.MasterDB, rk.Address, &assetCode, a.Config.IsTest,
		v.Now); err != nil {
		return err
	}

	return nil
}

// ModificationRequest handles an incoming Asset Modification and prepares a Creation response
func (a *Asset) ModificationRequest(ctx context.Context, w *node.ResponseWriter,
	itx *inspector.Transaction, rk *wallet.Key) error {
	ctx, span := trace.StartSpan(ctx, "handlers.Asset.Modification")
	defer span.End()

	msg, ok := itx.MsgProto.(*actions.AssetModification)
	if !ok {
		return errors.New("Could not assert as *actions.AssetModification")
	}

	v := ctx.Value(node.KeyValues).(*node.Values)

	// Validate all fields have valid values.
	if itx.RejectCode != 0 {
		node.LogWarn(ctx, "Asset modification invalid : %d %s", itx.RejectCode, itx.RejectText)
		return node.RespondRejectText(ctx, w, itx, rk, itx.RejectCode, itx.RejectText)
	}

	// Locate Asset
	ct, err := contract.Retrieve(ctx, a.MasterDB, rk.Address, a.Config.IsTest)
	if err != nil {
		return errors.Wrap(err, "Failed to retrieve contract")
	}

	if !ct.MovedTo.IsEmpty() {
		address := bitcoin.NewAddressFromRawAddress(ct.MovedTo, w.Config.Net)
		node.LogWarn(ctx, "Contract address changed : %s", address)
		return node.RespondReject(ctx, w, itx, rk, actions.RejectionsContractMoved)
	}

	if !contract.IsOperator(ctx, ct, itx.Inputs[0].Address) {
		address := bitcoin.NewAddressFromRawAddress(itx.Inputs[0].Address, w.Config.Net)
		node.LogVerbose(ctx, "Requestor is not operator : %s", address)
		return node.RespondReject(ctx, w, itx, rk, actions.RejectionsNotOperator)
	}

	assetCode, err := bitcoin.NewHash20(msg.AssetCode)
	if err != nil {
		node.LogVerbose(ctx, "Invalid asset code : 0x%x", msg.AssetCode)
		return node.RespondReject(ctx, w, itx, rk, actions.RejectionsMsgMalformed)
	}
	as, err := asset.Retrieve(ctx, a.MasterDB, rk.Address, assetCode)
	if err != nil {
		return errors.Wrap(err, "Failed to retrieve asset")
	}

	// Asset could not be found
	if as == nil {
		node.LogVerbose(ctx, "Asset ID not found: %s", assetCode)
		return node.RespondReject(ctx, w, itx, rk, actions.RejectionsAssetNotFound)
	}

	// Revision mismatch
	if as.Revision != msg.AssetRevision {
		node.LogVerbose(ctx, "Asset Revision does not match current: %s", assetCode)
		return node.RespondReject(ctx, w, itx, rk, actions.RejectionsAssetRevision)
	}

	// Check proposal if there was one
	proposed := false
	proposalType := uint32(0)
	votingSystem := uint32(0)

	if len(msg.RefTxID) != 0 { // Vote Result Action allowing these amendments
		proposed = true

		refTxId, err := bitcoin.NewHash32(msg.RefTxID)
		if err != nil {
			return errors.Wrap(err, "Failed to convert bitcoin.Hash32 to Hash32")
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
		voteTxId, err := bitcoin.NewHash32(voteResult.VoteTxId)
		if err != nil {
			node.LogWarn(ctx, "Invalid vote txid : 0x%x", voteResult.VoteTxId)
			return node.RespondReject(ctx, w, itx, rk, actions.RejectionsMsgMalformed)
		}

		vt, err := vote.Retrieve(ctx, a.MasterDB, rk.Address, voteTxId)
		if err == vote.ErrNotFound {
			node.LogWarn(ctx, "Vote not found : %s", voteResult.VoteTxId)
			return node.RespondReject(ctx, w, itx, rk, actions.RejectionsVoteNotFound)
		} else if err != nil {
			node.LogWarn(ctx, "Failed to retrieve vote : %s : %s", voteResult.VoteTxId, err)
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

		if len(vt.ProposedAmendments) == 0 {
			node.LogWarn(ctx, "Vote was not for specific amendments")
			return node.RespondReject(ctx, w, itx, rk, actions.RejectionsMsgMalformed)
		}

		if vt.AssetCode.IsZero() || !bytes.Equal(msg.AssetCode, vt.AssetCode.Bytes()) {
			node.LogWarn(ctx, "Vote was not for this asset code")
			return node.RespondReject(ctx, w, itx, rk, actions.RejectionsMsgMalformed)
		}

		// Verify proposal amendments match these amendments.
		if len(voteResult.ProposedAmendments) != len(msg.Amendments) {
			node.LogWarn(ctx, "Proposal has different count of amendments : %d != %d",
				len(voteResult.ProposedAmendments), len(msg.Amendments))
			return node.RespondReject(ctx, w, itx, rk, actions.RejectionsMsgMalformed)
		}

		for i, amendment := range voteResult.ProposedAmendments {
			if !amendment.Equal(msg.Amendments[i]) {
				node.LogWarn(ctx, "Proposal amendment %d doesn't match", i)
				return node.RespondReject(ctx, w, itx, rk, actions.RejectionsMsgMalformed)
			}
		}

		proposalType = vt.Type
		votingSystem = vt.VoteSystem
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

	node.Log(ctx, "Amending asset : %s", assetCode)

	if err := applyAssetAmendments(&ac, ct.VotingSystems, msg.Amendments, proposed,
		proposalType, votingSystem); err != nil {
		node.LogWarn(ctx, "Asset amendments failed : %s", err)
		code, ok := node.ErrorCode(err)
		if ok {
			return node.RespondReject(ctx, w, itx, rk, code)
		}
		return node.RespondReject(ctx, w, itx, rk, actions.RejectionsMsgMalformed)
	}

	var h *state.Holding
	updateHoldings := false
	if ac.AuthorizedTokenQty != as.AuthorizedTokenQty {
		updateHoldings = true

		// Check administration balance for token quantity reductions. Administration has to hold
		//   any tokens being "burned".
		h, err = holdings.GetHolding(ctx, a.MasterDB, rk.Address, assetCode, ct.AdminAddress, v.Now)
		if err != nil {
			return errors.Wrap(err, "Failed to get admin holding")
		}

		if ac.AuthorizedTokenQty < as.AuthorizedTokenQty {
			if err := holdings.AddDebit(h, itx.Hash, as.AuthorizedTokenQty-ac.AuthorizedTokenQty, true,
				v.Now); err != nil {
				node.LogWarn(ctx, "Failed to reduce administration holdings : %s", err)
				if err == holdings.ErrInsufficientHoldings {
					return node.RespondReject(ctx, w, itx, rk,
						actions.RejectionsInsufficientQuantity)
				} else {
					return errors.Wrap(err, "Failed to reduce holdings")
				}
			}
		} else {
			if err := holdings.AddDeposit(h, itx.Hash, ac.AuthorizedTokenQty-as.AuthorizedTokenQty,
				true, v.Now); err != nil {
				node.LogWarn(ctx, "Failed to increase administration holdings : %s", err)
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

	// Locate Asset
	if !itx.Inputs[0].Address.Equal(rk.Address) {
		address := bitcoin.NewAddressFromRawAddress(itx.Inputs[0].Address, w.Config.Net)
		return fmt.Errorf("Asset Creation not from contract : %s", address)
	}

	ct, err := contract.Retrieve(ctx, a.MasterDB, rk.Address, a.Config.IsTest)
	if err != nil {
		return errors.Wrap(err, "Failed to retrieve contract")
	}

	if !ct.MovedTo.IsEmpty() {
		address := bitcoin.NewAddressFromRawAddress(ct.MovedTo,
			w.Config.Net)
		return fmt.Errorf("Contract address changed : %s", address.String())
	}

	assetCode, err := bitcoin.NewHash20(msg.AssetCode)
	if err != nil {
		node.LogVerbose(ctx, "Invalid asset code : 0x%x", msg.AssetCode)
		return errors.Wrap(err, "invalid asset code")
	}
	as, err := asset.Retrieve(ctx, a.MasterDB, rk.Address, assetCode)
	if err != nil && err != asset.ErrNotFound {
		return errors.Wrap(err, "Failed to retrieve asset")
	}

	// Get request tx
	request, err := transactions.GetTx(ctx, a.MasterDB, &itx.Inputs[0].UTXO.Hash, a.Config.IsTest)
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
				return errors.Wrap(err, "Failed to convert bitcoin.Hash32 to Hash32")
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
			voteTxId, err := bitcoin.NewHash32(voteResult.VoteTxId)
			if err != nil {
				return errors.Wrap(err, "invalid vote txid")
			}

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

		na.AdminAddress = ct.AdminAddress

		// Add asset code if it hasn't been added yet. This will not add duplicates. This is
		//   required to handle the recovery case when the request will not be reprocessed.
		if err := contract.AddAssetCode(ctx, a.MasterDB, rk.Address, assetCode, a.Config.IsTest,
			v.Now); err != nil {
			return err
		}

		if err := asset.Create(ctx, a.MasterDB, rk.Address, assetCode, &na, v.Now); err != nil {
			return errors.Wrap(err, "Failed to create asset")
		}
		node.Log(ctx, "Created asset %d : %s", msg.AssetIndex, assetCode.String())

		// Update administration balance
		h, err := holdings.GetHolding(ctx, a.MasterDB, rk.Address, assetCode,
			ct.AdminAddress, v.Now)
		if err != nil {
			return errors.Wrap(err, "Failed to get admin holding")
		}
		holdings.AddDeposit(h, itx.Hash, msg.AuthorizedTokenQty, true,
			protocol.NewTimestamp(msg.Timestamp))
		holdings.FinalizeTx(h, itx.Hash, msg.AuthorizedTokenQty,
			protocol.NewTimestamp(msg.Timestamp))
		cacheItem, err := holdings.Save(ctx, a.MasterDB, rk.Address, assetCode, h)
		if err != nil {
			return errors.Wrap(err, "Failed to save holdings")
		}
		a.HoldingsChannel.Add(cacheItem)

		// Update Owner/Administrator Membership asset in contract
		if msg.AssetType == assets.CodeMembership {
			assetPayload, err := assets.Deserialize([]byte(msg.AssetType), msg.AssetPayload)
			if err != nil {
				node.LogWarn(ctx, "Failed to parse asset payload : %s", err)
				return node.RespondReject(ctx, w, itx, rk, actions.RejectionsMsgMalformed)
			}

			membership, ok := assetPayload.(*assets.Membership)
			if !ok {
				node.LogWarn(ctx, "Membership payload is wrong type")
				return node.RespondReject(ctx, w, itx, rk, actions.RejectionsMsgMalformed)
			}
			if membership.MembershipClass == "Owner" {
				updateContract := &contract.UpdateContract{
					OwnerMemberAsset: assetCode,
				}
				if err := contract.Update(ctx, a.MasterDB, rk.Address, updateContract,
					a.Config.IsTest, v.Now); err != nil {
					return errors.Wrap(err, "updating contract")
				}
			}
			if membership.MembershipClass == "Administrator" {
				updateContract := &contract.UpdateContract{
					AdminMemberAsset: assetCode,
				}
				if err := contract.Update(ctx, a.MasterDB, rk.Address, updateContract,
					a.Config.IsTest, v.Now); err != nil {
					return errors.Wrap(err, "updating contract")
				}
			}
		}
	} else {
		// Prepare update object
		ts := protocol.NewTimestamp(msg.Timestamp)
		ua := asset.UpdateAsset{
			Revision:  &msg.AssetRevision,
			Timestamp: &ts,
		}

		if !bytes.Equal(as.AssetPermissions[:], msg.AssetPermissions[:]) {
			ua.AssetPermissions = &msg.AssetPermissions
			node.Log(ctx, "Updating asset permissions (%s) : %s", assetCode,
				*ua.AssetPermissions)
		}
		if as.EnforcementOrdersPermitted != msg.EnforcementOrdersPermitted {
			ua.EnforcementOrdersPermitted = &msg.EnforcementOrdersPermitted
			node.Log(ctx, "Updating asset enforcement orders permitted (%s) : %t", assetCode,
				*ua.EnforcementOrdersPermitted)
		}
		if as.VoteMultiplier != msg.VoteMultiplier {
			ua.VoteMultiplier = &msg.VoteMultiplier
			node.Log(ctx, "Updating asset vote multiplier (%s) : %02x", assetCode,
				*ua.VoteMultiplier)
		}
		if as.AdministrationProposal != msg.AdministrationProposal {
			ua.AdministrationProposal = &msg.AdministrationProposal
			node.Log(ctx, "Updating asset administration proposal (%s) : %t", assetCode,
				*ua.AdministrationProposal)
		}
		if as.HolderProposal != msg.HolderProposal {
			ua.HolderProposal = &msg.HolderProposal
			node.Log(ctx, "Updating asset holder proposal (%s) : %t", assetCode,
				*ua.HolderProposal)
		}
		if as.AssetModificationGovernance != msg.AssetModificationGovernance {
			ua.AssetModificationGovernance = &msg.AssetModificationGovernance
			node.Log(ctx, "Updating asset modification governance (%s) : %d", assetCode,
				*ua.AssetModificationGovernance)
		}

		var h *state.Holding
		updateHoldings := false
		if as.AuthorizedTokenQty != msg.AuthorizedTokenQty {
			ua.AuthorizedTokenQty = &msg.AuthorizedTokenQty
			node.Log(ctx, "Updating asset token quantity %d : %s", *ua.AuthorizedTokenQty,
				assetCode)

			h, err = holdings.GetHolding(ctx, a.MasterDB, rk.Address, assetCode,
				ct.AdminAddress, v.Now)
			if err != nil {
				return errors.Wrap(err, "Failed to get admin holding")
			}

			if msg.AuthorizedTokenQty > as.AuthorizedTokenQty {
				node.Log(ctx, "Increasing token quantity by %d to %d : %s",
					msg.AuthorizedTokenQty-as.AuthorizedTokenQty, *ua.AuthorizedTokenQty, assetCode)
				holdings.FinalizeTx(h, itx.Hash, h.FinalizedBalance+(msg.AuthorizedTokenQty-as.AuthorizedTokenQty),
					protocol.NewTimestamp(msg.Timestamp))
			} else {
				node.Log(ctx, "Decreasing token quantity by %d to %d : %s",
					as.AuthorizedTokenQty-msg.AuthorizedTokenQty, *ua.AuthorizedTokenQty, assetCode)
				holdings.FinalizeTx(h, itx.Hash, h.FinalizedBalance-(as.AuthorizedTokenQty-msg.AuthorizedTokenQty),
					protocol.NewTimestamp(msg.Timestamp))
			}
			updateHoldings = true
			if err != nil {
				node.LogWarn(ctx, "Failed to update administration holding : %s", assetCode)
				return err
			}
		}
		if !bytes.Equal(as.AssetPayload, msg.AssetPayload) {
			ua.AssetPayload = &msg.AssetPayload
			node.Log(ctx, "Updating asset payload (%s) : %s", assetCode, *ua.AssetPayload)
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
			node.LogWarn(ctx, "Failed to update asset : %s", assetCode)
			return err
		}
		node.Log(ctx, "Updated asset %d : %s", msg.AssetIndex, assetCode)

		// Mark vote as "applied" if this amendment was a result of a vote.
		if vt != nil {
			node.Log(ctx, "Marking vote as applied : %s", vt.VoteTxId)
			if err := vote.MarkApplied(ctx, a.MasterDB, rk.Address, vt.VoteTxId, request.Hash,
				v.Now); err != nil {
				return errors.Wrap(err, "Failed to mark vote applied")
			}
		}

		// Update Owner/Administrator Membership asset in contract
		if msg.AssetType == assets.CodeMembership {
			assetPayload, err := assets.Deserialize([]byte(msg.AssetType), msg.AssetPayload)
			if err != nil {
				node.LogWarn(ctx, "Failed to parse asset payload : %s", err)
				return node.RespondReject(ctx, w, itx, rk, actions.RejectionsMsgMalformed)
			}

			membership, ok := assetPayload.(*assets.Membership)
			if !ok {
				node.LogWarn(ctx, "Membership payload is wrong type")
				return node.RespondReject(ctx, w, itx, rk, actions.RejectionsMsgMalformed)
			}

			if membership.MembershipClass == "Administrator" && !assetCode.Equal(&ct.AdminMemberAsset) {
				// Set contract AdminMemberAsset
				updateContract := &contract.UpdateContract{
					AdminMemberAsset: assetCode,
				}
				if err := contract.Update(ctx, a.MasterDB, rk.Address, updateContract,
					a.Config.IsTest, v.Now); err != nil {
					return errors.Wrap(err, "updating contract")
				}
			} else if membership.MembershipClass != "Administrator" && assetCode.Equal(&ct.AdminMemberAsset) {
				// Clear contract AdminMemberAsset
				updateContract := &contract.UpdateContract{
					AdminMemberAsset: &bitcoin.Hash20{}, // zero asset code
				}
				if err := contract.Update(ctx, a.MasterDB, rk.Address, updateContract,
					a.Config.IsTest, v.Now); err != nil {
					return errors.Wrap(err, "updating contract")
				}
			}

			if membership.MembershipClass == "Owner" && !assetCode.Equal(&ct.OwnerMemberAsset) {
				// Set contract OwnerMemberAsset
				updateContract := &contract.UpdateContract{
					OwnerMemberAsset: assetCode,
				}
				if err := contract.Update(ctx, a.MasterDB, rk.Address, updateContract,
					a.Config.IsTest, v.Now); err != nil {
					return errors.Wrap(err, "updating contract")
				}
			} else if membership.MembershipClass != "Owner" && assetCode.Equal(&ct.OwnerMemberAsset) {
				// Clear contract OwnerMemberAsset
				updateContract := &contract.UpdateContract{
					OwnerMemberAsset: &bitcoin.Hash20{}, // zero asset code
				}
				if err := contract.Update(ctx, a.MasterDB, rk.Address, updateContract,
					a.Config.IsTest, v.Now); err != nil {
					return errors.Wrap(err, "updating contract")
				}
			}
		}
	}

	return nil
}

func applyAssetAmendments(ac *actions.AssetCreation, votingSystems []*actions.VotingSystemField,
	amendments []*actions.AmendmentField, proposed bool, proposalType, votingSystem uint32) error {

	perms, err := permissions.PermissionsFromBytes(ac.AssetPermissions, len(votingSystems))
	if err != nil {
		return fmt.Errorf("Invalid asset permissions : %s", err)
	}

	var assetPayload assets.Asset

	for i, amendment := range amendments {
		applied := false
		var fieldPermissions permissions.Permissions
		fip, err := permissions.FieldIndexPathFromBytes(amendment.FieldIndexPath)
		if err != nil {
			return fmt.Errorf("Failed to read amendment %d field index path : %s", i, err)
		}
		if len(fip) == 0 {
			return fmt.Errorf("Amendment %d has no field specified", i)
		}

		switch fip[0] {
		case actions.AssetFieldAssetType:
			return node.NewError(actions.RejectionsAssetNotPermitted,
				"Asset type amendments prohibited")

		case actions.AssetFieldAssetPermissions:
			if _, err := permissions.PermissionsFromBytes(amendment.Data,
				len(votingSystems)); err != nil {
				return fmt.Errorf("AssetPermissions amendment value is invalid : %s", err)
			}

		case actions.AssetFieldAssetPayload:
			if len(fip) == 1 {
				return node.NewError(actions.RejectionsAssetNotPermitted,
					"Amendments on complex fields (AssetPayload) prohibited")
			}

			if assetPayload == nil {
				// Get payload object
				assetPayload, err = assets.Deserialize([]byte(ac.AssetType), ac.AssetPayload)
				if err != nil {
					return fmt.Errorf("Asset payload deserialize failed : %s %s", ac.AssetType, err)
				}
			}

			payloadPermissions, err := perms.SubPermissions(
				permissions.FieldIndexPath{actions.AssetFieldAssetPayload}, 0, false)

			fieldPermissions, err = assetPayload.ApplyAmendment(fip[1:], amendment.Operation,
				amendment.Data, payloadPermissions)
			if err != nil {
				return errors.Wrapf(err, "apply amendment %d", i)
			}
			if len(fieldPermissions) == 0 {
				return errors.New("Invalid field permissions")
			}

			switch assetPayload.(type) {
			case *assets.Membership:
				if fip[1] == assets.MembershipFieldMembershipClass {
					return node.NewError(actions.RejectionsAssetNotPermitted,
						"Amendments on MembershipClass prohibited")
				}
			}

			applied = true // Amendment already applied
		}

		if !applied {
			fieldPermissions, err = ac.ApplyAmendment(fip, amendment.Operation, amendment.Data,
				perms)
			if err != nil {
				return errors.Wrapf(err, "apply amendment %d", i)
			}
			if len(fieldPermissions) == 0 {
				return errors.New("Invalid field permissions")
			}
		}

		// fieldPermissions are the permissions that apply to the field that was changed in the
		// amendment.
		permission := fieldPermissions[0]
		if proposed {
			switch proposalType {
			case 0: // Administration
				if !permission.AdministrationProposal {
					return node.NewError(actions.RejectionsAssetPermissions,
						fmt.Sprintf("Field %s amendment not permitted by administration proposal",
							fip))
				}
			case 1: // Holder
				if !permission.HolderProposal {
					return node.NewError(actions.RejectionsAssetPermissions,
						fmt.Sprintf("Field %s amendment not permitted by holder proposal", fip))
				}
			case 2: // Administrative Matter
				if !permission.AdministrativeMatter {
					return node.NewError(actions.RejectionsAssetPermissions,
						fmt.Sprintf("Field %s amendment not permitted by administrative vote",
							fip))
				}
			default:
				return fmt.Errorf("Invalid proposal type : %d", proposalType)
			}

			if int(votingSystem) >= len(permission.VotingSystemsAllowed) {
				return fmt.Errorf("Field %s amendment voting system out of range : %d", fip,
					votingSystem)
			}
			if !permission.VotingSystemsAllowed[votingSystem] {
				return node.NewError(actions.RejectionsAssetPermissions,
					fmt.Sprintf("Field %s amendment not allowed using voting system %d", fip,
						votingSystem))
			}
		} else if !permission.Permitted {
			return node.NewError(actions.RejectionsAssetPermissions,
				fmt.Sprintf("Field %s amendment not permitted without proposal", fip))
		}
	}

	if assetPayload != nil {
		if err = assetPayload.Validate(); err != nil {
			return err
		}

		newPayload, err := assetPayload.Bytes()
		if err != nil {
			return err
		}

		ac.AssetPayload = newPayload
	}

	// Check validity of updated asset data
	if err := ac.Validate(); err != nil {
		return fmt.Errorf("Asset data invalid after amendments : %s", err)
	}

	return nil
}
