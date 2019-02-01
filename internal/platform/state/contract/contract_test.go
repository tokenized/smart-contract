package contract

import (
	"encoding/hex"
	"reflect"
	"testing"
	"time"

	"github.com/tokenized/smart-contract/pkg/protocol"
	"github.com/tokenized/smart-contract/pkg/wire"
)

func TestContract_NewContract(t *testing.T) {
	var tx *wire.MsgTx

	issuer := decodeAddress("1DNTgNSWtTestKs7j1DwaoxmSc4q9sEUsb")

	contractAddress := decodeAddress("1Cessj8TyzEypaVzp9V8oZhiMLokVDNSR5")

	cfh, err := hex.DecodeString("5891b5b522d5df086d0ff0b110fbd9d21bb4fc7163af34d08286a2e846f6be03")
	if err != nil {
		t.Fatal(err)
	}

	m := protocol.NewContractFormation()
	m.ContractFileHash = cfh

	c := NewContract(tx, contractAddress, issuer, nil)
	c = EditContract(c, &m)

	if c.CreatedAt == 0 {
		t.Errorf("got 0, but want a timestamp")
	}

	// clear the timestamp
	c.CreatedAt = 0

	wantContract := Contract{
		ID:               "1Cessj8TyzEypaVzp9V8oZhiMLokVDNSR5",
		IssuerAddress:    "1DNTgNSWtTestKs7j1DwaoxmSc4q9sEUsb",
		ContractFileHash: "5891b5b522d5df086d0ff0b110fbd9d21bb4fc7163af34d08286a2e846f6be03",
		Assets:           map[string]Asset{},
		Votes:            map[string]Vote{},
		Hashes:           []string{},
	}

	if !reflect.DeepEqual(*c, wantContract) {
		t.Errorf("got\n%#+v\nwant\n%#+v", *c, wantContract)
	}
}

func TestContract_IsIssuer(t *testing.T) {
	contractAddr := "1Cy7znvXpwTZZG5iqiZoMYtQXfThbLBadf"
	issuerAddr := "1CmQLd5vRdcvqXFaCeeLTcXZVHXzSzgscv"
	randomAddress := "13FzCGiNWaUHCWGvuLobWM7iaNyP3TJAJg"

	contract := Contract{
		ID:            contractAddr,
		IssuerAddress: issuerAddr,
	}

	tests := []struct {
		name    string
		address string
		want    bool
	}{
		{
			name:    "issuer",
			address: issuerAddr,
			want:    true,
		},
		{
			name:    "negative",
			address: randomAddress,
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := contract.IsIssuer(tt.address)
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestContract_IsOwner(t *testing.T) {
	contractAddr := "1Cy7znvXpwTZZG5iqiZoMYtQXfThbLBadf"
	issuerAddr := "1CmQLd5vRdcvqXFaCeeLTcXZVHXzSzgscv"
	assetID := "w840mxhrhupngqthd9quwtgsocaonv2f"

	// the address of the "User" (not the issuer)
	userAddress := "13FzCGiNWaUHCWGvuLobWM7iaNyP3TJAJg"

	contract := Contract{
		ID:            contractAddr,
		IssuerAddress: issuerAddr,
		Qty:           20,
		Assets: map[string]Asset{
			assetID: Asset{
				Holdings: map[string]Holding{
					issuerAddr: Holding{
						Address: issuerAddr,
						Balance: 19,
					},
					userAddress: Holding{
						Address: userAddress,
						Balance: 1,
					},
				},
			},
		},
	}

	tests := []struct {
		name    string
		address string
		want    bool
	}{
		{
			name:    "issuer",
			address: issuerAddr,
			want:    true,
		},
		{
			name:    "user",
			address: userAddress,
			want:    true,
		},
		{
			name:    "negative",
			address: "foo",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := contract.IsOwner(tt.address)
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestContract_CanVote(t *testing.T) {
	// the addresses we will receive in the outs of the TX
	contractAddr := "1CWjudGPuj1sHs3GuMkAGPEUP5YaJNqu8U"
	issuerAddr := "1HwvXtVEMDuvbrNHQCwWaV97ucBLr3zCgJ"
	userAddr := "1L9Vr7BCEeczDtSJiX3fHLG5VVQgHtB22o"
	unknownAddr := "1LmehXKRLdg28U1ZFzReuZwnvXY2xf1yYP"

	// build the existing asset
	asset := Asset{
		ID:   "1v2mwouuzz2x73ulv6o57llbx5udym6l",
		Qty:  43,
		Type: "GOO",
		Holdings: map[string]Holding{
			issuerAddr: Holding{
				Address: issuerAddr,
				Balance: 43,
			},
			userAddr: Holding{
				Address: issuerAddr,
				Balance: 43,
			},
		},
	}

	voteHash := "d2b2db94192a0e80f87fffe60c4a8b1b224f80b0ae46d0563f72e25c49b93758"

	// setup the existing contract
	expires := time.Now().Add(time.Hour * 1).UnixNano()

	contract := Contract{
		ID: contractAddr,
		Assets: map[string]Asset{
			asset.ID: asset,
		},
		Votes: map[string]Vote{
			voteHash: Vote{
				VoteCutOffTimestamp: expires,
			},
		},
		Qty: 1,
	}

	tests := []struct {
		name   string
		vote   Vote
		ballot Ballot
		want   uint8
	}{
		{
			name: "issuer",
			vote: Vote{
				VoteCutOffTimestamp: expires,
			},
			ballot: Ballot{
				Address: issuerAddr,
			},
			want: 0,
		},
		{
			name: "user",
			vote: Vote{
				VoteCutOffTimestamp: expires,
			},
			ballot: Ballot{
				Address: userAddr,
			},
			want: 0,
		},
		{
			name: "unknown address",
			vote: Vote{
				VoteCutOffTimestamp: expires,
			},
			ballot: Ballot{
				Address: unknownAddr,
			},
			want: protocol.RejectionCodeUnknownAddress,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := contract.CanVote(tt.vote, tt.ballot)

			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestContract_getUsers(t *testing.T) {
	// the addresses we will receive in the outs of the TX
	contractAddr := "1CWjudGPuj1sHs3GuMkAGPEUP5YaJNqu8U"
	issuerAddr := "1HwvXtVEMDuvbrNHQCwWaV97ucBLr3zCgJ"
	userAddr := "1L9Vr7BCEeczDtSJiX3fHLG5VVQgHtB22o"

	// build the existing asset
	asset := Asset{
		ID:   "1v2mwouuzz2x73ulv6o57llbx5udym6l",
		Qty:  43,
		Type: "GOO",
		Holdings: map[string]Holding{
			issuerAddr: Holding{
				Address: issuerAddr,
				Balance: 43,
			},
			userAddr: Holding{
				Address: issuerAddr,
				Balance: 43,
			},
		},
	}

	contract := Contract{
		ID: contractAddr,
		Assets: map[string]Asset{
			asset.ID: asset,
		},
	}

	got := contract.getUsers()

	want := []string{
		issuerAddr,
		userAddr,
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got\n%+v\nwant\n%+v", got, want)
	}
}
