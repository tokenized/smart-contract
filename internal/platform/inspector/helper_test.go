package inspector

import (
	"bytes"
	"encoding/hex"
	"errors"
	"strings"

	"github.com/btcsuite/btcd/txscript"
	"github.com/tokenized/smart-contract/pkg/protocol"
	"github.com/tokenized/smart-contract/pkg/wire"
)

func loadTXFromHex(raw string) wire.MsgTx {
	data := strings.Trim(string(raw), "\n ")

	// setup the tx
	b, err := hex.DecodeString(data)
	if err != nil {
		panic("Failed to decode payload")
	}

	return decodeTX(b)
}

func decodeTX(b []byte) wire.MsgTx {
	tx := wire.MsgTx{}
	buf := bytes.NewReader(b)

	if err := tx.Deserialize(buf); err != nil {
		panic("Failed to deserialize TX")
	}

	return tx
}

func loadMessageFromTX(tx wire.MsgTx) protocol.OpReturnMessage {
	for _, txOut := range tx.TxOut {
		m, err := newOpReturnMessage(txOut)
		if err != nil {
			panic("Broken OP_RETURN message")
		}

		if m == nil {
			// this isn't something we are interested in
			continue
		}

		return m
	}

	panic("No tokenized OP_RETURN")
}

func newOpReturnMessage(txOut *wire.TxOut) (protocol.OpReturnMessage, error) {
	if txOut.PkScript[0] != txscript.OP_RETURN {
		return nil, errors.New("Payload is not an OP_RETURN")
	}

	return protocol.New(txOut.PkScript)
}
