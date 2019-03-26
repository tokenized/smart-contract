package protocol

var (
	RejectionCodes = map[uint8][]byte{
		RejectionCodeInsufficientValue:          []byte("Fee Not Paid"),
		RejectionCodeIssuerAddress:              []byte("Issuer Address"),
		RejectionCodeDuplicateAssetCode:         []byte("Duplicate Asset Code"),
		RejectionCodeFixedQuantity:              []byte("Fixed Quantity"),
		RejectionCodeContractExists:             []byte("Contract Exists"),
		RejectionCodeContractNotDynamic:         []byte("Contract Not Dynamic"),
		RejectionCodeContractQtyReduction:       []byte("Contract Qty Reduction"),
		RejectionCodeContractAuthFlags:          []byte("Contract Auth Flags"),
		RejectionCodeContractExpiration:         []byte("Contract Expiration"),
		RejectionCodeContractUpdate:             []byte("Contract Update"),
		RejectionCodeVoteExists:                 []byte("Vote Exists"),
		RejectionCodeVoteNotFound:               []byte("Vote Not Found"),
		RejectionCodeVoteClosed:                 []byte("Vote Closed"),
		RejectionCodeAssetNotFound:              []byte("Asset Not Found"),
		RejectionCodeInsufficientAssets:         []byte("Insufficient Assets"),
		RejectionCodeTransferSelf:               []byte("Transfer Self"),
		RejectionCodeReceiverUnspecified:        []byte("Receiver Unspecified"),
		RejectionCodeUnknownAddress:             []byte("Unknown Address"),
		RejectionCodeFrozen:                     []byte("Frozen"),
		RejectionCodeContractRevision:           []byte("Contract Revision Incorrect"),
		RejectionCodeAssetRevision:              []byte("Asset Revision Incorrect"),
		RejectionCodeContractMissingNewIssuer:   []byte("Contract Issuer Change Missing"),
		RejectionCodeContractMissingNewOperator: []byte("Contract Operator Change Missing"),
		RejectionCodeContractMalformedAmendment: []byte("Contract Malformed Update"),
		RejectionCodeInvalidInitiative:          []byte("Invalid Initiative"),
		RejectionCodeMalFormedTransfer:          []byte("MalFormed Transfer"),
		RejectionCodeMalFormedSettle:            []byte("MalFormed Settle"),
	}
)
