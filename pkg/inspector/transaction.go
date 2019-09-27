package inspector

import (
	"bytes"
	"context"
	"fmt"

	"github.com/tokenized/smart-contract/pkg/bitcoin"
	"github.com/tokenized/smart-contract/pkg/logger"
	"github.com/tokenized/smart-contract/pkg/wire"
	"github.com/tokenized/specification/dist/golang/actions"
	"github.com/tokenized/specification/dist/golang/protocol"
)

var (
	// Incoming protocol message types (requests)
	incomingMessageTypes = map[string]bool{
		actions.CodeContractOffer:         true,
		actions.CodeContractAmendment:     true,
		actions.CodeAssetDefinition:       true,
		actions.CodeAssetModification:     true,
		actions.CodeTransfer:              true,
		actions.CodeProposal:              true,
		actions.CodeBallotCast:            true,
		actions.CodeOrder:                 true,
		actions.CodeContractAddressChange: true,
	}

	// Outgoing protocol message types (responses)
	outgoingMessageTypes = map[string]bool{
		actions.CodeAssetCreation:     true,
		actions.CodeContractFormation: true,
		actions.CodeSettlement:        true,
		actions.CodeVote:              true,
		actions.CodeBallotCounted:     true,
		actions.CodeResult:            true,
		actions.CodeFreeze:            true,
		actions.CodeThaw:              true,
		actions.CodeConfiscation:      true,
		actions.CodeReconciliation:    true,
		actions.CodeRejection:         true,
	}
)

// Transaction represents an ITX (Inspector Transaction) containing
// information about a transaction that is useful to the protocol.
type Transaction struct {
	Hash          *bitcoin.Hash32
	MsgTx         *wire.MsgTx
	MsgProto      actions.Action
	MsgProtoIndex uint32
	Inputs        []Input
	Outputs       []Output
	RejectCode    uint32
}

// Setup finds the tokenized message. It is required if the inspector transaction was created using
//   the NewBaseTransactionFromWire function.
func (itx *Transaction) Setup(ctx context.Context, isTest bool) error {
	// Find and deserialize protocol message
	var err error
	for i, txOut := range itx.MsgTx.TxOut {
		itx.MsgProto, err = protocol.Deserialize(txOut.PkScript, isTest)
		if err == nil {
			if err := itx.MsgProto.Validate(); err != nil {
				itx.RejectCode = actions.RejectionsMsgMalformed
				logger.Warn(ctx, "Protocol message is invalid : %s", err)
				return nil
			}
			itx.MsgProtoIndex = uint32(i)
			break // Tokenized output found
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
		itx.RejectCode = actions.RejectionsMsgMalformed
		return nil
	}

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
		output, err := buildOutput(itx.Hash, itx.MsgTx, n)
		if err != nil {
			return err
		}

		outputs = append(outputs, *output)
	}

	itx.Outputs = outputs
	return nil
}

func buildOutput(hash *bitcoin.Hash32, tx *wire.MsgTx, n int) (*Output, error) {
	txout := tx.TxOut[n]

	address, err := bitcoin.RawAddressFromLockingScript(txout.PkScript)
	if err != nil && err != bitcoin.ErrUnknownScriptTemplate {
		return nil, err
	}

	utxo := NewUTXOFromHashWire(hash, tx, uint32(n))

	output := Output{
		Address: address,
		UTXO:    utxo,
	}

	return &output, nil
}

// ParseInputs sets the Inputs property of the Transaction
func (itx *Transaction) ParseInputs(ctx context.Context, node NodeInterface) error {
	inputs := make([]Input, 0, len(itx.MsgTx.TxIn))

	for _, txin := range itx.MsgTx.TxIn {
		if txin.PreviousOutPoint.Index == 0xffffffff {
			// Empty coinbase input
			inputs = append(inputs, Input{
				UTXO: bitcoin.UTXO{
					Index: 0xffffffff,
				},
			})
			continue
		}

		// Lookup outpoint transaction
		h := txin.PreviousOutPoint.Hash
		inputTX, err := node.GetTX(ctx, &h)
		if err != nil {
			return err
		}

		input, err := buildInput(&h, inputTX, txin.PreviousOutPoint.Index)
		if err != nil {
			return err
		}

		inputs = append(inputs, *input)
	}

	itx.Inputs = inputs
	return nil
}

func buildInput(hash *bitcoin.Hash32, tx *wire.MsgTx, n uint32) (*Input, error) {
	utxo := NewUTXOFromHashWire(hash, tx, n)

	address, err := utxo.Address()
	if err != nil {
		return nil, err
	}

	// Build the Input
	input := Input{
		Address: address,
		UTXO:    utxo,
		FullTx:  tx,
	}

	return &input, nil
}

// Returns all the input hashes
func (itx *Transaction) InputHashes() []bitcoin.Hash32 {
	hashes := []bitcoin.Hash32{}

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

	_, ok := incomingMessageTypes[itx.MsgProto.Code()]
	return ok
}

// IsOutgoingMessageType returns true is the message type is one that we
// responded with, false otherwise.
func (itx *Transaction) IsOutgoingMessageType() bool {
	if !itx.IsTokenized() {
		return false
	}

	_, ok := outgoingMessageTypes[itx.MsgProto.Code()]
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

func (itx *Transaction) IsRelevant(contractAddress bitcoin.RawAddress) bool {
	for _, input := range itx.Inputs {
		if input.Address.Equal(contractAddress) {
			return true
		}
	}
	for _, output := range itx.Outputs {
		if output.Address.Equal(contractAddress) {
			return true
		}
	}
	return false
}

// ContractAddresses returns the contract address, which may include more than one
func (itx *Transaction) ContractAddresses() []bitcoin.RawAddress {
	return GetProtocolContractAddresses(itx, itx.MsgProto)
}

// // ContractPKHs returns the contract address, which may include more than one
// func (itx *Transaction) ContractPKHs() [][]byte {
// 	return GetProtocolContractPKHs(itx, itx.MsgProto)
// }

// Addresses returns all the PKH addresses involved in the transaction
func (itx *Transaction) Addresses() []bitcoin.RawAddress {
	addresses := make([]bitcoin.RawAddress, 0, len(itx.Inputs)+len(itx.Outputs))

	for _, input := range itx.Inputs {
		if !input.Address.IsEmpty() {
			addresses = append(addresses, input.Address)
		}
	}

	for _, output := range itx.Outputs {
		if !output.Address.IsEmpty() {
			addresses = append(addresses, output.Address)
		}
	}

	return addresses
}

// AddressesUnique returns the unique PKH addresses involved in a transaction
func (itx *Transaction) AddressesUnique() []bitcoin.RawAddress {
	return uniqueAddresses(itx.Addresses())
}

// uniqueAddresses is an isolated function used for testing
func uniqueAddresses(addresses []bitcoin.RawAddress) []bitcoin.RawAddress {
	result := make([]bitcoin.RawAddress, 0, len(addresses))

	// Spin over every address and check if it is found
	// in the list of unique addresses
	for _, address := range addresses {
		if len(result) == 0 {
			result = append(result, address)
			continue
		}

		seen := false
		for _, seenAddress := range result {
			// We have seen this address
			if seenAddress.Equal(address) {
				seen = true
				break
			}
		}

		if !seen {
			result = append(result, address)
		}
	}

	return result
}

func (itx *Transaction) Write(buf *bytes.Buffer) error {
	buf.WriteByte(0) // Version

	if err := itx.MsgTx.Serialize(buf); err != nil {
		return err
	}

	for i, _ := range itx.Inputs {
		if err := itx.Inputs[i].Write(buf); err != nil {
			return err
		}
	}

	buf.WriteByte(uint8(itx.RejectCode))
	return nil
}

func (itx *Transaction) Read(buf *bytes.Reader, isTest bool) error {
	version, err := buf.ReadByte() // Version
	if err != nil {
		return err
	}
	if version != 0 {
		return fmt.Errorf("Unknown version : %d", version)
	}

	msg := wire.MsgTx{}
	if err := msg.Deserialize(buf); err != nil {
		return err
	}
	itx.MsgTx = &msg
	itx.Hash = msg.TxHash()

	// Inputs
	itx.Inputs = make([]Input, len(msg.TxIn))
	for i, _ := range itx.Inputs {
		if err := itx.Inputs[i].Read(buf); err != nil {
			return err
		}
	}

	rejectCode, err := buf.ReadByte()
	if err != nil {
		return err
	}
	itx.RejectCode = uint32(rejectCode)

	// Outputs
	outputs := []Output{}
	for i := range itx.MsgTx.TxOut {
		output, err := buildOutput(itx.Hash, itx.MsgTx, i)

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
	for i, txOut := range itx.MsgTx.TxOut {
		itx.MsgProto, err = protocol.Deserialize(txOut.PkScript, isTest)
		if err == nil {
			itx.MsgProtoIndex = uint32(i)
			break // Tokenized output found
		}
	}

	return nil
}
