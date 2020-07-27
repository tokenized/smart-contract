package contract

import (
	"context"
	"testing"

	"github.com/tokenized/pkg/bitcoin"
	"github.com/tokenized/smart-contract/internal/platform/tests"
	"github.com/tokenized/specification/dist/golang/actions"
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

	cf := &actions.ContractFormation{
		ContractName: "Test Contract",
		ContractType: 1,
	}

	if err := SaveContractFormation(ctx, dbConn, ra, cf, true); err != nil {
		t.Fatalf("Failed to save contract formation : %s", err)
	}

	fetched, err := FetchContractFormation(ctx, dbConn, ra, true)
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
