package txbuilder

import (
	"bytes"

	"github.com/tokenized/smart-contract/pkg/txscript"
	"github.com/tokenized/smart-contract/pkg/wire"

	"github.com/btcsuite/btcd/btcec"
)

func sign(privateKey *btcec.PrivateKey,
	raw []byte,
	utxos []*TxOutput) ([]byte, error) {

	// deserialize the TX
	tx := &wire.MsgTx{}
	buf := bytes.NewReader(raw)

	if err := tx.Deserialize(buf); err != nil {
		return nil, err
	}

	// loop over the utxo's, and sign the tx inputs
	for i, utxo := range utxos {
		signature, err := txscript.SignatureScript(
			tx,
			i,
			utxo.PkScript,
			txscript.SigHashAll+SigHashForkID,
			privateKey,
			true,
			int64(utxo.Value))

		if err != nil {
			return nil, err
		}

		tx.TxIn[i].SignatureScript = signature
	}

	// serialize the TX
	w := bytes.Buffer{}
	if err := tx.Serialize(&w); err != nil {
		return nil, err
	}

	return w.Bytes(), nil
}
