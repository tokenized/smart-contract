package txbuilder

import (
	"bytes"
	"crypto/sha256"
	"errors"

	"github.com/tokenized/smart-contract/pkg/protocol"
	"golang.org/x/crypto/ripemd160"
)

const (
	OP_RETURN = 0x6a

	OP_DUP          = 0x76
	OP_HASH160      = 0xa9
	OP_PUSH_DATA_20 = 0x14
	OP_EQUALVERIFY  = 0x88
	OP_CHECKSIG     = 0xac
)

var (
	NotP2PKHScriptError = errors.New("Not a P2PKH script")
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

func PubKeyHashFromP2PKHSigScript(script []byte) ([]byte, error) {
	buf := bytes.NewBuffer(script)

	// Signature
	size, err := protocol.ParsePushDataScript(buf)
	if err != nil {
		return nil, newError(ErrorCodeNotP2PKHScript, err.Error())
	}

	signature := make([]byte, size)
	_, err = buf.Read(signature)
	if err != nil {
		return nil, newError(ErrorCodeNotP2PKHScript, err.Error())
	}

	// Public Key
	size, err = protocol.ParsePushDataScript(buf)
	if err != nil {
		return nil, newError(ErrorCodeNotP2PKHScript, err.Error())
	}

	publicKey := make([]byte, size)
	_, err = buf.Read(publicKey)
	if err != nil {
		return nil, newError(ErrorCodeNotP2PKHScript, err.Error())
	}

	// Hash public key
	hash256 := sha256.New()
	hash160 := ripemd160.New()

	hash256.Write(publicKey)
	hash160.Write(hash256.Sum(nil))
	return hash160.Sum(nil), nil
}
