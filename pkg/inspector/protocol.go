package inspector

import (
	"github.com/tokenized/pkg/bitcoin"
	"github.com/tokenized/specification/dist/golang/actions"
)

type Balance struct {
	Qty    uint64
	Frozen uint64
}

// GetProtocolTimestamp returns the timestamp of the action. It is only valid for "outgoing" actions.
func GetProtocolTimestamp(itx *Transaction, m actions.Action) *uint64 {
	switch msg := m.(type) {
	case *actions.AssetCreation:
		return &msg.Timestamp

	case *actions.ContractFormation:
		return &msg.Timestamp

	// Enforcement
	case *actions.Freeze:
		return &msg.Timestamp
	case *actions.Thaw:
		return &msg.Timestamp
	case *actions.Confiscation:
		return &msg.Timestamp
	case *actions.Reconciliation:
		return &msg.Timestamp

	// Governance
	case *actions.Vote:
		return &msg.Timestamp
	case *actions.BallotCounted:
		return &msg.Timestamp
	case *actions.Result:
		return &msg.Timestamp

	case *actions.Rejection:
		return &msg.Timestamp

	case *actions.Settlement:
		return &msg.Timestamp
	}

	return nil
}

func GetProtocolContractAddresses(itx *Transaction, m actions.Action) []bitcoin.RawAddress {
	if !itx.IsTokenized() {
		return nil
	}

	// Settlements may contain multiple contracts
	settlement, isSettlement := m.(*actions.Settlement)
	if isSettlement {
		result := []bitcoin.RawAddress{}
		for _, asset := range settlement.Assets {
			if int(asset.ContractIndex) < len(itx.Inputs) {
				result = append(result, itx.Inputs[asset.ContractIndex].Address)
			}
		}
		return result
	}

	// Some specific actions have the contract address as an input
	isOutgoing, ok := outgoingMessageTypes[m.Code()]
	if ok && isOutgoing {
		return []bitcoin.RawAddress{itx.Inputs[0].Address}
	}

	// Default behavior is contract as first output
	isIncoming, ok := incomingMessageTypes[m.Code()]
	if ok && isIncoming {
		return []bitcoin.RawAddress{itx.Outputs[0].Address}
	}

	return nil
}

// func GetProtocolContractPKHs(itx *Transaction, m actions.Action) [][]byte {
//
// 	addresses := make([][]byte, 1)
//
// 	// Settlements may contain a second contract, although optional
// 	if m.Code() == actions.CodeSettlement {
// 		addressPKH, ok := itx.Inputs[0].Address.(*bitcoin.RawAddressPKH)
// 		if ok {
// 			addresses = append(addresses, addressPKH.PKH())
// 		}
//
// 		if len(itx.Inputs) > 1 && !itx.Inputs[1].Address.Equal(itx.Inputs[0].Address) {
// 			addressPKH, ok := itx.Inputs[1].Address.(*bitcoin.RawAddressPKH)
// 			if ok {
// 				addresses = append(addresses, addressPKH.PKH())
// 			}
// 		}
//
// 		return addresses
// 	}
//
// 	// Some specific actions have the contract address as an input
// 	isOutgoing, ok := outgoingMessageTypes[m.Code()]
// 	if ok && isOutgoing {
// 		addressPKH, ok := itx.Inputs[0].Address.(*bitcoin.RawAddressPKH)
// 		if ok {
// 			addresses = append(addresses, addressPKH.PKH())
// 		}
// 		return addresses
// 	}
//
// 	// Default behavior is contract as first output
// 	addressPKH, ok := itx.Outputs[0].Address.(*bitcoin.RawAddressPKH)
// 	if ok {
// 		addresses = append(addresses, addressPKH.PKH())
// 	}
//
// 	// TODO Transfers/Settlements can contain multiple contracts in inputs and outputs
//
// 	return addresses
// }

func GetProtocolAddresses(itx *Transaction, m actions.Action, contractAddress bitcoin.RawAddress) []bitcoin.RawAddress {

	addresses := []bitcoin.RawAddress{}

	// input messages have contract address at output[0], and the input
	// address at input[0].
	//
	// output messages have contract address at input[0], and the receiver
	// at output[0]
	//
	// exceptions to this are
	//
	// - CO, which has an optional operator address
	// - Swap (T4)  output[0] and output[1] are contract addresses
	// - Settlement (T4) - input[0] and input[1] are contract addresses
	//
	if m.Code() == actions.CodeContractOffer {
		addresses = append(addresses, itx.Inputs[0].Address)

		// is there an operator address?
		if len(itx.Inputs) > 1 && !itx.Inputs[1].Address.Equal(itx.Inputs[0].Address) {
			addresses = append(addresses, itx.Inputs[1].Address)
		}

		return addresses
	}

	// if m.Code() == protocol.CodeSwap {
	// addresses = append(addresses, itx.Inputs[0].Address)
	// addresses = append(addresses, itx.Inputs[1].Address)

	// return addresses
	// }

	if m.Code() == actions.CodeSettlement {
		addresses = append(addresses, itx.Outputs[0].Address)
		addresses = append(addresses, itx.Outputs[1].Address)

		return addresses
	}

	// if this is an input message?
	switch m.Code() {
	case actions.CodeContractOffer,
		actions.CodeContractAmendment,
		actions.CodeAssetDefinition,
		actions.CodeAssetModification,
		actions.CodeTransfer,
		actions.CodeProposal,
		actions.CodeBallotCast,
		actions.CodeOrder:

		if m.Code() == actions.CodeTransfer {
			addresses = append(addresses, itx.Outputs[1].Address)
			addresses = append(addresses, itx.Outputs[2].Address)

		} else {
			addresses = append(addresses, itx.Inputs[0].Address)
		}

		return addresses
	}

	// output messages.
	//
	// output[0] can be change to the contract, so the recipient would be
	// output[1] in that case.
	if m.Code() == actions.CodeResult {
		addresses = append(addresses, itx.Outputs[0].Address)
	} else if m.Code() == actions.CodeConfiscation {
		addresses = append(addresses, itx.Outputs[0].Address)
		addresses = append(addresses, itx.Outputs[1].Address)
	} else {
		if itx.Outputs[0].Address.Equal(contractAddress) {
			// change to contract, so receiver is 2nd output
			addresses = append(addresses, itx.Outputs[1].Address)
		} else {
			// no change, so receiver is 1st output
			addresses = append(addresses, itx.Outputs[0].Address)
		}
	}

	return addresses
}
