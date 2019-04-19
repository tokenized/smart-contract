package tests

import (
	"testing"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/tokenized/smart-contract/internal/asset"
	"github.com/tokenized/smart-contract/internal/platform/tests"
	"github.com/tokenized/smart-contract/pkg/inspector"
	"github.com/tokenized/smart-contract/pkg/txbuilder"
	"github.com/tokenized/smart-contract/pkg/wire"
	"github.com/tokenized/specification/dist/golang/protocol"
)

// TestTransfers is the entry point for testing transfer functions.
func TestTransfers(t *testing.T) {
	defer tests.Recover(t)

	t.Run("transferTokens", transferTokens)
}

func transferTokens(t *testing.T) {
	ctx := test.Context

	fundingTx := wire.NewMsgTx(2)
	fundingTx.TxOut = append(fundingTx.TxOut, wire.NewTxOut(100002, txbuilder.P2PKHScriptForPKH(issuerKey.Address.ScriptAddress())))
	test.RPCNode.AddTX(ctx, fundingTx)

	// Create Transfer message
	transferAmount := uint64(250)
	transferData := protocol.Transfer{}

	assetTransferData := protocol.AssetTransfer{
		ContractIndex: 0, // first output
		AssetType:     protocol.CodeShareCommon,
		AssetCode:     testAssetCode,
	}

	assetTransferData.AssetSenders = append(assetTransferData.AssetSenders, protocol.QuantityIndex{Index: 0, Quantity: transferAmount})
	assetTransferData.AssetReceivers = append(assetTransferData.AssetReceivers, protocol.TokenReceiver{Index: 1, Quantity: transferAmount})

	transferData.Assets = append(transferData.Assets, assetTransferData)

	// Build transfer transaction
	transferTx := wire.NewMsgTx(2)

	var transferInputHash chainhash.Hash
	transferInputHash = fundingTx.TxHash()

	// From issuer
	transferTx.TxIn = append(transferTx.TxIn, wire.NewTxIn(wire.NewOutPoint(&transferInputHash, 0), make([]byte, 130)))

	// To contract
	transferTx.TxOut = append(transferTx.TxOut, wire.NewTxOut(256, txbuilder.P2PKHScriptForPKH(test.ContractKey.Address.ScriptAddress())))

	// To user
	transferTx.TxOut = append(transferTx.TxOut, wire.NewTxOut(256, txbuilder.P2PKHScriptForPKH(userKey.Address.ScriptAddress())))

	// Data output
	script, err := protocol.Serialize(&transferData)
	if err != nil {
		t.Fatalf("\t%s\tFailed to serialize transfer : %v", tests.Failed, err)
	}
	transferTx.TxOut = append(transferTx.TxOut, wire.NewTxOut(0, script))

	transferItx, err := inspector.NewTransactionFromWire(ctx, transferTx)
	if err != nil {
		t.Fatalf("\t%s\tFailed to create transfer itx : %v", tests.Failed, err)
	}

	err = transferItx.Promote(ctx, test.RPCNode)
	if err != nil {
		t.Fatalf("\t%s\tFailed to promote transfer itx : %v", tests.Failed, err)
	}

	test.RPCNode.AddTX(ctx, transferTx)

	err = a.Trigger(ctx, "SEE", transferItx)
	if err == nil {
		t.Fatalf("\t%s\tAccepted transfer with insufficient value", tests.Failed)
	}

	if len(responses) != 0 {
		t.Fatalf("\t%s\tHandle asset transfer created reject response", tests.Failed)
	}

	t.Logf("Underfunded asset transfer rejected with no response")

	// Adjust amount to contract to be appropriate
	transferTx.TxOut[0].Value = 1000

	transferItx, err = inspector.NewTransactionFromWire(ctx, transferTx)
	if err != nil {
		t.Fatalf("\t%s\tFailed to create transfer itx : %v", tests.Failed, err)
	}

	err = transferItx.Promote(ctx, test.RPCNode)
	if err != nil {
		t.Fatalf("\t%s\tFailed to promote transfer itx : %v", tests.Failed, err)
	}

	// Resubmit
	test.RPCNode.AddTX(ctx, transferTx)

	err = a.Trigger(ctx, "SEE", transferItx)
	if err != nil {
		t.Fatalf("\t%s\tFailed to accept transfer : %v", tests.Failed, err)
	}

	t.Logf("Transfer accepted")

	// Check the response
	checkResponse(t, "T2")

	// Check issuer and user balance
	contractPKH := protocol.PublicKeyHashFromBytes(test.ContractKey.Address.ScriptAddress())
	as, err := asset.Retrieve(ctx, test.MasterDB, contractPKH, &testAssetCode)
	if err != nil {
		t.Fatalf("\t%s\tFailed to retrieve asset : %v", tests.Failed, err)
	}

	issuerPKH := protocol.PublicKeyHashFromBytes(issuerKey.Address.ScriptAddress())
	issuerBalance := asset.GetBalance(ctx, as, issuerPKH)
	if issuerBalance != testTokenQty-transferAmount {
		t.Fatalf("\t%s\tIssuer token balance incorrect : %d != %d", tests.Failed, issuerBalance, testTokenQty-transferAmount)
	}

	userPKH := protocol.PublicKeyHashFromBytes(userKey.Address.ScriptAddress())
	userBalance := asset.GetBalance(ctx, as, userPKH)
	if userBalance != transferAmount {
		t.Fatalf("\t%s\tUser token balance incorrect : %d != %d", tests.Failed, userBalance, transferAmount)
	}

	t.Logf("Issuer asset balance : %d", issuerBalance)
	t.Logf("User asset balance : %d", userBalance)
}
