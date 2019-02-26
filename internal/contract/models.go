package contract

// NewContract defines what we require when creating a Contract record.
type NewContract struct {
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
}

// UpdateContract defines what information may be provided to modify an
// existing Contract.
type UpdateContract struct {
	ContractName                string  `json:"name"`
	ContractFileHash            string  `json:"hash"`
	GoverningLaw                string  `json:"law"`
	Jurisdiction                string  `json:"jurisdiction"`
	ContractExpiration          uint64  `json:"contract_expiration"`
	URI                         string  `json:"uri"`
	Revision                    uint16  `json:"revision"`
	IssuerID                    string  `json:"issuer_id"`
	IssuerType                  string  `json:"issuer_type"`
	ContractOperatorID          string  `json:"tokenizer_id"`
	AuthorizationFlags          []byte  `json:"authorization_flags"`
	VotingSystem                string  `json:"voting_system"`
	InitiativeThreshold         float32 `json:"initiative_threshold"`
	InitiativeThresholdCurrency string  `json:"initiative_threshold_currency"`
	Qty                         uint64  `json:"qty"`
}
