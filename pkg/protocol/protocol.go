package protocol

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"time"
)

const (
	// OpReturn (OP_RETURN) is a script opcode is used to mark a transaction
	// output as invalid, and can be used to add data to a TX.
	OpReturn = 0x6a

	// ProtocolID is the current protocol ID
	ProtocolID = "tokenized.com"

	// Version of the Tokenized protocol.
	Version = uint8(0)
)

// PayloadMessage is the interface for messages that are derived from
// payloads, such as asset types and message types.
type PayloadMessage interface {
	Type() string
	Serialize() ([]byte, error)
	Write(b []byte) (int, error)
}

// OpReturnMessage implements a base interface for all message types.
type OpReturnMessage interface {
	Type() string
	String() string
	PayloadMessage() (PayloadMessage, error)
	serialize() ([]byte, error)
	write(b []byte) (int, error)
}

// Deserialize returns a message, as an OpReturnMessage, from the OP_RETURN script.
func Deserialize(b []byte) (OpReturnMessage, error) {
	buf := bytes.NewBuffer(b)

	var opCode byte
	var err error

	// Parse OP_RETURN op code
	err = binary.Read(buf, defaultEndian, &opCode)
	if err != nil {
		return nil, err
	}

	if opCode != OpReturn {
		return nil, fmt.Errorf("Not an op return output : %02x", opCode)
	}

	// Parse push op code for op return protocol ID
	err = binary.Read(buf, defaultEndian, &opCode)
	if err != nil {
		return nil, err
	}

	if int(opCode) != len(ProtocolID) {
		return nil, fmt.Errorf("Push not correct size for protocol ID : %02x", opCode)
	}

	// Parse protocol ID
	protocolID := make([]byte, len(ProtocolID))
	_, err = buf.Read(protocolID)
	if err != nil {
		return nil, err
	}

	if !bytes.Equal(protocolID, []byte(ProtocolID)) {
		return nil, fmt.Errorf("Invalid protocol ID : %s", string(protocolID))
	}

	// Parse push op code for payload length + 3 for version and message type code
	var payloadSize uint64
	payloadSize, err = ParsePushDataScript(buf)
	if err != nil {
		return nil, err
	}

	if uint64(buf.Len()) < payloadSize {
		return nil, fmt.Errorf("Payload push op code is too large for message : %d", payloadSize)
	}

	// Parse version
	var version uint8
	err = binary.Read(buf, defaultEndian, &version)
	if err != nil {
		return nil, err
	}

	if version != Version {
		return nil, fmt.Errorf("Unsupported version : %02x", version)
	}

	// Parse message type code
	code := make([]byte, 2)
	_, err = buf.Read(code)
	if err != nil {
		return nil, err
	}

	msg := TypeMapping(string(code))
	if msg == nil {
		return nil, fmt.Errorf("Unknown code : %s", code)
	}

	if _, err := msg.write(b); err != nil {
		return nil, err
	}

	return msg, nil
}

// Serialize returns a complete op return script including the specified payload.
func Serialize(msg OpReturnMessage) ([]byte, error) {
	var buf bytes.Buffer
	var err error

	// Write OP_RETURN op code
	err = binary.Write(&buf, defaultEndian, OpReturn)
	if err != nil {
		return nil, err
	}

	// Write signature push op code
	protocolIDSize := uint8(len(ProtocolID))
	err = binary.Write(&buf, defaultEndian, protocolIDSize)
	if err != nil {
		return nil, err
	}

	// Write protocol ID
	_, err = buf.Write([]byte(ProtocolID))
	if err != nil {
		return nil, err
	}

	var payload []byte
	payload, err = msg.serialize()
	if err != nil {
		return nil, err
	}

	// Write push op code for payload length + 3 for version and message type code
	_, err = buf.Write(PushDataScript(uint64(len(payload)) + 3))
	if err != nil {
		return nil, err
	}

	// Write version
	err = binary.Write(&buf, defaultEndian, Version)
	if err != nil {
		return nil, err
	}

	// Write message type code
	_, err = buf.Write([]byte(msg.Type()))
	if err != nil {
		return nil, err
	}

	// Write payload
	_, err = buf.Write(payload)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// Code returns the identifying code from the OP_RETURN payload.
func Code(b []byte) (string, error) {
	if len(b) < 9 || b[0] != OpReturn {
		return "", errors.New("Not an OP_RETURN payload")
	}

	offset := 7

	if b[1] < 0x4c {
		offset = 6
	}

	return string(b[offset : offset+2]), nil
}

// ------------------------------------------------------------------------------------------------
// TxId represents a Bitcoin transaction ID. (Double SHA256 of tx data)
type TxId struct {
	data [32]byte
}

func TxIdFromBytes(data []byte) TxId {
	var result TxId
	copy(result.data[:], data)
	return result
}

// Bytes returns the byte slice for the TxId.
func (id *TxId) Bytes() []byte { return id.data[:] }

// Serialize returns a byte slice with the TxId in it.
func (id *TxId) Serialize() ([]byte, error) { return id.data[:], nil }

// Write reads a TxId from a bytes.Buffer
func (id *TxId) Write(buf *bytes.Buffer) error {
	return readLen(buf, id.data[:])
}

// ------------------------------------------------------------------------------------------------
// PublicKeyHash represents a Bitcoin Public Key Hash. Often used as an address to receive transactions.
type PublicKeyHash struct {
	data [20]byte
}

// PublicKeyHashFromBytes returns a PublicKeyHash with the specified bytes.
func PublicKeyHashFromBytes(data []byte) PublicKeyHash {
	var result PublicKeyHash
	copy(result.data[:], data)
	return result
}

// Bytes returns a byte slice containing the PublicKeyHash.
func (hash *PublicKeyHash) Bytes() []byte { return hash.data[:] }

// Serialize returns a byte slice with the PublicKeyHash in it.
func (hash *PublicKeyHash) Serialize() ([]byte, error) { return hash.data[:], nil }

// Write reads a PublicKeyHash from a bytes.Buffer
func (hash *PublicKeyHash) Write(buf *bytes.Buffer) error {
	return readLen(buf, hash.data[:])
}

// ------------------------------------------------------------------------------------------------
// AssetCode represents a unique identifier for a Tokenized asset.
type AssetCode struct {
	data [32]byte
}

// IsZero returns true if the AssetCode is all zeroes. (empty)
func (id *AssetCode) IsZero() bool {
	return bytes.Equal(id.data[:], []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0})
}

// AssetCodeFromBytes returns a AssetCode with the specified bytes.
func AssetCodeFromBytes(data []byte) AssetCode {
	var result AssetCode
	copy(result.data[:], data)
	return result
}

// Bytes returns a byte slice containing the AssetCode.
func (code *AssetCode) Bytes() []byte { return code.data[:] }

// Serialize returns a byte slice with the AssetCode in it.
func (code *AssetCode) Serialize() ([]byte, error) { return code.data[:], nil }

// Write reads a AssetCode from a bytes.Buffer
func (code *AssetCode) Write(buf *bytes.Buffer) error {
	return readLen(buf, code.data[:])
}

// ------------------------------------------------------------------------------------------------
// ContractCode represents a unique identifier for a Tokenized static contract.
type ContractCode struct {
	data [32]byte
}

// IsZero returns true if the ContractCode is all zeroes. (empty)
func (id *ContractCode) IsZero() bool {
	return bytes.Equal(id.data[:], []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0})
}

// AssetCodeFromBytes returns a ContractCode with the specified bytes.
func ContractCodeFromBytes(data []byte) ContractCode {
	var result ContractCode
	copy(result.data[:], data)
	return result
}

// Bytes returns a byte slice containing the ContractCode.
func (code *ContractCode) Bytes() []byte { return code.data[:] }

// Serialize returns a byte slice with the ContractCode in it.
func (code *ContractCode) Serialize() ([]byte, error) { return code.data[:], nil }

// Write reads a ContractCode from a bytes.Buffer
func (code *ContractCode) Write(buf *bytes.Buffer) error {
	return readLen(buf, code.data[:])
}

// ------------------------------------------------------------------------------------------------
// Timestamp represents a time in the Tokenized protocol.
type Timestamp struct {
	nanoseconds uint64 // nanoseconds since Unix epoch
}

// CurrentTimestamp returns a Timestamp containing the current time.
func CurrentTimestamp() Timestamp {
	return Timestamp{nanoseconds: uint64(time.Now().UnixNano())}
}

// Nano returns the nanoseconds since the Unix epoch for the Timestamp.
func (time *Timestamp) Nano() uint64 { return time.nanoseconds }

// Serialize returns a byte slice with the Timestamp in it.
func (time *Timestamp) Serialize() ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := write(buf, &time.nanoseconds); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Write reads a Timestamp from a bytes.Buffer
func (time *Timestamp) Write(buf *bytes.Buffer) error {
	if err := read(buf, &time.nanoseconds); err != nil {
		return err
	}
	return nil
}
