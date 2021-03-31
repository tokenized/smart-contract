package state

import (
	"github.com/tokenized/pkg/bitcoin"
	"github.com/tokenized/specification/dist/golang/actions"
	"github.com/tokenized/specification/dist/golang/assets"
	"github.com/tokenized/specification/dist/golang/protocol"
)

// Contract represents a Smart Contract.
type Contract struct {
	Address      bitcoin.RawAddress `json:"Address,omitempty"`
	Revision     uint32             `json:"Revision,omitempty"`
	CreatedAt    protocol.Timestamp `json:"CreatedAt,omitempty"`
	UpdatedAt    protocol.Timestamp `json:"UpdatedAt,omitempty"`
	Timestamp    protocol.Timestamp `json:"Timestamp,omitempty"`
	FreezePeriod protocol.Timestamp `json:"FreezePeriod,omitempty"`

	AdminAddress    bitcoin.RawAddress `json:"AdminAddress,omitempty"`
	OperatorAddress bitcoin.RawAddress `json:"OperatorAddress,omitempty"`
	MasterAddress   bitcoin.RawAddress `json:"MasterAddress,omitempty"`
	MovedTo         bitcoin.RawAddress `json:"MovedTo,omitempty"`

	AdminMemberAsset bitcoin.Hash20 `json:"AdminMemberAsset,omitempty"`
	OwnerMemberAsset bitcoin.Hash20 `json:"OwnerMemberAsset,omitempty"`

	ContractType uint32 `json:"ContractType,omitempty"`
	ContractFee  uint64 `json:"ContractFee,omitempty"`

	ContractExpiration  protocol.Timestamp `json:"ContractExpiration,omitempty"`
	RestrictedQtyAssets uint64             `json:"RestrictedQtyAssets,omitempty"`

	VotingSystems          []*actions.VotingSystemField `json:"VotingSystems,omitempty"`
	AdministrationProposal bool                         `json:"AdministrationProposal,omitempty"`
	HolderProposal         bool                         `json:"HolderProposal,omitempty"`

	BodyOfAgreementType uint32 `json:"BodyOfAgreementType,omitempty"`

	Oracles []*actions.OracleField `json:"Oracles,omitempty"`

	AssetCodes []*bitcoin.Hash20 `json:"AssetCodes,omitempty"`

	FullOracles []Oracle `json:"_,omitempty"`
}

type Agreement struct {
	Chapters    []*actions.ChapterField     `json:"Chapters,omitempty"`
	Definitions []*actions.DefinedTermField `json:"Definitions,omitempty"`
	Revision    uint32                      `json:"Revision,omitempty"`
	CreatedAt   protocol.Timestamp          `json:"CreatedAt,omitempty"`
	Timestamp   protocol.Timestamp          `json:"Timestamp,omitempty"`
	UpdatedAt   protocol.Timestamp          `json:"UpdatedAt,omitempty"`
}

type Oracle struct {
	Address   bitcoin.RawAddress `json:"address,omitempty"`
	URL       string             `json:"url,omitempty"`
	PublicKey bitcoin.PublicKey  `json:"public_key,omitempty"`
}

type Asset struct {
	Code      *bitcoin.Hash20    `json:"Code,omitempty"`
	Revision  uint32             `json:"Revision,omitempty"`
	CreatedAt protocol.Timestamp `json:"CreatedAt,omitempty"`
	UpdatedAt protocol.Timestamp `json:"UpdatedAt,omitempty"`
	Timestamp protocol.Timestamp `json:"Timestamp,omitempty"`

	AssetType                   string             `json:"AssetType,omitempty"`
	AssetIndex                  uint64             `json:"AssetIndex,omitempty"`
	AssetPermissions            []byte             `json:"AssetPermissions,omitempty"`
	TradeRestrictions           []string           `json:"TradeRestrictions,omitempty"`
	EnforcementOrdersPermitted  bool               `json:"EnforcementOrdersPermitted,omitempty"`
	VotingRights                bool               `json:"VotingRights,omitempty"`
	VoteMultiplier              uint32             `json:"VoteMultiplier,omitempty"`
	AdministrationProposal      bool               `json:"AdministrationProposal,omitempty"`
	HolderProposal              bool               `json:"HolderProposal,omitempty"`
	AssetModificationGovernance uint32             `json:"AssetModificationGovernance,omitempty"`
	AuthorizedTokenQty          uint64             `json:"AuthorizedTokenQty,omitempty"`
	AssetPayload                []byte             `json:"AssetPayload,omitempty"`
	FreezePeriod                protocol.Timestamp `json:"FreezePeriod,omitempty"`
}

func (a Asset) TransfersPermitted() bool {
	if a.AssetType == assets.CodeCurrency {
		return true
	}

	data, err := assets.Deserialize([]byte(a.AssetType), a.AssetPayload)
	if err != nil {
		return false
	}

	switch as := data.(type) {
	case *assets.Membership:
		return as.TransfersPermitted
	case *assets.ShareCommon:
		return as.TransfersPermitted
	case *assets.Coupon:
		return as.TransfersPermitted
	case *assets.LoyaltyPoints:
		return as.TransfersPermitted
	case *assets.TicketAdmission:
		return as.TransfersPermitted
	case *assets.CasinoChip:
		return as.TransfersPermitted
	case *assets.BondFixedRate:
		return as.TransfersPermitted
	}

	return true
}

type Holding struct {
	Address bitcoin.RawAddress `json:"Address,omitempty"`
	// Balance after all pending changes have been finalized
	PendingBalance uint64 `json:"PendingBalance,omitempty"`
	// Balance without pending changes
	FinalizedBalance uint64                            `json:"FinalizedBalance,omitempty"`
	HoldingStatuses  map[bitcoin.Hash32]*HoldingStatus `json:"HoldingStatuses,omitempty"`
	CreatedAt        protocol.Timestamp                `json:"CreatedAt,omitempty"`
	UpdatedAt        protocol.Timestamp                `json:"UpdatedAt,omitempty"`
}

type HoldingStatus struct {
	// Code F = Freeze, R = Pending Receive, S = Pending Send
	Code byte `json:"Code,omitempty"`

	Expires        protocol.Timestamp `json:"Expires,omitempty"`
	Amount         uint64             `json:"Amount,omitempty"`
	TxId           *bitcoin.Hash32    `json:"TxId,omitempty"`
	SettleQuantity uint64             `json:"SettleQuantity,omitempty"`

	// Balance has been posted to the chain and is not reversible without a reconcile.
	// Note: This is currently not used as address balances are locked during multi-contract
	//   transfers so a bad state can never be posted.
	Posted bool `json:"Posted,omitempty"`
}

type Vote struct {
	Type               uint32                    `json:"Type,omitempty"`
	VoteSystem         uint32                    `json:"VoteSystem,omitempty"`
	ContractWideVote   bool                      `json:"ContractWideVote,omitempty"`
	AssetType          string                    `json:"AssetType,omitempty"`
	AssetCode          *bitcoin.Hash20           `json:"AssetCode,omitempty"`
	ProposedAmendments []*actions.AmendmentField `json:"ProposedAmendments,omitempty"`

	VoteTxId     *bitcoin.Hash32    `json:"VoteTxId,omitempty"`
	ProposalTxId *bitcoin.Hash32    `json:"ProposalTxId,omitempty"`
	TokenQty     uint64             `json:"TokenQty,omitempty"`
	Expires      protocol.Timestamp `json:"Expires,omitempty"`
	Timestamp    protocol.Timestamp `json:"Timestamp,omitempty"`
	CreatedAt    protocol.Timestamp `json:"CreatedAt,omitempty"`
	UpdatedAt    protocol.Timestamp `json:"UpdatedAt,omitempty"`

	OptionTally []uint64           `json:"OptionTally,omitempty"`
	Result      string             `json:"Result,omitempty"`
	AppliedTxId *bitcoin.Hash32    `json:"AppliedTxId,omitempty"`
	CompletedAt protocol.Timestamp `json:"CompletedAt,omitempty"`

	Ballots    map[bitcoin.Hash20]Ballot `json:"-"` // json can only encode string maps
	BallotList []Ballot                  `json:"Ballots,omitempty"`
}

type Ballot struct {
	Address   bitcoin.RawAddress `json:"Address,omitempty"`
	Vote      string             `json:"Vote,omitempty"`
	Quantity  uint64             `json:"Quantity,omitempty"`
	Timestamp protocol.Timestamp `json:"Timestamp,omitempty"`
}

// PendingTransfer defines the information required to monitor pending multi-contract transfers.
type PendingTransfer struct {
	TransferTxId *bitcoin.Hash32    `json:"TransferTxId,omitempty"`
	Timeout      protocol.Timestamp `json:"Timeout,omitempty"`
}
