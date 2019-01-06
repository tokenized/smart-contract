package protocol

const (
	// FlagAmendment indicates that the Issuer or Operator can amend the
	// Contract or Asset. This flag does (Except for the Authorization
	// Flags).
	//
	// This flag can be applied to the AuthorizationFlags of a Contract or
	// an Asset.
	FlagAmendment = 1 << 0

	// FlagVoteRequired indicates that a Token Owner Vote is required for
	// amendments (excluding Authorization Flags).
	//
	// This flag can be applied to the AuthorizationFlags of a Contract or
	// an Asset.
	FlagVoteRequired = 1 << 1

	// FlagAmendAuthFlags indicates that the Issuer or Operator can amend
	// the Authorization Flags.
	//
	// This flag can be applied to the AuthorizationFlags of a Contract or
	// an Asset.
	FlagAmendAuthFlags = 1 << 2

	// FlagReferendumAuthFlags indicates that Authorization Flag amendments
	// are permitted with a successful Referendum.
	//
	// This flag can be applied to the AuthorizationFlags of a Contract or
	// an Asset.
	FlagReferendumAuthFlags = 1 << 3

	// FlagInitiatives allows Users to propose Initiatives.
	//
	// This flag can be applied to the AuthorizationFlags of a Contract or
	// an Asset.
	FlagInitiatives = 1 << 4

	// FlagContractEnforcement indicates if the Contract permits Issuer or
	// Operator to issue Enforcement Orders,
	//
	// This flag is only used with Contract Authorization Flags.
	FlagContractEnforcement = 1 << 5

	// FlagContractWhitelist indicates that transfers are restricted to a
	// Whitelist.
	//
	// This is not yet implemented.
	//
	// This flag is only used with Contract Authorization Flags.
	FlagContractWhitelist = 1 << 6

	// FlagAssetAmendment indicates that the Issuer or Operator can amend
	// the Asset. This flag does (Except for the Authorization Flags).
	//
	// This flag can be applied to the AuthorizationFlags of an Asset.
	FlagAssetAmendment = 1 << 0

	// FlagAssetVoteRequired indicates that a Token Owner Vote is required for
	// amendments (excluding Authorization Flags).
	//
	// This flag can be applied to the AuthorizationFlags of a Contract or
	// an Asset.
	FlagAssetVoteRequired = 1 << 1

	// FlagAssetAmendAuthFlags indicates that the Issuer or Operator can
	// amend the Authorization Flags.
	//
	// This flag can be applied to the AuthorizationFlags of a Contract or
	// an Asset.
	FlagAssetAmendAuthFlags = 1 << 2

	// FlagAssetReferendumAuthFlags indicates that Authorization Flag
	// amendments are permitted with a successful Referendum.
	//
	// This flag can be applied to the AuthorizationFlags of an Asset.
	FlagAssetReferendumAuthFlags = 1 << 3

	// FlagAssetInitiatives indicates that Token Owners can propose
	// Initiatives to modify an Asset.
	//
	// This flag is only used with Asset Authorization Flags.
	FlagAssetInitiatives = 1 << 4

	// FlagAssetInitiativesModify permits the Issuer to modify the Asset.
	//
	// This flag is only used with Asset Authorization Flags.
	FlagAssetInitiativesModify = 1 << 5

	// FlagAssetTransfersPermitted indicates that transfers are restricted
	// to a Whitelist.
	//
	// This is not yet implemented.
	//
	// This flag is only used with Asset Authorization Flags.
	FlagAssetTransfersPermitted = 1 << 6
)

// IsAuthorized returns true if the given flags match the existing state,
// false otherwise.
func IsAuthorized(state uint16, flags uint16) bool {
	return state&flags == flags
}
