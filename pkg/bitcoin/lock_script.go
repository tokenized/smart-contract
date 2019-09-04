package bitcoin

// AddressFromLockingScript returns the address associated with the specified locking script.
func AddressFromLockingScript(lockingScript []byte, net Network) (Address, error) {
	ra, err := RawAddressFromLockingScript(lockingScript)
	if err != nil {
		return Address{}, err
	}
	return NewAddressFromRawAddress(ra, net), nil
}

// RawAddressFromLockingScript returns the script template associated with the specified locking
//   script.
func RawAddressFromLockingScript(lockingScript []byte) (RawAddress, error) {
	var result RawAddress
	script := lockingScript
	switch script[0] {
	case OP_DUP: // PKH or RPH
		if len(script) < 25 {
			return result, ErrUnknownScriptTemplate
		}
		script = script[1:]
		switch script[0] {
		case OP_HASH160: // PKH
			if len(script) != 24 {
				return result, ErrUnknownScriptTemplate
			}
			script = script[1:]

			if script[0] != OP_PUSH_DATA_20 {
				return result, ErrUnknownScriptTemplate
			}
			script = script[1:]

			pkh := script[:ScriptHashLength]
			script = script[ScriptHashLength:]

			if script[0] != OP_EQUALVERIFY {
				return result, ErrUnknownScriptTemplate
			}
			script = script[1:]

			if script[0] != OP_CHECKSIG {
				return result, ErrUnknownScriptTemplate
			}
			script = script[1:]

			err := result.SetPKH(pkh)
			return result, err

		case OP_3: // RPH
			if len(script) != 33 {
				return result, ErrUnknownScriptTemplate
			}
			script = script[1:]

			if script[0] != OP_SPLIT {
				return result, ErrUnknownScriptTemplate
			}
			script = script[1:]

			if script[0] != OP_NIP {
				return result, ErrUnknownScriptTemplate
			}
			script = script[1:]

			if script[0] != OP_1 {
				return result, ErrUnknownScriptTemplate
			}
			script = script[1:]

			if script[0] != OP_SPLIT {
				return result, ErrUnknownScriptTemplate
			}
			script = script[1:]

			if script[0] != OP_SWAP {
				return result, ErrUnknownScriptTemplate
			}
			script = script[1:]

			if script[0] != OP_SPLIT {
				return result, ErrUnknownScriptTemplate
			}
			script = script[1:]

			if script[0] != OP_DROP {
				return result, ErrUnknownScriptTemplate
			}
			script = script[1:]

			if script[0] != OP_HASH160 {
				return result, ErrUnknownScriptTemplate
			}
			script = script[1:]

			if script[0] != OP_PUSH_DATA_20 {
				return result, ErrUnknownScriptTemplate
			}
			script = script[1:]

			rph := script[:ScriptHashLength]
			script = script[ScriptHashLength:]

			if script[0] != OP_EQUALVERIFY {
				return result, ErrUnknownScriptTemplate
			}
			script = script[1:]

			if script[0] != OP_SWAP {
				return result, ErrUnknownScriptTemplate
			}
			script = script[1:]

			if script[0] != OP_CHECKSIG {
				return result, ErrUnknownScriptTemplate
			}
			script = script[1:]

			err := result.SetRPH(rph)
			return result, err

		}
	case OP_HASH160: // P2SH
		if len(script) != 23 {
			return result, ErrUnknownScriptTemplate
		}
		script = script[1:]

		if script[0] != OP_PUSH_DATA_20 {
			return result, ErrUnknownScriptTemplate
		}
		script = script[1:]

		sh := script[:ScriptHashLength]
		script = script[ScriptHashLength:]

		if script[0] != OP_EQUAL {
			return result, ErrUnknownScriptTemplate
		}
		script = script[1:]

		err := result.SetSH(sh)
		return result, err

	}

	return result, ErrUnknownScriptTemplate
}

func (ra RawAddress) LockingScript() ([]byte, error) {
	switch ra.scriptType {
	case ScriptTypePKH:
		result := make([]byte, 0, 25)

		result = append(result, OP_DUP)
		result = append(result, OP_HASH160)

		// Push public key hash
		result = append(result, OP_PUSH_DATA_20) // Single byte push op code of 20 bytes
		result = append(result, ra.data...)

		result = append(result, OP_EQUALVERIFY)
		result = append(result, OP_CHECKSIG)
		return result, nil

	case ScriptTypeSH:
		result := make([]byte, 0, 23)

		result = append(result, OP_HASH160)

		// Push script hash
		result = append(result, OP_PUSH_DATA_20) // Single byte push op code of 20 bytes
		result = append(result, ra.data...)

		result = append(result, OP_EQUAL)
		return result, nil

	case ScriptTypeRPH:
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
		result = append(result, ra.data...)

		result = append(result, OP_EQUALVERIFY)
		result = append(result, OP_SWAP)
		result = append(result, OP_CHECKSIG)
		return result, nil
	}

	return nil, ErrUnknownScriptTemplate
}
