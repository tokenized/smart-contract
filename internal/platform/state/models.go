package state

import (
	"github.com/tokenized/smart-contract/pkg/bitcoin"
	"github.com/tokenized/specification/dist/golang/actions"
	"github.com/tokenized/specification/dist/golang/protocol"
)

// Contract represents a Smart Contract.
type Contract struct {
	Address      *bitcoin.ConcreteRawAddress `json:"Address,omitempty"`
	Revision     uint32                      `json:"Revision,omitempty"`
	CreatedAt    protocol.Timestamp          `json:"CreatedAt,omitempty"`
	UpdatedAt    protocol.Timestamp          `json:"UpdatedAt,omitempty"`
	Timestamp    protocol.Timestamp          `json:"Timestamp,omitempty"`
	FreezePeriod protocol.Timestamp          `json:"FreezePeriod,omitempty"`

	AdministrationAddress *bitcoin.ConcreteRawAddress `json:"AdministrationAddress,omitempty"`
	OperatorAddress       *bitcoin.ConcreteRawAddress `json:"OperatorAddress,omitempty"`
	MasterAddress         *bitcoin.ConcreteRawAddress `json:"MasterAddress,omitempty"`
	MovedTo               *bitcoin.ConcreteRawAddress `json:"MovedTo,omitempty"`

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
	ContractAuthFlags         []byte                       `json:"ContractAuthFlags,omitempty"`
	ContractFee               uint64                       `json:"ContractFee,omitempty"`
	VotingSystems             []*actions.VotingSystemField `json:"VotingSystems,omitempty"`
	RestrictedQtyAssets       uint64                       `json:"RestrictedQtyAssets,omitempty"`
	AdministrationProposal    bool                         `json:"AdministrationProposal,omitempty"`
	HolderProposal            bool                         `json:"HolderProposal,omitempty"`
	Oracles                   []*actions.OracleField       `json:"Oracles,omitempty"`

	AssetCodes []*protocol.AssetCode `json:"AssetCodes,omitempty"`

	FullOracles []bitcoin.PublicKey `json:"_,omitempty"`
}

type Asset struct {
	Code      *protocol.AssetCode `json:"Code,omitempty"`
	Revision  uint32              `json:"Revision,omitempty"`
	CreatedAt protocol.Timestamp  `json:"CreatedAt,omitempty"`
	UpdatedAt protocol.Timestamp  `json:"UpdatedAt,omitempty"`
	Timestamp protocol.Timestamp  `json:"Timestamp,omitempty"`

	AssetType                   string             `json:"AssetType,omitempty"`
	AssetIndex                  uint64             `json:"AssetIndex,omitempty"`
	AssetAuthFlags              []byte             `json:"AssetAuthFlags,omitempty"`
	TransfersPermitted          bool               `json:"TransfersPermitted,omitempty"`
	TradeRestrictions           []string           `json:"TradeRestrictions,omitempty"`
	EnforcementOrdersPermitted  bool               `json:"EnforcementOrdersPermitted,omitempty"`
	VotingRights                bool               `json:"VotingRights,omitempty"`
	VoteMultiplier              uint32             `json:"VoteMultiplier,omitempty"`
	AdministrationProposal      bool               `json:"AdministrationProposal,omitempty"`
	HolderProposal              bool               `json:"HolderProposal,omitempty"`
	AssetModificationGovernance uint32             `json:"AssetModificationGovernance,omitempty"`
	TokenQty                    uint64             `json:"TokenQty,omitempty"`
	AssetPayload                []byte             `json:"AssetPayload,omitempty"`
	FreezePeriod                protocol.Timestamp `json:"FreezePeriod,omitempty"`
}

type Holding struct {
	Address *bitcoin.ConcreteRawAddress `json:"Address,omitempty"`
	// Balance after all pending changes have been finalized
	PendingBalance uint64 `json:"PendingBalance,omitempty"`
	// Balance without pending changes
	FinalizedBalance uint64                           `json:"FinalizedBalance,omitempty"`
	HoldingStatuses  map[protocol.TxId]*HoldingStatus `json:"HoldingStatuses,omitempty"`
	CreatedAt        protocol.Timestamp               `json:"CreatedAt,omitempty"`
	UpdatedAt        protocol.Timestamp               `json:"UpdatedAt,omitempty"`
}

type HoldingStatus struct {
	// Code F = Freeze, R = Pending Receive, S = Pending Send
	Code byte `json:"Code,omitempty"`

	Expires        protocol.Timestamp `json:"Expires,omitempty"`
	Amount         uint64             `json:"Amount,omitempty"`
	TxId           *protocol.TxId     `json:"TxId,omitempty"`
	SettleQuantity uint64             `json:"SettleQuantity,omitempty"`

	// Balance has been posted to the chain and is not reversible without a reconcile.
	Posted bool `json:"Posted,omitempty"`
}

type Vote struct {
	Initiator          uint32                    `json:"Initiator,omitempty"`
	VoteSystem         uint32                    `json:"VoteSystem,omitempty"`
	ContractWideVote   bool                      `json:"ContractWideVote,omitempty"`
	AssetSpecificVote  bool                      `json:"AssetSpecificVote,omitempty"`
	AssetType          string                    `json:"AssetType,omitempty"`
	AssetCode          *protocol.AssetCode       `json:"AssetCode,omitempty"`
	Specific           bool                      `json:"Specific,omitempty"`
	ProposedAmendments []*actions.AmendmentField `json:"ProposedAmendments,omitempty"`

	VoteTxId     *protocol.TxId     `json:"VoteTxId,omitempty"`
	ProposalTxId *protocol.TxId     `json:"ProposalTxId,omitempty"`
	TokenQty     uint64             `json:"TokenQty,omitempty"`
	Expires      protocol.Timestamp `json:"Expires,omitempty"`
	Timestamp    protocol.Timestamp `json:"Timestamp,omitempty"`
	CreatedAt    protocol.Timestamp `json:"CreatedAt,omitempty"`
	UpdatedAt    protocol.Timestamp `json:"UpdatedAt,omitempty"`

	OptionTally []uint64           `json:"OptionTally,omitempty"`
	Result      string             `json:"Result,omitempty"`
	AppliedTxId *protocol.TxId     `json:"AppliedTxId,omitempty"`
	CompletedAt protocol.Timestamp `json:"CompletedAt,omitempty"`

	Ballots []*Ballot `json:"Ballots,omitempty"`
}

type Ballot struct {
	Address   *bitcoin.ConcreteRawAddress `json:"Address,omitempty"`
	Vote      string                      `json:"Vote,omitempty"`
	Quantity  uint64                      `json:"Quantity,omitempty"`
	Timestamp protocol.Timestamp          `json:"Timestamp,omitempty"`
}

// PendingTransfer defines the information required to monitor pending multi-contract transfers.
type PendingTransfer struct {
	TransferTxId *protocol.TxId     `json:"TransferTxId,omitempty"`
	Timeout      protocol.Timestamp `json:"Timeout,omitempty"`
}
