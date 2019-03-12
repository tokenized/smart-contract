package vote

import "github.com/tokenized/smart-contract/pkg/protocol"

// import (
// "reflect"

// "github.com/tokenized/smart-contract/pkg/protocol"
// )

// TODO protocol.Initiative.VoteSystem is now an index into the contract's voting systems
func ValidateInitiative(msg *protocol.Initiative) bool {
	return true
	// // An Initiative can never change auth flags.
	// if msg.VoteType == 'F' {
	// return false
	// }

	// // There must be at least 1 option.
	// if len(msg.VoteOptions) == 0 {
	// return false
	// }

	// // Ensure a have a valid vote type was received
	// vt := msg.VoteType

	// // Invalid vote type
	// if vt != 'C' && vt != 'A' && vt != 'P' {
	// return false
	// }

	// // C, A, F must always be binary options (A, B)
	// if vt != 'P' && !reflect.DeepEqual(msg.VoteOptions, []byte{'A', 'B'}) {
	// return false
	// }

	// // Pass
	// return true
}

// TODO Implement ValidateReferendum
func ValidateReferendum(msg *protocol.Referendum) bool {
	return true
	// // There must be at least 1 option.
	// if msg.VoteOptions.Len == 0 {
	// return false
	// }

	// // Ensure a have a valid vote type was received
	// vt := msg.VoteType

	// // Invalid vote type
	// if vt != 'C' && vt != 'A' && vt != 'P' && vt != 'F' {
	// return false
	// }

	// // C, A, F must always be binary options (A, B)
	// if vt != 'P' && !reflect.DeepEqual(msg.VoteOptions, []byte{'A', 'B'}) {
	// return false
	// }

	// // Pass
	// return true
}
