package txbuilder

import (
	"fmt"

	"github.com/btcsuite/btcutil"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
)

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

func getHashString(txHash []byte, index uint32) string {
	hash, err := chainhash.NewHash(txHash)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("out:%s:%d", hash.String(), index)
}

func (t TxOutput) GetHashString() string {
	return getHashString(t.TransactionHash, t.Index)
}
