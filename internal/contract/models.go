package contract

// Contract represents a Smart Contract.
type Contract struct {
	ID                          string  `json:"id"`
	CreatedAt                   int64   `json:"created_at"`
	UpdatedAt                   int64   `json:"updated_at"`
	TxHeadCount                 int     `json:"tx_head_count"`
	IssuerAddress               string  `json:"issuer_address"`
	OperatorAddress             string  `json:"operator_address"`
	Revision                    uint16  `json:"revision"`
	ContractName                string  `json:"name"`
	ContractFileHash            string  `json:"hash"`
	GoverningLaw                string  `json:"law"`
	Jurisdiction                string  `json:"jurisdiction"`
	ContractExpiration          uint64  `json:"contract_expiration"`
	URI                         string  `json:"uri"`
	IssuerID                    string  `json:"issuer_id"`
	IssuerType                  string  `json:"issuer_type"`
	ContractOperatorID          string  `json:"tokenizer_id"`
	AuthorizationFlags          []byte  `json:"authorization_flags"`
	VotingSystem                string  `json:"voting_system"`
	InitiativeThreshold         float32 `json:"initiative_threshold"`
	InitiativeThresholdCurrency string  `json:"initiative_threshold_currency"`
	Qty                         uint64  `json:"qty"`
	// Assets                      map[string]Asset `json:"assets"`
	// Votes                       map[string]Vote  `json:"votes"`
	Hashes []string `json:"hashes"`
}
