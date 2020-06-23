package inspector

import (
	"bytes"
	"context"
	"encoding/hex"
	"strings"

	"github.com/tokenized/pkg/bitcoin"
	"github.com/tokenized/pkg/txbuilder"
	"github.com/tokenized/pkg/wire"

	"github.com/tokenized/specification/dist/golang/actions"
	"github.com/tokenized/specification/dist/golang/protocol"

	"github.com/pkg/errors"
)

/**
 * Inspector Service
 *
 * What is my purpose?
 * - You look at Bitcoin transactions that I give you
 * - You tell me if they contain return data of interest
 * - You give me back special transaction objects (ITX objects)
 */

var (
	// ErrDecodeFail Failed to decode a transaction payload
	ErrDecodeFail = errors.New("Failed to decode payload")

	// ErrInvalidProtocol The op return data was invalid
	ErrInvalidProtocol = errors.New("Invalid protocol message")

	// ErrMissingInputs
	ErrMissingInputs = errors.New("Message is missing inputs")

	// ErrMissingOutputs
	ErrMissingOutputs = errors.New("Message is missing outputs")

	ErrNegativeFee  = errors.New("Negative fee")
	ErrUnpromotedTx = errors.New("Unpromoted tx")
	ErrIncompleteTx = errors.New("Incomplete tx")

	// prefixP2PKH Pay to PKH prefix
	prefixP2PKH = []byte{0x76, 0xA9}
)

// NodeInterface represents a configured bitcoin node that is capable
// of looking up transactions and parameters for its network
type NodeInterface interface {
	SaveTX(context.Context, *wire.MsgTx) error
	GetTX(context.Context, *bitcoin.Hash32) (*wire.MsgTx, error)
	GetOutputs(context.Context, []wire.OutPoint) ([]bitcoin.UTXO, error)
}

// NewTransaction builds an ITX from a raw transaction.
func NewTransaction(ctx context.Context, raw string, isTest bool) (*Transaction, error) {
	data := strings.Trim(string(raw), "\n ")

	b, err := hex.DecodeString(data)
	if err != nil {
		return nil, errors.Wrap(ErrDecodeFail, "decoding string")
	}

	// Set up the Wire transaction
	tx := wire.MsgTx{}
	buf := bytes.NewReader(b)
	if err := tx.Deserialize(buf); err != nil {
		return nil, errors.Wrap(ErrDecodeFail, "deserializing wire message")
	}

	return NewTransactionFromWire(ctx, &tx, isTest)
}

// NewTransactionFromHash builds an ITX from a transaction hash
func NewTransactionFromHash(ctx context.Context, node NodeInterface, hash string, isTest bool) (*Transaction, error) {
	h, err := bitcoin.NewHash32FromStr(hash)
	if err != nil {
		return nil, err
	}

	tx, err := node.GetTX(ctx, h)
	if err != nil {
		return nil, err
	}

	return NewTransactionFromHashWire(ctx, h, tx, isTest)
}

// NewTransactionFromWire builds an ITX from a wire Msg Tx
func NewTransactionFromWire(ctx context.Context, tx *wire.MsgTx, isTest bool) (*Transaction, error) {
	hash := tx.TxHash()
	return NewTransactionFromHashWire(ctx, hash, tx, isTest)
}

// NewTransactionFromWire builds an ITX from a wire Msg Tx
func NewTransactionFromHashWire(ctx context.Context, hash *bitcoin.Hash32, tx *wire.MsgTx, isTest bool) (*Transaction, error) {
	// Must have inputs
	if len(tx.TxIn) == 0 {
		return nil, errors.Wrap(ErrMissingInputs, "parsing transaction")
	}

	// Must have outputs
	if len(tx.TxOut) == 0 {
		return nil, errors.Wrap(ErrMissingOutputs, "parsing transaction")
	}

	// Find and deserialize protocol message
	var msg actions.Action
	var msgProtoIndex uint32
	var err error
	for i, txOut := range tx.TxOut {
		msg, err = protocol.Deserialize(txOut.PkScript, isTest)
		if err == nil {
			msgProtoIndex = uint32(i)
			break // Tokenized output found
		}
	}

	return &Transaction{
		Hash:          hash,
		MsgTx:         tx.Copy(),
		MsgProto:      msg,
		MsgProtoIndex: msgProtoIndex,
	}, nil
}

// NewBaseTransactionFromHashWire creates a non-setup transaction from an already calculated tx hash
// and a wire tx. This is the same as NewTransactionFromHashWire except Setup must be called
// before the transaction is usable.
func NewBaseTransactionFromHashWire(ctx context.Context, hash *bitcoin.Hash32, tx *wire.MsgTx) (*Transaction, error) {
	return &Transaction{
		Hash:  hash,
		MsgTx: tx.Copy(),
	}, nil
}

// NewBaseTransactionFromWire creates a non-setup transaction from a wire tx. This is the same as
// NewTransactionFromWire except Setup must be called before the transaction is usable.
func NewBaseTransactionFromWire(ctx context.Context, tx *wire.MsgTx) (*Transaction, error) {
	return &Transaction{
		Hash:  tx.TxHash(),
		MsgTx: tx.Copy(),
	}, nil
}

// NewUTXOFromWire returns a new UTXO from a wire message.
func NewUTXOFromWire(tx *wire.MsgTx, index uint32) bitcoin.UTXO {
	return bitcoin.UTXO{
		Hash:          *tx.TxHash(),
		Index:         index,
		LockingScript: tx.TxOut[index].PkScript,
		Value:         tx.TxOut[index].Value,
	}
}

// NewUTXOFromHashWire returns a new UTXO from a wire message.
func NewUTXOFromHashWire(hash *bitcoin.Hash32, tx *wire.MsgTx, index uint32) bitcoin.UTXO {
	return bitcoin.UTXO{
		Hash:          *hash,
		Index:         index,
		LockingScript: tx.TxOut[index].PkScript,
		Value:         tx.TxOut[index].Value,
	}
}

func NewTransactionFromTxBuilder(ctx context.Context, tx *txbuilder.TxBuilder, isTest bool) (*Transaction, error) {
	result, err := NewTransactionFromWire(ctx, tx.MsgTx, isTest)
	if err != nil {
		return result, errors.Wrap(err, "new from wire")
	}

	utxos := make([]bitcoin.UTXO, 0, len(tx.Inputs))
	for i, input := range tx.Inputs {
		if tx.MsgTx.TxIn[i].PreviousOutPoint.Index == 0xffffffff {
			continue // skip coinbase inputs
		}
		utxos = append(utxos, bitcoin.UTXO{
			Hash:          tx.MsgTx.TxIn[i].PreviousOutPoint.Hash,
			Index:         tx.MsgTx.TxIn[i].PreviousOutPoint.Index,
			Value:         input.Value,
			LockingScript: input.LockingScript,
		})
	}

	if err := result.PromoteFromUTXOs(ctx, utxos); err != nil {
		return result, errors.Wrap(err, "promote")
	}

	return result, nil
}
