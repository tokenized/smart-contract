package inspector

import (
	"github.com/btcsuite/btcutil"
	"github.com/tokenized/smart-contract/pkg/protocol"
)

type Balance struct {
	Qty    uint64
	Frozen uint64
}

// func GetProtocolQuantity(itx *Transaction, m protocol.OpReturnMessage, address btcutil.Address) Balance {

// b := Balance{}

// switch m.Type() {
// case protocol.CodeAssetCreation:
// o := m.(*protocol.AssetCreation)
// b.Qty = o.TokenQty

// case protocol.CodeSettlement:
// o := m.(*protocol.Settlement)

// // which token balance do we want? is this address for party1 or
// // party2?
// if address.String() == itx.Outputs[0].Address.String() {
// b.Qty = o.Party1TokenQty
// } else {
// b.Qty = o.Party2TokenQty
// }

// case protocol.CodeFreeze:
// o := m.(*protocol.Freeze)

// // this makes the assumption that the amount frozen is the amount
// // that is held.
// //
// // See https://bitbucket.org/tokenized/contract/issues/55/review-freeze
// b.Qty = o.Qty
// b.Frozen = o.Qty

// case protocol.CodeThaw:
// o := m.(*protocol.Thaw)

// b.Qty = o.Qty
// b.Frozen = 0

// case protocol.CodeConfiscation:
// o := m.(*protocol.Confiscation)

// if address.String() == itx.Outputs[0].Address.String() {
// b.Qty = o.TargetsQty
// } else {
// b.Qty = o.DepositsQty
// }

// case protocol.CodeReconciliation:
// o := m.(*protocol.Reconciliation)
// b.Qty = o.TargetAddressQty

// }

// return b
// }

func GetProtocolContractAddresses(itx *Transaction, m protocol.OpReturnMessage) []btcutil.Address {

	addresses := []btcutil.Address{}

	// Swaps contain a second contract
	// if m.Type() == protocol.CodeSwap {
	// addresses = append(addresses, itx.Outputs[0].Address)
	// addresses = append(addresses, itx.Outputs[1].Address)
	// return addresses
	// }

	// Settlements may contain a second contract, although optional
	if m.Type() == protocol.CodeSettlement {
		addresses = append(addresses, itx.Inputs[0].Address)

		if len(itx.Inputs) > 1 && itx.Inputs[1].Address.String() != itx.Inputs[0].Address.String() {
			addresses = append(addresses, itx.Inputs[1].Address)
		}

		return addresses
	}

	// Some specific actions have the contract address as an input
	switch m.Type() {
	case protocol.CodeBallotCounted,
		protocol.CodeFreeze,
		protocol.CodeThaw,
		protocol.CodeConfiscation,
		protocol.CodeReconciliation,
		protocol.CodeRejection:

		addresses = append(addresses, itx.Inputs[0].Address)
		return addresses
	}

	// Default behavior is contract as first output
	addresses = append(addresses, itx.Outputs[0].Address)
	return addresses
}

func GetProtocolAddresses(itx *Transaction, m protocol.OpReturnMessage, contractAddress btcutil.Address) []btcutil.Address {

	addresses := []btcutil.Address{}

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
	if m.Type() == protocol.CodeContractOffer {
		addresses = append(addresses, itx.Inputs[0].Address)

		// is there an operator address?
		if len(itx.Inputs) > 1 && itx.Inputs[1].Address.String() != itx.Inputs[0].Address.String() {

			addresses = append(addresses, itx.Inputs[1].Address)
		}

		return addresses
	}

	// if m.Type() == protocol.CodeSwap {
	// addresses = append(addresses, itx.Inputs[0].Address)
	// addresses = append(addresses, itx.Inputs[1].Address)

	// return addresses
	// }

	if m.Type() == protocol.CodeSettlement {
		addresses = append(addresses, itx.Outputs[0].Address)
		addresses = append(addresses, itx.Outputs[1].Address)

		return addresses
	}

	// if this is an input message?
	switch m.Type() {
	case protocol.CodeContractOffer,
		protocol.CodeContractAmendment,
		protocol.CodeAssetDefinition,
		protocol.CodeAssetModification,
		protocol.CodeTransfer,
		protocol.CodeInitiative,
		protocol.CodeReferendum,
		protocol.CodeBallotCast,
		protocol.CodeOrder:

		if m.Type() == protocol.CodeTransfer {
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
	if m.Type() == protocol.CodeResult {
		addresses = append(addresses, itx.Outputs[0].Address)
	} else if m.Type() == protocol.CodeConfiscation {
		addresses = append(addresses, itx.Outputs[0].Address)
		addresses = append(addresses, itx.Outputs[1].Address)
	} else {
		if itx.Outputs[0].Address.String() == contractAddress.String() {
			// change to contract, so receiver is 2nd output
			addresses = append(addresses, itx.Outputs[1].Address)
		} else {
			// no change, so receiver is 1st output
			addresses = append(addresses, itx.Outputs[0].Address)
		}
	}

	return addresses
}
