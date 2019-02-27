package vote

// NewVote defines what we require when creating a Vote record.
type NewVote struct {
	Address              string  `json:"address"`
	AssetType            string  `json:"asset_type"`
	AssetID              string  `json:"asset_id"`
	VoteType             byte    `json:"vote_type"`
	VoteOptions          []uint8 `json:"vote_options"`
	VoteMax              uint8   `json:"vote_max"`
	VoteLogic            byte    `json:"vote_logic"`
	ProposalDescription  string  `json:"proposal_description"`
	ProposalDocumentHash string  `json:"proposal_document_hash"`
	VoteCutOffTimestamp  int64   `json:"vote_cut_off_timestamp"`
}

// UpdateVote defines what information may be provided to modify an existing
// Vote. All fields are optional so clients can send just the fields they want
// changed. It uses pointer fields so we can differentiate between a field that
// was not provided and a field that was provided as explicitly blank. Normally
// we do not want to use pointers to basic types but we make exceptions around
// marshalling/unmarshalling.
type UpdateVote struct {
}
