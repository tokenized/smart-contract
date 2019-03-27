package protocol

const (
	// RejectionCodeOK is not an error
	RejectionCodeOK uint8 = iota

	// RejectionCodeMalformed is returned when a request is malformed.
	RejectionCodeMalformed

	// RejectionCodeInsufficientValue is sent when the issuer has sent
	// insufficient value to fund the transaction.
	RejectionCodeInsufficientValue

	// RejectionCodeIssuerAddress is sent when the message was received from
	// a PKH not associated with the Contract.
	RejectionCodeIssuerAddress

	// RejectionCodeDuplicateAssetCode is sent when the issuer attempted to
	// add a duplicate Asset Code.
	RejectionCodeDuplicateAssetCode

	// RejectionCodeFixedQuantity is sent when the issuer attempted to
	// change increase the quantity of a contract beyond the fixed quantity
	// permitted.
	RejectionCodeFixedQuantity

	// RejectionCodeContractExists is sent when a CO is received, but a
	// Contract already exists at the address.
	RejectionCodeContractExists

	// RejectionCodeContractNotDynamic is sent when a CO is received, but the
	// contract type is not "D" (Dynamic)
	RejectionCodeContractNotDynamic

	// RejectionCodeContractQtyReduction is sent when a  CA tries to reduce
	// the number of assets below the number of assets the contract has.
	RejectionCodeContractQtyReduction

	// RejectionCodeContractAuthFlags is sent when a CA tries to change the
	// authorization flags, but the authorization flags do not permit the
	// change.
	RejectionCodeContractAuthFlags

	// RejectionCodeContractExpiration is sent when a CA tries to modify the
	// Contract, but the auth flags do not permit the update.
	RejectionCodeContractExpiration

	// RejectionCodeContractUpdate is sent when a CA tries to modify the
	// Contract, but the auth flags do not permit the update.
	RejectionCodeContractUpdate

	// RejectionCodeVoteExists is returned when an existing proposal
	// already exists.
	RejectionCodeVoteExists

	// RejectionCodeVoteNotFound is returned when a vote could not be found.
	RejectionCodeVoteNotFound

	// RejectionCodeVoteClosed is returned when a vote is received after the
	// cutoff time.
	RejectionCodeVoteClosed

	// RejectionCodeAssetNotFound is returned when a specified asset is not
	// found.
	RejectionCodeAssetNotFound

	// RejectionCodeInsufficientAssets is returned when there are
	// insufficient assets for the operation.
	RejectionCodeInsufficientAssets

	// RejectionCodeTransferSelf cannot transfer with self
	RejectionCodeTransferSelf

	// RejectionCodeReceiverUnspecified is returned when a required receiver
	// is not specified.
	RejectionCodeReceiverUnspecified

	// RejectionCodeUnknownAddress is returned when an operation was requested
	// by an address that is not known to the contract.
	RejectionCodeUnknownAddress

	// RejectionCodeFrozen is returned when an operation is attempted on a
	// frozen asset holding.
	RejectionCodeFrozen

	// RejectionCodeContractRevision is returned when the incorrect contract
	// revision is sent.
	RejectionCodeContractRevision

	// RejectionCodeAssetRevision is returned when the incorrect asset
	// revision is sent.
	RejectionCodeAssetRevision

	// RejectionCodeMissingNewIssuer is returned when an issuer change was requested, but the new issuer was not provided.
	RejectionCodeContractMissingNewIssuer

	// RejectionCodeContractMissingNewOperator is returned when an operator change was requested, but the new operator was not provided.
	RejectionCodeContractMissingNewOperator

	RejectionCodeInvalidProposal
)
