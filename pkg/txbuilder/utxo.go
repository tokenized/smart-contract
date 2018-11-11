package txbuilder

import (
	"fmt"

	"github.com/tokenized/smart-contract/pkg/wire"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcutil"
)

// UTXO holds all required UTXO details.
type UTXO struct {
	Hash     chainhash.Hash
	PkScript []byte
	Index    uint32
	Value    uint64
}

// NewUTXO returns a new UTXO.
func NewUTXO(hash chainhash.Hash, n uint32, pkScript []byte, value uint64) UTXO {
	return UTXO{
		Hash:     hash,
		Index:    n,
		PkScript: pkScript,
		Value:    value,
	}
}

// NewUTXOFromTX returns a new UTXO with details from the TX.
func NewUTXOFromTX(tx wire.MsgTx, n uint32) UTXO {
	return UTXO{
		Hash:     tx.TxHash(),
		Index:    n,
		PkScript: tx.TxOut[n].PkScript,
		Value:    uint64(tx.TxOut[n].Value),
	}
}

// PublicAddress returns the public address from a P2PKH script.
//
// The script will look something like this
//
// OP_DUP OP_HASH160 <address hash> OP_EQUALVERIFY OP_CHECKSIG
func (u UTXO) PublicAddress(params *chaincfg.Params) (btcutil.Address, error) {
	if len(u.PkScript) != 25 &&
		u.PkScript[0] != txscript.OP_DUP &&
		u.PkScript[1] != txscript.OP_HASH160 {

		return nil, fmt.Errorf("Invalid pkScript %s : %v", u.Hash, u.PkScript)
	}

	return btcutil.NewAddressPubKeyHash(u.PkScript[3:23], params)
}
