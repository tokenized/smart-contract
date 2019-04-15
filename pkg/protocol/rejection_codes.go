package protocol

const (
	// Success - No failure. This code should not be used in a reject message.
	RejectSuccess = 0

	// MsgMalformed - The OP_RETURN message is malformed. It doesn't pass data validation or something about it isn't according to protocol.
	RejectMsgMalformed = 1

	// TxMalformed - The Bitcoin tx is malformed. Incorrect inputs/outputs or something similar.
	RejectTxMalformed = 2

	// Timeout - A dependency, other contract/service, has failed to complete before the smart contract's timeout.
	RejectTimeout = 3

	// ContractExists - The contract already exists and can't be recreated.
	RejectContractExists = 10

	// ContractNotDynamic - Sent when a CO is received, but the contract type is not "D" (Dynamic)
	RejectContractNotDynamic = 11

	// ContractAssetQtyReduction - Sent when a CA tries to reduce the number of assets below the number of assets the contract has.
	RejectContractAssetQtyReduction = 12

	// ContractFixedQuantity - Sent when the issuer attempted to increase the quantity of assets in a contract beyond the fixed quantity permitted.
	RejectContractFixedQuantity = 13

	// ContractAuthFlags - The contract auth flags don't permit the action requested.
	RejectContractAuthFlags = 14

	// ContractExpired - The contract is expired so can no longer accept requests.
	RejectContractExpired = 15

	// ContractFrozen - The contract is frozen and the request is not permitted while frozen.
	RejectContractFrozen = 16

	// ContractRevision - The revision in a contract amendment is incorrect.
	RejectContractRevision = 17

	// ContractNotPermitted - Action not permitted by contract.
	RejectContractNotPermitted = 18

	// AssetCodeExists - The asset code specified already exists and can't be reused.
	RejectAssetCodeExists = 20

	// AssetNotFound - The asset code is not found.
	RejectAssetNotFound = 21

	// AssetAuthFlags - The asset auth flags don't permit the action requested.
	RejectAssetAuthFlags = 22

	// AssetFrozen - The asset is frozen and the request is not permitted while frozen.
	RejectAssetFrozen = 23

	// AssetRevision - The revision in an asset amendment is incorrect.
	RejectAssetRevision = 24

	// AssetNotPermitted - Action not permitted by asset.
	RejectAssetNotPermitted = 25

	// TransferSelf - Transfers with the sender and receiver addresses the same are not permitted.
	RejectTransferSelf = 30

	// TransferExpired - The transfer has expired.
	RejectTransferExpired = 31

	// HoldingsFrozen - Holdings are frozen, so the request can't be completed.
	RejectHoldingsFrozen = 32

	// HolderProposalProhibited - Holders are not permitted to make proposals.
	RejectHolderProposalProhibited = 40

	// ProposalConflicts - The proposal conflicts with an unapplied proposal.
	RejectProposalConflicts = 41

	// VoteNotFound - The vote ID referenced is not found.
	RejectVoteNotFound = 42

	// VoteClosed - The vote has closed and ballots are no longer permitted.
	RejectVoteClosed = 43

	// BallotAlreadyCounted - The ballot has already been counted for this address.
	RejectBallotAlreadyCounted = 44

	// VoteSystemNotPermitted - The voting system isn't permitted for this request.
	RejectVoteSystemNotPermitted = 45

	// InsufficientTxFeeFunding - Insufficient bitcoin quantities for response transaction fees.
	RejectInsufficientTxFeeFunding = 60

	// InsufficientValue - Insufficient bitcoin quantity in inputs to fund request.
	RejectInsufficientValue = 61

	// InsufficientQuantity - Insufficient token holdings to for request.
	RejectInsufficientQuantity = 62

	// NotIssuer - The requestor is not the issuer and is required to be for this request.
	RejectNotIssuer = 70

	// NotOperator - The requestor is not the operator and is required to be for this request.
	RejectNotOperator = 71

	// UnauthorizedAddress - The address specified is not permitted for this request.
	RejectUnauthorizedAddress = 72

	// InvalidSignature - The signature provided is not valid. This is for signatures included within OP_RETURN data. Not bitcoin transaction signature scripts.
	RejectInvalidSignature = 80
)
