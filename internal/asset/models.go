package asset

// NewAsset defines what we require when creating a Asset record.
type NewAsset struct {
	IssuerAddress  string `json:"string"`
	ID             string `json:"id"`
	Type           string `json:"type"`
	VotingSystem   string `json:"voting_system"`
	VoteMultiplier uint8  `json:"vote_multiplier"`
	Qty            uint64 `json:"qty"`
}

// UpdateAsset defines what information may be provided to modify an
// existing Asset.
type UpdateAsset struct {
	Type           string `json:"type"`
	Revision       uint16 `json:"revision"`
	VotingSystem   string `json:"voting_system"`
	VoteMultiplier uint8  `json:"vote_multiplier"`
	Qty            uint64 `json:"qty"`
}
