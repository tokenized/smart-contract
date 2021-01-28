package contract

import (
	"github.com/tokenized/pkg/bitcoin"
	"github.com/tokenized/specification/dist/golang/actions"
	"github.com/tokenized/specification/dist/golang/protocol"
)

// NewContract defines what we require when creating a Contract record.
type NewContract struct {
	Timestamp protocol.Timestamp `json:"Timestamp,omitempty"`

	AdminAddress    bitcoin.RawAddress `json:"AdminAddress,omitempty"`
	OperatorAddress bitcoin.RawAddress `json:"OperatorAddress,omitempty"`
	MasterAddress   bitcoin.RawAddress `json:"MasterAddress,omitempty"`

	ContractType uint32 `json:"ContractType,omitempty"`
	ContractFee  uint64 `json:"ContractFee,omitempty"`

	ContractExpiration  protocol.Timestamp `json:"ContractExpiration,omitempty"`
	RestrictedQtyAssets uint64             `json:"RestrictedQtyAssets,omitempty"`

	VotingSystems          []*actions.VotingSystemField `json:"VotingSystems,omitempty"`
	AdministrationProposal bool                         `json:"AdministrationProposal,omitempty"`
	HolderProposal         bool                         `json:"HolderProposal,omitempty"`

	Oracles []*actions.OracleField `json:"Oracles,omitempty"`
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

	AdminAddress    *bitcoin.RawAddress `json:"AdminAddress,omitempty"`
	OperatorAddress *bitcoin.RawAddress `json:"OperatorAddress,omitempty"`

	AdminMemberAsset *bitcoin.Hash20 `json:"AdminMemberAsset,omitempty"`
	OwnerMemberAsset *bitcoin.Hash20 `json:"OwnerMemberAsset,omitempty"`

	FreezePeriod *protocol.Timestamp `json:"FreezePeriod,omitempty"`

	ContractType *uint32 `json:"ContractType,omitempty"`
	ContractFee  *uint64 `json:"ContractFee,omitempty"`

	ContractExpiration  *protocol.Timestamp `json:"ContractExpiration,omitempty"`
	RestrictedQtyAssets *uint64             `json:"RestrictedQtyAssets,omitempty"`

	VotingSystems          *[]*actions.VotingSystemField `json:"VotingSystems,omitempty"`
	AdministrationProposal *bool                         `json:"AdministrationProposal,omitempty"`
	HolderProposal         *bool                         `json:"HolderProposal,omitempty"`

	Oracles *[]*actions.OracleField `json:"Oracles,omitempty"`
}
