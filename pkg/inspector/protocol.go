package inspector

import (
	"github.com/tokenized/smart-contract/pkg/bitcoin"
	"github.com/tokenized/specification/dist/golang/actions"
)

type Balance struct {
	Qty    uint64
	Frozen uint64
}

func GetProtocolQuantity(itx *Transaction, m actions.Action, address bitcoin.RawAddress) Balance {

	return Balance{
		Qty:    0,
		Frozen: 0,
	}
	/*
		b := Balance{}

		switch m.Code() {
		case protocol.CodeAssetCreation:
			o := m.(*protocol.AssetCreation)
			b.Qty = o.Qty

		case protocol.CodeSettlement:
			o := m.(*protocol.Settlement)

			// which token balance do we want? is this address for party1 or
			// party2?
			if address.String() == itx.Outputs[0].Address.String() {
				b.Qty = o.Party1TokenQty
			} else {
				b.Qty = o.Party2TokenQty
			}

		case protocol.CodeFreeze:
			o := m.(*protocol.Freeze)

			// this makes the assumption that the amount frozen is the amount
			// that is held.
			//
			// See https://bitbucket.org/tokenized/contract/issues/55/review-freeze
			b.Qty = o.Qty
			b.Frozen = o.Qty

		case protocol.CodeThaw:
			o := m.(*protocol.Thaw)

			b.Qty = o.Qty
			b.Frozen = 0

		case protocol.CodeConfiscation:
			o := m.(*protocol.Confiscation)

			if address.String() == itx.Outputs[0].Address.String() {
				b.Qty = o.TargetsQty
			} else {
				b.Qty = o.DepositsQty
			}

		case protocol.CodeReconciliation:
			o := m.(*protocol.Reconciliation)
			b.Qty = o.TargetAddressQty

		}

		return b
	*/
}

func GetProtocolContractAddresses(itx *Transaction, m actions.Action) []bitcoin.RawAddress {

	addresses := []bitcoin.RawAddress{}

	// Settlements may contain a second contract, although optional
	if m.Code() == actions.CodeSettlement {
		addresses = append(addresses, itx.Inputs[0].Address)

		if len(itx.Inputs) > 1 && !itx.Inputs[1].Address.Equal(itx.Inputs[0].Address) {
			addresses = append(addresses, itx.Inputs[1].Address)
		}

		return addresses
	}

	// Some specific actions have the contract address as an input
	isOutgoing, ok := outgoingMessageTypes[m.Code()]
	if ok && isOutgoing {
		addresses = append(addresses, itx.Inputs[0].Address)
		return addresses
	}

	// Default behavior is contract as first output
	addresses = append(addresses, itx.Outputs[0].Address)

	// TODO Transfers/Settlements can contain multiple contracts in inputs and outputs

	return addresses
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
