package state

import "github.com/tokenized/smart-contract/pkg/inspector"

// Contract represents a Smart Contract.
type Contract struct {
	ID                          string           `json:"id"`
	CreatedAt                   int64            `json:"created_at"`
	UpdatedAt                   int64            `json:"updated_at"`
	TxHeadCount                 int              `json:"tx_head_count"`
	IssuerAddress               string           `json:"issuer_address"`
	OperatorAddress             string           `json:"operator_address"`
	Revision                    uint16           `json:"revision"`
	ContractName                string           `json:"name"`
	ContractFileHash            string           `json:"hash"`
	GoverningLaw                string           `json:"law"`
	Jurisdiction                string           `json:"jurisdiction"`
	ContractExpiration          uint64           `json:"contract_expiration"`
	URI                         string           `json:"uri"`
	IssuerID                    string           `json:"issuer_id"`
	IssuerType                  string           `json:"issuer_type"`
	ContractOperatorID          string           `json:"tokenizer_id"`
	AuthorizationFlags          []byte           `json:"authorization_flags"`
	VotingSystem                string           `json:"voting_system"`
	InitiativeThreshold         float32          `json:"initiative_threshold"`
	InitiativeThresholdCurrency string           `json:"initiative_threshold_currency"`
	Qty                         uint64           `json:"qty"`
	Assets                      map[string]Asset `json:"assets"`
	Votes                       map[string]Vote  `json:"votes"`
	Hashes                      []string         `json:"hashes"`
}

type Asset struct {
	ID                 string             `json:"id"`
	Type               string             `json:"type"`
	Revision           uint16             `json:"revision"`
	AuthorizationFlags []byte             `json:"auth_flags"`
	VotingSystem       string             `json:"voting_system"`
	VoteMultiplier     uint8              `json:"vote_multiplier"`
	Qty                uint64             `json:"qty"`
	TxnFeeType         byte               `json:"txn_fee_type"`
	TxnFeeCurrency     string             `json:"txn_fee_currency"`
	TxnFeeVar          float32            `json:"txn_fee_var,omitempty"`
	TxnFeeFixed        float32            `json:"txn_fee_fixed,omitempty"`
	Holdings           map[string]Holding `json:"holdings"`
	CreatedAt          int64              `json:"created_at"`
}

type Holding struct {
	Address       string         `json:"address"`
	Balance       uint64         `json:"balance"`
	HoldingStatus *HoldingStatus `json:"order_status,omitempty"`
	CreatedAt     int64          `json:"created_at"`
}

type HoldingStatus struct {
	Code    string `json:"code"`
	Expires uint64 `json:"expires,omitempty"`
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
	CreatedAt            int64             `json:"created_at"`
}

type Ballot struct {
	Address   string  `json:"address"`
	AssetType string  `json:"asset_type"`
	AssetID   string  `json:"asset_id"`
	VoteTxnID string  `json:"vote_txn_id"`
	Vote      []uint8 `json:"vote"`
	CreatedAt int64   `json:"created_at"`
}

type Result map[uint8]uint64
