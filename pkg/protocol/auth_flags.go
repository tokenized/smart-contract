package protocol

const (
	// ContractIssuerUpdate indicates that the Issuer can amend/update the
	// Contract (Except for the Authorization Flags).
	ContractIssuerUpdate = 1 << 0

	// ContractOwnerAmendments indicates that a Token Owner Vote is required
	// for Contract Amendments/updates (excluding Contract Authorization
	// Flags).
	ContractOwnerAmendments = 1 << 1

	// ContractAuthFlagsIssuer indicates that the Issuer can amend the
	// Contract's Authorization Flags.
	ContractAuthFlagsIssuer = 1 << 2

	// ContractAuthFlagReferendum indicates that Contract Authorization Flag
	// amendments permitted with Referendum (Token Owner Vote. Voting Type
	// Specified in Contract Formation).
	ContractAuthFlagReferendum = 1 << 3

	// ContractUserInitiatives allows Users to propose Initiatives to amend
	// the Contract.
	ContractUserInitiatives = 1 << 4

	// ContractQuantityUpdate permits Issuer to Create new Assets.
	ContractQuantityUpdate = 1 << 5

	// ContractReferendum indicates that a Referendum required for new Asset
	// Creations.
	ContractReferendum = 1 << 6

	// ContractAssetConfiscate indicates that the Issuer can confiscate
	// existing Assets.
	ContractAssetConfiscate = 1 << 7

	// ContractAssetFreezeThaw indicates that the Issuer can freeze/thaw the
	// Asset state (The Smart Contract will ignore user requests).
	ContractAssetFreezeThaw = 1 << 8

	// ContractAssetWhitelist indicates that Asset trading is restricted to
	// a Whitelist.
	ContractAssetWhitelist = 1 << 9

	// ContractBindingInitiatives indicates that Initiatives are binding.
	ContractBindingInitiatives = 1 << 10

	// ContractExpirationUpdate indicates that the Contract Expiration Date
	// Can be Amended.
	ContractExpirationUpdate = 1 << 11

	// AssetIssuerModification permits the Issuer to modify the Asset
	AssetIssuerModification = 1 << 0

	// AssetVoteRequired means the Token Owner Vote required for Asset
	// Amendments/updates (excluding Asset Authorization Flags)
	AssetVoteRequired = 1 << 1

	// AssetIsserAmendFlags indicates that the Issuer can amend the Asset's
	// Authorization Flags.
	AssetIsserAmendFlags = 1 << 2

	// AssetAuthFlagAmendment permits Asset Authorization Flag amendments
	// permitted with Token Owner Vote.
	AssetAuthFlagAmendment = 1 << 3

	// AssetUserInitiative permits Users to propose initiatives to modify
	// the Asset.
	AssetUserInitiative = 1 << 4

	// AssetIssuerMintBurn permits the Issuer to mint/burn tokens.
	AssetIssuerMintBurn = 1 << 5

	// AssetTokenOwnerVote Token Owner Vote required for new token issuances.
	AssetTokenOwnerVote = 1 << 6

	// AssetIssuerConfiscate permits the Issuer to confiscate existing
	// tokens.
	AssetIssuerConfiscate = 1 << 7

	// AssetIsserFreeThaw permits the Issuer to freeze/thaw the token state
	// (The Smart Contract will ignore user requests).
	AssetIsserFreeThaw = 1 << 8

	// AssetUserTransfer allows the user to transfer existing tokens.
	AssetUserTransfer = 1 << 9
)

// IsAuthorized returns true if the given flags match the existing state,
// false otherwise.
func IsAuthorized(state uint16, flags uint16) bool {
	return state&flags == flags
}
