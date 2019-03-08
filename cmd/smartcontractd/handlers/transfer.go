package handlers

import (
	"context"
	"errors"
	"log"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcutil"
	"github.com/tokenized/smart-contract/internal/asset"
	"github.com/tokenized/smart-contract/internal/platform/db"
	"github.com/tokenized/smart-contract/internal/platform/node"
	"github.com/tokenized/smart-contract/internal/platform/protomux"
	"github.com/tokenized/smart-contract/internal/platform/wallet"
	"github.com/tokenized/smart-contract/pkg/inspector"
	"github.com/tokenized/smart-contract/pkg/logger"
	"github.com/tokenized/smart-contract/pkg/protocol"
	"github.com/tokenized/smart-contract/pkg/txbuilder"
	"go.opencensus.io/trace"
)

type Transfer struct {
	MasterDB *db.DB
	Config   *node.Config
}

// SendRequest handles an incoming Send request and prepares a Settlement response
func (t *Transfer) SendRequest(ctx context.Context, mux protomux.Handler, itx *inspector.Transaction, rk *wallet.RootKey) error {
	ctx, span := trace.StartSpan(ctx, "handlers.Transfer.Send")
	defer span.End()

	msg, ok := itx.MsgProto.(*protocol.Send)
	if !ok {
		return errors.New("Could not assert as *protocol.Send")
	}

	dbConn := t.MasterDB

	v := ctx.Value(node.KeyValues).(*node.Values)

	// Locate Asset
	contractAddr := rk.Address
	assetID := string(msg.AssetID)
	as, err := asset.Retrieve(ctx, dbConn, contractAddr.String(), assetID)
	if err != nil {
		logger.Warn(ctx, "%s : Failed to retrieve asset : %s\n", v.TraceID, assetID)
		return err
	}

	// Asset could not be found
	if as == nil {
		logger.Warn(ctx, "%s : Asset not found : %s\n", v.TraceID, assetID)
		return node.RespondReject(ctx, mux, itx, rk, protocol.RejectionCodeAssetNotFound)
	}

	// Validate transaction
	if len(itx.Outputs) < 2 {
		logger.Warn(ctx, "%s : Not enough outputs : %s\n", v.TraceID, assetID)
		return node.RespondReject(ctx, mux, itx, rk, protocol.RejectionCodeReceiverUnspecified)
	}

	// Party 1 (Sender), Party 2 (Receiver)
	party1Addr := itx.Inputs[0].Address
	party2Addr := itx.Outputs[1].Address

	// Cannot transfer to self
	if party1Addr.String() == party2Addr.String() {
		logger.Warn(ctx, "%s : Cannot transfer to self : party=%s asset=%s\n", v.TraceID, party1Addr, assetID)
		return node.RespondReject(ctx, mux, itx, rk, protocol.RejectionCodeTransferSelf)
	}

	// Check available balance
	if !asset.CheckBalance(ctx, as, party1Addr.String(), msg.TokenQty) {
		logger.Warn(ctx, "%s : Insufficient funds %d : party=%s asset=%s\n", v.TraceID, msg.TokenQty, party1Addr, assetID)
		return node.RespondReject(ctx, mux, itx, rk, protocol.RejectionCodeInsufficientAssets)
	}

	// Check available unfrozen balance
	if !asset.CheckBalanceFrozen(ctx, as, party1Addr.String(), msg.TokenQty, v.Now) {
		logger.Warn(ctx, "%s : Frozen funds %d : party=%s asset=%s\n", v.TraceID, msg.TokenQty, party1Addr, assetID)
		return node.RespondReject(ctx, mux, itx, rk, protocol.RejectionCodeFrozen)
	}

	logger.Info(ctx, "%s : Sending %d : %s %s\n", v.TraceID, msg.TokenQty, contractAddr.String(), string(msg.AssetID))

	// Find balances
	party1Balance := asset.GetBalance(ctx, as, party1Addr.String())
	party2Balance := asset.GetBalance(ctx, as, party2Addr.String())

	// Modify balances
	party1Balance -= msg.TokenQty
	party2Balance += msg.TokenQty

	// Settlement <- Send
	settlement := protocol.NewSettlement()
	settlement.AssetType = msg.AssetType
	settlement.AssetID = msg.AssetID
	settlement.Party1TokenQty = party1Balance
	settlement.Party2TokenQty = party2Balance
	settlement.Timestamp = uint64(v.Now.Unix())

	// Build outputs
	// 1 - Party 1 Address (Change)
	// 2 - Party 2 Address
	// 3 - Contract Address
	// 4 - Fee
	outs := []node.Output{{
		Address: party1Addr,
		Value:   t.Config.DustLimit,
		Change:  true,
	}, {
		Address: party2Addr,
		Value:   t.Config.DustLimit,
	}, {
		Address: contractAddr,
		Value:   t.Config.DustLimit,
	}}

	// Add fee output
	if fee := node.OutputFee(ctx, t.Config); fee != nil {
		outs = append(outs, *fee)
	}

	// Respond with a settlement
	return node.RespondSuccess(ctx, mux, itx, rk, &settlement, outs)
}

// ExchangeRequest handles an incoming Exchange request and prepares a Settlement response
func (t *Transfer) ExchangeRequest(ctx context.Context, mux protomux.Handler, itx *inspector.Transaction, rk *wallet.RootKey) error {
	ctx, span := trace.StartSpan(ctx, "handlers.Transfer.Exchange")
	defer span.End()

	msg, ok := itx.MsgProto.(*protocol.Exchange)
	if !ok {
		return errors.New("Could not assert as *protocol.Exchange")
	}

	dbConn := t.MasterDB

	v := ctx.Value(node.KeyValues).(*node.Values)

	// Locate Asset
	contractAddr := rk.Address
	assetID := string(msg.Party1AssetID)
	as, err := asset.Retrieve(ctx, dbConn, contractAddr.String(), assetID)
	if err != nil {
		return err
	}

	// Asset could not be found
	if as == nil {
		log.Printf("%s : Asset ID not found: %+v %+v\n", v.TraceID, contractAddr, assetID)
		return node.RespondReject(ctx, mux, itx, rk, protocol.RejectionCodeAssetNotFound)
	}

	// Validate transaction
	if len(itx.Inputs) < 2 {
		log.Printf("%s : Not enough inputs: %+v %+v\n", v.TraceID, contractAddr, assetID)
		return node.RespondReject(ctx, mux, itx, rk, protocol.RejectionCodeReceiverUnspecified)
	}

	if len(itx.Outputs) < 3 {
		log.Printf("%s : Not enough outputs: %+v %+v\n", v.TraceID, contractAddr, assetID)
		return node.RespondReject(ctx, mux, itx, rk, protocol.RejectionCodeReceiverUnspecified)
	}

	// Party 1 (Sender), Party 2 (Receiver)
	party1Addr := itx.Inputs[0].Address
	party2Addr := itx.Inputs[1].Address

	// Cannot transfer to self
	if party1Addr.String() == party2Addr.String() {
		log.Printf("%s : Cannot transfer to own self : contract=%+v asset=%+v party=%+v\n", v.TraceID, contractAddr, assetID, party1Addr)
		return node.RespondReject(ctx, mux, itx, rk, protocol.RejectionCodeTransferSelf)
	}

	// Check available balance
	if !asset.CheckBalance(ctx, as, party1Addr.String(), msg.Party1TokenQty) {
		log.Printf("%s : Insufficient funds: contract=%+v asset=%+v party=%+v\n", v.TraceID, contractAddr, assetID, party1Addr)
		return node.RespondReject(ctx, mux, itx, rk, protocol.RejectionCodeInsufficientAssets)
	}

	// Check available unfrozen balance
	if !asset.CheckBalanceFrozen(ctx, as, party1Addr.String(), msg.Party1TokenQty, v.Now) {
		log.Printf("%s : Frozen funds: contract=%+v asset=%+v party=%+v\n", v.TraceID, contractAddr, assetID, party1Addr)
		return node.RespondReject(ctx, mux, itx, rk, protocol.RejectionCodeFrozen)
	}

	// Find balances
	party1Balance := asset.GetBalance(ctx, as, party1Addr.String())
	party2Balance := asset.GetBalance(ctx, as, party2Addr.String())

	// Modify balances
	party1Balance -= msg.Party1TokenQty
	party2Balance += msg.Party1TokenQty

	// Settlement <- Exchange
	settlement := protocol.NewSettlement()
	settlement.AssetType = msg.Party1AssetType
	settlement.AssetID = msg.Party1AssetID
	settlement.Party1TokenQty = party1Balance
	settlement.Party2TokenQty = party2Balance
	settlement.Timestamp = uint64(v.Now.Unix())

	// Build outputs
	// 1 - Party 1 Address (Change + Value)
	// 2 - Party 2 Address
	// 3 - Contract Address
	// 4 - Fee
	outs := []node.Output{{
		Address: party1Addr,
		Value:   t.Config.DustLimit,
		Change:  true,
	}, {
		Address: party2Addr,
		Value:   t.Config.DustLimit,
	}, {
		Address: contractAddr,
		Value:   t.Config.DustLimit,
	}}

	// Add fee output
	if fee := node.OutputFee(ctx, t.Config); fee != nil {
		outs = append(outs, *fee)
	}

	// Optional exchange fee.
	if msg.ExchangeFeeFixed > 0 {
		exAddr := string(msg.ExchangeFeeAddress)
		addr, err := btcutil.DecodeAddress(exAddr, &chaincfg.MainNetParams)
		if err != nil {
			return err
		}

		// Convert BCH to Satoshi's
		exo := node.Output{
			Address: addr,
			Value:   txbuilder.ConvertBCHToSatoshis(msg.ExchangeFeeFixed),
		}

		outs = append(outs, exo)
	}

	// Respond with a settlement
	return node.RespondSuccess(ctx, mux, itx, rk, &settlement, outs)
}

// SwapRequest handles an incoming Swap request and prepares a Settlement response
func (t *Transfer) SwapRequest(ctx context.Context, mux protomux.Handler, itx *inspector.Transaction, rk *wallet.RootKey) error {
	ctx, span := trace.StartSpan(ctx, "handlers.Transfer.Swap")
	defer span.End()

	// TODO(srg) - This feature is incomplete

	return nil
}

// SettlementResponse handles an outgoing Settlement action and writes it to the state
func (t *Transfer) SettlementResponse(ctx context.Context, mux protomux.Handler, itx *inspector.Transaction, rk *wallet.RootKey) error {
	ctx, span := trace.StartSpan(ctx, "handlers.Transfer.Settlement")
	defer span.End()

	msg, ok := itx.MsgProto.(*protocol.Settlement)
	if !ok {
		return errors.New("Could not assert as *protocol.Settlement")
	}

	dbConn := t.MasterDB

	v := ctx.Value(node.KeyValues).(*node.Values)

	// Locate Asset
	contractAddr := rk.Address
	assetID := string(msg.AssetID)
	as, err := asset.Retrieve(ctx, dbConn, contractAddr.String(), assetID)
	if err != nil {
		return err
	}

	// Asset could not be found
	if as == nil {
		log.Printf("%s : Asset ID not found: %+v %+v\n", v.TraceID, contractAddr, assetID)
		return node.ErrNoResponse
	}

	// Validate transaction
	if len(itx.Outputs) < 2 {
		log.Printf("%s : Not enough outputs: %+v %+v\n", v.TraceID, contractAddr, assetID)
		return node.ErrNoResponse
	}

	// Party 1 (Sender), Party 2 (Receiver)
	party1PKH := itx.Outputs[0].Address.String()
	party2PKH := itx.Outputs[1].Address.String()

	logger.Info(ctx, "%s : Settling transfer : %s %s\n", v.TraceID, contractAddr.String(), string(msg.AssetID))
	logger.Info(ctx, "%s : Party 1 %s : %d tokens\n", v.TraceID, party1PKH, msg.Party1TokenQty)
	logger.Info(ctx, "%s : Party 2 %s : %d tokens\n", v.TraceID, party2PKH, msg.Party2TokenQty)

	newBalances := map[string]uint64{
		party1PKH: msg.Party1TokenQty,
		party2PKH: msg.Party2TokenQty,
	}

	// Update asset
	ua := asset.UpdateAsset{
		NewBalances: newBalances,
	}

	if err := asset.Update(ctx, dbConn, contractAddr.String(), assetID, &ua, v.Now); err != nil {
		return err
	}

	return nil
}
