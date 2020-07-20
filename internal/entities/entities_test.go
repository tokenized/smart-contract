package entities

import (
	"context"
	"testing"

	"github.com/tokenized/pkg/bitcoin"
	"github.com/tokenized/pkg/wire"
	"github.com/tokenized/smart-contract/internal/platform/tests"
	"github.com/tokenized/smart-contract/pkg/inspector"
	"github.com/tokenized/specification/dist/golang/actions"
	"github.com/tokenized/specification/dist/golang/protocol"
)

func TestSaveFetch(t *testing.T) {
	ctx := context.Background()
	dbConn := tests.NewMasterDB(t)

	key, err := bitcoin.GenerateKey(bitcoin.MainNet)
	if err != nil {
		t.Fatalf("Failed to generate key : %s", err)
	}

	ra, err := key.RawAddress()
	if err != nil {
		t.Fatalf("Failed to create address : %s", err)
	}

	lockingScript, err := ra.LockingScript()
	if err != nil {
		t.Fatalf("Failed to locking script : %s", err)
	}

	cf := &actions.ContractFormation{
		ContractName: "Test Contract",
		ContractType: 1,
	}

	opReturn, err := protocol.Serialize(cf, true)
	if err != nil {
		t.Fatalf("Failed to serialize contract formation : %s", err)
	}

	itx := &inspector.Transaction{
		MsgTx:         wire.NewMsgTx(1),
		MsgProto:      cf,
		MsgProtoIndex: 1,
		Inputs: []inspector.Input{
			inspector.Input{
				Address: ra,
			},
		},
	}

	itx.MsgTx.AddTxOut(wire.NewTxOut(2000, lockingScript))

	itx.MsgTx.AddTxOut(wire.NewTxOut(0, opReturn))

	itx.Hash = itx.MsgTx.TxHash()

	if err := SaveContractFormation(ctx, dbConn, itx, true); err != nil {
		t.Fatalf("Failed to save contract formation : %s", err)
	}

	fetched, err := FetchEntity(ctx, dbConn, ra, true)
	if err != nil {
		t.Fatalf("Failed to fetch contract formation : %s", err)
	}

	if fetched.ContractName != cf.ContractName {
		t.Errorf("Wrong contract name : got \"%s\", wanted \"%s\"", fetched.ContractName,
			cf.ContractName)
	}

	if fetched.ContractType != cf.ContractType {
		t.Errorf("Wrong contract type : got %d, wanted %d", fetched.ContractType,
			cf.ContractType)
	}
}
