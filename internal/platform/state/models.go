package state

import (
	"github.com/tokenized/smart-contract/pkg/protocol"
)

// Contract represents a Smart Contract.
type Contract struct {
	ID        protocol.PublicKeyHash `json:"id,omitempty"`
	Revision  uint32                 `json:"revision,omitempty"`
	CreatedAt protocol.Timestamp     `json:"created_at,omitempty"`
	UpdatedAt protocol.Timestamp     `json:"updated_at,omitempty"`
	Timestamp protocol.Timestamp     `json:"timestamp,omitempty"`

	IssuerPKH   protocol.PublicKeyHash `json:"issuer_pkh,omitempty"`
	OperatorPKH protocol.PublicKeyHash `json:"operator_pkh,omitempty"`

	ContractName           string                  `json:"contract_name,omitempty"`
	BodyOfAgreementType    uint8                   `json:"body_of_agreement_type,omitempty"`
	BodyOfAgreement        []byte                  `json:"body_of_agreement,omitempty"`
	ContractType           string                  `json:"contract_type,omitempty"`
	SupportingDocsFileType uint8                   `json:"supporting_docs_file_type,omitempty"`
	SupportingDocs         []byte                  `json:"supporting_docs,omitempty"`
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
	IssuerProposal         bool                    `json:"issuer_proposal,omitempty"`
	HolderProposal         bool                    `json:"holder_proposal,omitempty"`
	Registries             []protocol.Registry     `json:"registries,omitempty"`

	AssetCodes []protocol.AssetCode `json:"asset_codes,omitempty"`
}

type Asset struct {
	ID        protocol.AssetCode `json:"id,omitempty"`
	Revision  uint32             `json:"revision,omitempty"`
	CreatedAt protocol.Timestamp `json:"created_at,omitempty"`
	UpdatedAt protocol.Timestamp `json:"updated_at,omitempty"`
	Timestamp protocol.Timestamp `json:"timestamp,omitempty"`

	AssetType                   string          `json:"asset_type,omitempty"`
	AssetAuthFlags              []byte          `json:"asset_auth_flags,omitempty"`
	TransfersPermitted          bool            `json:"transfers_permitted,omitempty"`
	TradeRestrictions           protocol.Polity `json:"trade_restrictions,omitempty"`
	EnforcementOrdersPermitted  bool            `json:"enforcement_orders_permitted,omitempty"`
	VotingRights                bool            `json:"voting_rights,omitempty"`
	VoteMultiplier              uint8           `json:"vote_multiplier,omitempty"`
	IssuerProposal              bool            `json:"issuer_proposal,omitempty"`
	HolderProposal              bool            `json:"holder_proposal,omitempty"`
	AssetModificationGovernance bool            `json:"asset_modification_governance,omitempty"`
	TokenQty                    uint64          `json:"token_qty,omitempty"`
	AssetPayload                []byte          `json:"asset_payload,omitempty"`

	Holdings []Holding `json:"holdings,omitempty"`
}

type Holding struct {
	PKH             protocol.PublicKeyHash `json:"address,omit_empty"`
	Balance         uint64                 `json:"balance,omit_empty"`
	HoldingStatuses []HoldingStatus        `json:"order_status,omitempty"`
	CreatedAt       protocol.Timestamp     `json:"created_at,omit_empty"`
}

type HoldingStatus struct {
	Code    string             `json:"code,omit_empty"`
	Expires protocol.Timestamp `json:"expires,omitempty"`
	Balance uint64             `json:"balance,omit_empty"`
	TxId    protocol.TxId      `json:"tx_id,omit_empty"`
}

type Vote struct {
	VoteTxId     protocol.TxId      `json:"vote_tx_id_hash,omit_empty"`
	ProposalTxId protocol.TxId      `json:"proposal_tx_id_hash,omit_empty"`
	TokenQty     uint64             `json:"token_qty,omit_empty"`
	Expires      protocol.Timestamp `json:"expires,omit_empty"`
	Timestamp    protocol.Timestamp `json:"timestamp,omit_empty"`
	CreatedAt    protocol.Timestamp `json:"created_at,omit_empty"`
	UpdatedAt    protocol.Timestamp `json:"updated_at,omit_empty"`

	OptionTally []uint64           `json:"option_tally,omit_empty"`
	Result      string             `json:"result,omit_empty"`
	CompletedAt protocol.Timestamp `json:"completed_at,omit_empty"`

	Ballots []*Ballot `json:"ballots,omit_empty"`
}

type Ballot struct {
	PKH       protocol.PublicKeyHash `json:"pkh,omit_empty"`
	Vote      string                 `json:"vote,omit_empty"`
	Quantity  uint64                 `json:"quantity,omit_empty"`
	Timestamp protocol.Timestamp     `json:"timestamp,omit_empty"`
}
