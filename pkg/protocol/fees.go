package protocol

import (
	"encoding/json"
	"fmt"

	"github.com/tokenized/smart-contract/pkg/txbuilder"
	"github.com/tokenized/smart-contract/pkg/wire"

	"github.com/pkg/errors"
)

// EstimatedResponse calculates information about the contract's response to a request.
//   fees is the sum of all contract related fees including base contract fee, proposal fee, and others.
// Returns
//   estimated size of response tx in bytes.
//   estimated funding needed, not including contract/proposal fees.
//   error if there were any
func EstimatedResponse(requestTx *wire.MsgTx, inputIndex int, dustLimit, fees uint64) (int, uint64, error) {
	// Find Tokenized OP_RETURN
	var err error
	var opReturn OpReturnMessage
	found := false
	for _, output := range requestTx.TxOut {
		opReturn, err = Deserialize(output.PkScript)
		if err == nil {
			found = true
			break
		}
	}
	if !found {
		return 0, 0, errors.New("Tokenized OP_RETURN not found")
	}

	// Build sample response and calculate values based on expected response inputs and outputs.
	var response OpReturnMessage
	size := txbuilder.BaseTxSize
	value := uint64(0)

	switch request := opReturn.(type) {
	case *ContractOffer:
		contractFormation := ContractFormation{}
		response = &contractFormation

		// 1 input from contract
		size += wire.VarIntSerializeSize(uint64(1)) + txbuilder.EstimatedInputSize

		// P2PKH dust output to contract, contract fee, and op return output
		if fees > 0 {
			size += wire.VarIntSerializeSize(uint64(3)) + txbuilder.P2PKHOutputSize
			value += fees
		} else {
			size += wire.VarIntSerializeSize(uint64(2)) + txbuilder.P2PKHOutputSize
		}
		value += dustLimit

	case *AssetDefinition:
		assetDefinition := AssetDefinition{}
		response = &assetDefinition

		// 1 input from contract
		size += wire.VarIntSerializeSize(uint64(1)) + txbuilder.EstimatedInputSize

		// P2PKH dust output to contract, and op return output
		if fees > 0 {
			size += wire.VarIntSerializeSize(uint64(3)) + txbuilder.P2PKHOutputSize
			value += fees
		} else {
			size += wire.VarIntSerializeSize(uint64(2)) + txbuilder.P2PKHOutputSize
		}
		value += dustLimit

	case *Proposal:
		if inputIndex == 0 {
			// First input funds vote (initiation) message
			vote := Vote{}
			response = &vote

			// 1 input from contract
			size += wire.VarIntSerializeSize(uint64(1)) + txbuilder.EstimatedInputSize

			// P2PKH dust output to contract, and op return output
			if fees > 0 {
				size += wire.VarIntSerializeSize(uint64(3)) + txbuilder.P2PKHOutputSize
				value += fees
			} else {
				size += wire.VarIntSerializeSize(uint64(2)) + txbuilder.P2PKHOutputSize
			}
			value += dustLimit
		} else {
			// Second input funds vote result message
			voteResult := Result{}
			response = &voteResult

			voteResult.OptionTally = make([]uint64, len(request.VoteOptions))
			voteResult.Result = " "
			for len(voteResult.Result) < len(request.VoteOptions) {
				voteResult.Result += " "
			}

			// 1 input from contract
			size += wire.VarIntSerializeSize(uint64(1)) + txbuilder.EstimatedInputSize

			// P2PKH dust output to contract, and op return output
			if fees > 0 {
				size += wire.VarIntSerializeSize(uint64(3)) + txbuilder.P2PKHOutputSize
				value += fees
			} else {
				size += wire.VarIntSerializeSize(uint64(2)) + txbuilder.P2PKHOutputSize
			}
			value += dustLimit
		}

	case *BallotCast:
		counted := BallotCounted{}
		response = &counted

		// 1 input from contract
		size += wire.VarIntSerializeSize(uint64(1)) + txbuilder.EstimatedInputSize

		// P2PKH dust output to contract, and op return output
		if fees > 0 {
			size += wire.VarIntSerializeSize(uint64(3)) + txbuilder.P2PKHOutputSize
			value += fees
		} else {
			size += wire.VarIntSerializeSize(uint64(2)) + txbuilder.P2PKHOutputSize
		}
		value += dustLimit

	default:
		return 0, 0, errors.New("Unsupported request type")
	}

	if err = convert(opReturn, response); err != nil {
		return 0, 0, errors.Wrap(err, "Failed to convert request to response")
	}

	script, err := Serialize(response)
	if err != nil {
		return 0, 0, errors.Wrap(err, "Failed to serialize response")
	}

	// OP_RETURN output size
	size += txbuilder.OutputBaseSize + wire.VarIntSerializeSize(uint64(len(script))) + len(script)

	return size, value, nil
}

// Convert assigns all available compatible values with matching member names from one object to
//   another.
// The dst object needs to be a pointer so that it can be written to.
// Members of these objects that are "specialized", like a struct containing only a string, need
//   to have json.Marshaler and json.UnMarshaler interfaces implemented.
func convert(src interface{}, dst interface{}) error {
	// Marshal source object to json.
	var data []byte
	var err error
	data, err = json.Marshal(src)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("Failed json marshal"))
	}

	// Unmarshal json back into destination object.
	err = json.Unmarshal(data, dst)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("Failed json unmarshal"))
	}

	return nil
}
