package request

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcutil"
	"github.com/tokenized/smart-contract/internal/platform/config"
	"github.com/tokenized/smart-contract/internal/platform/state/contract"
	"github.com/tokenized/smart-contract/pkg/protocol"
	"github.com/tokenized/smart-contract/pkg/txbuilder"
)

type orderHandler struct {
	Fee config.Fee
}

func newOrderHandler(fee config.Fee) orderHandler {
	return orderHandler{
		Fee: fee,
	}
}

func (h orderHandler) handle(ctx context.Context,
	r contractRequest) (*contractResponse, error) {

	order, ok := r.m.(*protocol.Order)
	if !ok {
		return nil, errors.New("Not *protocol.Order")
	}

	// Contract
	c := r.contract

	// Asset check
	assetKey := string(order.AssetID)
	asset, ok := c.Assets[assetKey]
	if !ok {
		return nil, fmt.Errorf("order : Asset ID not found : contract=%s assetID=%s", c.ID, order.AssetID)
	}

	// Holdings check
	targetAddr := string(order.TargetAddress)
	_, ok = asset.Holdings[targetAddr]
	if !ok {
		return nil, fmt.Errorf("order : Holding not found contract=%s assetID=%s target=%s", c.ID, assetKey, targetAddr)
	}

	// Apply logic based on Compliance Action type
	var err error
	var resp *contractResponse

	switch order.ComplianceAction {
	case protocol.ComplianceActionFreeze:
		resp, err = h.freeze(c, order)
	case protocol.ComplianceActionThaw:
		resp, err = h.thaw(c, order)
	case protocol.ComplianceActionConfiscation:
		resp, err = h.confiscate(c, order)
	default:
		return nil, fmt.Errorf("Unknown enforcement : %v", order.ComplianceAction)
	}

	return resp, err
}

// freeze sets the state of a holding to frozen.
func (h orderHandler) freeze(c contract.Contract,
	order *protocol.Order) (*contractResponse, error) {

	// Freeze <- Order
	freeze := protocol.NewFreeze()
	freeze.AssetID = order.AssetID
	freeze.AssetType = order.AssetType
	freeze.Timestamp = uint64(time.Now().Unix())
	freeze.Qty = order.Qty
	freeze.Message = order.Message
	freeze.Expiration = order.Expiration

	contractAddr, err := c.Address()
	if err != nil {
		return nil, err
	}

	// Outputs
	outputs, err := h.buildFreezeThawOutputs(c, order)
	if err != nil {
		return nil, err
	}

	cr := contractResponse{
		Contract:      c,
		Message:       &freeze,
		outs:          outputs,
		changeAddress: contractAddr,
	}

	return &cr, nil
}

// thaw reverses the freeze operation on a holding.
func (h orderHandler) thaw(c contract.Contract,
	order *protocol.Order) (*contractResponse, error) {

	// Thaw <- Order
	thaw := protocol.NewThaw()
	thaw.AssetID = order.AssetID
	thaw.AssetType = order.AssetType
	thaw.Timestamp = uint64(time.Now().Unix())
	thaw.Qty = order.Qty
	thaw.Message = order.Message

	contractAddr, err := c.Address()
	if err != nil {
		return nil, err
	}

	// Outputs
	outputs, err := h.buildFreezeThawOutputs(c, order)
	if err != nil {
		return nil, err
	}

	cr := contractResponse{
		Contract:      c,
		Message:       &thaw,
		outs:          outputs,
		changeAddress: contractAddr,
	}

	return &cr, nil
}

// confiscate performs a confiscation of assets.
func (h orderHandler) confiscate(c contract.Contract,
	order *protocol.Order) (*contractResponse, error) {

	// Asset
	assetKey := string(order.AssetID)
	asset := c.Assets[assetKey]

	// Target Holding
	targetAddr := string(order.TargetAddress)
	targetHolding := asset.Holdings[targetAddr]
	targetBalance := targetHolding.Balance

	// Depositor Holding
	depositKey := string(order.DepositAddress)
	depositHolding, ok := asset.Holdings[depositKey]
	depositBalance := uint64(0)
	if ok {
		depositBalance = depositHolding.Balance
	}

	// Transfer the qty from the target to the deposit
	qty := order.Qty

	// Trying to take more than is held by the target, limit
	// to the amount they are holding.
	if targetBalance < qty {
		qty = targetBalance
	}

	// Modify balances
	targetBalance -= qty
	depositBalance += qty

	// Confiscation <- Order
	confiscation := protocol.NewConfiscation()
	confiscation.AssetID = order.AssetID
	confiscation.AssetType = order.AssetType
	confiscation.Timestamp = uint64(time.Now().Unix())
	confiscation.Message = order.Message
	confiscation.TargetsQty = targetBalance
	confiscation.DepositsQty = depositBalance

	// Outputs
	outputs, err := h.buildConfiscateOutputs(c, order)
	if err != nil {
		return nil, err
	}

	contractAddr, err := c.Address()
	if err != nil {
		return nil, err
	}

	cr := contractResponse{
		Contract:      c,
		Message:       &confiscation,
		outs:          outputs,
		changeAddress: contractAddr,
	}

	return &cr, nil
}

func (h orderHandler) buildFreezeThawOutputs(contract contract.Contract,
	order *protocol.Order) ([]txbuilder.TxOutput, error) {

	contractAddr, err := contract.Address()
	if err != nil {
		return nil, err
	}

	targetAddr, err := btcutil.DecodeAddress(string(order.TargetAddress),
		&chaincfg.MainNetParams)
	if err != nil {
		return nil, err
	}

	// Alleged Target's Public Address
	// Contract's Public Address
	// Contract Fee Address
	outs := []txbuilder.TxOutput{
		txbuilder.TxOutput{
			Address: targetAddr,
			Value:   546,
		},
		txbuilder.TxOutput{
			Address: contractAddr,
			Value:   546, // address will receive change, if any
		},
	}

	// optional contract fee
	if h.Fee.Value > 0 {
		o := txbuilder.TxOutput{
			Address: h.Fee.Address,
			Value:   h.Fee.Value,
		}

		outs = append(outs, o)
	}

	return outs, nil
}

func (h orderHandler) buildConfiscateOutputs(contract contract.Contract,
	order *protocol.Order) ([]txbuilder.TxOutput, error) {

	// we need a txout to the target
	targetAddr, err := btcutil.DecodeAddress(string(order.TargetAddress),
		&chaincfg.MainNetParams)
	if err != nil {
		return nil, err
	}

	depositAddr, err := btcutil.DecodeAddress(string(order.DepositAddress),
		&chaincfg.MainNetParams)
	if err != nil {
		return nil, err
	}

	contractAddr, err := contract.Address()
	if err != nil {
		return nil, err
	}

	outs := []txbuilder.TxOutput{
		txbuilder.TxOutput{
			Address: targetAddr,
			Value:   546,
		},
		txbuilder.TxOutput{
			Address: depositAddr,
			Value:   546,
		},
		txbuilder.TxOutput{
			Address: contractAddr,
			Value:   546, // address will receive change, if any
		},
	}

	// optional contract fee
	if h.Fee.Value > 0 {
		o := txbuilder.TxOutput{
			Address: h.Fee.Address,
			Value:   h.Fee.Value,
		}

		outs = append(outs, o)
	}

	return outs, nil
}
