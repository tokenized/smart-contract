package request

import (
	"reflect"
	"testing"
	"time"

	"github.com/btcsuite/btcutil"
	"github.com/tokenized/smart-contract/internal/app/state/contract"
	"github.com/tokenized/smart-contract/pkg/protocol"
	"github.com/tokenized/smart-contract/pkg/txbuilder"
)

func TestBallotCastHandler_handle(t *testing.T) {
	ctx := newSilentContext()

	// this is the TX we receive. This isn't actually a ballot cast tx, but
	// we can get around it for the test.
	// tx := loadFixtureTX("exchange-signed_8dc75891.txn")

	// the addresses we will receive in the outs of the TX
	contractAddr := "1CWjudGPuj1sHs3GuMkAGPEUP5YaJNqu8U"
	issuerAddr := "1HwvXtVEMDuvbrNHQCwWaV97ucBLr3zCgJ"
	userAddr := "1L9Vr7BCEeczDtSJiX3fHLG5VVQgHtB22o"

	senders := []btcutil.Address{
		decodeAddress(userAddr),
	}

	// receivers from the TX outs
	receivers := []txbuilder.TxOutput{
		txbuilder.TxOutput{
			Address: decodeAddress(contractAddr),
			Value:   2200,
		},
		// Receiver{
		// 	Address: decodeAddress(userAddr),
		// 	Value:   546,
		// },
		// Receiver{
		// 	Address: decodeAddress(party2Addr),
		// 	Value:   76802,
		// },
		// Receiver{
		// 	Address: decodeAddress(feeAddr),
		// 	Value:   546,
		// },
	}

	// build the existing asset
	asset := contract.Asset{
		ID:   "1v2mwouuzz2x73ulv6o57llbx5udym6l",
		Qty:  43,
		Type: "GOO",
		Holdings: map[string]contract.Holding{
			issuerAddr: contract.Holding{
				Address: issuerAddr,
				Balance: 43,
			},
			userAddr: contract.Holding{
				Address: issuerAddr,
				Balance: 43,
			},
		},
	}

	voteHash := "d2b2db94192a0e80f87fffe60c4a8b1b224f80b0ae46d0563f72e25c49b93758"

	// setup the existing contract
	expires := time.Now().Add(time.Hour * 1).UnixNano()

	c := contract.Contract{
		ID: contractAddr,
		Assets: map[string]contract.Asset{
			asset.ID: asset,
		},
		Votes: map[string]contract.Vote{
			voteHash: contract.Vote{
				VoteCutOffTimestamp: expires,
			},
		},
		Qty: 1,
	}

	ballotCast := protocol.NewBallotCast()
	ballotCast.VoteTxnID = []byte(voteHash)
	ballotCast.AssetID = []byte(asset.ID)
	ballotCast.AssetType = []byte("GOO")
	ballotCast.Vote = []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}

	hash := "3c597097711bc7b3c8b24f87d622cb47612ff91288017e8aa9a5e23d71e45322"
	req := contractRequest{
		hash: newHash(hash),
		// tx:        &tx,
		contract:  c,
		senders:   senders,
		receivers: receivers,
		m:         &ballotCast,
	}

	h := newBallotCastHandler()

	resp, err := h.handle(ctx, req)
	if err != nil {
		t.Fatal(err)
	}

	if resp.Message != nil {
		t.Fatalf("expected nil message, result is only sent after expiration")
	}

	gotVote, ok := resp.Contract.Votes[voteHash]
	if !ok {
		t.Fatalf("Vote not found on contract")
	}

	// clear timestamps
	gotVote.CreatedAt = 0

	for k, b := range gotVote.Ballots {
		b.CreatedAt = 0
		gotVote.Ballots[k] = b
	}

	wantVote := contract.Vote{
		// Address:
		// AssetType:
		// AssetID:
		// VoteType:
		// VoteOptions:
		// VoteMax:0
		// VoteLogic:
		// ProposalDescription:
		// ProposalDocumentHash:
		VoteCutOffTimestamp: expires,
		// RefTxnIDHash:
		Ballots: []contract.Ballot{
			contract.Ballot{
				Address:   userAddr,
				AssetType: "GOO",
				AssetID:   asset.ID,
				VoteTxnID: voteHash,
				Vote:      ballotCast.Vote,
			},
		},
	}

	if !reflect.DeepEqual(gotVote, wantVote) {
		t.Fatalf("got\n%#+v\nwant\n%#+v", gotVote, wantVote)
	}
}
