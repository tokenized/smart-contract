package inspector

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"sync"

	"github.com/tokenized/pkg/bitcoin"
	"github.com/tokenized/pkg/logger"
	"github.com/tokenized/pkg/wire"

	"github.com/tokenized/specification/dist/golang/actions"
	"github.com/tokenized/specification/dist/golang/protocol"

	"github.com/pkg/errors"
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
	RejectText    string

	lock sync.RWMutex
}

func (itx *Transaction) String(net bitcoin.Network) string {
	result := fmt.Sprintf("TxId: %s (%d bytes)\n", itx.Hash.String(), itx.MsgTx.SerializeSize())
	result += fmt.Sprintf("  Version: %d\n", itx.MsgTx.Version)
	result += "  Inputs:\n\n"
	for i, input := range itx.MsgTx.TxIn {
		result += fmt.Sprintf("    Outpoint: %d - %s\n", input.PreviousOutPoint.Index,
			input.PreviousOutPoint.Hash.String())
		result += fmt.Sprintf("    Script: %x\n", input.SignatureScript)
		result += fmt.Sprintf("    Sequence: %x\n", input.Sequence)

		if !itx.Inputs[i].Address.IsEmpty() {
			result += fmt.Sprintf("    Address: %s\n",
				bitcoin.NewAddressFromRawAddress(itx.Inputs[i].Address, net).String())
		}
		result += fmt.Sprintf("    Value: %d\n\n", itx.Inputs[i].UTXO.Value)
	}
	result += "  Outputs:\n\n"
	for i, output := range itx.MsgTx.TxOut {
		result += fmt.Sprintf("    Value: %.08f\n", float32(output.Value)/100000000.0)
		result += fmt.Sprintf("    Script: %x\n", output.PkScript)
		if !itx.Outputs[i].Address.IsEmpty() {
			result += fmt.Sprintf("    Address: %s\n",
				bitcoin.NewAddressFromRawAddress(itx.Outputs[i].Address, net).String())
		}
		result += "\n"
	}
	result += fmt.Sprintf("  LockTime: %d\n", itx.MsgTx.LockTime)
	return result
}

// Setup finds the tokenized message. It is required if the inspector transaction was created using
//   the NewBaseTransactionFromWire function.
func (itx *Transaction) Setup(ctx context.Context, isTest bool) error {
	itx.lock.Lock()
	defer itx.lock.Unlock()

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
	itx.lock.RLock()
	defer itx.lock.RUnlock()

	if itx.MsgProto == nil {
		return nil
	}

	if err := itx.MsgProto.Validate(); err != nil {
		logger.Warn(ctx, "Protocol message is invalid : %s", err)
		itx.RejectCode = actions.RejectionsMsgMalformed
		itx.RejectText = err.Error()
		return nil
	}

	return nil
}

// PromoteFromUTXOs will populate the inputs and outputs accordingly using UTXOs instead of a node.
func (itx *Transaction) PromoteFromUTXOs(ctx context.Context, utxos []bitcoin.UTXO) error {
	itx.lock.Lock()

	if err := itx.ParseOutputsWithoutNode(ctx); err != nil {
		itx.lock.Unlock()
		return err
	}

	if err := itx.ParseInputsFromUTXOs(ctx, utxos); err != nil {
		itx.lock.Unlock()
		return err
	}

	itx.lock.Unlock()
	itx.lock.RLock()
	return nil
}

// Promote will populate the inputs and outputs accordingly
func (itx *Transaction) Promote(ctx context.Context, node NodeInterface) error {
	itx.lock.Lock()

	if err := itx.ParseOutputs(ctx, node); err != nil {
		itx.lock.Unlock()
		return err
	}

	if err := itx.ParseInputs(ctx, node); err != nil {
		itx.lock.Unlock()
		return err
	}

	itx.lock.Unlock()
	itx.lock.RLock()
	return nil
}

// IsPromoted returns true if inputs and outputs are populated.
func (itx *Transaction) IsPromoted(ctx context.Context) bool {
	itx.lock.RLock()
	defer itx.lock.RUnlock()

	return len(itx.Inputs) > 0 && len(itx.Outputs) > 0
}

// ParseOutputsWithoutNode sets the Outputs property of the Transaction
func (itx *Transaction) ParseOutputsWithoutNode(ctx context.Context) error {
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

// ParseInputsFromUTXOs sets the Inputs property of the Transaction
func (itx *Transaction) ParseInputsFromUTXOs(ctx context.Context, utxos []bitcoin.UTXO) error {

	// Build inputs
	inputs := make([]Input, 0, len(itx.MsgTx.TxIn))
	offset := 0
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

		if !txin.PreviousOutPoint.Hash.Equal(&utxos[offset].Hash) ||
			txin.PreviousOutPoint.Index != utxos[offset].Index {
			return errors.New("Mismatched UTXO")
		}

		input, err := buildInput(utxos[offset])
		if err != nil {
			return errors.Wrap(err, "build input")
		}

		inputs = append(inputs, *input)
		offset++
	}

	itx.Inputs = inputs
	return nil
}

// ParseInputs sets the Inputs property of the Transaction
func (itx *Transaction) ParseInputs(ctx context.Context, node NodeInterface) error {

	// Fetch input transactions from RPC
	outpoints := make([]wire.OutPoint, 0, len(itx.MsgTx.TxIn))
	for _, txin := range itx.MsgTx.TxIn {
		if txin.PreviousOutPoint.Index != 0xffffffff {
			outpoints = append(outpoints, txin.PreviousOutPoint)
		}
	}

	utxos, err := node.GetOutputs(ctx, outpoints)
	if err != nil {
		return err
	}

	return itx.ParseInputsFromUTXOs(ctx, utxos)
}

func buildInput(utxo bitcoin.UTXO) (*Input, error) {
	address, err := utxo.Address()
	if err != nil {
		return nil, err
	}

	// Build the Input
	input := Input{
		Address: address,
		UTXO:    utxo,
	}

	return &input, nil
}

// GetPublicKeyForInput attempts to find a public key in the locking and unlocking scripts.
// Currently supports P2PK and P2PKH.
func (itx *Transaction) GetPublicKeyForInput(index int) (bitcoin.PublicKey, error) {
	// P2PKH script contains public key in unlock script
	if itx.Inputs[index].Address.Type() == bitcoin.ScriptTypePKH {
		pk, err := bitcoin.PublicKeyFromUnlockingScript(itx.MsgTx.TxIn[index].SignatureScript)
		if err != nil {
			return bitcoin.PublicKey{}, errors.Wrap(err, "unlock script")
		}

		publicKey, err := bitcoin.PublicKeyFromBytes(pk)
		if err != nil {
			return bitcoin.PublicKey{}, errors.Wrap(err, "parse public key")
		}

		return publicKey, nil
	}

	// P2PK script contains public key in locking script
	if itx.Inputs[index].Address.Type() == bitcoin.ScriptTypePKH {
		pk, err := bitcoin.PublicKeyFromLockingScript(itx.Inputs[index].UTXO.LockingScript)
		if err != nil {
			return bitcoin.PublicKey{}, errors.Wrap(err, "locking script")
		}

		publicKey, err := bitcoin.PublicKeyFromBytes(pk)
		if err != nil {
			return bitcoin.PublicKey{}, errors.Wrap(err, "parse public key")
		}

		return publicKey, nil
	}

	return bitcoin.PublicKey{}, errors.Wrap(bitcoin.ErrWrongType, "not found")
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
	itx.lock.RLock()
	defer itx.lock.RUnlock()
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

func (itx *Transaction) Fee() (uint64, error) {
	result := uint64(0)

	if len(itx.Inputs) != len(itx.MsgTx.TxIn) {
		return 0, ErrUnpromotedTx
	}

	for _, input := range itx.Inputs {
		result += input.UTXO.Value
	}

	for _, output := range itx.MsgTx.TxOut {
		if output.Value > result {
			return 0, ErrNegativeFee
		}
		result -= output.Value
	}

	return result, nil
}

func (itx *Transaction) FeeRate() (float32, error) {
	fee, err := itx.Fee()
	if err != nil {
		return 0.0, err
	}

	size := itx.MsgTx.SerializeSize()
	if size == 0 {
		return 0.0, ErrIncompleteTx
	}

	return float32(fee) / float32(size), nil
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

// Addresses returns all the addresses involved in the transaction
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

// AddressesUnique returns the unique addresses involved in a transaction
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

// PKHs returns all the PKHs involved in the transaction. This includes hashes of the public keys in
//   inputs.
func (itx *Transaction) PKHs() ([]bitcoin.Hash20, error) {
	result := make([]bitcoin.Hash20, 0)

	for _, input := range itx.MsgTx.TxIn {
		pubkeys, err := bitcoin.PubKeysFromSigScript(input.SignatureScript)
		if err != nil {
			return nil, err
		}
		for _, pubkey := range pubkeys {
			pkh, err := bitcoin.NewHash20FromData(pubkey)
			if err != nil {
				return nil, err
			}
			result = append(result, *pkh)
		}
	}

	for _, output := range itx.MsgTx.TxOut {
		pkhs, err := bitcoin.PKHsFromLockingScript(output.PkScript)
		if err != nil {
			return nil, err
		}
		result = append(result, pkhs...)
	}

	return result, nil
}

// PKHsUnique returns the unique PKH addresses involved in a transaction
func (itx *Transaction) PKHsUnique() ([]bitcoin.Hash20, error) {
	result, err := itx.PKHs()
	if err != nil {
		return nil, err
	}
	return uniquePKHs(result), nil
}

// uniquePKHs is an isolated function used for testing
func uniquePKHs(pkhs []bitcoin.Hash20) []bitcoin.Hash20 {
	result := make([]bitcoin.Hash20, 0, len(pkhs))

	// Spin over every address and check if it is found
	// in the list of unique addresses
	for _, pkh := range pkhs {
		if len(result) == 0 {
			result = append(result, pkh)
			continue
		}

		seen := false
		for _, seenPKH := range result {
			// We have seen this address
			if seenPKH.Equal(&pkh) {
				seen = true
				break
			}
		}

		if !seen {
			result = append(result, pkh)
		}
	}

	return result
}

func (itx *Transaction) Write(w io.Writer) error {
	// Version
	if _, err := w.Write([]byte{2}); err != nil {
		return errors.Wrap(err, "version")
	}

	if err := itx.MsgTx.Serialize(w); err != nil {
		return errors.Wrap(err, "tx")
	}

	if err := binary.Write(w, binary.LittleEndian, uint32(len(itx.Inputs))); err != nil {
		return errors.Wrap(err, "inputs count")
	}

	for i, _ := range itx.Inputs {
		if err := itx.Inputs[i].Write(w); err != nil {
			return err
		}
	}

	if _, err := w.Write([]byte{uint8(itx.RejectCode)}); err != nil {
		return errors.Wrap(err, "reject code")
	}
	return nil
}

func (itx *Transaction) Read(r io.Reader, isTest bool) error {
	// Version
	var version [1]byte
	if _, err := r.Read(version[:]); err != nil {
		return err
	}
	if version[0] != 0 && version[0] != 1 && version[0] != 2 {
		return fmt.Errorf("Unknown version : %d", version[0])
	}

	msg := wire.MsgTx{}
	if err := msg.Deserialize(r); err != nil {
		return err
	}
	itx.MsgTx = &msg
	itx.Hash = msg.TxHash()

	// Inputs
	var count uint32
	if version[0] >= 2 {
		if err := binary.Read(r, binary.LittleEndian, &count); err != nil {
			return errors.Wrap(err, "inputs count")
		}
	} else {
		count = uint32(len(msg.TxIn))
	}

	itx.Inputs = make([]Input, count)
	for i, _ := range itx.Inputs {
		if err := itx.Inputs[i].Read(version[0], r); err != nil {
			return err
		}
	}

	var rejectCode [1]byte
	if _, err := r.Read(rejectCode[:]); err != nil {
		return err
	}
	itx.RejectCode = uint32(rejectCode[0])

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
	var err error
	for i, txOut := range itx.MsgTx.TxOut {
		itx.MsgProto, err = protocol.Deserialize(txOut.PkScript, isTest)
		if err == nil {
			itx.MsgProtoIndex = uint32(i)
			break // Tokenized output found
		}
	}

	return nil
}
