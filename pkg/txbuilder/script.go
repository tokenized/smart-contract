package txbuilder

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"golang.org/x/crypto/ripemd160"
)

const (
	OP_RETURN = 0x6a

	OP_DUP          = 0x76
	OP_HASH160      = 0xa9
	OP_PUSH_DATA_20 = 0x14
	OP_EQUALVERIFY  = 0x88
	OP_CHECKSIG     = 0xac

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
	NotP2PKHScriptError = errors.New("Not a P2PKH script")
)

var (
	endian = binary.LittleEndian
)

// IsOpReturn returns true if the script is an OP_RETURN output script.
func IsOpReturnScript(script []byte) bool {
	return len(script) > 0 && script[0] == OP_RETURN
}

// IsP2PKH returns true if the script is a P2PKH (Pay 2 Public Key Hash) output script.
func IsP2PKHScript(script []byte) bool {
	if len(script) != 25 {
		return false
	}

	offset := 0
	if script[offset] != OP_DUP {
		return false
	}
	offset++

	if script[offset] != OP_HASH160 {
		return false
	}
	offset++

	if script[offset] != OP_PUSH_DATA_20 { // Single byte push op for 20 bytes
		return false
	}
	offset += 21 // Skip push op code and data

	if script[offset] != OP_EQUALVERIFY {
		return false
	}
	offset++

	if script[offset] != OP_CHECKSIG {
		return false
	}
	offset++

	return true
}

func P2PKHScriptForPKH(pkh []byte) []byte {
	result := make([]byte, 0, 25)

	result = append(result, OP_DUP)
	result = append(result, OP_HASH160)
	result = append(result, OP_PUSH_DATA_20) // Single byte push op code of 20 bytes

	result = append(result, pkh...) // Address

	result = append(result, OP_EQUALVERIFY)
	result = append(result, OP_CHECKSIG)
	return result
}

func PubKeyHashFromP2PKH(script []byte) ([]byte, error) {
	if len(script) != 25 {
		return nil, newError(ErrorCodeNotP2PKHScript, "Wrong length")
	}

	offset := 0
	if script[offset] != OP_DUP {
		return nil, newError(ErrorCodeNotP2PKHScript, "Missing OP_DUP")
	}
	offset++

	if script[offset] != OP_HASH160 {
		return nil, newError(ErrorCodeNotP2PKHScript, "Missing OP_HASH160")
	}
	offset++

	if script[offset] != OP_PUSH_DATA_20 { // Single byte push op for 20 bytes
		return nil, newError(ErrorCodeNotP2PKHScript, "Missing OP_PUSH_DATA_20")
	}
	offset += 21 // Skip push op code and data

	if script[offset] != OP_EQUALVERIFY {
		return nil, newError(ErrorCodeNotP2PKHScript, "Missing OP_EQUALVERIFY")
	}
	offset++

	if script[offset] != OP_CHECKSIG {
		return nil, newError(ErrorCodeNotP2PKHScript, "Missing OP_CHECKSIG")
	}
	offset++

	return script[3:23], nil
}

func PubKeyFromP2PKHSigScript(script []byte) ([]byte, error) {
	buf := bytes.NewBuffer(script)

	// Signature
	size, err := ParsePushDataScript(buf)
	if err != nil {
		return nil, newError(ErrorCodeNotP2PKHScript, err.Error())
	}

	signature := make([]byte, size)
	_, err = buf.Read(signature)
	if err != nil {
		return nil, newError(ErrorCodeNotP2PKHScript, err.Error())
	}

	// Public Key
	size, err = ParsePushDataScript(buf)
	if err != nil {
		return nil, newError(ErrorCodeNotP2PKHScript, err.Error())
	}

	publicKey := make([]byte, size)
	_, err = buf.Read(publicKey)
	if err != nil {
		return nil, newError(ErrorCodeNotP2PKHScript, err.Error())
	}

	return publicKey, nil
}

func PubKeyHashFromP2PKHSigScript(script []byte) ([]byte, error) {
	publicKey, err := PubKeyFromP2PKHSigScript(script)
	if err != nil {
		return nil, err
	}

	// Hash public key
	hash160 := ripemd160.New()
	hash256 := sha256.Sum256(publicKey)
	hash160.Write(hash256[:])
	return hash160.Sum(nil), nil
}

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
