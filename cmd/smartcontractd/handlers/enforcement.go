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
	"github.com/tokenized/smart-contract/pkg/inspector"
	"github.com/tokenized/smart-contract/pkg/wallet"
	"github.com/tokenized/specification/dist/golang/actions"
	"github.com/tokenized/specification/dist/golang/protocol"

	"github.com/pkg/errors"
	"go.opencensus.io/trace"
)

type Enforcement struct {
	MasterDB        *db.DB
	Config          *node.Config
	HoldingsChannel *holdings.CacheChannel
}

// OrderRequest handles an incoming Order request and prepares a Confiscation response
func (e *Enforcement) OrderRequest(ctx context.Context, w *node.ResponseWriter,
	itx *inspector.Transaction, rk *wallet.Key) error {

	ctx, span := trace.StartSpan(ctx, "handlers.Enforcement.Order")
	defer span.End()

	msg, ok := itx.MsgProto.(*actions.Order)
	if !ok {
		return errors.New("Could not assert as *actions.Order")
	}

	// Validate all fields have valid values.
	if itx.RejectCode != 0 {
		node.LogWarn(ctx, "Order invalid : %d %s", itx.RejectCode, itx.RejectText)
		return node.RespondRejectText(ctx, w, itx, rk, itx.RejectCode, itx.RejectText)
	}

	ct, err := contract.Retrieve(ctx, e.MasterDB, rk.Address, e.Config.IsTest)
	if err != nil {
		return errors.Wrap(err, "Failed to retrieve contract")
	}

	if !ct.MovedTo.IsEmpty() {
		address := bitcoin.NewAddressFromRawAddress(ct.MovedTo, w.Config.Net)
		node.LogWarn(ctx, "Contract address changed : %s", address.String())
		return node.RespondReject(ctx, w, itx, rk, actions.RejectionsContractMoved)
	}

	v := ctx.Value(node.KeyValues).(*node.Values)

	if ct.ContractExpiration.Nano() != 0 && ct.ContractExpiration.Nano() < v.Now.Nano() {
		node.LogWarn(ctx, "Contract expired : %s", ct.ContractExpiration.String())
		return node.RespondReject(ctx, w, itx, rk, actions.RejectionsContractExpired)
	}

	if !contract.IsOperator(ctx, ct, itx.Inputs[0].Address) {
		address := bitcoin.NewAddressFromRawAddress(itx.Inputs[0].Address, w.Config.Net)
		node.LogWarn(ctx, "Requestor PKH is not administration or operator : %s", address.String())
		return node.RespondReject(ctx, w, itx, rk, actions.RejectionsNotOperator)
	}

	// Validate enforcement authority public key and signature
	if len(msg.OrderSignature) > 0 || msg.SignatureAlgorithm != 0 {
		if msg.SignatureAlgorithm != 1 {
			node.LogWarn(ctx, "Invalid authority sig algo : %02x", msg.SignatureAlgorithm)
			return node.RespondReject(ctx, w, itx, rk, actions.RejectionsMsgMalformed)
		}

		authorityPubKey, err := bitcoin.PublicKeyFromBytes(msg.AuthorityPublicKey)
		if err != nil {
			node.LogWarn(ctx, "Failed to parse authority pub key : %s", err)
			return node.RespondReject(ctx, w, itx, rk, actions.RejectionsMsgMalformed)
		}

		authoritySig, err := bitcoin.SignatureFromBytes(msg.OrderSignature)
		if err != nil {
			node.LogWarn(ctx, "Failed to parse authority signature : %s", err)
			return node.RespondReject(ctx, w, itx, rk, actions.RejectionsMsgMalformed)
		}

		sigHash, err := protocol.OrderAuthoritySigHash(ctx, rk.Address, msg)
		if err != nil {
			return errors.Wrap(err, "Failed to calculate authority sig hash")
		}

		if !authoritySig.Verify(*sigHash, authorityPubKey) {
			node.LogWarn(ctx, "Authority Sig Verify Failed")
			return node.RespondReject(ctx, w, itx, rk, actions.RejectionsInvalidSignature)
		}
	}

	// Apply logic based on Compliance Action type
	switch msg.ComplianceAction {
	case actions.ComplianceActionFreeze:
		return e.OrderFreezeRequest(ctx, w, itx, rk)
	case actions.ComplianceActionThaw:
		return e.OrderThawRequest(ctx, w, itx, rk)
	case actions.ComplianceActionConfiscation:
		return e.OrderConfiscateRequest(ctx, w, itx, rk)
	case actions.ComplianceActionReconciliation:
		return e.OrderReconciliationRequest(ctx, w, itx, rk)
	default:
		return fmt.Errorf("Unknown compliance action : %s", string(msg.ComplianceAction))
	}
}

// OrderFreezeRequest is a helper of Order
func (e *Enforcement) OrderFreezeRequest(ctx context.Context, w *node.ResponseWriter,
	itx *inspector.Transaction, rk *wallet.Key) error {

	ctx, span := trace.StartSpan(ctx, "handlers.Enforcement.OrderFreezeRequest")
	defer span.End()

	msg, ok := itx.MsgProto.(*actions.Order)
	if !ok {
		return errors.New("Could not assert as *protocol.Order")
	}

	v := ctx.Value(node.KeyValues).(*node.Values)

	ct, err := contract.Retrieve(ctx, e.MasterDB, rk.Address, e.Config.IsTest)
	if err != nil {
		return errors.Wrap(err, "Failed to retrieve contract")
	}

	if !contract.IsOperator(ctx, ct, itx.Inputs[0].Address) {
		address := bitcoin.NewAddressFromRawAddress(itx.Inputs[0].Address, w.Config.Net)
		node.LogVerbose(ctx, "Requestor is not operator : %s", address.String())
		return node.RespondReject(ctx, w, itx, rk, actions.RejectionsNotOperator)
	}

	// Freeze <- Order
	freeze := actions.Freeze{
		Timestamp: v.Now.Nano(),
	}

	err = node.Convert(ctx, msg, &freeze)
	if err != nil {
		return errors.Wrap(err, "Failed to convert freeze order to freeze")
	}

	full := false
	if len(msg.TargetAddresses) == 0 {
		node.LogWarn(ctx, "No freeze target addresses specified")
		return node.RespondReject(ctx, w, itx, rk, actions.RejectionsMsgMalformed)
	} else if len(msg.TargetAddresses) == 1 && bytes.Equal(msg.TargetAddresses[0].Address, rk.Address.Bytes()) {
		full = true
		freeze.Quantities = append(freeze.Quantities, &actions.QuantityIndexField{Index: 0, Quantity: 0})
	}

	// Outputs
	// 1..n - Target Addresses
	// n+1  - Contract Address
	// n+2  - Contract Fee (change)
	if len(msg.InstrumentCode) == 0 {
		if !full {
			node.LogWarn(ctx, "Zero instrument code in non-full freeze")
			return node.RespondReject(ctx, w, itx, rk, actions.RejectionsMsgMalformed)
		}
	} else {
		instrumentCode, err := bitcoin.NewHash20(msg.InstrumentCode)
		if err != nil {
			node.LogVerbose(ctx, "Invalid instrument code : 0x%x", msg.InstrumentCode)
			return node.RespondReject(ctx, w, itx, rk, actions.RejectionsMsgMalformed)
		}

		as, err := instrument.Retrieve(ctx, e.MasterDB, rk.Address, instrumentCode)
		if err != nil {
			node.LogWarn(ctx, "Instrument ID not found : %s : %s", instrumentCode, err)
			return node.RespondReject(ctx, w, itx, rk, actions.RejectionsInstrumentNotFound)
		}

		if !as.EnforcementOrdersPermitted {
			node.LogWarn(ctx, "Enforcement orders not permitted on instrument : %s", instrumentCode)
			return node.RespondReject(ctx, w, itx, rk, actions.RejectionsInstrumentNotPermitted)
		}

		if !full {
			outputIndex := uint32(0)
			used := make(map[bitcoin.Hash20]bool)

			// Validate target addresses
			for _, target := range msg.TargetAddresses {
				targetAddress, err := bitcoin.DecodeRawAddress(target.Address)
				if err != nil {
					return errors.Wrap(err, "Failed to read target address")
				}
				address := bitcoin.NewAddressFromRawAddress(targetAddress,
					w.Config.Net)
				node.Log(ctx, "Freeze order request : %s %s", instrumentCode, address)

				if target.Quantity == 0 {
					node.LogWarn(ctx, "Zero quantity order is invalid : %s %s", instrumentCode,
						address.String())
					return node.RespondReject(ctx, w, itx, rk, actions.RejectionsMsgMalformed)
				}

				hash, err := targetAddress.Hash()
				if err != nil {
					node.LogWarn(ctx, "Invalid freeze address : %s %s", instrumentCode, address)
					return node.RespondReject(ctx, w, itx, rk, actions.RejectionsMsgMalformed)
				}
				_, exists := used[*hash]
				if exists {
					node.LogWarn(ctx, "Address used more than once : %s %s", instrumentCode,
						address)
					return node.RespondReject(ctx, w, itx, rk, actions.RejectionsMsgMalformed)
				}

				used[*hash] = true

				// Notify target address
				w.AddOutput(ctx, targetAddress, 0)

				freeze.Quantities = append(freeze.Quantities,
					&actions.QuantityIndexField{Index: outputIndex, Quantity: target.Quantity})
				outputIndex++
			}
		}
	}

	// Add contract output
	w.AddOutput(ctx, rk.Address, 0)

	// Add fee output
	w.AddContractFee(ctx, ct.ContractFee)

	// Respond with a freeze action
	if err := node.RespondSuccess(ctx, w, itx, rk, &freeze); err != nil {
		return errors.Wrap(err, "Failed to respond")
	}

	return nil
}

// OrderThawRequest is a helper of Order
func (e *Enforcement) OrderThawRequest(ctx context.Context, w *node.ResponseWriter,
	itx *inspector.Transaction, rk *wallet.Key) error {

	ctx, span := trace.StartSpan(ctx, "handlers.Enforcement.OrderThawRequest")
	defer span.End()

	msg, ok := itx.MsgProto.(*actions.Order)
	if !ok {
		return errors.New("Could not assert as *protocol.Order")
	}

	v := ctx.Value(node.KeyValues).(*node.Values)

	ct, err := contract.Retrieve(ctx, e.MasterDB, rk.Address, e.Config.IsTest)
	if err != nil {
		return errors.Wrap(err, "Failed to retrieve contract")
	}

	if !contract.IsOperator(ctx, ct, itx.Inputs[0].Address) {
		address := bitcoin.NewAddressFromRawAddress(itx.Inputs[0].Address,
			w.Config.Net)
		node.LogVerbose(ctx, "Requestor is not operator : %s", address)
		return node.RespondReject(ctx, w, itx, rk, actions.RejectionsNotOperator)
	}

	// Get Freeze Tx
	hash, err := bitcoin.NewHash32(msg.FreezeTxId)
	freezeTx, err := transactions.GetTx(ctx, e.MasterDB, hash, e.Config.IsTest)
	if err != nil {
		return fmt.Errorf("Failed to retrieve freeze tx for thaw : %s : %s", msg.FreezeTxId, err)
	}

	// Get Freeze Op Return
	freeze, ok := freezeTx.MsgProto.(*actions.Freeze)
	if !ok {
		return fmt.Errorf("Failed to assert freeze tx op return : %s", msg.FreezeTxId)
	}

	// Thaw <- Order
	thaw := actions.Thaw{
		FreezeTxId: msg.FreezeTxId,
		Timestamp:  v.Now.Nano(),
	}

	full := false
	if len(freeze.Quantities) == 0 {
		node.LogWarn(ctx, "No freeze target addresses specified")
		return node.RespondReject(ctx, w, itx, rk, actions.RejectionsMsgMalformed)
	} else if len(freeze.Quantities) == 1 &&
		freezeTx.Outputs[freeze.Quantities[0].Index].Address.Equal(rk.Address) {
		full = true
	}

	// Outputs
	// 1..n - Target Addresses
	// n+1  - Contract Address
	// n+2  - Contract Fee (change)
	if len(freeze.InstrumentCode) == 0 {
		if !full {
			node.LogWarn(ctx, "Zero instrument code in non-full freeze")
			return node.RespondReject(ctx, w, itx, rk, actions.RejectionsMsgMalformed)
		}
	} else {
		instrumentCode, err := bitcoin.NewHash20(freeze.InstrumentCode)
		if err != nil {
			node.LogVerbose(ctx, "Invalid instrument code : %s", err)
			return node.RespondReject(ctx, w, itx, rk, actions.RejectionsMsgMalformed)
		}

		as, err := instrument.Retrieve(ctx, e.MasterDB, rk.Address, instrumentCode)
		if err != nil {
			node.LogWarn(ctx, "Instrument not found: %s", instrumentCode)
			return node.RespondReject(ctx, w, itx, rk, actions.RejectionsInstrumentNotFound)
		}

		if !as.EnforcementOrdersPermitted {
			node.LogWarn(ctx, "Enforcement orders not permitted on instrument : %s", instrumentCode)
			return node.RespondReject(ctx, w, itx, rk, actions.RejectionsInstrumentNotPermitted)
		}

		if !full {
			// Validate target addresses
			txid, err := bitcoin.NewHash32(msg.FreezeTxId)
			if err != nil {
				node.LogVerbose(ctx, "Invalid freeze txid : %s", err)
				return node.RespondReject(ctx, w, itx, rk, actions.RejectionsMsgMalformed)
			}

			for _, quantity := range freeze.Quantities {
				address := bitcoin.NewAddressFromRawAddress(freezeTx.Outputs[quantity.Index].Address,
					w.Config.Net)
				h, err := holdings.GetHolding(ctx, e.MasterDB, rk.Address,
					instrumentCode, freezeTx.Outputs[quantity.Index].Address, v.Now)
				if err != nil {
					return errors.Wrap(err, "Failed to get holding")
				}

				err = holdings.CheckFreeze(h, txid, quantity.Quantity)
				if err != nil {
					node.LogWarn(ctx, "Freeze holding status invalid : %s : %s", address, err)
					return node.RespondReject(ctx, w, itx, rk, actions.RejectionsMsgMalformed)
				}

				node.Log(ctx, "Thaw order request : %s %s", instrumentCode, address)

				// Notify target address
				w.AddOutput(ctx, freezeTx.Outputs[quantity.Index].Address, 0)
			}
		}
	}

	// Add contract output
	w.AddOutput(ctx, rk.Address, 0)

	// Add fee output
	w.AddContractFee(ctx, ct.ContractFee)

	// Respond with a thaw action
	return node.RespondSuccess(ctx, w, itx, rk, &thaw)
}

// OrderConfiscateRequest is a helper of Order
func (e *Enforcement) OrderConfiscateRequest(ctx context.Context, w *node.ResponseWriter,
	itx *inspector.Transaction, rk *wallet.Key) error {
	ctx, span := trace.StartSpan(ctx, "handlers.Enforcement.OrderConfiscateRequest")
	defer span.End()

	msg, ok := itx.MsgProto.(*actions.Order)
	if !ok {
		return errors.New("Could not assert as *protocol.Order")
	}

	v := ctx.Value(node.KeyValues).(*node.Values)

	ct, err := contract.Retrieve(ctx, e.MasterDB, rk.Address, e.Config.IsTest)
	if err != nil {
		return errors.Wrap(err, "Failed to retrieve contract")
	}

	if !contract.IsOperator(ctx, ct, itx.Inputs[0].Address) {
		address := bitcoin.NewAddressFromRawAddress(itx.Inputs[0].Address, w.Config.Net)
		node.LogVerbose(ctx, "Requestor is not operator : %s", address.String())
		return node.RespondReject(ctx, w, itx, rk, actions.RejectionsNotOperator)
	}

	instrumentCode, err := bitcoin.NewHash20(msg.InstrumentCode)
	if err != nil {
		node.LogVerbose(ctx, "Invalid instrument code : 0x%x", msg.InstrumentCode)
		return node.RespondReject(ctx, w, itx, rk, actions.RejectionsMsgMalformed)
	}

	as, err := instrument.Retrieve(ctx, e.MasterDB, rk.Address, instrumentCode)
	if err != nil {
		node.LogWarn(ctx, "Instrument not found : %s", instrumentCode)
		return node.RespondReject(ctx, w, itx, rk, actions.RejectionsInstrumentNotFound)
	}

	if !as.EnforcementOrdersPermitted {
		node.LogWarn(ctx, "Enforcement orders not permitted on instrument : %s", instrumentCode)
		return node.RespondReject(ctx, w, itx, rk, actions.RejectionsInstrumentNotPermitted)
	}

	hds := make(map[bitcoin.Hash20]*state.Holding)

	// Confiscation <- Order
	confiscation := actions.Confiscation{}

	if err := node.Convert(ctx, msg, &confiscation); err != nil {
		return errors.Wrap(err, "Failed to convert confiscation order to confiscation")
	}

	confiscation.Timestamp = v.Now.Nano()
	confiscation.Quantities = make([]*actions.QuantityIndexField, 0, len(msg.TargetAddresses))

	// Build outputs
	// 1..n - Target Addresses
	// n+1  - Deposit Address
	// n+2  - Contract Address
	// n+3  - Contract Fee (change)

	// Validate deposit address, and increase balance by confiscation.DepositQty and increase
	// DepositQty by previous balance
	depositAddress, err := bitcoin.DecodeRawAddress(msg.DepositAddress)
	if err != nil {
		return errors.Wrap(err, "Failed to read deposit address")
	}

	// Holdings check
	depositAmount := uint64(0)

	// Validate target addresses
	outputIndex := uint32(0)
	for _, target := range msg.TargetAddresses {
		targetAddress, err := bitcoin.DecodeRawAddress(target.Address)
		if err != nil {
			return errors.Wrap(err, "Failed to read target address")
		}
		address := bitcoin.NewAddressFromRawAddress(targetAddress, w.Config.Net)

		if target.Quantity == 0 {
			node.LogWarn(ctx, "Zero quantity confiscation order is invalid : %s %s", instrumentCode,
				address)
			return node.RespondReject(ctx, w, itx, rk, actions.RejectionsMsgMalformed)
		}

		hash, err := targetAddress.Hash()
		if err != nil {
			node.LogWarn(ctx, "Invalid confiscation address : %s %s", instrumentCode, address)
			return node.RespondReject(ctx, w, itx, rk, actions.RejectionsMsgMalformed)
		}
		_, exists := hds[*hash]
		if exists {
			node.LogWarn(ctx, "Address used more than once : %s %s", instrumentCode, address)
			return node.RespondReject(ctx, w, itx, rk, actions.RejectionsMsgMalformed)
		}

		h, err := holdings.GetHolding(ctx, e.MasterDB, rk.Address, instrumentCode, targetAddress, v.Now)
		if err != nil {
			return errors.Wrap(err, "Failed to get holding")
		}

		err = holdings.AddDebit(h, itx.Hash, target.Quantity, true, v.Now)
		if err != nil {
			node.LogWarn(ctx, "Failed confiscation for holding : %s %s : %s", instrumentCode,
				address.String(), err)
			if err == holdings.ErrInsufficientHoldings {
				return node.RespondReject(ctx, w, itx, rk, actions.RejectionsInsufficientQuantity)
			} else {
				return node.RespondReject(ctx, w, itx, rk, actions.RejectionsMsgMalformed)
			}
		}

		hds[*hash] = h
		depositAmount += target.Quantity

		confiscation.Quantities = append(confiscation.Quantities,
			&actions.QuantityIndexField{Index: outputIndex, Quantity: h.PendingBalance})

		node.Log(ctx, "Confiscation order request : %s %s", instrumentCode, address)

		// Notify target address
		w.AddOutput(ctx, targetAddress, 0)
		outputIndex++
	}

	hash, err := depositAddress.Hash()
	if err != nil {
		address := bitcoin.NewAddressFromRawAddress(depositAddress, w.Config.Net)
		node.LogWarn(ctx, "Invalid deposit address : %s %s", instrumentCode, address)
		return node.RespondReject(ctx, w, itx, rk, actions.RejectionsMsgMalformed)
	}
	_, exists := hds[*hash]
	if exists {
		address := bitcoin.NewAddressFromRawAddress(depositAddress, w.Config.Net)
		node.LogWarn(ctx, "Deposit address already used : %s %s", instrumentCode, address)
		return node.RespondReject(ctx, w, itx, rk, actions.RejectionsMsgMalformed)
	}

	depositHolding, err := holdings.GetHolding(ctx, e.MasterDB, rk.Address, instrumentCode,
		depositAddress, v.Now)
	if err != nil {
		return errors.Wrap(err, "Failed to get holding")
	}
	err = holdings.AddDeposit(depositHolding, itx.Hash, depositAmount, true, v.Now)
	if err != nil {
		address := bitcoin.NewAddressFromRawAddress(depositAddress,
			w.Config.Net)
		node.LogWarn(ctx, "Failed confiscation deposit : %s %s : %s", instrumentCode, address, err)
		return node.RespondReject(ctx, w, itx, rk, actions.RejectionsMsgMalformed)
	}
	hds[*hash] = depositHolding
	confiscation.DepositQty = depositHolding.PendingBalance

	// Notify deposit address
	w.AddOutput(ctx, depositAddress, 0)

	// Add contract output
	w.AddOutput(ctx, rk.Address, 0)

	// Add fee output
	w.AddContractFee(ctx, ct.ContractFee)

	// Respond with a confiscation action
	err = node.RespondSuccess(ctx, w, itx, rk, &confiscation)
	if err != nil {
		return err
	}

	for _, h := range hds {
		cacheItem, err := holdings.Save(ctx, e.MasterDB, rk.Address, instrumentCode, h)
		if err != nil {
			return errors.Wrap(err, "Failed to save holding")
		}
		e.HoldingsChannel.Add(cacheItem)
	}
	node.Log(ctx, "Updated holdings : %s", instrumentCode)
	return nil
}

// OrderReconciliationRequest is a helper of Order
func (e *Enforcement) OrderReconciliationRequest(ctx context.Context, w *node.ResponseWriter,
	itx *inspector.Transaction, rk *wallet.Key) error {

	ctx, span := trace.StartSpan(ctx, "handlers.Enforcement.OrderReconciliationRequest")
	defer span.End()

	msg, ok := itx.MsgProto.(*actions.Order)
	if !ok {
		return errors.New("Could not assert as *actions.Order")
	}

	v := ctx.Value(node.KeyValues).(*node.Values)

	ct, err := contract.Retrieve(ctx, e.MasterDB, rk.Address, e.Config.IsTest)
	if err != nil {
		return errors.Wrap(err, "Failed to retrieve contract")
	}

	if !contract.IsOperator(ctx, ct, itx.Inputs[0].Address) {
		address := bitcoin.NewAddressFromRawAddress(itx.Inputs[0].Address, w.Config.Net)
		node.LogVerbose(ctx, "Requestor is not operator : %s", address.String())
		return node.RespondReject(ctx, w, itx, rk, actions.RejectionsNotOperator)
	}

	instrumentCode, err := bitcoin.NewHash20(msg.InstrumentCode)
	if err != nil {
		node.LogVerbose(ctx, "Invalid instrument code : 0x%x", msg.InstrumentCode)
		return node.RespondReject(ctx, w, itx, rk, actions.RejectionsMsgMalformed)
	}

	as, err := instrument.Retrieve(ctx, e.MasterDB, rk.Address, instrumentCode)
	if err != nil {
		node.LogWarn(ctx, "Instrument not found: %s", instrumentCode)
		return node.RespondReject(ctx, w, itx, rk, actions.RejectionsInstrumentNotFound)
	}

	if !as.EnforcementOrdersPermitted {
		node.LogWarn(ctx, "Enforcement orders not permitted on instrument : %s", instrumentCode)
		return node.RespondReject(ctx, w, itx, rk, actions.RejectionsInstrumentNotPermitted)
	}

	// Reconciliation <- Order
	reconciliation := actions.Reconciliation{}

	err = node.Convert(ctx, msg, &reconciliation)
	if err != nil {
		return errors.Wrap(err, "Failed to convert reconciliation order to reconciliation")
	}

	reconciliation.Timestamp = v.Now.Nano()
	reconciliation.Quantities = make([]*actions.QuantityIndexField, 0, len(msg.TargetAddresses))
	hds := make(map[bitcoin.Hash20]*state.Holding)

	// Build outputs
	// 1..n - Target Addresses
	// n+1  - Contract Address
	// n+2  - Contract Fee (change)

	// Validate target addresses
	outputIndex := uint32(0)
	addressOutputIndex := make([]uint32, 0, len(msg.TargetAddresses))
	outputs := make([]node.Output, 0, len(msg.TargetAddresses))
	for _, target := range msg.TargetAddresses {
		targetAddress, err := bitcoin.DecodeRawAddress(target.Address)
		if err != nil {
			return errors.Wrap(err, "Failed to read target address")
		}
		address := bitcoin.NewAddressFromRawAddress(targetAddress, w.Config.Net)

		if target.Quantity == 0 {
			node.LogWarn(ctx, "Zero quantity reconciliation order is invalid : %s %s",
				instrumentCode, address)
			return node.RespondReject(ctx, w, itx, rk, actions.RejectionsMsgMalformed)
		}

		hash, err := targetAddress.Hash()
		if err != nil {
			node.LogWarn(ctx, "Invalid reconcile address : %s %s", instrumentCode, address)
			return node.RespondReject(ctx, w, itx, rk, actions.RejectionsMsgMalformed)
		}
		_, exists := hds[*hash]
		if exists {
			node.LogWarn(ctx, "Address used more than once : %s %s", instrumentCode, address)
			return node.RespondReject(ctx, w, itx, rk, actions.RejectionsMsgMalformed)
		}

		h, err := holdings.GetHolding(ctx, e.MasterDB, rk.Address, instrumentCode, targetAddress, v.Now)
		if err != nil {
			return errors.Wrap(err, "Failed to get holding")
		}

		err = holdings.AddDebit(h, itx.Hash, target.Quantity, true, v.Now)
		if err != nil {
			node.LogWarn(ctx, "Failed reconciliation for holding : %s %s : %s", instrumentCode,
				address, err)
			if err == holdings.ErrInsufficientHoldings {
				return node.RespondReject(ctx, w, itx, rk, actions.RejectionsInsufficientQuantity)
			} else {
				return node.RespondReject(ctx, w, itx, rk, actions.RejectionsMsgMalformed)
			}
		}

		hds[*hash] = h

		reconciliation.Quantities = append(reconciliation.Quantities,
			&actions.QuantityIndexField{Index: outputIndex, Quantity: h.PendingBalance})

		node.Log(ctx, "Reconciliation order request : %s %s", instrumentCode, address)

		// Notify target address
		outputs = append(outputs, node.Output{Address: targetAddress, Value: 0})
		addressOutputIndex = append(addressOutputIndex, outputIndex)
		outputIndex++
	}

	// Update outputs with bitcoin dispersions
	for _, quantity := range msg.BitcoinDispersions {
		if int(quantity.Index) >= len(msg.TargetAddresses) {
			node.LogWarn(ctx, "Invalid bitcoin dispersion index : %s %d", instrumentCode,
				quantity.Index)
			return node.RespondReject(ctx, w, itx, rk, actions.RejectionsMsgMalformed)
		}

		outputs[addressOutputIndex[quantity.Index]].Value += quantity.Quantity
	}

	// Add outputs to response writer
	for _, output := range outputs {
		w.AddOutput(ctx, output.Address, output.Value)
	}

	// Add contract output
	w.AddOutput(ctx, rk.Address, 0)

	// Add fee output
	w.AddContractFee(ctx, ct.ContractFee)

	// Respond with a reconciliation action
	err = node.RespondSuccess(ctx, w, itx, rk, &reconciliation)
	if err != nil {
		return err
	}

	for _, h := range hds {
		cacheItem, err := holdings.Save(ctx, e.MasterDB, rk.Address, instrumentCode, h)
		if err != nil {
			return errors.Wrap(err, "Failed to save holding")
		}
		e.HoldingsChannel.Add(cacheItem)
	}
	return nil
}

// FreezeResponse handles an outgoing Freeze action and writes it to the state
func (e *Enforcement) FreezeResponse(ctx context.Context, w *node.ResponseWriter,
	itx *inspector.Transaction, rk *wallet.Key) error {

	ctx, span := trace.StartSpan(ctx, "handlers.Enforcement.Freeze")
	defer span.End()

	msg, ok := itx.MsgProto.(*actions.Freeze)
	if !ok {
		return errors.New("Could not assert as *actions.Freeze")
	}

	if !itx.Inputs[0].Address.Equal(rk.Address) {
		address := bitcoin.NewAddressFromRawAddress(itx.Inputs[0].Address,
			w.Config.Net)
		return fmt.Errorf("Freeze not from contract : %s", address.String())
	}

	ct, err := contract.Retrieve(ctx, e.MasterDB, rk.Address, e.Config.IsTest)
	if err != nil {
		return errors.Wrap(err, "Failed to retrieve contract")
	}

	if !ct.MovedTo.IsEmpty() {
		address := bitcoin.NewAddressFromRawAddress(ct.MovedTo,
			w.Config.Net)
		return fmt.Errorf("Contract address changed : %s", address.String())
	}

	full := false
	if len(msg.Quantities) == 0 {
		return fmt.Errorf("No freeze addresses specified")
	} else if len(msg.Quantities) == 1 &&
		itx.Outputs[msg.Quantities[0].Index].Address.Equal(rk.Address) {
		full = true
	}

	if len(msg.InstrumentCode) == 0 {
		if !full {
			return fmt.Errorf("Zero instrument code in non-full freeze")
		} else {
			// Contract wide freeze
			ts := protocol.NewTimestamp(msg.FreezePeriod)
			uc := contract.UpdateContract{FreezePeriod: &ts}
			if err := contract.Update(ctx, e.MasterDB, rk.Address, &uc, e.Config.IsTest,
				protocol.NewTimestamp(msg.Timestamp)); err != nil {
				return errors.Wrap(err, "Failed to update contract freeze period")
			}
		}
	} else {
		instrumentCode, err := bitcoin.NewHash20(msg.InstrumentCode)
		if err != nil {
			return errors.Wrap(err, "invalid instrument code")
		}

		if full {
			// Instrument wide freeze
			ts := protocol.NewTimestamp(msg.FreezePeriod)
			ua := instrument.UpdateInstrument{FreezePeriod: &ts}
			if err := instrument.Update(ctx, e.MasterDB, rk.Address, instrumentCode, &ua,
				protocol.NewTimestamp(msg.Timestamp)); err != nil {
				return errors.Wrap(err, "Failed to update instrument freeze period")
			}
		} else {
			hds := make(map[bitcoin.Hash20]*state.Holding)
			timestamp := protocol.NewTimestamp(msg.Timestamp)
			freezePeriod := protocol.NewTimestamp(msg.FreezePeriod)

			// Validate target addresses
			for _, quantity := range msg.Quantities {
				if int(quantity.Index) >= len(itx.Outputs) {
					return fmt.Errorf("Freeze quantity index out of range : %d/%d", quantity.Index,
						len(itx.Outputs))
				}

				hash, err := itx.Outputs[quantity.Index].Address.Hash()
				if err != nil {
					return errors.Wrap(err, "Invalid freeze address")
				}
				_, exists := hds[*hash]
				if exists {
					address := bitcoin.NewAddressFromRawAddress(itx.Outputs[quantity.Index].Address,
						w.Config.Net)
					node.LogWarn(ctx, "Address used more than once : %s %s", instrumentCode, address)
					return fmt.Errorf("Address used more than once : %s %s", instrumentCode, address)
				}

				h, err := holdings.GetHolding(ctx, e.MasterDB, rk.Address, instrumentCode,
					itx.Outputs[quantity.Index].Address, timestamp)
				if err != nil {
					return errors.Wrap(err, "Failed to get holding")
				}

				err = holdings.AddFreeze(h, itx.Hash, quantity.Quantity, freezePeriod, timestamp)
				if err != nil {
					address := bitcoin.NewAddressFromRawAddress(itx.Outputs[quantity.Index].Address,
						w.Config.Net)
					node.LogWarn(ctx, "Failed to add freeze to holding : %s %s : %s", instrumentCode,
						address, err)
					return fmt.Errorf("Failed to add freeze to holding : %s %s : %s", instrumentCode,
						address, err)
				}

				hds[*hash] = h
			}

			for _, h := range hds {
				cacheItem, err := holdings.Save(ctx, e.MasterDB, rk.Address, instrumentCode, h)
				if err != nil {
					return errors.Wrap(err, "Failed to save holding")
				}
				e.HoldingsChannel.Add(cacheItem)
			}
		}
	}

	// Save Tx for thaw action.
	if err := transactions.AddTx(ctx, e.MasterDB, itx); err != nil {
		return errors.Wrap(err, "Failed to save tx")
	}

	node.Log(ctx, "Processed Freeze : %s", itx.Hash)
	return nil
}

// ThawResponse handles an outgoing Thaw action and writes it to the state
func (e *Enforcement) ThawResponse(ctx context.Context, w *node.ResponseWriter,
	itx *inspector.Transaction, rk *wallet.Key) error {

	ctx, span := trace.StartSpan(ctx, "handlers.Enforcement.Thaw")
	defer span.End()

	msg, ok := itx.MsgProto.(*actions.Thaw)
	if !ok {
		return errors.New("Could not assert as *protocol.Thaw")
	}

	if !itx.Inputs[0].Address.Equal(rk.Address) {
		address := bitcoin.NewAddressFromRawAddress(itx.Inputs[0].Address,
			w.Config.Net)
		return fmt.Errorf("Thaw not from contract : %s", address.String())
	}

	ct, err := contract.Retrieve(ctx, e.MasterDB, rk.Address, e.Config.IsTest)
	if err != nil {
		return errors.Wrap(err, "Failed to retrieve contract")
	}

	if !ct.MovedTo.IsEmpty() {
		address := bitcoin.NewAddressFromRawAddress(ct.MovedTo,
			w.Config.Net)
		return fmt.Errorf("Contract address changed : %s", address.String())
	}

	// Get Freeze Tx
	hash, _ := bitcoin.NewHash32(msg.FreezeTxId)
	freezeTx, err := transactions.GetTx(ctx, e.MasterDB, hash, e.Config.IsTest)
	if err != nil {
		return fmt.Errorf("Failed to retrieve freeze tx for thaw : %s : %s", msg.FreezeTxId, err)
	}

	// Get Freeze Op Return
	freeze, ok := freezeTx.MsgProto.(*actions.Freeze)
	if !ok {
		return fmt.Errorf("Failed to assert freeze tx op return : %s", msg.FreezeTxId)
	}

	full := false
	if len(freeze.Quantities) == 0 {
		return fmt.Errorf("No freeze addresses specified")
	} else if len(freeze.Quantities) == 1 &&
		freezeTx.Outputs[freeze.Quantities[0].Index].Address.Equal(rk.Address) {
		full = true
	}

	if len(freeze.InstrumentCode) == 0 {
		if !full {
			return fmt.Errorf("Zero instrument code in non-full freeze")
		} else {
			// Contract wide freeze
			var zeroTimestamp protocol.Timestamp
			uc := contract.UpdateContract{FreezePeriod: &zeroTimestamp}
			if err := contract.Update(ctx, e.MasterDB, rk.Address, &uc, e.Config.IsTest,
				protocol.NewTimestamp(msg.Timestamp)); err != nil {
				return errors.Wrap(err, "Failed to clear contract freeze period")
			}
		}
	} else {
		instrumentCode, err := bitcoin.NewHash20(freeze.InstrumentCode)
		if err != nil {
			return errors.Wrap(err, "invalid instrument code")
		}

		if full {
			// Instrument wide freeze
			var zeroTimestamp protocol.Timestamp
			ua := instrument.UpdateInstrument{FreezePeriod: &zeroTimestamp}
			if err := instrument.Update(ctx, e.MasterDB, rk.Address, instrumentCode, &ua,
				protocol.NewTimestamp(msg.Timestamp)); err != nil {
				return errors.Wrap(err, "Failed to clear instrument freeze period")
			}
		} else {
			hds := make(map[bitcoin.Hash20]*state.Holding)
			timestamp := protocol.NewTimestamp(msg.Timestamp)

			// Validate target addresses
			for _, quantity := range freeze.Quantities {
				if int(quantity.Index) >= len(freezeTx.Outputs) {
					return fmt.Errorf("Freeze quantity index out of range : %d/%d", quantity.Index,
						len(freezeTx.Outputs))
				}

				hash, err := itx.Outputs[quantity.Index].Address.Hash()
				if err != nil {
					address := bitcoin.NewAddressFromRawAddress(itx.Outputs[quantity.Index].Address,
						w.Config.Net)
					node.LogWarn(ctx, "Invalid freeze address : %s %s", instrumentCode, address)
					return fmt.Errorf("Invalid freeze address : %s %s", instrumentCode, address)
				}
				_, exists := hds[*hash]
				if exists {
					address := bitcoin.NewAddressFromRawAddress(itx.Outputs[quantity.Index].Address,
						w.Config.Net)
					node.LogWarn(ctx, "Address used more than once : %s %s", instrumentCode, address)
					return fmt.Errorf("Address used more than once : %s %s", instrumentCode, address)
				}

				h, err := holdings.GetHolding(ctx, e.MasterDB, rk.Address, instrumentCode,
					itx.Outputs[quantity.Index].Address, timestamp)
				if err != nil {
					return errors.Wrap(err, "Failed to get holding")
				}

				err = holdings.RevertStatus(h, freezeTx.Hash)
				if err != nil {
					address := bitcoin.NewAddressFromRawAddress(itx.Outputs[quantity.Index].Address,
						w.Config.Net)
					node.LogWarn(ctx, "Failed thaw for holding : %s %s : %s", instrumentCode, address,
						err)
					return fmt.Errorf("Failed thaw for holding : %s %s : %s", instrumentCode, address,
						err)
				}

				hds[*hash] = h
			}

			for _, h := range hds {
				cacheItem, err := holdings.Save(ctx, e.MasterDB, rk.Address, instrumentCode, h)
				if err != nil {
					return errors.Wrap(err, "Failed to save holding")
				}
				e.HoldingsChannel.Add(cacheItem)
			}
		}
	}

	node.Log(ctx, "Processed Thaw : %s", itx.Hash)
	return nil
}

// ConfiscationResponse handles an outgoing Confiscation action and writes it to the state
func (e *Enforcement) ConfiscationResponse(ctx context.Context, w *node.ResponseWriter,
	itx *inspector.Transaction, rk *wallet.Key) error {

	ctx, span := trace.StartSpan(ctx, "handlers.Enforcement.Confiscation")
	defer span.End()

	msg, ok := itx.MsgProto.(*actions.Confiscation)
	if !ok {
		return errors.New("Could not assert as *protocol.Confiscation")
	}

	// Locate Instrument
	if !itx.Inputs[0].Address.Equal(rk.Address) {
		address := bitcoin.NewAddressFromRawAddress(itx.Inputs[0].Address,
			w.Config.Net)
		return fmt.Errorf("Confiscation not from contract : %s", address.String())
	}

	ct, err := contract.Retrieve(ctx, e.MasterDB, rk.Address, e.Config.IsTest)
	if err != nil {
		return errors.Wrap(err, "Failed to retrieve contract")
	}

	if !ct.MovedTo.IsEmpty() {
		address := bitcoin.NewAddressFromRawAddress(ct.MovedTo,
			w.Config.Net)
		return fmt.Errorf("Contract address changed : %s", address.String())
	}

	// Apply confiscations
	hds := make(map[bitcoin.Hash20]*state.Holding)

	instrumentCode, err := bitcoin.NewHash20(msg.InstrumentCode)
	if err != nil {
		return errors.Wrap(err, "invalid instrument code")
	}
	timestamp := protocol.NewTimestamp(msg.Timestamp)

	highestIndex := uint32(0)
	for _, quantity := range msg.Quantities {
		hash, err := itx.Outputs[quantity.Index].Address.Hash()
		if err != nil {
			address := bitcoin.NewAddressFromRawAddress(itx.Outputs[quantity.Index].Address,
				w.Config.Net)
			node.LogWarn(ctx, "Invalid confiscation address : %s %s", instrumentCode, address)
			return fmt.Errorf("Invalid confiscation address : %s %s", instrumentCode, address)
		}
		_, exists := hds[*hash]
		if exists {
			address := bitcoin.NewAddressFromRawAddress(itx.Outputs[quantity.Index].Address,
				w.Config.Net)
			node.LogWarn(ctx, "Address used more than once : %s %s", instrumentCode, address)
			return fmt.Errorf("Address used more than once : %s %s", instrumentCode, address)
		}

		h, err := holdings.GetHolding(ctx, e.MasterDB, rk.Address, instrumentCode,
			itx.Outputs[quantity.Index].Address, timestamp)
		if err != nil {
			return errors.Wrap(err, "Failed to get holding")
		}

		err = holdings.FinalizeTx(h, &itx.Inputs[0].UTXO.Hash, quantity.Quantity, timestamp)
		if err != nil {
			address := bitcoin.NewAddressFromRawAddress(itx.Outputs[quantity.Index].Address,
				w.Config.Net)
			node.LogWarn(ctx, "Failed confiscation finalize for holding : %s %s : %s",
				instrumentCode, address, err)
			return fmt.Errorf("Failed confiscation finalize for holding : %s %s : %s",
				instrumentCode, address, err)
		}

		hds[*hash] = h

		if quantity.Index > highestIndex {
			highestIndex = quantity.Index
		}
	}

	// Update deposit balance
	h, err := holdings.GetHolding(ctx, e.MasterDB, rk.Address, instrumentCode,
		itx.Outputs[highestIndex+1].Address, timestamp)
	if err != nil {
		return errors.Wrap(err, "Failed to get deposit holding")
	}

	err = holdings.FinalizeTx(h, &itx.Inputs[0].UTXO.Hash, msg.DepositQty, timestamp)
	if err != nil {
		address := bitcoin.NewAddressFromRawAddress(itx.Outputs[highestIndex+1].Address,
			w.Config.Net)
		node.LogWarn(ctx, "Failed confiscation finalize for holding : %s %s : %s",
			instrumentCode, address, err)
		return fmt.Errorf("Failed confiscation finalize for holding : %s %s : %s",
			instrumentCode, address, err)
	}

	hash, err := itx.Outputs[highestIndex+1].Address.Hash()
	if err != nil {
		address := bitcoin.NewAddressFromRawAddress(itx.Outputs[highestIndex+1].Address,
			w.Config.Net)
		node.LogWarn(ctx, "Invalid deposit address : %s %s", instrumentCode, address)
		return fmt.Errorf("Invalid deposit address : %s %s", instrumentCode, address)
	}
	hds[*hash] = h

	for _, h := range hds {
		cacheItem, err := holdings.Save(ctx, e.MasterDB, rk.Address, instrumentCode, h)
		if err != nil {
			return errors.Wrap(err, "Failed to save holding")
		}
		e.HoldingsChannel.Add(cacheItem)
	}

	node.Log(ctx, "Processed Confiscation : %s", instrumentCode)
	return nil
}

// ReconciliationResponse handles an outgoing Reconciliation action and writes it to the state
func (e *Enforcement) ReconciliationResponse(ctx context.Context, w *node.ResponseWriter,
	itx *inspector.Transaction, rk *wallet.Key) error {

	ctx, span := trace.StartSpan(ctx, "handlers.Enforcement.Reconciliation")
	defer span.End()

	msg, ok := itx.MsgProto.(*actions.Reconciliation)
	if !ok {
		return errors.New("Could not assert as *protocol.Reconciliation")
	}

	hds := make(map[bitcoin.Hash20]*state.Holding)

	if !itx.Inputs[0].Address.Equal(rk.Address) {
		address := bitcoin.NewAddressFromRawAddress(itx.Inputs[0].Address,
			w.Config.Net)
		return fmt.Errorf("Reconciliation not from contract : %s", address.String())
	}

	ct, err := contract.Retrieve(ctx, e.MasterDB, rk.Address, e.Config.IsTest)
	if err != nil {
		return errors.Wrap(err, "Failed to retrieve contract")
	}

	if !ct.MovedTo.IsEmpty() {
		address := bitcoin.NewAddressFromRawAddress(ct.MovedTo,
			w.Config.Net)
		return fmt.Errorf("Contract address changed : %s", address.String())
	}

	// Apply reconciliations
	highestIndex := uint32(0)
	instrumentCode, err := bitcoin.NewHash20(msg.InstrumentCode)
	if err != nil {
		node.LogVerbose(ctx, "Invalid instrument code : 0x%x", msg.InstrumentCode)
		return node.RespondReject(ctx, w, itx, rk, actions.RejectionsMsgMalformed)
	}

	timestamp := protocol.NewTimestamp(msg.Timestamp)
	for _, quantity := range msg.Quantities {
		hash, err := itx.Outputs[quantity.Index].Address.Hash()
		if err != nil {
			address := bitcoin.NewAddressFromRawAddress(itx.Outputs[quantity.Index].Address,
				w.Config.Net)
			node.LogWarn(ctx, "Invalid reconciliation address : %s %s", instrumentCode, address)
			return fmt.Errorf("Invalid reconciliation address : %s %s", instrumentCode, address)
		}
		_, exists := hds[*hash]
		if exists {
			address := bitcoin.NewAddressFromRawAddress(itx.Outputs[quantity.Index].Address,
				w.Config.Net)
			node.LogWarn(ctx, "Address used more than once : %s %s", instrumentCode, address)
			return fmt.Errorf("Address used more than once : %s %s", instrumentCode, address)
		}

		h, err := holdings.GetHolding(ctx, e.MasterDB, rk.Address, instrumentCode,
			itx.Outputs[quantity.Index].Address, timestamp)
		if err != nil {
			return errors.Wrap(err, "Failed to get holding")
		}

		err = holdings.FinalizeTx(h, &itx.Inputs[0].UTXO.Hash, quantity.Quantity, timestamp)
		if err != nil {
			address := bitcoin.NewAddressFromRawAddress(itx.Outputs[quantity.Index].Address,
				w.Config.Net)
			node.LogWarn(ctx, "Failed reconciliation finalize for holding : %s %s : %s",
				instrumentCode, address, err)
			return fmt.Errorf("Failed reconciliation finalize for holding : %s %s : %s",
				instrumentCode, address, err)
		}

		hds[*hash] = h

		if quantity.Index > highestIndex {
			highestIndex = quantity.Index
		}
	}

	for _, h := range hds {
		cacheItem, err := holdings.Save(ctx, e.MasterDB, rk.Address, instrumentCode, h)
		if err != nil {
			return errors.Wrap(err, "Failed to save holding")
		}
		e.HoldingsChannel.Add(cacheItem)
	}

	node.Log(ctx, "Processed Confiscation : %s", instrumentCode)
	return nil
}
