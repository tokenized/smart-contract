package inspector

import (
	"context"

	"github.com/tokenized/smart-contract/pkg/protocol"
	"github.com/tokenized/smart-contract/pkg/wire"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcutil"
)

var (
	// Incoming protocol message types (requests)
	incomingMessageTypes = map[string]bool{
		protocol.CodeContractOffer:     true,
		protocol.CodeContractAmendment: true,
		protocol.CodeAssetDefinition:   true,
		protocol.CodeAssetModification: true,
		protocol.CodeTransfer:          true,
		protocol.CodeInitiative:        true,
		protocol.CodeReferendum:        true,
		protocol.CodeBallotCast:        true,
		protocol.CodeOrder:             true,
	}

	// Outgoing protocol message types (responses)
	outgoingMessageTypes = map[string]bool{
		protocol.CodeAssetCreation:     true,
		protocol.CodeContractFormation: true,
		protocol.CodeSettlement:        true,
		protocol.CodeVote:              true,
		protocol.CodeBallotCounted:     true,
		protocol.CodeResult:            true,
		protocol.CodeFreeze:            true,
		protocol.CodeThaw:              true,
		protocol.CodeConfiscation:      true,
		protocol.CodeReconciliation:    true,
		protocol.CodeRejection:         true,
	}
)

// Transaction represents an ITX (Inspector Transaction) containing
// information about a transaction that is useful to the protocol.
type Transaction struct {
	Hash     chainhash.Hash
	MsgTx    *wire.MsgTx
	MsgProto protocol.OpReturnMessage
	Inputs   []Input
	Outputs  []Output
}

// Promote will populate the inputs and outputs accordingly
func (itx *Transaction) Promote(ctx context.Context, node NodeInterface, netParams *chaincfg.Params) error {

	if err := itx.ParseOutputs(ctx, netParams); err != nil {
		return err
	}

	if err := itx.ParseInputs(ctx, node, netParams); err != nil {
		return err
	}

	return nil
}

// IsPromoted returns true if inputs and outputs are populated.
func (itx *Transaction) IsPromoted(ctx context.Context) bool {
	return len(itx.Inputs) > 0 && len(itx.Outputs) > 0
}

// ParseOutputs sets the Outputs property of the Transaction
func (itx *Transaction) ParseOutputs(ctx context.Context, netParams *chaincfg.Params) error {
	outputs := []Output{}

	for n := range itx.MsgTx.TxOut {
		output, err := buildOutput(itx.MsgTx, n, netParams)

		if err != nil {
			return err
		}

		if output == nil {
			continue
		}

		outputs = append(outputs, *output)
	}

	itx.Outputs = outputs
	return nil
}

func buildOutput(tx *wire.MsgTx, n int, netParams *chaincfg.Params) (*Output, error) {
	txout := tx.TxOut[n]

	// Zero value output
	if txout.Value == 0 {
		return nil, nil
	}

	// Skip non-P2PKH scripts
	if !isPayToPublicKeyHash(txout.PkScript) {
		return nil, nil
	}

	utxo := NewUTXOFromWire(tx, uint32(n))

	address, err := utxo.PublicAddress(netParams)
	if err != nil {
		return nil, err
	}

	output := Output{
		Address: address,
		Index:   utxo.Index,
		Value:   utxo.Value,
		UTXO:    utxo,
	}

	return &output, nil
}

// ParseInputs sets the Inputs property of the Transaction
func (itx *Transaction) ParseInputs(ctx context.Context, node NodeInterface, netParams *chaincfg.Params) error {
	inputs := []Input{}

	for _, txin := range itx.MsgTx.TxIn {
		h := txin.PreviousOutPoint.Hash

		inputTX, err := node.GetTX(ctx, &h)
		if err != nil {
			return err
		}

		input, err := buildInput(inputTX, txin.PreviousOutPoint.Index, netParams)
		if err != nil {
			return err
		}

		inputs = append(inputs, *input)
	}

	itx.Inputs = inputs
	return nil
}

func buildInput(tx *wire.MsgTx, n uint32, netParams *chaincfg.Params) (*Input, error) {
	utxo := NewUTXOFromWire(tx, n)

	address, err := utxo.PublicAddress(netParams)
	if err != nil {
		return nil, err
	}

	// Build the Input
	input := Input{
		Address: address,
		Index:   utxo.Index,
		Value:   utxo.Value,
		UTXO:    utxo,
		FullTx:  tx,
	}

	return &input, nil
}

// Returns all the input hashes
func (itx *Transaction) InputHashes() []chainhash.Hash {
	hashes := []chainhash.Hash{}

	for _, txin := range itx.MsgTx.TxIn {
		hashes = append(hashes, txin.PreviousOutPoint.Hash)
	}

	return hashes
}

// IsTokenized determines if the inspected transaction is using the Tokenized protocol.
func (itx *Transaction) IsTokenized() bool {
	return itx.MsgProto != nil
}

// IsIncomingMessageType returns true is the message type is one that we
// want to process, false otherwise.
func (itx *Transaction) IsIncomingMessageType() bool {
	if !itx.IsTokenized() {
		return false
	}

	_, ok := incomingMessageTypes[itx.MsgProto.Type()]
	return ok
}

// IsOutgoingMessageType returns true is the message type is one that we
// responded with, false otherwise.
func (itx *Transaction) IsOutgoingMessageType() bool {
	if !itx.IsTokenized() {
		return false
	}

	_, ok := outgoingMessageTypes[itx.MsgProto.Type()]
	return ok
}

// UTXOs returns all the unspent transaction outputs created by this tx
func (itx *Transaction) UTXOs() UTXOs {
	utxos := UTXOs{}

	for _, output := range itx.Outputs {
		utxos = append(utxos, output.UTXO)
	}

	return utxos
}

// ContractAddresses returns the contract address, which may include more than one
func (itx *Transaction) ContractAddresses() []btcutil.Address {
	return GetProtocolContractAddresses(itx, itx.MsgProto)
}

// Addresses returns all the PKH addresses involved in the transaction
func (itx *Transaction) Addresses() []btcutil.Address {
	l := len(itx.Inputs) + len(itx.Outputs)
	addresses := make([]btcutil.Address, l, l)

	for i, input := range itx.Inputs {
		addresses[i] = input.Address
	}

	for i, output := range itx.Outputs {
		addresses[i+len(itx.Inputs)] = output.Address
	}

	return addresses
}

// AddressesUnique returns the unique PKH addresses involved in a transaction
func (itx *Transaction) AddressesUnique() []btcutil.Address {
	return uniqueAddresses(itx.Addresses())
}

// uniqueAddresses is an isolated function used for testing
func uniqueAddresses(s []btcutil.Address) []btcutil.Address {
	u := []btcutil.Address{}

	// Spin over every address and check if it is found
	// in the list of unique addresses
	for _, v := range s {
		if len(u) == 0 {
			u = append(u, v)
			continue
		}

		var seen bool

		for _, x := range u {
			// We have seen this address
			if x.String() == v.String() {
				seen = true
				continue
			}
		}

		if !seen {
			u = append(u, v)
		}
	}

	return u
}
