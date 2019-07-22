package bitcoin

// AddressFromLockingScript returns the address associated with the specified locking script.
func AddressFromLockingScript(lockingScript []byte) (Address, error) {
	script := lockingScript
	switch script[0] {
	case OP_DUP: // PKH or RPH
		if len(script) < 25 {
			return nil, ErrUnknownScriptTemplate
		}
		script = script[1:]
		switch script[0] {
		case OP_HASH160: // PKH
			if len(script) != 24 {
				return nil, ErrUnknownScriptTemplate
			}
			script = script[1:]

			if script[0] != OP_PUSH_DATA_20 {
				return nil, ErrUnknownScriptTemplate
			}
			script = script[1:]

			pkh := script[:hashLength]
			script = script[hashLength:]

			if script[0] != OP_EQUALVERIFY {
				return nil, ErrUnknownScriptTemplate
			}
			script = script[1:]

			if script[0] != OP_CHECKSIG {
				return nil, ErrUnknownScriptTemplate
			}
			script = script[1:]

			return AddressPKHFromBytes(pkh)

		case OP_3: // RPH
			if len(script) != 33 {
				return nil, ErrUnknownScriptTemplate
			}
			script = script[1:]

			if script[0] != OP_SPLIT {
				return nil, ErrUnknownScriptTemplate
			}
			script = script[1:]

			if script[0] != OP_NIP {
				return nil, ErrUnknownScriptTemplate
			}
			script = script[1:]

			if script[0] != OP_1 {
				return nil, ErrUnknownScriptTemplate
			}
			script = script[1:]

			if script[0] != OP_SPLIT {
				return nil, ErrUnknownScriptTemplate
			}
			script = script[1:]

			if script[0] != OP_SWAP {
				return nil, ErrUnknownScriptTemplate
			}
			script = script[1:]

			if script[0] != OP_SPLIT {
				return nil, ErrUnknownScriptTemplate
			}
			script = script[1:]

			if script[0] != OP_DROP {
				return nil, ErrUnknownScriptTemplate
			}
			script = script[1:]

			if script[0] != OP_HASH160 {
				return nil, ErrUnknownScriptTemplate
			}
			script = script[1:]

			if script[0] != OP_PUSH_DATA_20 {
				return nil, ErrUnknownScriptTemplate
			}
			script = script[1:]

			rph := script[:hashLength]
			script = script[hashLength:]

			if script[0] != OP_EQUALVERIFY {
				return nil, ErrUnknownScriptTemplate
			}
			script = script[1:]

			if script[0] != OP_SWAP {
				return nil, ErrUnknownScriptTemplate
			}
			script = script[1:]

			if script[0] != OP_CHECKSIG {
				return nil, ErrUnknownScriptTemplate
			}
			script = script[1:]

			return AddressRPHFromBytes(rph)

		}
	case OP_HASH160: // P2SH
		if len(script) != 23 {
			return nil, ErrUnknownScriptTemplate
		}
		script = script[1:]

		if script[0] != OP_PUSH_DATA_20 {
			return nil, ErrUnknownScriptTemplate
		}
		script = script[1:]

		sh := script[:hashLength]
		script = script[hashLength:]

		if script[0] != OP_EQUAL {
			return nil, ErrUnknownScriptTemplate
		}
		script = script[1:]

		return AddressSHFromBytes(sh)

	case OP_FALSE: // MultiPKH
		// 35 = 1 min number push + 4 op codes outside of pkh if statements + 30 per pkh
		if len(script) < 35 {
			return nil, ErrUnknownScriptTemplate
		}
		script = script[1:]

		if script[0] != OP_TOALTSTACK {
			return nil, ErrUnknownScriptTemplate
		}
		script = script[1:]

		// Loop through pkhs
		pkhs := make([][]byte, 0, len(script)/30)
		for script[0] == OP_IF {
			script = script[1:]

			if script[0] != OP_DUP {
				return nil, ErrUnknownScriptTemplate
			}
			script = script[1:]

			if script[0] != OP_HASH160 {
				return nil, ErrUnknownScriptTemplate
			}
			script = script[1:]

			if script[0] != OP_PUSH_DATA_20 {
				return nil, ErrUnknownScriptTemplate
			}
			script = script[1:]

			pkhs = append(pkhs, script[:hashLength])
			script = script[hashLength:]

			if script[0] != OP_EQUALVERIFY {
				return nil, ErrUnknownScriptTemplate
			}
			script = script[1:]

			if script[0] != OP_CHECKSIGVERIFY {
				return nil, ErrUnknownScriptTemplate
			}
			script = script[1:]

			if script[0] != OP_FROMALTSTACK {
				return nil, ErrUnknownScriptTemplate
			}
			script = script[1:]

			if script[0] != OP_1ADD {
				return nil, ErrUnknownScriptTemplate
			}
			script = script[1:]

			if script[0] != OP_TOALTSTACK {
				return nil, ErrUnknownScriptTemplate
			}
			script = script[1:]

			if script[0] != OP_ENDIF {
				return nil, ErrUnknownScriptTemplate
			}
			script = script[1:]

			if len(script) == 0 {
				return nil, ErrUnknownScriptTemplate
			}
		}

		if len(script) < 3 {
			return nil, ErrUnknownScriptTemplate
		}

		// Parse required signature count
		required, length, err := ParsePushNumberScript(script)
		if err != nil {
			return nil, ErrUnknownScriptTemplate
		}
		script = script[length:]

		if len(script) != 2 {
			return nil, ErrUnknownScriptTemplate
		}

		if script[0] != OP_FROMALTSTACK {
			return nil, ErrUnknownScriptTemplate
		}
		script = script[1:]

		if script[0] != OP_GREATERTHANOREQUAL {
			return nil, ErrUnknownScriptTemplate
		}
		script = script[1:]

		return AddressMultiPKHFromBytes(uint16(required), pkhs)

	}

	return nil, ErrUnknownScriptTemplate
}

func (a *AddressPKH) LockingScript() []byte {
	result := make([]byte, 0, 25)

	result = append(result, OP_DUP)
	result = append(result, OP_HASH160)

	// Push public key hash
	result = append(result, OP_PUSH_DATA_20) // Single byte push op code of 20 bytes
	result = append(result, a.pkh...)

	result = append(result, OP_EQUALVERIFY)
	result = append(result, OP_CHECKSIG)
	return result
}

func (a *AddressSH) LockingScript() []byte {
	result := make([]byte, 0, 23)

	result = append(result, OP_HASH160)

	// Push script hash
	result = append(result, OP_PUSH_DATA_20) // Single byte push op code of 20 bytes
	result = append(result, a.sh...)

	result = append(result, OP_EQUAL)
	return result
}

func (a *AddressMultiPKH) LockingScript() []byte {
	// 14 = 10 max number push + 4 op codes outside of pkh if statements
	// 30 = 10 op codes + 20 byte pkh per pkh
	result := make([]byte, 0, 14+(len(a.pkhs)*30))

	result = append(result, OP_FALSE)
	result = append(result, OP_TOALTSTACK)

	for _, pkh := range a.pkhs {
		// Check if this pkh has a signature
		result = append(result, OP_IF)

		// Check signature against this pkh
		result = append(result, OP_DUP)
		result = append(result, OP_HASH160)

		// Push public key hash
		result = append(result, OP_PUSH_DATA_20) // Single byte push op code of 20 bytes
		result = append(result, pkh...)

		result = append(result, OP_EQUALVERIFY)
		result = append(result, OP_CHECKSIGVERIFY)

		// Add 1 to count of valid signatures
		result = append(result, OP_FROMALTSTACK)
		result = append(result, OP_1ADD)
		result = append(result, OP_TOALTSTACK)

		result = append(result, OP_ENDIF)
	}

	// Check required signature count
	result = append(result, PushNumberScript(int64(a.required))...)
	result = append(result, OP_FROMALTSTACK)
	result = append(result, OP_GREATERTHANOREQUAL)

	return result
}

func (a *AddressRPH) LockingScript() []byte {
	result := make([]byte, 0, 34)

	result = append(result, OP_DUP)
	result = append(result, OP_3)
	result = append(result, OP_SPLIT)
	result = append(result, OP_NIP)
	result = append(result, OP_1)
	result = append(result, OP_SPLIT)
	result = append(result, OP_SWAP)
	result = append(result, OP_SPLIT)
	result = append(result, OP_DROP)
	result = append(result, OP_HASH160)

	// Push r hash
	result = append(result, OP_PUSH_DATA_20) // Single byte push op code of 20 bytes
	result = append(result, a.rph...)

	result = append(result, OP_EQUALVERIFY)
	result = append(result, OP_SWAP)
	result = append(result, OP_CHECKSIG)
	return result
}
