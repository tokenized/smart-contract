package state

import (
	"context"

	"github.com/tokenized/smart-contract/internal/app/state/contract"
)

type StateInterface interface {
	Write(context.Context, contract.Contract) error
	Read(context.Context, string) (*contract.Contract, error)
	WriteHard(context.Context, contract.Contract) error
	ReadHard(context.Context, string) (*contract.Contract, error)
}
