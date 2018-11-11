package contract

import (
	"time"

	"github.com/tokenized/smart-contract/pkg/protocol"
	"github.com/tokenized/smart-contract/pkg/txbuilder"
)

type Vote struct {
	Address              string         `json:"address"`
	AssetType            string         `json:"asset_type"`
	AssetID              string         `json:"asset_id"`
	VoteType             byte           `json:"vote_type"`
	VoteOptions          []byte         `json:"vote_options"`
	VoteMax              uint8          `json:"vote_max"`
	VoteLogic            byte           `json:"vote_logic"`
	ProposalDescription  string         `json:"proposal_description"`
	ProposalDocumentHash string         `json:"proposal_document_hash"`
	VoteCutOffTimestamp  int64          `json:"vote_cut_off_timestamp"`
	RefTxnIDHash         string         `json:"ref_txn_id_hash"`
	Ballots              []Ballot       `json:"ballots"`
	UTXO                 txbuilder.UTXO `json:"utxo"`
	Result               *BallotResult  `json:"result,omitempty"`
	CreatedAt            int64          `json:"created_at"`
}

func NewVote() Vote {
	return Vote{
		CreatedAt: time.Now().UnixNano(),
		Ballots:   []Ballot{},
	}
}

func NewVoteFromProtocolVote(address string, m *protocol.Vote) Vote {
	v := NewVote()
	v.Address = address
	v.AssetType = string(m.AssetType)
	v.AssetID = string(m.AssetID)
	v.VoteType = m.VoteType
	v.VoteOptions = m.VoteOptions
	v.VoteMax = m.VoteMax
	v.VoteLogic = m.VoteLogic
	v.ProposalDescription = string(m.ProposalDescription)
	v.ProposalDocumentHash = string(v.ProposalDocumentHash)
	v.VoteCutOffTimestamp = v.VoteCutOffTimestamp

	return v
}

func (v Vote) IsOpen(ts time.Time) bool {
	return ts.UnixNano() < v.VoteCutOffTimestamp
}
