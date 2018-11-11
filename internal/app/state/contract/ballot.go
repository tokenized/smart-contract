package contract

import (
	"time"

	"github.com/tokenized/smart-contract/pkg/protocol"
	"github.com/btcsuite/btcutil"
)

type Ballot struct {
	Address   string `json:"address"`
	AssetType string `json:"asset_type"`
	AssetID   string `json:"asset_id"`
	VoteTxnID string `json:"vote_txn_id"`
	Vote      []byte `json:"vote"`
	CreatedAt int64  `json:"created_at"`
}

func NewBallotFromBallotCast(address btcutil.Address,
	m *protocol.BallotCast) Ballot {

	return Ballot{
		Address:   address.EncodeAddress(),
		AssetType: string(m.AssetType),
		AssetID:   string(m.AssetID),
		VoteTxnID: string(m.VoteTxnID),
		Vote:      m.Vote,
		CreatedAt: time.Now().UnixNano(),
	}
}
