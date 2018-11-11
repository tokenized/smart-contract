package txbuilder

import (
	"github.com/tokenized/smart-contract/pkg/txscript"
	"github.com/tokenized/smart-contract/pkg/wire"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
)

func Create(spendOuts []*TxOutput,
	privateKey *PrivateKey,
	spendOutputs []TxOutput) (*wire.MsgTx, error) {

	var txOuts []*wire.TxOut
	for _, spendOutput := range spendOutputs {
		switch spendOutput.Type {
		case OutputTypeP2PK:
			pkScript, err := txscript.NewScriptBuilder().
				AddOp(txscript.OP_DUP).
				AddOp(txscript.OP_HASH160).
				AddData(spendOutput.Address.ScriptAddress()).
				AddOp(txscript.OP_EQUALVERIFY).
				AddOp(txscript.OP_CHECKSIG).
				Script()
			if err != nil {
				return nil, err
			}
			txOuts = append(txOuts, wire.NewTxOut(int64(spendOutput.Value), pkScript))
		case OutputTypeReturn:
			pkScript, err := txscript.NewScriptBuilder().
				AddOp(txscript.OP_RETURN).
				AddData(spendOutput.Data).
				Script()
			if err != nil {
				return nil, err
			}
			txOuts = append(txOuts, wire.NewTxOut(0, pkScript))
		}
	}

	var txIns []*wire.TxIn
	var totalValue uint64
	for _, spendOut := range spendOuts {
		hash, err := chainhash.NewHash(spendOut.TransactionHash)
		if err != nil {
			return nil, err
		}
		newTxIn := wire.NewTxIn(&wire.OutPoint{
			Hash:  *hash,
			Index: uint32(spendOut.Index),
		}, nil)
		txIns = append(txIns, newTxIn)
		totalValue += spendOut.Value
	}

	var tx = &wire.MsgTx{
		Version:  wire.TxVersion,
		TxIn:     txIns,
		TxOut:    txOuts,
		LockTime: 0,
	}

	for i := 0; i < len(spendOuts); i++ {
		signature, err := txscript.SignatureScript(
			tx,
			i,
			spendOuts[i].PkScript,
			txscript.SigHashAll+SigHashForkID,
			privateKey.GetBtcEcPrivateKey(),
			true,
			int64(spendOuts[i].Value),
		)
		if err != nil {
			return nil, err
		}
		txIns[i].SignatureScript = signature
	}
	return tx, nil
}

func CreateUnsigned(spendOuts []*TxOutput,
	spendOutputs []TxOutput) (*wire.MsgTx, error) {

	var txOuts []*wire.TxOut
	for _, spendOutput := range spendOutputs {
		switch spendOutput.Type {
		case OutputTypeP2PK:
			pkScript, err := txscript.NewScriptBuilder().
				AddOp(txscript.OP_DUP).
				AddOp(txscript.OP_HASH160).
				AddData(spendOutput.Address.ScriptAddress()).
				AddOp(txscript.OP_EQUALVERIFY).
				AddOp(txscript.OP_CHECKSIG).
				Script()
			if err != nil {
				return nil, err
			}
			txOuts = append(txOuts, wire.NewTxOut(int64(spendOutput.Value), pkScript))
		case OutputTypeReturn:
			pkScript, err := txscript.NewScriptBuilder().
				AddOp(txscript.OP_RETURN).
				AddData(spendOutput.Data).
				Script()
			if err != nil {
				return nil, err
			}
			txOuts = append(txOuts, wire.NewTxOut(0, pkScript))

		}
	}

	var txIns []*wire.TxIn
	var totalValue uint64
	for _, spendOut := range spendOuts {
		hash, err := chainhash.NewHash(spendOut.TransactionHash)
		if err != nil {
			return nil, err
		}
		newTxIn := wire.NewTxIn(&wire.OutPoint{
			Hash:  *hash,
			Index: uint32(spendOut.Index),
		}, nil)
		txIns = append(txIns, newTxIn)
		totalValue += spendOut.Value
	}

	var tx = &wire.MsgTx{
		Version:  wire.TxVersion,
		TxIn:     txIns,
		TxOut:    txOuts,
		LockTime: 0,
	}

	return tx, nil
}
