package protocol

var (
	RejectionCodes = map[uint8][]byte{
		1:  []byte("Fee Not Paid"),
		2:  []byte("Issuer Address"),
		3:  []byte("Duplicate Asset ID"),
		4:  []byte("Fixed Quantity"),
		5:  []byte("Contract Exists"),
		6:  []byte("Contract Not Dynamic"),
		7:  []byte("Contract Qty Reduction"),
		8:  []byte("Contract Auth Flags"),
		9:  []byte("Contract Expiration"),
		10: []byte("Contract Update"),
		11: []byte("Vote Exists"),
		12: []byte("Vote Not Found"),
		13: []byte("Vote Closed"),
		14: []byte("Asset Not Found"),
		15: []byte("Insufficient Assets"),
		16: []byte("Transfer Self"),
		17: []byte("Receiver Unspecified"),
		18: []byte("Unknown Address"),
		19: []byte("Frozen"),
		20: []byte("Contract Revision Incorrect"),
		21: []byte("Asset Revision Incorrect"),
		22: []byte("Contract Issuer Change Missing"),
		23: []byte("Contract Malformed Update"),
		24: []byte("Contract Malformed Update"),
		25: []byte("Invalid Initiative"),
	}
)
