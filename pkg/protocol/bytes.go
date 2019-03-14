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
