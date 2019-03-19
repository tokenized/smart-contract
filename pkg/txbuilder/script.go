package txbuilder

import (
	"errors"
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
		return nil, NotP2PKHScriptError
	}

	offset := 0
	if script[offset] != OP_DUP {
		return nil, NotP2PKHScriptError
	}
	offset++

	if script[offset] != OP_HASH160 {
		return nil, NotP2PKHScriptError
	}
	offset++

	if script[offset] != OP_PUSH_DATA_20 { // Single byte push op for 20 bytes
		return nil, NotP2PKHScriptError
	}
	offset += 21 // Skip push op code and data

	if script[offset] != OP_EQUALVERIFY {
		return nil, NotP2PKHScriptError
	}
	offset++

	if script[offset] != OP_CHECKSIG {
		return nil, NotP2PKHScriptError
	}
	offset++

	return script[3:23], nil
}
