package vote

// NewVote defines what we require when creating a Vote record.
type NewVote struct {
	Address              string  `json:"address,omit_empty"`
	AssetType            string  `json:"asset_type,omit_empty"`
	AssetID              string  `json:"asset_id,omit_empty"`
	VoteType             byte    `json:"vote_type,omit_empty"`
	VoteOptions          []uint8 `json:"vote_options,omit_empty"`
	VoteMax              uint8   `json:"vote_max,omit_empty"`
	VoteLogic            byte    `json:"vote_logic,omit_empty"`
	ProposalDescription  string  `json:"proposal_description,omit_empty"`
	ProposalDocumentHash string  `json:"proposal_document_hash,omit_empty"`
	VoteCutOffTimestamp  int64   `json:"vote_cut_off_timestamp,omit_empty"`
}

// UpdateVote defines what information may be provided to modify an existing
// Vote. All fields are optional so clients can send just the fields they want
// changed. It uses pointer fields so we can differentiate between a field that
// was not provided and a field that was provided as explicitly blank. Normally
// we do not want to use pointers to basic types but we make exceptions around
// marshalling/unmarshalling.
type UpdateVote struct {
}
