package state

import (
	"sync"

	"github.com/tokenized/pkg/bitcoin"
	"github.com/tokenized/specification/dist/golang/actions"
	"github.com/tokenized/specification/dist/golang/instruments"
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

	// Note: Leave JSON name as AdminMemberAsset for backwards compatibility
	AdminMemberInstrument bitcoin.Hash20 `json:"AdminMemberAsset,omitempty"`

	// Note: Leave JSON name as OwnerMemberAsset for backwards compatibility
	OwnerMemberInstrument bitcoin.Hash20 `json:"OwnerMemberAsset,omitempty"`

	ContractType uint32 `json:"ContractType,omitempty"`
	ContractFee  uint64 `json:"ContractFee,omitempty"`

	ContractExpiration protocol.Timestamp `json:"ContractExpiration,omitempty"`

	// Note: Leave JSON name as RestrictedQtyAssets for backwards compatibility
	RestrictedQtyInstruments uint64 `json:"RestrictedQtyAssets,omitempty"`

	VotingSystems          []*actions.VotingSystemField `json:"VotingSystems,omitempty"`
	AdministrationProposal bool                         `json:"AdministrationProposal,omitempty"`
	HolderProposal         bool                         `json:"HolderProposal,omitempty"`

	BodyOfAgreementType uint32 `json:"BodyOfAgreementType,omitempty"`

	Oracles []*actions.OracleField `json:"Oracles,omitempty"`

	// Note: Leave JSON name as AssetCodes for backwards compatibility
	InstrumentCodes []*bitcoin.Hash20 `json:"AssetCodes,omitempty"`

	FullOracles []*Oracle `json:"_,omitempty"`
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
	Address  bitcoin.RawAddress `json:"address,omitempty"`
	Services []*Service         `json:"services,omitempty"`

	sync.RWMutex
}

type Service struct {
	ServiceType uint32            `json:"service_type,omitempty"`
	URL         string            `json:"url,omitempty"`
	PublicKey   bitcoin.PublicKey `json:"public_key,omitempty"`
}

func (o *Oracle) GetService(serviceType uint32) *Service {
	o.RLock()
	defer o.RUnlock()

	for _, service := range o.Services {
		if service.ServiceType == serviceType {
			return service
		}
	}

	return nil
}

type Instrument struct {
	Code      *bitcoin.Hash20    `json:"Code,omitempty"`
	Revision  uint32             `json:"Revision,omitempty"`
	CreatedAt protocol.Timestamp `json:"CreatedAt,omitempty"`
	UpdatedAt protocol.Timestamp `json:"UpdatedAt,omitempty"`
	Timestamp protocol.Timestamp `json:"Timestamp,omitempty"`

	InstrumentType                   string             `json:"InstrumentType,omitempty"`
	InstrumentIndex                  uint64             `json:"InstrumentIndex,omitempty"`
	InstrumentPermissions            []byte             `json:"InstrumentPermissions,omitempty"`
	TradeRestrictions                []string           `json:"TradeRestrictions,omitempty"`
	EnforcementOrdersPermitted       bool               `json:"EnforcementOrdersPermitted,omitempty"`
	VotingRights                     bool               `json:"VotingRights,omitempty"`
	VoteMultiplier                   uint32             `json:"VoteMultiplier,omitempty"`
	AdministrationProposal           bool               `json:"AdministrationProposal,omitempty"`
	HolderProposal                   bool               `json:"HolderProposal,omitempty"`
	InstrumentModificationGovernance uint32             `json:"InstrumentModificationGovernance,omitempty"`
	AuthorizedTokenQty               uint64             `json:"AuthorizedTokenQty,omitempty"`
	InstrumentPayload                []byte             `json:"InstrumentPayload,omitempty"`
	FreezePeriod                     protocol.Timestamp `json:"FreezePeriod,omitempty"`
}

func (a Instrument) TransfersPermitted() bool {
	if a.InstrumentType == instruments.CodeCurrency {
		return true
	}

	data, err := instruments.Deserialize([]byte(a.InstrumentType), a.InstrumentPayload)
	if err != nil {
		return false
	}

	switch as := data.(type) {
	case *instruments.Membership:
		return as.TransfersPermitted
	case *instruments.ShareCommon:
		return as.TransfersPermitted
	case *instruments.Coupon:
		return as.TransfersPermitted
	case *instruments.LoyaltyPoints:
		return as.TransfersPermitted
	case *instruments.TicketAdmission:
		return as.TransfersPermitted
	case *instruments.CasinoChip:
		return as.TransfersPermitted
	case *instruments.BondFixedRate:
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
	Type             uint32 `json:"Type,omitempty"`
	VoteSystem       uint32 `json:"VoteSystem,omitempty"`
	ContractWideVote bool   `json:"ContractWideVote,omitempty"`

	// Note: Leave JSON name as AssetType for backwards compatibility
	InstrumentType string `json:"AssetType,omitempty"`

	// Note: Leave JSON name as AssetCode for backwards compatibility
	InstrumentCode *bitcoin.Hash20 `json:"AssetCode,omitempty"`

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
