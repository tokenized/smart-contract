package bitcoin

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

const (
	OP_FALSE              = 0x00
	OP_1                  = 0x51
	OP_3                  = 0x53
	OP_RETURN             = 0x6a
	OP_DUP                = 0x76
	OP_HASH160            = 0xa9
	OP_PUSH_DATA_20       = 0x14
	OP_EQUAL              = 0x87
	OP_EQUALVERIFY        = 0x88
	OP_GREATERTHANOREQUAL = 0xa2
	OP_CHECKSIG           = 0xac
	OP_CHECKSIGVERIFY     = 0xad
	OP_IF                 = 0x63
	OP_ENDIF              = 0x68
	OP_TOALTSTACK         = 0x6b
	OP_FROMALTSTACK       = 0x6c
	OP_1ADD               = 0x8b
	OP_SPLIT              = 0x7f
	OP_NIP                = 0x77
	OP_SWAP               = 0x7c
	OP_DROP               = 0x75

	// OP_MAX_SINGLE_BYTE_PUSH_DATA represents the max length for a single byte push
	OP_MAX_SINGLE_BYTE_PUSH_DATA = byte(0x4b)

	// OP_PUSH_DATA_1 represent the OP_PUSHDATA1 opcode.
	OP_PUSH_DATA_1 = byte(0x4c)

	// OP_PUSH_DATA_2 represents the OP_PUSHDATA2 opcode.
	OP_PUSH_DATA_2 = byte(0x4d)

	// OP_PUSH_DATA_4 represents the OP_PUSHDATA4 opcode.
	OP_PUSH_DATA_4 = byte(0x4e)

	// OP_PUSH_DATA_1_MAX is the maximum number of bytes that can be used in the
	// OP_PUSHDATA1 opcode.
	OP_PUSH_DATA_1_MAX = uint64(255)

	// OP_PUSH_DATA_2_MAX is the maximum number of bytes that can be used in the
	// OP_PUSHDATA2 opcode.
	OP_PUSH_DATA_2_MAX = uint64(65535)
)

var (
	endian = binary.LittleEndian
)

// PushDataScript prepares a push data script based on the given size
func PushDataScript(size uint64) []byte {
	if size <= uint64(OP_MAX_SINGLE_BYTE_PUSH_DATA) {
		return []byte{byte(size)} // Single byte push
	} else if size < OP_PUSH_DATA_1_MAX {
		return []byte{OP_PUSH_DATA_1, byte(size)}
	} else if size < OP_PUSH_DATA_2_MAX {
		var buf bytes.Buffer
		binary.Write(&buf, endian, OP_PUSH_DATA_2)
		binary.Write(&buf, endian, uint16(size))
		return buf.Bytes()
	}

	var buf bytes.Buffer
	binary.Write(&buf, endian, OP_PUSH_DATA_4)
	binary.Write(&buf, endian, uint32(size))
	return buf.Bytes()
}

// ParsePushDataScript will parse a push data script and return its size
func ParsePushDataScript(buf io.Reader) (uint64, error) {
	var opCode byte
	err := binary.Read(buf, endian, &opCode)
	if err != nil {
		return 0, err
	}

	if opCode <= OP_MAX_SINGLE_BYTE_PUSH_DATA {
		return uint64(opCode), nil
	}

	switch opCode {
	case OP_PUSH_DATA_1:
		var size uint8
		err := binary.Read(buf, endian, &size)
		if err != nil {
			return 0, err
		}
		return uint64(size), nil
	case OP_PUSH_DATA_2:
		var size uint16
		err := binary.Read(buf, endian, &size)
		if err != nil {
			return 0, err
		}
		return uint64(size), nil
	case OP_PUSH_DATA_4:
		var size uint32
		err := binary.Read(buf, endian, &size)
		if err != nil {
			return 0, err
		}
		return uint64(size), nil
	default:
		return 0, fmt.Errorf("Invalid push data op code : 0x%02x", opCode)
	}
}

// PushNumberScript returns a section of script that will push the specified number onto the stack.
// Example encodings:
//       127 -> [0x7f]
//      -127 -> [0xff]
//       128 -> [0x80 0x00]
//      -128 -> [0x80 0x80]
//       129 -> [0x81 0x00]
//      -129 -> [0x81 0x80]
//       256 -> [0x00 0x01]
//      -256 -> [0x00 0x81]
//     32767 -> [0xff 0x7f]
//    -32767 -> [0xff 0xff]
//     32768 -> [0x00 0x80 0x00]
//    -32768 -> [0x00 0x80 0x80]
func PushNumberScript(n int64) []byte {
	// OP_FALSE, OP_0
	if n == 0 {
		return []byte{0x00}
	}

	// OP_1NEGATE
	if n == -1 {
		return []byte{0x4f}
	}

	// Single byte number push op codes
	if n > 0 && n <= 16 {
		return []byte{0x50 + byte(n)}
	}

	// Take the absolute value and keep track of whether it was originally
	// negative.
	isNegative := n < 0
	if isNegative {
		n = -n
	}

	// Encode to little endian.  The maximum number of encoded bytes is 9
	// (8 bytes for max int64 plus a potential byte for sign extension).
	result := make([]byte, 0, 10)
	for n > 0 {
		result = append(result, byte(n&0xff))
		n >>= 8
	}

	// When the most significant byte already has the high bit set, an
	// additional high byte is required to indicate whether the number is
	// negative or positive.  The additional byte is removed when converting
	// back to an integral and its high bit is used to denote the sign.
	//
	// Otherwise, when the most significant byte does not already have the
	// high bit set, use it to indicate the value is negative, if needed.
	if result[len(result)-1]&0x80 != 0 {
		extraByte := byte(0x00)
		if isNegative {
			extraByte = 0x80
		}
		result = append(result, extraByte)

	} else if isNegative {
		result[len(result)-1] |= 0x80
	}

	// Push this value onto the stack (single byte push op)
	return append([]byte{byte(len(result))}, result...)
}

// ParsePushNumberScript reads a number out of script and returns the value, the bytes of script it
//   used, and an error if one occured.
func ParsePushNumberScript(b []byte) (int64, int, error) {
	if len(b) == 0 {
		return 0, 0, errors.New("Script empty")
	}

	// Zero
	if b[0] == OP_FALSE {
		return 0, 1, nil
	}

	// Negative one
	if b[0] == 0x4f {
		return -1, 1, nil
	}

	// 1 - 16 (single byte number op codes)
	if b[0] >= 0x51 && b[0] <= 0x60 {
		return int64(b[0] - 0x50), 1, nil
	}

	if b[0] > 10 {
		return 0, 0, errors.New("Invalid push op for number")
	}

	length := int(b[0])
	b = b[1:]

	// Decode from little endian.
	var result int64
	for i, val := range b {
		result |= int64(val) << uint8(8*i)
	}

	// When the most significant byte of the input bytes has the sign bit
	// set, the result is negative.  So, remove the sign bit from the result
	// and make it negative.
	if b[len(b)-1]&0x80 != 0 {
		// The maximum length of v has already been determined to be 4
		// above, so uint8 is enough to cover the max possible shift
		// value of 24.
		result &= ^(int64(0x80) << uint8(8*(len(b)-1)))
		result = -result
	}

	return result, length, nil
}

func PubKeyFromP2PKHSigScript(script []byte) ([]byte, error) {
	buf := bytes.NewBuffer(script)

	// Signature
	size, err := ParsePushDataScript(buf)
	if err != nil {
		return nil, err
	}

	signature := make([]byte, size)
	_, err = buf.Read(signature)
	if err != nil {
		return nil, err
	}

	// Public Key
	size, err = ParsePushDataScript(buf)
	if err != nil {
		return nil, err
	}

	publicKey := make([]byte, size)
	_, err = buf.Read(publicKey)
	if err != nil {
		return nil, err
	}

	return publicKey, nil
}

func PubKeyHashFromP2PKHSigScript(script []byte) ([]byte, error) {
	publicKey, err := PubKeyFromP2PKHSigScript(script)
	if err != nil {
		return nil, err
	}

	// Hash public key
	return Hash160(publicKey), nil
}
