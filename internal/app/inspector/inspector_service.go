package inspector

/**
 * Inspector Service
 *
 * What is my purpose?
 * - You look at Bitcoin transactions that I give you
 * - You tell me if they contain return data of interest
 * - You give me back special transaction objects (ITX objects)
 */

import (
	"bytes"
	"errors"

	"github.com/tokenized/smart-contract/internal/app/network"
	"github.com/tokenized/smart-contract/pkg/protocol"
	"github.com/tokenized/smart-contract/pkg/txbuilder"
	"github.com/tokenized/smart-contract/pkg/wire"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/txscript"
)

var (
	targetVersion = []byte{0x0, 0x0, 0x0, 0x20}
)

type InspectorService struct {
	Network network.NetworkInterface
	Builder txbuilder.UTXOSetBuilder
}

func NewInspectorService(network network.NetworkInterface) InspectorService {
	builder := txbuilder.NewUTXOSetBuilder(network)

	return InspectorService{
		Network: network,
		Builder: builder,
	}
}

// Creating a new transaction from scratch, wire.MsgTx is omitted
//
func (s InspectorService) CreateTransaction(inputs txbuilder.UTXOs,
	outputs []txbuilder.TxOutput,
	msg protocol.OpReturnMessage) *Transaction {

	t := &Transaction{
		Inputs:   inputs,
		Outputs:  outputs,
		MsgProto: msg,
	}

	return t
}

// Primary purpose of this service. Does the supplied transaction concern us?
//
func (s InspectorService) MakeTransaction(tx *wire.MsgTx) (*Transaction, error) {
	msg, err := s.findTokenizedProtocol(tx)
	if err != nil {
		return nil, err
	}

	if msg == nil {
		// we didn't find one of our OP_RETURN messages, nothing to do.
		return nil, nil
	}

	// Outputs
	outputs, err := s.getOutputs(tx)
	if err != nil {
		return nil, err
	}

	t := &Transaction{
		Outputs:  outputs,
		MsgProto: msg,
		MsgTx:    tx,
	}

	return t, nil
}

func (s InspectorService) PromoteTransaction(tx *Transaction) (*Transaction, error) {
	// Inputs
	inputs, err := s.Builder.Build(tx.MsgTx)
	if err != nil {
		return nil, err
	}

	tx.Inputs = inputs

	// Input addreses
	inputAddresses, err := inputs.Addresses()
	if err != nil {
		return nil, err
	}

	tx.InputAddrs = inputAddresses

	// get all unspents outputs
	allUtxos, err := s.Builder.BuildFromOutputs(tx.MsgTx)
	if err != nil {
		return nil, err
	}

	tx.UTXOs = allUtxos

	return tx, nil
}

func (s InspectorService) getOutputs(tx *wire.MsgTx) ([]txbuilder.TxOutput, error) {
	outputs := []txbuilder.TxOutput{}

	for i, txOut := range tx.TxOut {
		if txOut.Value == 0 {
			continue
		}

		utxo := txbuilder.NewUTXOFromTX(*tx, uint32(i))

		address, err := utxo.PublicAddress(&chaincfg.MainNetParams)
		if err != nil {
			return nil, err
		}

		output := txbuilder.TxOutput{
			Index:   uint32(i),
			Value:   uint64(txOut.Value),
			Address: address,
		}

		outputs = append(outputs, output)
	}

	return outputs, nil
}

// findTokenizedProtocol returns a special Tokenized OP_RETURN message if
// one is found, otherwise nil.
func (s InspectorService) findTokenizedProtocol(tx *wire.MsgTx) (protocol.OpReturnMessage, error) {

	for _, txOut := range tx.TxOut {
		if !s.isTokenizedOpReturn(txOut.PkScript) {
			continue
		}

		m, err := s.newProtocolMessage(txOut)
		if err != nil {
			return nil, err
		}

		if m == nil {
			continue
		}

		return m, nil
	}

	return nil, nil
}

// Returns a Tokenized protocol instance
func (s InspectorService) newProtocolMessage(txOut *wire.TxOut) (protocol.OpReturnMessage, error) {
	if txOut.PkScript[0] != txscript.OP_RETURN {
		return nil, errors.New("Payload is not an OP_RETURN")
	}

	return protocol.New(txOut.PkScript)
}

func (s InspectorService) isTokenizedOpReturn(pkScript []byte) bool {
	if len(pkScript) < 20 {
		// this isn't long enough to be a sane message
		return false
	}

	version := make([]byte, 4, 4)

	// get the version. Where that is, depends on the message structure.
	if pkScript[1] < 0x4c {
		version = pkScript[2:6]
	} else {
		version = pkScript[3:7]
	}

	return bytes.Equal(version, targetVersion)
}
