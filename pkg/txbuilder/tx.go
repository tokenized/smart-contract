package txbuilder

import (
    "fmt"

    "github.com/btcsuite/btcd/chaincfg/chainhash"
    "github.com/btcsuite/btcutil"
    "github.com/tokenized/smart-contract/pkg/wire"
)

const (
    OutputTypeP2PK TxOutputType = iota
    OutputTypeReturn
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

type TxInput struct {
    PkHash    []byte
    PrevIndex int64
    PrevHash  string
    Value     uint64
}

type TxOutput struct {
    Type            TxOutputType
    PkScript        []byte
    PkHash          []byte
    Address         btcutil.Address
    Index           uint32
    Value           uint64
    TransactionHash []byte
    Data            []byte
}

func (t TxOutput) GetHashString() string {
    hash, err := chainhash.NewHash(t.TransactionHash)
    if err != nil {
        return ""
    }
    return fmt.Sprintf("out:%s:%d", hash.String(), t.Index)
}

type TxOutputType uint

func (s TxOutputType) String() string {
    switch s {
    case OutputTypeP2PK:
        return StringP2pk
    case OutputTypeReturn:
        return StringReturn
    default:
        return "unknown"
    }
}
