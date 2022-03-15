package handlers

import (
	"bytes"
	"context"
	"fmt"

	"github.com/tokenized/pkg/bitcoin"
	"github.com/tokenized/smart-contract/internal/contract"
	"github.com/tokenized/smart-contract/internal/holdings"
	"github.com/tokenized/smart-contract/internal/instrument"
	"github.com/tokenized/smart-contract/internal/platform/db"
	"github.com/tokenized/smart-contract/internal/platform/node"
	"github.com/tokenized/smart-contract/internal/platform/state"
	"github.com/tokenized/smart-contract/internal/transactions"
	"github.com/tokenized/smart-contract/internal/vote"
	"github.com/tokenized/smart-contract/pkg/inspector"
	"github.com/tokenized/smart-contract/pkg/wallet"
	"github.com/tokenized/specification/dist/golang/actions"
	"github.com/tokenized/specification/dist/golang/instruments"
	"github.com/tokenized/specification/dist/golang/permissions"
	"github.com/tokenized/specification/dist/golang/protocol"

	"github.com/pkg/errors"
	"go.opencensus.io/trace"
)

type Instrument struct {
	MasterDB        *db.DB
	Config          *node.Config
	HoldingsChannel *holdings.CacheChannel
}

// DefinitionRequest handles an incoming Instrument Definition and prepares a Creation response
func (a *Instrument) DefinitionRequest(ctx context.Context, w *node.ResponseWriter,
	itx *inspector.Transaction, rk *wallet.Key) error {
	ctx, span := trace.StartSpan(ctx, "handlers.Instrument.Definition")
	defer span.End()

	msg, ok := itx.MsgProto.(*actions.InstrumentDefinition)
	if !ok {
		return errors.New("Could not assert as *actions.InstrumentDefinition")
	}

	v := ctx.Value(node.KeyValues).(*node.Values)

	// Validate all fields have valid values.
	if itx.RejectCode != 0 {
		node.LogWarn(ctx, "Instrument definition invalid : %d %s", itx.RejectCode, itx.RejectText)
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

	if _, err = permissions.PermissionsFromBytes(msg.InstrumentPermissions,
		len(ct.VotingSystems)); err != nil {
		node.LogWarn(ctx, "Invalid instrument permissions : %s", err)
		return node.RespondReject(ctx, w, itx, rk, actions.RejectionsMsgMalformed)
	}

	// Verify administration is sender of tx.
	if !itx.Inputs[0].Address.Equal(ct.AdminAddress) {
		address := bitcoin.NewAddressFromRawAddress(itx.Inputs[0].Address, w.Config.Net)
		node.LogWarn(ctx, "Only administration can create instruments: %s", address)
		return node.RespondReject(ctx, w, itx, rk, actions.RejectionsNotAdministration)
	}

	// Generate Instrument ID
	instrumentCode := protocol.InstrumentCodeFromContract(rk.Address, uint64(len(ct.InstrumentCodes)))

	// Locate Instrument
	_, err = instrument.Retrieve(ctx, a.MasterDB, rk.Address, &instrumentCode)
	if err != instrument.ErrNotFound {
		if err == nil {
			node.LogWarn(ctx, "Instrument already exists : %s", instrumentCode.String())
			return node.RespondReject(ctx, w, itx, rk, actions.RejectionsInstrumentCodeExists)
		} else {
			return errors.Wrap(err, "retrieve instrument")
		}
	}

	// Allowed to have more instruments
	if !contract.CanHaveMoreInstruments(ctx, ct) {
		address := bitcoin.NewAddressFromRawAddress(rk.Address, w.Config.Net)
		node.LogWarn(ctx, "Number of instruments exceeds contract Qty: %s %s", address.String(),
			instrumentCode.String())
		return node.RespondReject(ctx, w, itx, rk, actions.RejectionsContractFixedQuantity)
	}

	// Validate payload
	instrumentPayload, err := instruments.Deserialize([]byte(msg.InstrumentType), msg.InstrumentPayload)
	if err != nil {
		node.LogWarn(ctx, "Failed to parse instrument payload : %s", err)
		return node.RespondReject(ctx, w, itx, rk, actions.RejectionsMsgMalformed)
	}

	if err := instrumentPayload.Validate(); err != nil {
		node.LogWarn(ctx, "Instrument %s payload is invalid : %s", msg.InstrumentType, err)
		return node.RespondRejectText(ctx, w, itx, rk, actions.RejectionsMsgMalformed, err.Error())
	}

	// Only one Owner/Administrator Membership instrument allowed
	if msg.InstrumentType == instruments.CodeMembership &&
		(!ct.AdminMemberInstrument.IsZero() || !ct.OwnerMemberInstrument.IsZero()) {
		membership, ok := instrumentPayload.(*instruments.Membership)
		if !ok {
			node.LogWarn(ctx, "Membership payload is wrong type")
			return node.RespondReject(ctx, w, itx, rk, actions.RejectionsMsgMalformed)
		}
		if membership.MembershipClass == "Owner" && !ct.OwnerMemberInstrument.IsZero() {
			node.LogWarn(ctx, "Only one Owner Membership instrument allowed")
			return node.RespondReject(ctx, w, itx, rk, actions.RejectionsContractNotPermitted)
		}
		if membership.MembershipClass == "Administrator" && !ct.AdminMemberInstrument.IsZero() {
			node.LogWarn(ctx, "Only one Administrator Membership instrument allowed")
			return node.RespondReject(ctx, w, itx, rk, actions.RejectionsContractNotPermitted)
		}
	}

	address := bitcoin.NewAddressFromRawAddress(rk.Address, w.Config.Net)
	node.Log(ctx, "Accepting instrument creation request : %s %s", address.String(), instrumentCode.String())

	// Instrument Creation <- Instrument Definition
	ac := actions.InstrumentCreation{}

	err = node.Convert(ctx, &msg, &ac)
	if err != nil {
		return err
	}

	ac.Timestamp = v.Now.Nano()
	ac.InstrumentCode = instrumentCode.Bytes()
	ac.InstrumentIndex = uint64(len(ct.InstrumentCodes))

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

	// Add the instrument code now rather than when the instrument creation is processed in case another instrument
	//   definition is received before then.
	if err := contract.AddInstrumentCode(ctx, a.MasterDB, rk.Address, &instrumentCode, a.Config.IsTest,
		v.Now); err != nil {
		return err
	}

	return nil
}

// ModificationRequest handles an incoming Instrument Modification and prepares a Creation response
func (a *Instrument) ModificationRequest(ctx context.Context, w *node.ResponseWriter,
	itx *inspector.Transaction, rk *wallet.Key) error {
	ctx, span := trace.StartSpan(ctx, "handlers.Instrument.Modification")
	defer span.End()

	msg, ok := itx.MsgProto.(*actions.InstrumentModification)
	if !ok {
		return errors.New("Could not assert as *actions.InstrumentModification")
	}

	v := ctx.Value(node.KeyValues).(*node.Values)

	// Validate all fields have valid values.
	if itx.RejectCode != 0 {
		node.LogWarn(ctx, "Instrument modification invalid : %d %s", itx.RejectCode, itx.RejectText)
		return node.RespondRejectText(ctx, w, itx, rk, itx.RejectCode, itx.RejectText)
	}

	// Locate Instrument
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

	instrumentCode, err := bitcoin.NewHash20(msg.InstrumentCode)
	if err != nil {
		node.LogVerbose(ctx, "Invalid instrument code : 0x%x", msg.InstrumentCode)
		return node.RespondReject(ctx, w, itx, rk, actions.RejectionsMsgMalformed)
	}
	as, err := instrument.Retrieve(ctx, a.MasterDB, rk.Address, instrumentCode)
	if err != nil {
		return errors.Wrap(err, "Failed to retrieve instrument")
	}

	// Instrument could not be found
	if as == nil {
		node.LogVerbose(ctx, "Instrument ID not found: %s", instrumentCode)
		return node.RespondReject(ctx, w, itx, rk, actions.RejectionsInstrumentNotFound)
	}

	// Revision mismatch
	if as.Revision != msg.InstrumentRevision {
		node.LogVerbose(ctx, "Instrument Revision does not match current: %s", instrumentCode)
		return node.RespondReject(ctx, w, itx, rk, actions.RejectionsInstrumentRevision)
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

		if vt.InstrumentCode.IsZero() || !bytes.Equal(msg.InstrumentCode, vt.InstrumentCode.Bytes()) {
			node.LogWarn(ctx, "Vote was not for this instrument code")
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

	// Instrument Creation <- Instrument Modification
	ac := actions.InstrumentCreation{}

	err = node.Convert(ctx, as, &ac)
	if err != nil {
		return errors.Wrap(err, "Failed to convert state instrument to instrument creation")
	}

	ac.InstrumentRevision = as.Revision + 1
	ac.Timestamp = v.Now.Nano()
	ac.InstrumentCode = msg.InstrumentCode // Instrument code not in state data

	node.Log(ctx, "Amending instrument : %s", instrumentCode)

	if err := applyInstrumentAmendments(&ac, ct.VotingSystems, msg.Amendments, proposed,
		proposalType, votingSystem); err != nil {
		node.LogWarn(ctx, "Instrument amendments failed : %s", err)
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
		h, err = holdings.GetHolding(ctx, a.MasterDB, rk.Address, instrumentCode, ct.AdminAddress, v.Now)
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
		cacheItem, err := holdings.Save(ctx, a.MasterDB, rk.Address, instrumentCode, h)
		if err != nil {
			return errors.Wrap(err, "Failed to save holdings")
		}
		a.HoldingsChannel.Add(cacheItem)
	}

	return nil
}

// CreationResponse handles an outgoing Instrument Creation and writes it to the state
func (a *Instrument) CreationResponse(ctx context.Context, w *node.ResponseWriter,
	itx *inspector.Transaction, rk *wallet.Key) error {

	ctx, span := trace.StartSpan(ctx, "handlers.Instrument.Definition")
	defer span.End()

	msg, ok := itx.MsgProto.(*actions.InstrumentCreation)
	if !ok {
		return errors.New("Could not assert as *actions.InstrumentCreation")
	}

	v := ctx.Value(node.KeyValues).(*node.Values)

	// Locate Instrument
	if !itx.Inputs[0].Address.Equal(rk.Address) {
		address := bitcoin.NewAddressFromRawAddress(itx.Inputs[0].Address, w.Config.Net)
		return fmt.Errorf("Instrument Creation not from contract : %s", address)
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

	instrumentCode, err := bitcoin.NewHash20(msg.InstrumentCode)
	if err != nil {
		node.LogVerbose(ctx, "Invalid instrument code : 0x%x", msg.InstrumentCode)
		return errors.Wrap(err, "invalid instrument code")
	}
	as, err := instrument.Retrieve(ctx, a.MasterDB, rk.Address, instrumentCode)
	if err != nil && err != instrument.ErrNotFound {
		return errors.Wrap(err, "Failed to retrieve instrument")
	}

	// Get request tx
	request, err := transactions.GetTx(ctx, a.MasterDB, &itx.Inputs[0].UTXO.Hash, a.Config.IsTest)
	if err != nil {
		return errors.Wrap(err, "Failed to retrieve request tx")
	}
	var vt *state.Vote
	var modification *actions.InstrumentModification
	if request != nil {
		var ok bool
		modification, ok = request.MsgProto.(*actions.InstrumentModification)

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

	// Create or update Instrument
	if as == nil {
		// Prepare creation object
		na := instrument.NewInstrument{}

		if err = node.Convert(ctx, &msg, &na); err != nil {
			return err
		}

		na.AdminAddress = ct.AdminAddress

		// Add instrument code if it hasn't been added yet. This will not add duplicates. This is
		//   required to handle the recovery case when the request will not be reprocessed.
		if err := contract.AddInstrumentCode(ctx, a.MasterDB, rk.Address, instrumentCode, a.Config.IsTest,
			v.Now); err != nil {
			return err
		}

		if err := instrument.Create(ctx, a.MasterDB, rk.Address, instrumentCode, &na, v.Now); err != nil {
			return errors.Wrap(err, "Failed to create instrument")
		}
		node.Log(ctx, "Created instrument %d : %s", msg.InstrumentIndex, instrumentCode.String())

		// Update administration balance
		h, err := holdings.GetHolding(ctx, a.MasterDB, rk.Address, instrumentCode,
			ct.AdminAddress, v.Now)
		if err != nil {
			return errors.Wrap(err, "Failed to get admin holding")
		}
		holdings.AddDeposit(h, itx.Hash, msg.AuthorizedTokenQty, true,
			protocol.NewTimestamp(msg.Timestamp))
		holdings.FinalizeTx(h, itx.Hash, msg.AuthorizedTokenQty,
			protocol.NewTimestamp(msg.Timestamp))
		cacheItem, err := holdings.Save(ctx, a.MasterDB, rk.Address, instrumentCode, h)
		if err != nil {
			return errors.Wrap(err, "Failed to save holdings")
		}
		a.HoldingsChannel.Add(cacheItem)

		// Update Owner/Administrator Membership instrument in contract
		if msg.InstrumentType == instruments.CodeMembership {
			instrumentPayload, err := instruments.Deserialize([]byte(msg.InstrumentType), msg.InstrumentPayload)
			if err != nil {
				node.LogWarn(ctx, "Failed to parse instrument payload : %s", err)
				return node.RespondReject(ctx, w, itx, rk, actions.RejectionsMsgMalformed)
			}

			membership, ok := instrumentPayload.(*instruments.Membership)
			if !ok {
				node.LogWarn(ctx, "Membership payload is wrong type")
				return node.RespondReject(ctx, w, itx, rk, actions.RejectionsMsgMalformed)
			}
			if membership.MembershipClass == "Owner" {
				updateContract := &contract.UpdateContract{
					OwnerMemberInstrument: instrumentCode,
				}
				if err := contract.Update(ctx, a.MasterDB, rk.Address, updateContract,
					a.Config.IsTest, v.Now); err != nil {
					return errors.Wrap(err, "updating contract")
				}
			}
			if membership.MembershipClass == "Administrator" {
				updateContract := &contract.UpdateContract{
					AdminMemberInstrument: instrumentCode,
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
		ua := instrument.UpdateInstrument{
			Revision:  &msg.InstrumentRevision,
			Timestamp: &ts,
		}

		if !bytes.Equal(as.InstrumentPermissions[:], msg.InstrumentPermissions[:]) {
			ua.InstrumentPermissions = &msg.InstrumentPermissions
			node.Log(ctx, "Updating instrument permissions (%s) : %s", instrumentCode,
				*ua.InstrumentPermissions)
		}
		if as.EnforcementOrdersPermitted != msg.EnforcementOrdersPermitted {
			ua.EnforcementOrdersPermitted = &msg.EnforcementOrdersPermitted
			node.Log(ctx, "Updating instrument enforcement orders permitted (%s) : %t", instrumentCode,
				*ua.EnforcementOrdersPermitted)
		}
		if as.VoteMultiplier != msg.VoteMultiplier {
			ua.VoteMultiplier = &msg.VoteMultiplier
			node.Log(ctx, "Updating instrument vote multiplier (%s) : %02x", instrumentCode,
				*ua.VoteMultiplier)
		}
		if as.AdministrationProposal != msg.AdministrationProposal {
			ua.AdministrationProposal = &msg.AdministrationProposal
			node.Log(ctx, "Updating instrument administration proposal (%s) : %t", instrumentCode,
				*ua.AdministrationProposal)
		}
		if as.HolderProposal != msg.HolderProposal {
			ua.HolderProposal = &msg.HolderProposal
			node.Log(ctx, "Updating instrument holder proposal (%s) : %t", instrumentCode,
				*ua.HolderProposal)
		}
		if as.InstrumentModificationGovernance != msg.InstrumentModificationGovernance {
			ua.InstrumentModificationGovernance = &msg.InstrumentModificationGovernance
			node.Log(ctx, "Updating instrument modification governance (%s) : %d", instrumentCode,
				*ua.InstrumentModificationGovernance)
		}

		var h *state.Holding
		updateHoldings := false
		if as.AuthorizedTokenQty != msg.AuthorizedTokenQty {
			ua.AuthorizedTokenQty = &msg.AuthorizedTokenQty
			node.Log(ctx, "Updating instrument token quantity %d : %s", *ua.AuthorizedTokenQty,
				instrumentCode)

			h, err = holdings.GetHolding(ctx, a.MasterDB, rk.Address, instrumentCode,
				ct.AdminAddress, v.Now)
			if err != nil {
				return errors.Wrap(err, "Failed to get admin holding")
			}

			if msg.AuthorizedTokenQty > as.AuthorizedTokenQty {
				node.Log(ctx, "Increasing token quantity by %d to %d : %s",
					msg.AuthorizedTokenQty-as.AuthorizedTokenQty, *ua.AuthorizedTokenQty, instrumentCode)
				holdings.FinalizeTx(h, itx.Hash, h.FinalizedBalance+(msg.AuthorizedTokenQty-as.AuthorizedTokenQty),
					protocol.NewTimestamp(msg.Timestamp))
			} else {
				node.Log(ctx, "Decreasing token quantity by %d to %d : %s",
					as.AuthorizedTokenQty-msg.AuthorizedTokenQty, *ua.AuthorizedTokenQty, instrumentCode)
				holdings.FinalizeTx(h, itx.Hash, h.FinalizedBalance-(as.AuthorizedTokenQty-msg.AuthorizedTokenQty),
					protocol.NewTimestamp(msg.Timestamp))
			}
			updateHoldings = true
			if err != nil {
				node.LogWarn(ctx, "Failed to update administration holding : %s", instrumentCode)
				return err
			}
		}
		if !bytes.Equal(as.InstrumentPayload, msg.InstrumentPayload) {
			ua.InstrumentPayload = &msg.InstrumentPayload
			node.Log(ctx, "Updating instrument payload (%s) : %s", instrumentCode, *ua.InstrumentPayload)
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
			cacheItem, err := holdings.Save(ctx, a.MasterDB, rk.Address, instrumentCode, h)
			if err != nil {
				return errors.Wrap(err, "Failed to save holdings")
			}
			a.HoldingsChannel.Add(cacheItem)
		}
		if err := instrument.Update(ctx, a.MasterDB, rk.Address, instrumentCode, &ua, v.Now); err != nil {
			node.LogWarn(ctx, "Failed to update instrument : %s", instrumentCode)
			return err
		}
		node.Log(ctx, "Updated instrument %d : %s", msg.InstrumentIndex, instrumentCode)

		// Mark vote as "applied" if this amendment was a result of a vote.
		if vt != nil {
			node.Log(ctx, "Marking vote as applied : %s", vt.VoteTxId)
			if err := vote.MarkApplied(ctx, a.MasterDB, rk.Address, vt.VoteTxId, request.Hash,
				v.Now); err != nil {
				return errors.Wrap(err, "Failed to mark vote applied")
			}
		}

		// Update Owner/Administrator Membership instrument in contract
		if msg.InstrumentType == instruments.CodeMembership {
			instrumentPayload, err := instruments.Deserialize([]byte(msg.InstrumentType), msg.InstrumentPayload)
			if err != nil {
				node.LogWarn(ctx, "Failed to parse instrument payload : %s", err)
				return node.RespondReject(ctx, w, itx, rk, actions.RejectionsMsgMalformed)
			}

			membership, ok := instrumentPayload.(*instruments.Membership)
			if !ok {
				node.LogWarn(ctx, "Membership payload is wrong type")
				return node.RespondReject(ctx, w, itx, rk, actions.RejectionsMsgMalformed)
			}

			if membership.MembershipClass == "Administrator" && !instrumentCode.Equal(&ct.AdminMemberInstrument) {
				// Set contract AdminMemberInstrument
				updateContract := &contract.UpdateContract{
					AdminMemberInstrument: instrumentCode,
				}
				if err := contract.Update(ctx, a.MasterDB, rk.Address, updateContract,
					a.Config.IsTest, v.Now); err != nil {
					return errors.Wrap(err, "updating contract")
				}
			} else if membership.MembershipClass != "Administrator" && instrumentCode.Equal(&ct.AdminMemberInstrument) {
				// Clear contract AdminMemberInstrument
				updateContract := &contract.UpdateContract{
					AdminMemberInstrument: &bitcoin.Hash20{}, // zero instrument code
				}
				if err := contract.Update(ctx, a.MasterDB, rk.Address, updateContract,
					a.Config.IsTest, v.Now); err != nil {
					return errors.Wrap(err, "updating contract")
				}
			}

			if membership.MembershipClass == "Owner" && !instrumentCode.Equal(&ct.OwnerMemberInstrument) {
				// Set contract OwnerMemberInstrument
				updateContract := &contract.UpdateContract{
					OwnerMemberInstrument: instrumentCode,
				}
				if err := contract.Update(ctx, a.MasterDB, rk.Address, updateContract,
					a.Config.IsTest, v.Now); err != nil {
					return errors.Wrap(err, "updating contract")
				}
			} else if membership.MembershipClass != "Owner" && instrumentCode.Equal(&ct.OwnerMemberInstrument) {
				// Clear contract OwnerMemberInstrument
				updateContract := &contract.UpdateContract{
					OwnerMemberInstrument: &bitcoin.Hash20{}, // zero instrument code
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

func applyInstrumentAmendments(ac *actions.InstrumentCreation, votingSystems []*actions.VotingSystemField,
	amendments []*actions.AmendmentField, proposed bool, proposalType, votingSystem uint32) error {

	perms, err := permissions.PermissionsFromBytes(ac.InstrumentPermissions, len(votingSystems))
	if err != nil {
		return fmt.Errorf("Invalid instrument permissions : %s", err)
	}

	var instrumentPayload instruments.Instrument

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
		case actions.InstrumentFieldInstrumentType:
			return node.NewError(actions.RejectionsInstrumentNotPermitted,
				"Instrument type amendments prohibited")

		case actions.InstrumentFieldInstrumentPermissions:
			if _, err := permissions.PermissionsFromBytes(amendment.Data,
				len(votingSystems)); err != nil {
				return fmt.Errorf("InstrumentPermissions amendment value is invalid : %s", err)
			}

		case actions.InstrumentFieldInstrumentPayload:
			if len(fip) == 1 {
				return node.NewError(actions.RejectionsInstrumentNotPermitted,
					"Amendments on complex fields (InstrumentPayload) prohibited")
			}

			if instrumentPayload == nil {
				// Get payload object
				instrumentPayload, err = instruments.Deserialize([]byte(ac.InstrumentType), ac.InstrumentPayload)
				if err != nil {
					return fmt.Errorf("Instrument payload deserialize failed : %s %s", ac.InstrumentType, err)
				}
			}

			payloadPermissions, err := perms.SubPermissions(
				permissions.FieldIndexPath{actions.InstrumentFieldInstrumentPayload}, 0, false)

			fieldPermissions, err = instrumentPayload.ApplyAmendment(fip[1:], amendment.Operation,
				amendment.Data, payloadPermissions)
			if err != nil {
				return errors.Wrapf(err, "apply amendment %d", i)
			}
			if len(fieldPermissions) == 0 {
				return errors.New("Invalid field permissions")
			}

			switch instrumentPayload.(type) {
			case *instruments.Membership:
				if fip[1] == instruments.MembershipFieldMembershipClass {
					return node.NewError(actions.RejectionsInstrumentNotPermitted,
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
					return node.NewError(actions.RejectionsInstrumentPermissions,
						fmt.Sprintf("Field %s amendment not permitted by administration proposal",
							fip))
				}
			case 1: // Holder
				if !permission.HolderProposal {
					return node.NewError(actions.RejectionsInstrumentPermissions,
						fmt.Sprintf("Field %s amendment not permitted by holder proposal", fip))
				}
			case 2: // Administrative Matter
				if !permission.AdministrativeMatter {
					return node.NewError(actions.RejectionsInstrumentPermissions,
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
				return node.NewError(actions.RejectionsInstrumentPermissions,
					fmt.Sprintf("Field %s amendment not allowed using voting system %d", fip,
						votingSystem))
			}
		} else if !permission.Permitted {
			return node.NewError(actions.RejectionsInstrumentPermissions,
				fmt.Sprintf("Field %s amendment not permitted without proposal", fip))
		}
	}

	if instrumentPayload != nil {
		if err = instrumentPayload.Validate(); err != nil {
			return err
		}

		newPayload, err := instrumentPayload.Bytes()
		if err != nil {
			return err
		}

		ac.InstrumentPayload = newPayload
	}

	// Check validity of updated instrument data
	if err := ac.Validate(); err != nil {
		return fmt.Errorf("Instrument data invalid after amendments : %s", err)
	}

	return nil
}
