package vote

import (
	"reflect"

	"github.com/tokenized/smart-contract/pkg/protocol"
)

func ValidateInitiative(msg *protocol.Initiative) bool {

	// An Initiative can never change auth flags.
	if msg.VoteType == 'F' {
		return false
	}

	// There must be at least 1 option.
	if len(msg.VoteOptions) == 0 {
		return false
	}

	// Ensure a have a valid vote type was received
	vt := msg.VoteType

	// Invalid vote type
	if vt != 'C' && vt != 'A' && vt != 'P' {
		return false
	}

	// C, A, F must always be binary options (A, B)
	if vt != 'P' && !reflect.DeepEqual(msg.VoteOptions, []byte{'A', 'B'}) {
		return false
	}

	// Pass
	return true
}
