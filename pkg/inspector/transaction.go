package inspector

import (
	"bytes"
	"context"

	"github.com/tokenized/smart-contract/pkg/logger"
	"github.com/tokenized/smart-contract/pkg/wire"
	"github.com/tokenized/specification/dist/golang/protocol"

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
		protocol.CodeProposal:          true,
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
	Hash               chainhash.Hash
	MsgTx              *wire.MsgTx
	MsgProto           protocol.OpReturnMessage
	Inputs             []Input
	Outputs            []Output
	DataIsValid        bool
	SignaturesAreValid bool
}

// Setup finds the tokenized message. It is required if the inspector transaction was created using
//   the NewBaseTransactionFromWire function.
func (itx *Transaction) Setup(ctx context.Context, isTest bool) error {
	// Find and deserialize protocol message
	var msg protocol.OpReturnMessage
	var err error
	for _, txOut := range itx.MsgTx.TxOut {
		msg, err = protocol.Deserialize(txOut.PkScript, isTest)
		if err == nil {
			itx.MsgProto = msg
			if err := msg.Validate(); err != nil {
				itx.DataIsValid = false
				logger.Warn(ctx, "Protocol message is invalid : %s", err)
				return nil
			}
			itx.DataIsValid = true
			return nil // Tokenized output found
		}
	}

	return nil
}

// Validate checks the validity of the data in the protocol message.
func (itx *Transaction) Validate(ctx context.Context) error {
	if itx.MsgProto == nil {
		return nil
	}

	if err := itx.MsgProto.Validate(); err != nil {
		logger.Warn(ctx, "Protocol message is invalid : %s", err)
		itx.DataIsValid = false
		return nil
	}

	itx.DataIsValid = true
	return nil
}

// Promote will populate the inputs and outputs accordingly
func (itx *Transaction) Promote(ctx context.Context, node NodeInterface) error {

	if err := itx.ParseOutputs(ctx, node); err != nil {
		return err
	}

	if err := itx.ParseInputs(ctx, node); err != nil {
		return err
	}

	return nil
}

// IsPromoted returns true if inputs and outputs are populated.
func (itx *Transaction) IsPromoted(ctx context.Context) bool {
	return len(itx.Inputs) > 0 && len(itx.Outputs) > 0
}

// ParseOutputs sets the Outputs property of the Transaction
func (itx *Transaction) ParseOutputs(ctx context.Context, node NodeInterface) error {
	outputs := make([]Output, 0, len(itx.MsgTx.TxOut))

	for n := range itx.MsgTx.TxOut {
		output, err := buildOutput(&itx.Hash, itx.MsgTx, n, node.GetChainParams())

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

func buildOutput(hash *chainhash.Hash, tx *wire.MsgTx, n int, netParams *chaincfg.Params) (*Output, error) {
	txout := tx.TxOut[n]

	// Zero value output
	if txout.Value == 0 {
		return nil, nil
	}

	// Skip non-P2PKH scripts
	if !isPayToPublicKeyHash(txout.PkScript) {
		return nil, nil
	}

	utxo := NewUTXOFromHashWire(hash, tx, uint32(n))

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
func (itx *Transaction) ParseInputs(ctx context.Context, node NodeInterface) error {
	inputs := make([]Input, 0, len(itx.MsgTx.TxIn))

	for _, txin := range itx.MsgTx.TxIn {
		h := txin.PreviousOutPoint.Hash

		inputTX, err := node.GetTX(ctx, &h)
		if err != nil {
			return err
		}

		input, err := buildInput(&h, inputTX, txin.PreviousOutPoint.Index, node.GetChainParams())
		if err != nil {
			return err
		}

		inputs = append(inputs, *input)
	}

	itx.Inputs = inputs
	return nil
}

func buildInput(hash *chainhash.Hash, tx *wire.MsgTx, n uint32, netParams *chaincfg.Params) (*Input, error) {
	utxo := NewUTXOFromHashWire(hash, tx, n)

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

func (itx *Transaction) IsRelevant(contractAddress btcutil.Address) bool {
	for _, input := range itx.Inputs {
		if bytes.Equal(input.Address.ScriptAddress(), contractAddress.ScriptAddress()) {
			return true
		}
	}
	for _, output := range itx.Outputs {
		if bytes.Equal(output.Address.ScriptAddress(), contractAddress.ScriptAddress()) {
			return true
		}
	}
	return false
}

// ContractAddresses returns the contract address, which may include more than one
func (itx *Transaction) ContractAddresses() []btcutil.Address {
	return GetProtocolContractAddresses(itx, itx.MsgProto)
}

// ContractAddresses returns the contract address, which may include more than one
func (itx *Transaction) ContractPKHs() [][]byte {
	return GetProtocolContractPKHs(itx, itx.MsgProto)
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

func (itx *Transaction) Write(buf *bytes.Buffer) error {
	if err := itx.MsgTx.Serialize(buf); err != nil {
		return err
	}

	for i, _ := range itx.Inputs {
		if err := itx.Inputs[i].Write(buf); err != nil {
			return err
		}
	}

	if itx.DataIsValid {
		buf.WriteByte(0xff)
	} else {
		buf.WriteByte(0)
	}
	if itx.SignaturesAreValid {
		buf.WriteByte(0xff)
	} else {
		buf.WriteByte(0)
	}

	return nil
}

func (itx *Transaction) Read(buf *bytes.Buffer, netParams *chaincfg.Params, isTest bool) error {
	msg := wire.MsgTx{}
	if err := msg.Deserialize(buf); err != nil {
		return err
	}
	itx.MsgTx = &msg
	itx.Hash = msg.TxHash()

	// Inputs
	itx.Inputs = make([]Input, len(msg.TxIn))
	for i, _ := range itx.Inputs {
		if err := itx.Inputs[i].Read(buf, netParams); err != nil {
			return err
		}
	}

	byte, err := buf.ReadByte()
	if err != nil {
		return err
	}
	itx.DataIsValid = byte != 0
	byte, err = buf.ReadByte()
	if err != nil {
		return err
	}
	itx.SignaturesAreValid = byte != 0

	// Outputs
	outputs := []Output{}
	for i := range itx.MsgTx.TxOut {
		output, err := buildOutput(&itx.Hash, itx.MsgTx, i, netParams)

		if err != nil {
			return err
		}

		if output == nil {
			continue
		}

		outputs = append(outputs, *output)
	}

	itx.Outputs = outputs

	// Protocol Message
	for _, txOut := range itx.MsgTx.TxOut {
		itx.MsgProto, err = protocol.Deserialize(txOut.PkScript, isTest)
		if err == nil {
			break // Tokenized output found
		}
	}

	return nil
}
