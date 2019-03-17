package protocol

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

const (
	// OpMaxSingleBytePushdata represents the max length for a single byte push
	OpMaxSingleBytePushdata = byte(0x4b)

	// OpPushdata1 represent the OP_PUSHDATA1 opcode.
	OpPushdata1 = byte(0x4c)

	// OpPushdata2 represents the OP_PUSHDATA2 opcode.
	OpPushdata2 = byte(0x4d)

	// OpPushdata4 represents the OP_PUSHDATA4 opcode.
	OpPushdata4 = byte(0x4e)

	// OpPushdata1Max is the maximum number of bytes that can be used in the
	// OP_PUSHDATA1 opcode.
	OpPushdata1Max = uint64(255)

	// OpPushdata2Max is the maximum number of bytes that can be used in the
	// OP_PUSHDATA2 opcode.
	OpPushdata2Max = uint64(65535)
)

var (
	defaultEndian = binary.BigEndian
)

// pad returns a []byte with the given length.
func pad(b []byte, l int) []byte {
	if len(b) == l {
		return b
	}

	padding := []byte{}
	c := l - len(b)

	for i := 0; i < c; i++ {
		padding = append(padding, 0)
	}

	return append(b, padding...)
}

// lpad returns a []byte with the given length, padded on the left.
func lpad(b []byte, l int) []byte {
	if len(b) == l {
		return b
	}

	padding := []byte{}
	c := l - len(b)

	for i := 0; i < c; i++ {
		padding = append(padding, 0)
	}

	return append(padding, b...)
}

// write writes the  value to the buffer.
func write(buf io.Writer, v interface{}) error {
	return binary.Write(buf, defaultEndian, v)
}

// read fills the value with the appropriate number of bytes from the buffer.
//
// This is useful for fixed size types such as int, float etc.
func read(buf io.Reader, v interface{}) error {
	return binary.Read(buf, defaultEndian, v)
}

// readLen reads the number of bytes from the buffer to fill the slice of
// []byte.
func readLen(buf io.Reader, b []byte) error {
	_, err := io.ReadFull(buf, b)
	return err
}

// PushDataScript prepares a push data script based on the given size
func PushDataScript(size uint64) []byte {
	if size <= uint64(OpMaxSingleBytePushdata) {
		return []byte{byte(size)} // Single byte push
	} else if size < OpPushdata1Max {
		return []byte{OpPushdata1, byte(size)}
	} else if size < OpPushdata2Max {
		var buf bytes.Buffer
		binary.Write(&buf, defaultEndian, OpPushdata2)
		binary.Write(&buf, defaultEndian, uint16(size))
		return buf.Bytes()
	}

	var buf bytes.Buffer
	binary.Write(&buf, defaultEndian, OpPushdata4)
	binary.Write(&buf, defaultEndian, uint32(size))
	return buf.Bytes()
}

// ParsePushDataScript will parse a push data script and return its size
func ParsePushDataScript(buf io.Reader) (uint64, error) {
	var opCode byte
	err := binary.Read(buf, defaultEndian, &opCode)
	if err != nil {
		return 0, err
	}

	if opCode <= OpMaxSingleBytePushdata {
		return uint64(opCode), nil
	}

	switch opCode {
	case OpPushdata1:
		var size uint8
		err := binary.Read(buf, defaultEndian, &size)
		if err != nil {
			return 0, err
		}
		return uint64(size), nil
	case OpPushdata2:
		var size uint16
		err := binary.Read(buf, defaultEndian, &size)
		if err != nil {
			return 0, err
		}
		return uint64(size), nil
	case OpPushdata4:
		var size uint32
		err := binary.Read(buf, defaultEndian, &size)
		if err != nil {
			return 0, err
		}
		return uint64(size), nil
	default:
		return 0, fmt.Errorf("Invalid push data op code : 0x%02x", opCode)
	}
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
