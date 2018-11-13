package vote

import (
	"reflect"
	"testing"

	"github.com/tokenized/smart-contract/internal/app/state/contract"
)

func TestVoteService_generateResult(t *testing.T) {
	// the assetID that the "User" is creating a Vote on
	assetID := "w840mxhrhupngqthd9quwtgsocaonv2f"

	wrongAssetID := "FOO"

	// the address of the "User" (not the issuer)
	userAddress := "13FzCGiNWaUHCWGvuLobWM7iaNyP3TJAJg"

	// the address of the issuer
	issuerAddr := "1CmQLd5vRdcvqXFaCeeLTcXZVHXzSzgscv"

	otherUserAddress := "1DnoezsMcKZeQrXVW7eqU5v8HRKmnPSYd2"

	c := contract.Contract{
		Assets: map[string]contract.Asset{
			assetID: contract.Asset{
				Holdings: map[string]contract.Holding{
					issuerAddr: contract.Holding{
						Address: issuerAddr,
						Balance: 15,
					},
					userAddress: contract.Holding{
						Address: userAddress,
						Balance: 5,
					},
				},
			},
		},
	}

	tests := []struct {
		name     string
		contract contract.Contract
		vote     contract.Vote
		want     contract.BallotResult
	}{
		{
			// "Y" = 89 (0x59), "N" = 78 (0x4e)
			name:     "yes no vote (YN)",
			contract: c,
			vote: contract.Vote{
				Address: assetID,
				VoteOptions: []byte{
					0x59, 0x4c, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
				},
				VoteLogic: '0',
				VoteMax:   1,
				Ballots: []contract.Ballot{
					contract.Ballot{
						Address: userAddress,
						AssetID: assetID,
						Vote:    []byte{0x59},
					},
				},
			},
			want: contract.BallotResult{
				0x59: 5,
			},
		},
		{
			// "Y" = 89 (0x59), "N" = 78 (0x4e)
			name:     "yes no vote (YN), one rejected",
			contract: c,
			vote: contract.Vote{
				Address: assetID,
				VoteOptions: []byte{
					0x59, 0x4c, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
				},
				VoteLogic: '0',
				VoteMax:   1,
				Ballots: []contract.Ballot{
					contract.Ballot{
						Address: userAddress,
						AssetID: assetID,
						Vote:    []byte{0x59},
					},
					contract.Ballot{
						Address: otherUserAddress,
						AssetID: wrongAssetID,
						Vote:    []byte{0x59},
					},
				},
			},
			want: contract.BallotResult{
				0x59: 5,
			},
		},
		{
			// "Y" = 89 (0x59), "N" = 78 (0x4e)
			name:     "yes no vote (YN), wrong option",
			contract: c,
			vote: contract.Vote{
				Address: assetID,
				VoteOptions: []byte{
					0x59, 0x4c, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
				},
				VoteLogic: '0',
				VoteMax:   1,
				Ballots: []contract.Ballot{
					contract.Ballot{
						Address: userAddress,
						AssetID: assetID,
						Vote:    []byte{0x59},
					},
					contract.Ballot{
						Address: otherUserAddress,
						AssetID: assetID,
						Vote:    []byte{0x03},
					},
				},
			},
			want: contract.BallotResult{
				0x59: 5,
			},
		},
		{
			// Options : ABCDEFGHIJKLMNOP, 2 max choices
			name:     "16 selections, 2 max",
			contract: c,
			vote: contract.Vote{
				Address: assetID,
				VoteOptions: []byte{
					65, 66, 67, 68, 69, 70, 71, 72,
					73, 74, 75, 76, 77, 78, 79, 80,
				},
				VoteLogic: '0',
				VoteMax:   2,
				Ballots: []contract.Ballot{
					contract.Ballot{
						Address: userAddress,
						AssetID: assetID,
						Vote:    []byte{73, 65},
					},
				},
			},
			want: contract.BallotResult{
				0x49: 5,
				0x41: 5,
			},
		},
		{
			// Options : ABCDEFGHIJKLMNOP, 2 max choices
			name:     "16 selections, 2 max, weighted voting",
			contract: c,
			vote: contract.Vote{
				Address: assetID,
				VoteOptions: []byte{
					65, 66, 67, 68, 69, 70, 71, 72,
					73, 74, 75, 76, 77, 78, 79, 80,
				},
				VoteLogic: '1',
				VoteMax:   4,
				Ballots: []contract.Ballot{
					contract.Ballot{
						Address: userAddress,
						AssetID: assetID,
						Vote:    []byte{80, 79, 78, 77},
					},
				},
			},
			want: contract.BallotResult{
				80: 20,
				79: 15,
				78: 10,
				77: 5,
			},
		},
		{
			// Options : ABCDEFGHIJKLMNOP, 2 max choices
			name:     "16 selections, 2 max, weighted voting, contract vote",
			contract: c,
			vote: contract.Vote{
				VoteOptions: []byte{
					65, 66, 67, 68, 69, 70, 71, 72,
					73, 74, 75, 76, 77, 78, 79, 80,
				},
				VoteLogic: '1',
				VoteMax:   4,
				Ballots: []contract.Ballot{
					contract.Ballot{
						Address: userAddress,
						AssetID: assetID,
						Vote:    []byte{80, 79, 78, 77},
					},
				},
			},
			want: contract.BallotResult{
				80: 20,
				79: 15,
				78: 10,
				77: 5,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewVoteService()

			result := s.generateResult(tt.contract, tt.vote)

			if !reflect.DeepEqual(result, tt.want) {
				t.Errorf("got\n%#+v\nwant\n%#+v", result, tt.want)
			}
		})
	}
}
