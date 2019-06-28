package contract

import (
	"github.com/tokenized/specification/dist/golang/protocol"
)

// NewContract defines what we require when creating a Contract record.
type NewContract struct {
	Timestamp protocol.Timestamp `json:"timestamp,omitempty"`

	AdministrationPKH protocol.PublicKeyHash `json:"administration_pkh,omitempty"`
	OperatorPKH       protocol.PublicKeyHash `json:"operator_pkh,omitempty"`
	MasterPKH         protocol.PublicKeyHash `json:"master_pkh,omitempty"`

	ContractName           string                  `json:"contract_name,omitempty"`
	BodyOfAgreementType    uint8                   `json:"body_of_agreement_type,omitempty"`
	BodyOfAgreement        []byte                  `json:"body_of_agreement,omitempty"`
	ContractType           string                  `json:"contract_type,omitempty"`
	SupportingDocs         []protocol.Document     `json:"supporting_docs,omitempty"`
	GoverningLaw           string                  `json:"governing_law,omitempty"`
	Jurisdiction           string                  `json:"jurisdiction,omitempty"`
	ContractExpiration     protocol.Timestamp      `json:"contract_expiration,omitempty"`
	ContractURI            string                  `json:"contract_uri,omitempty"`
	Issuer                 protocol.Entity         `json:"issuer,omitempty"`
	IssuerLogoURL          string                  `json:"issuer_logo_url,omitempty"`
	ContractOperator       protocol.Entity         `json:"contract_operator,omitempty"`
	ContractAuthFlags      []byte                  `json:"contract_auth_flags,omitempty"`
	ContractFee            uint64                  `json:"contract_fee,omitempty"`
	VotingSystems          []protocol.VotingSystem `json:"voting_systems,omitempty"`
	RestrictedQtyAssets    uint64                  `json:"restricted_qty_assets,omitempty"`
	AdministrationProposal bool                    `json:"administration_proposal,omitempty"`
	HolderProposal         bool                    `json:"holder_proposal,omitempty"`
	Oracle                 []protocol.Oracle       `json:"oracle,omitempty"`
}

// UpdateContract defines what information may be provided to modify an existing
// Contract. All fields are optional so clients can send just the fields they want
// changed. It uses pointer fields so we can differentiate between a field that
// was not provided and a field that was provided as explicitly blank. Normally
// we do not want to use pointers to basic types but we make exceptions around
// marshalling/unmarshalling.
type UpdateContract struct {
	Revision  *uint32             `json:"revision,omitempty"`
	Timestamp *protocol.Timestamp `json:"timestamp,omitempty"`

	AdministrationPKH *protocol.PublicKeyHash `json:"administration_pkh,omitempty"`
	OperatorPKH       *protocol.PublicKeyHash `json:"operator_pkh,omitempty"`

	ContractName           *string                  `json:"contract_name,omitempty"`
	BodyOfAgreementType    *uint8                   `json:"body_of_agreement_type,omitempty"`
	BodyOfAgreement        *[]byte                  `json:"body_of_agreement,omitempty"`
	ContractType           *string                  `json:"contract_type,omitempty"`
	SupportingDocs         *[]protocol.Document     `json:"supporting_docs,omitempty"`
	GoverningLaw           *string                  `json:"governing_law,omitempty"`
	Jurisdiction           *string                  `json:"jurisdiction,omitempty"`
	ContractExpiration     *protocol.Timestamp      `json:"contract_expiration,omitempty"`
	ContractURI            *string                  `json:"contract_uri,omitempty"`
	Issuer                 *protocol.Entity         `json:"issuer,omitempty"`
	IssuerLogoURL          *string                  `json:"issuer_logo_url,omitempty"`
	ContractOperator       *protocol.Entity         `json:"contract_operator,omitempty"`
	ContractAuthFlags      *[]byte                  `json:"contract_auth_flags,omitempty"`
	ContractFee            *uint64                  `json:"contract_fee,omitempty"`
	VotingSystems          *[]protocol.VotingSystem `json:"voting_systems,omitempty"`
	RestrictedQtyAssets    *uint64                  `json:"restricted_qty_assets,omitempty"`
	AdministrationProposal *bool                    `json:"administration_proposal,omitempty"`
	HolderProposal         *bool                    `json:"holder_proposal,omitempty"`
	Oracles                *[]protocol.Oracle       `json:"oracle,omitempty"`

	FreezePeriod *protocol.Timestamp `json:"freeze_period,omitempty"`
}