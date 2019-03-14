package protocol

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
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
// payloads, such as asset types.
type PayloadMessage interface {
	Type() string
	serialize() ([]byte, error)
	write(b []byte) (int, error)
}

// New returns the approriate PayloadMessage for the
// given code.
func New(code []byte) (PayloadMessage, error) {
	s := string(code)

	switch s {
	case CodeShareCommon:
		result := ShareCommon{}
		return &result, nil
	}

	return nil, fmt.Errorf("No asset type for code %s", code)
}

// OpReturnMessage implements a base interface for all message types.
type OpReturnMessage interface {
	PayloadMessage
	String() string
	PayloadMessage() (PayloadMessage, error)
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
func Serialize(msg PayloadMessage) ([]byte, error) {
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

// WriteVarChar writes a variable character string
func WriteVarChar(buf *bytes.Buffer, value string, sizeBits int) error {
	if err := WriteVariableSize(buf, uint64(len(value)), sizeBits, 0); err != nil {
		return err
	}

	if _, err := buf.Write([]byte(value)); err != nil {
		return err
	}
	return nil
}

// ReadVarChar reads a variable character string
func ReadVarChar(buf *bytes.Buffer, sizeBits int) (string, error) {
	size, err := ReadVariableSize(buf, sizeBits, 0)
	if err != nil {
		return "", err
	}

	data := make([]byte, size)
	err = readLen(buf, data)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// WriteFixedChar writes a fixed character string
func WriteFixedChar(buf *bytes.Buffer, value string, size uint64) error {
	if uint64(len(value)) > size {
		return fmt.Errorf("FixedChar too long %d > %d", len(value), size)
	}
	if _, err := buf.Write([]byte(value)); err != nil {
		return err
	}

	// Pad with zeroes
	if uint64(len(value)) < size {
		padCount := size - uint64(len(value))
		empty := make([]byte, padCount)
		for i := uint64(0); i < padCount; i++ {
			empty[i] = 0
		}

		if _, err := buf.Write(empty); err != nil {
			return err
		}
	}
	return nil
}

// ReadFixedChar reads a fixed character string
func ReadFixedChar(buf *bytes.Buffer, size uint64) (string, error) {
	var err error
	data := make([]byte, size)
	err = readLen(buf, data)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// WriteVarBin writes a variable binary value
func WriteVarBin(buf *bytes.Buffer, value []byte, sizeBits int) error {
	if err := WriteVariableSize(buf, uint64(len(value)), sizeBits, 0); err != nil {
		return err
	}

	if _, err := buf.Write(value); err != nil {
		return err
	}
	return nil
}

// ReadVarBin reads a variable binary value
func ReadVarBin(buf *bytes.Buffer, sizeBits int) ([]byte, error) {
	size, err := ReadVariableSize(buf, sizeBits, 0)
	if err != nil {
		return []byte{}, err
	}

	data := make([]byte, size)
	err = readLen(buf, data)
	if err != nil {
		return []byte{}, err
	}
	return data, nil
}

// WriteFixedBin writes a fixed binary value
func WriteFixedBin(buf *bytes.Buffer, value []byte, size uint64) error {
	if uint64(len(value)) > size {
		return fmt.Errorf("FixedBin too long %d > %d", len(value), size)
	}
	if _, err := buf.Write(value); err != nil {
		return err
	}

	// Pad with zeroes
	if uint64(len(value)) < size {
		padCount := size - uint64(len(value))
		empty := make([]byte, padCount)
		for i := uint64(0); i < padCount; i++ {
			empty[i] = 0
		}

		if _, err := buf.Write(empty); err != nil {
			return err
		}
	}
	return nil
}

// WriteVariableSize writes a size/length to the buffer using the specified size unsigned integer.
// defaultSizeBits is used if sizeBits is zero.
func WriteVariableSize(buf *bytes.Buffer, size uint64, sizeBits int, defaultSizeBits int) error {
	if sizeBits == 0 {
		sizeBits = defaultSizeBits
	}
	if size >= 2<<uint64(sizeBits) {
		return fmt.Errorf("Size beyond size bits limit (%d) : %d", (2<<uint64(sizeBits))-1, size)
	}

	var err error
	switch sizeBits {
	case 8:
		err = write(buf, uint8(size))
	case 16:
		err = write(buf, uint16(size))
	case 32:
		err = write(buf, uint32(size))
	case 64:
		err = write(buf, uint64(size))
	default:
		return fmt.Errorf("Invalid variable size bits : %d", sizeBits)
	}
	if err != nil {
		return err
	}

	return nil
}

// ReadVariableSize reads a size/length from the buffer using the specified size unsigned integer.
// defaultSizeBits is used if sizeBits is zero.
func ReadVariableSize(buf *bytes.Buffer, sizeBits int, defaultSizeBits int) (uint64, error) {
	if sizeBits == 0 {
		sizeBits = defaultSizeBits
	}

	var err error
	var size uint64
	switch sizeBits {
	case 8:
		var size8 uint8
		err = read(buf, &size8)
		size = uint64(size8)
	case 16:
		var size16 uint16
		err = read(buf, &size16)
		size = uint64(size16)
	case 32:
		var size32 uint32
		err = read(buf, &size32)
		size = uint64(size32)
	case 64:
		err = read(buf, &size)
	default:
		err = fmt.Errorf("Invalid variable size bits : %d", sizeBits)
	}
	return size, err
}
