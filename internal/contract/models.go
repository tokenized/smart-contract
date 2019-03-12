package contract

import "github.com/tokenized/smart-contract/internal/platform/state"

// NewContract defines what we require when creating a Contract record.
type NewContract struct {
	IssuerAddress   string
	OperatorAddress string

	ContractName               string
	ContractFileType           uint8
	ContractFile               []byte
	GoverningLaw               string
	Jurisdiction               string
	ContractExpiration         uint64
	ContractURI                string
	IssuerName                 string
	IssuerType                 byte
	IssuerLogoURL              string
	ContractOperatorID         string
	ContractAuthFlags          []byte
	VotingSystems              []state.VotingSystem
	RestrictedQtyAssets        uint64
	ReferendumProposal         bool
	InitiativeProposal         bool
	Registries                 []state.Registry
	UnitNumber                 string
	BuildingNumber             string
	Street                     string
	SuburbCity                 string
	TerritoryStateProvinceCode string
	CountryCode                string
	PostalZIPCode              string
	EmailAddress               string
	PhoneNumber                string
	KeyRoles                   []state.KeyRole
	NotableRoles               []state.NotableRole
}

// UpdateContract defines what information may be provided to modify an existing
// Contract. All fields are optional so clients can send just the fields they want
// changed. It uses pointer fields so we can differentiate between a field that
// was not provided and a field that was provided as explicitly blank. Normally
// we do not want to use pointers to basic types but we make exceptions around
// marshalling/unmarshalling.
type UpdateContract struct {
	IssuerAddress   *string
	OperatorAddress *string

	ContractName               *string
	ContractFileType           *uint8
	ContractFile               *[]byte
	GoverningLaw               *string
	Jurisdiction               *string
	ContractExpiration         *uint64
	ContractURI                *string
	IssuerName                 *string
	IssuerType                 *byte
	IssuerLogoURL              *string
	ContractOperatorID         *string
	ContractAuthFlags          *[]byte
	VotingSystems              *[]state.VotingSystem
	RestrictedQtyAssets        *uint64
	ReferendumProposal         *bool
	InitiativeProposal         *bool
	Registries                 *[]state.Registry
	UnitNumber                 *string
	BuildingNumber             *string
	Street                     *string
	SuburbCity                 *string
	TerritoryStateProvinceCode *string
	CountryCode                *string
	PostalZIPCode              *string
	EmailAddress               *string
	PhoneNumber                *string
	KeyRoles                   *[]state.KeyRole
	NotableRoles               *[]state.NotableRole
}
