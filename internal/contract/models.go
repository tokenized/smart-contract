package contract

import (
	"github.com/tokenized/smart-contract/pkg/bitcoin"
	"github.com/tokenized/specification/dist/golang/actions"
	"github.com/tokenized/specification/dist/golang/protocol"
)

// NewContract defines what we require when creating a Contract record.
type NewContract struct {
	Timestamp protocol.Timestamp `json:"Timestamp,omitempty"`

	AdministrationAddress bitcoin.RawAddress `json:"AdministrationAddress,omitempty"`
	OperatorAddress       bitcoin.RawAddress `json:"OperatorAddress,omitempty"`
	MasterAddress         bitcoin.RawAddress `json:"MasterAddress,omitempty"`

	ContractName              string                       `json:"ContractName,omitempty"`
	BodyOfAgreementType       uint32                       `json:"BodyOfAgreementType,omitempty"`
	BodyOfAgreement           []byte                       `json:"BodyOfAgreement,omitempty"`
	ContractType              string                       `json:"ContractType,omitempty"`
	SupportingDocs            []*actions.DocumentField     `json:"SupportingDocs,omitempty"`
	GoverningLaw              string                       `json:"GoverningLaw,omitempty"`
	Jurisdiction              string                       `json:"Jurisdiction,omitempty"`
	ContractExpiration        protocol.Timestamp           `json:"ContractExpiration,omitempty"`
	ContractURI               string                       `json:"ContractURI,omitempty"`
	Issuer                    *actions.EntityField         `json:"Issuer,omitempty"`
	IssuerLogoURL             string                       `json:"IssuerLogoURL,omitempty"`
	ContractOperator          *actions.EntityField         `json:"ContractOperator,omitempty"`
	AdminOracle               *actions.OracleField         `json:"AdminOracle,omitempty"`
	AdminOracleSignature      []byte                       `json:"AdminOracleSignature,omitempty"`
	AdminOracleSigBlockHeight uint32                       `json:"AdminOracleSigBlockHeight,omitempty"`
	ContractPermissions       []byte                       `json:"ContractPermissions,omitempty"`
	ContractFee               uint64                       `json:"ContractFee,omitempty"`
	VotingSystems             []*actions.VotingSystemField `json:"VotingSystems,omitempty"`
	RestrictedQtyAssets       uint64                       `json:"RestrictedQtyAssets,omitempty"`
	AdministrationProposal    bool                         `json:"AdministrationProposal,omitempty"`
	HolderProposal            bool                         `json:"HolderProposal,omitempty"`
	Oracle                    []*actions.OracleField       `json:"Oracle,omitempty"`
}

// UpdateContract defines what information may be provided to modify an existing
// Contract. All fields are optional so clients can send just the fields they want
// changed. It uses pointer fields so we can differentiate between a field that
// was not provided and a field that was provided as explicitly blank. Normally
// we do not want to use pointers to basic types but we make exceptions around
// marshalling/unmarshalling.
type UpdateContract struct {
	Revision  *uint32             `json:"Revision,omitempty"`
	Timestamp *protocol.Timestamp `json:"Timestamp,omitempty"`

	AdministrationAddress *bitcoin.RawAddress `json:"AdministrationAddress,omitempty"`
	OperatorAddress       *bitcoin.RawAddress `json:"OperatorAddress,omitempty"`

	AdminMemberAsset *protocol.AssetCode `json:"AdminMemberAsset,omitempty"`

	ContractName              *string                       `json:"ContractName,omitempty"`
	BodyOfAgreementType       *uint32                       `json:"BodyOfAgreementType,omitempty"`
	BodyOfAgreement           *[]byte                       `json:"BodyOfAgreement,omitempty"`
	ContractType              *string                       `json:"ContractType,omitempty"`
	SupportingDocs            *[]*actions.DocumentField     `json:"SupportingDocs,omitempty"`
	GoverningLaw              *string                       `json:"GoverningLaw,omitempty"`
	Jurisdiction              *string                       `json:"Jurisdiction,omitempty"`
	ContractExpiration        *protocol.Timestamp           `json:"ContractExpiration,omitempty"`
	ContractURI               *string                       `json:"ContractURI,omitempty"`
	Issuer                    *actions.EntityField          `json:"Issuer,omitempty"`
	IssuerLogoURL             *string                       `json:"IssuerLogoURL,omitempty"`
	ContractOperator          *actions.EntityField          `json:"ContractOperator,omitempty"`
	AdminOracle               *actions.OracleField          `json:"AdminOracle,omitempty"`
	AdminOracleSignature      *[]byte                       `json:"AdminOracleSignature,omitempty"`
	AdminOracleSigBlockHeight *uint32                       `json:"AdminOracleSigBlockHeight,omitempty"`
	ContractPermissions       *[]byte                       `json:"ContractPermissions,omitempty"`
	ContractFee               *uint64                       `json:"ContractFee,omitempty"`
	VotingSystems             *[]*actions.VotingSystemField `json:"VotingSystems,omitempty"`
	RestrictedQtyAssets       *uint64                       `json:"RestrictedQtyAssets,omitempty"`
	AdministrationProposal    *bool                         `json:"AdministrationProposal,omitempty"`
	HolderProposal            *bool                         `json:"HolderProposal,omitempty"`
	Oracles                   *[]*actions.OracleField       `json:"Oracles,omitempty"`

	FreezePeriod *protocol.Timestamp `json:"FreezePeriod,omitempty"`
}
