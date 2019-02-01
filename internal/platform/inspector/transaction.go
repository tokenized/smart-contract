package inspector

import (
	"github.com/tokenized/smart-contract/pkg/protocol"
	"github.com/tokenized/smart-contract/pkg/txbuilder"
	"github.com/tokenized/smart-contract/pkg/wire"

	"github.com/btcsuite/btcutil"
)

type Transaction struct {
	Hash       []byte
	InputAddrs []btcutil.Address
	Inputs     txbuilder.UTXOs
	UTXOs      txbuilder.UTXOs
	Outputs    []txbuilder.TxOutput
	MsgTx      *wire.MsgTx
	MsgProto   protocol.OpReturnMessage
}
