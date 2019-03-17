package contract

import (
	"github.com/tokenized/smart-contract/internal/platform/state"
	"github.com/tokenized/smart-contract/pkg/protocol"
)

// NewContract defines what we require when creating a Contract record.
type NewContract struct {
	Timestamp protocol.Timestamp `json:"timestamp,omitempty"`

	Issuer   protocol.PublicKeyHash `json:"issuer,omitempty"`
	Operator protocol.PublicKeyHash `json:"operator,omitempty"`

	ContractName               string               `json:"contract_name,omitempty"`
	ContractFileType           uint8                `json:"contract_file_type,omitempty"`
	ContractFile               []byte               `json:"contract_file,omitempty"`
	GoverningLaw               string               `json:"governing_law,omitempty"`
	Jurisdiction               string               `json:"jurisdiction,omitempty"`
	ContractExpiration         protocol.Timestamp   `json:"contract_expiration,omitempty"`
	ContractURI                string               `json:"contract_uri,omitempty"`
	IssuerName                 string               `json:"issuer_name,omitempty"`
	IssuerType                 byte                 `json:"issuer_type,omitempty"`
	IssuerLogoURL              string               `json:"issuer_logo_url,omitempty"`
	ContractOperatorID         string               `json:"contract_operator_id,omitempty"`
	ContractAuthFlags          [16]byte             `json:"contract_auth_flags,omitempty"`
	VotingSystems              []state.VotingSystem `json:"voting_systems,omitempty"`
	RestrictedQtyAssets        uint64               `json:"restricted_qty_assets,omitempty"`
	ReferendumProposal         bool                 `json:"referendum_proposal,omitempty"`
	InitiativeProposal         bool                 `json:"initiative_proposal,omitempty"`
	Registries                 []state.Registry     `json:"registries,omitempty"`
	UnitNumber                 string               `json:"unit_number,omitempty"`
	BuildingNumber             string               `json:"building_number,omitempty"`
	Street                     string               `json:"street,omitempty"`
	SuburbCity                 string               `json:"suburb_city,omitempty"`
	TerritoryStateProvinceCode string               `json:"territory_state_province_code,omitempty"`
	CountryCode                string               `json:"country_code,omitempty"`
	PostalZIPCode              string               `json:"postal_zipcode,omitempty"`
	EmailAddress               string               `json:"email_address,omitempty"`
	PhoneNumber                string               `json:"phone_number,omitempty"`
	KeyRoles                   []state.KeyRole      `json:"key_roles,omitempty"`
	NotableRoles               []state.NotableRole  `json:"notable_roles,omitempty"`
}

// UpdateContract defines what information may be provided to modify an existing
// Contract. All fields are optional so clients can send just the fields they want
// changed. It uses pointer fields so we can differentiate between a field that
// was not provided and a field that was provided as explicitly blank. Normally
// we do not want to use pointers to basic types but we make exceptions around
// marshalling/unmarshalling.
type UpdateContract struct {
	Revision  *uint64             `json:"revision,omitempty"`
	Timestamp *protocol.Timestamp `json:"timestamp,omitempty"`

	Issuer   *protocol.PublicKeyHash `json:"issuer,omitempty"`
	Operator *protocol.PublicKeyHash `json:"operator,omitempty"`

	ContractName               *string               `json:"contract_name,omitempty"`
	ContractFileType           *uint8                `json:"contract_file_type,omitempty"`
	ContractFile               *[]byte               `json:"contract_file,omitempty"`
	GoverningLaw               *string               `json:"governing_law,omitempty"`
	Jurisdiction               *string               `json:"jurisdiction,omitempty"`
	ContractExpiration         *protocol.Timestamp   `json:"contract_expiration,omitempty"`
	ContractURI                *string               `json:"contract_uri,omitempty"`
	IssuerName                 *string               `json:"issuer_name,omitempty"`
	IssuerType                 *byte                 `json:"issuer_type,omitempty"`
	IssuerLogoURL              *string               `json:"issuer_logo_url,omitempty"`
	ContractOperatorID         *string               `json:"contract_operator_id,omitempty"`
	ContractAuthFlags          *[16]byte             `json:"contract_auth_flags,omitempty"`
	VotingSystems              *[]state.VotingSystem `json:"voting_systems,omitempty"`
	RestrictedQtyAssets        *uint64               `json:"restricted_qty_assets,omitempty"`
	ReferendumProposal         *bool                 `json:"referendum_proposal,omitempty"`
	InitiativeProposal         *bool                 `json:"initiative_proposal,omitempty"`
	Registries                 *[]state.Registry     `json:"registries,omitempty"`
	UnitNumber                 *string               `json:"unit_number,omitempty"`
	BuildingNumber             *string               `json:"building_number,omitempty"`
	Street                     *string               `json:"street,omitempty"`
	SuburbCity                 *string               `json:"suburb_city,omitempty"`
	TerritoryStateProvinceCode *string               `json:"territory_state_province_code,omitempty"`
	CountryCode                *string               `json:"country_code,omitempty"`
	PostalZIPCode              *string               `json:"postal_zip_code,omitempty"`
	EmailAddress               *string               `json:"email_address,omitempty"`
	PhoneNumber                *string               `json:"phone_number,omitempty"`
	KeyRoles                   *[]state.KeyRole      `json:"key_roles,omitempty"`
	NotableRoles               *[]state.NotableRole  `json:"notable_roles,omitempty"`
}
