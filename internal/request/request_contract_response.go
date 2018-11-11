package request

import (
	"github.com/btcsuite/btcutil"
	"github.com/tokenized/smart-contract/internal/app/state/contract"
	"github.com/tokenized/smart-contract/pkg/protocol"
	"github.com/tokenized/smart-contract/pkg/txbuilder"
)

type contractResponse struct {
	Contract      contract.Contract
	Message       protocol.OpReturnMessage
	outs          []txbuilder.TxOutput
	Responses     []contractResponse
	changeAddress btcutil.Address
}
