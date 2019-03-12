package state

import (
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/tokenized/smart-contract/pkg/inspector"
)

// Contract represents a Smart Contract.
type Contract struct {
	ID              string
	Revision        uint64
	CreatedAt       uint64
	UpdatedAt       uint64
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
	VotingSystems              []VotingSystem
	RestrictedQtyAssets        uint64
	ReferendumProposal         bool
	InitiativeProposal         bool
	Registries                 []Registry
	IssuerAddressSpecified     bool
	UnitNumber                 string
	BuildingNumber             string
	Street                     string
	SuburbCity                 string
	TerritoryStateProvinceCode string
	CountryCode                string
	PostalZIPCode              string
	EmailAddress               string
	PhoneNumber                string
	KeyRoles                   []KeyRole
	NotableRoles               []NotableRole

	Assets map[string]Asset
	Votes  map[string]Vote
}

type Asset struct {
	ID        string
	Revision  uint64
	CreatedAt uint64
	UpdatedAt uint64

	AssetType                   string
	AssetAuthFlags              []byte
	TransfersPermitted          bool
	TradeRestrictions           []byte
	EnforcementOrdersPermitted  bool
	VoteMultiplier              uint8
	ReferendumProposal          bool
	InitiativeProposal          bool
	AssetModificationGovernance bool
	TokenQty                    uint64
	ContractFeeCurrency         []byte
	ContractFeeVar              float32
	ContractFeeFixed            float32
	AssetPayload                []byte

	Holdings map[string]Holding
}

type Holding struct {
	Address         string          `json:"address"`
	Balance         uint64          `json:"balance"`
	HoldingStatuses []HoldingStatus `json:"order_status,omitempty"`
	CreatedAt       uint64          `json:"created_at"`
}

type HoldingStatus struct {
	Code    string `json:"code"`
	Expires uint64 `json:"expires,omitempty"`
	Balance uint64 `json:"balance"`
	TxId    chainhash.Hash
}

type Vote struct {
	Address              string            `json:"address"`
	AssetType            string            `json:"asset_type"`
	AssetID              string            `json:"asset_id"`
	VoteType             byte              `json:"vote_type"`
	VoteOptions          []uint8           `json:"vote_options"`
	VoteMax              uint8             `json:"vote_max"`
	VoteLogic            byte              `json:"vote_logic"`
	ProposalDescription  string            `json:"proposal_description"`
	ProposalDocumentHash string            `json:"proposal_document_hash"`
	VoteCutOffTimestamp  int64             `json:"vote_cut_off_timestamp"`
	RefTxnIDHash         string            `json:"ref_txn_id_hash"`
	Ballots              map[string]Ballot `json:"ballots"`
	UTXO                 inspector.UTXO    `json:"utxo"`
	Result               *Result           `json:"result,omitempty"`
	UsedBy               string            `json:"used_by,omit_empty"`
	CreatedAt            uint64            `json:"created_at"`
}

type Ballot struct {
	Address   string  `json:"address"`
	AssetType string  `json:"asset_type"`
	AssetID   string  `json:"asset_id"`
	VoteTxnID string  `json:"vote_txn_id"`
	Vote      []uint8 `json:"vote"`
	CreatedAt uint64  `json:"created_at"`
}

type Result map[uint8]uint64

type VotingSystem struct {
	Name                        string  `json:"name"`
	System                      []byte  `json:"system"`
	Method                      byte    `json:"method"`
	Logic                       byte    `json:"logic"`
	ThresholdPercentage         uint8   `json:"threshold_percent"`
	VoteMultiplierPermitted     byte    `json:"multiplier_permitted"`
	InitiativeThreshold         float32 `json:"threshold"`
	InitiativeThresholdCurrency []byte  `json:"threshold_currency"`
}

type Registry struct {
	Name      string `json:"name"`
	URL       string `json:"url"`
	PublicKey string `json:"public_key"`
}

type KeyRole struct {
	Type byte   `json:"type"`
	Name string `json:"name"`
}

type NotableRole struct {
	Type byte   `json:"type"`
	Name string `json:"name"`
}
