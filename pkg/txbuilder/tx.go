package txbuilder

import (
	"github.com/tokenized/smart-contract/pkg/wire"
)

type Tx struct {
	Type       TxOutputType
	SelfPkHash []byte
	MsgTx      *wire.MsgTx
	Inputs     []*TxInput
}

type TxOutSortByValue []*TxOutput

func (txOuts TxOutSortByValue) Len() int      { return len(txOuts) }
func (txOuts TxOutSortByValue) Swap(i, j int) { txOuts[i], txOuts[j] = txOuts[j], txOuts[i] }
func (txOuts TxOutSortByValue) Less(i, j int) bool {
	return txOuts[i].Value > txOuts[j].Value
}
