package request

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/tokenized/smart-contract/pkg/protocol"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
)

func TestIsTokenizedOpReturn(t *testing.T) {
	hash := "d4f9f93735d5008725373a26b7d8ffb749bb6c9d47fc063a2391b1155cf8e7a5"
	tx := loadFixtureTX(fmt.Sprintf("%v.txt", hash))

	pkScript := tx.TxOut[1].PkScript

	got := isTokenizedOpReturn(pkScript)

	want := true

	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestBuildContractFormationFromContractAmendment(t *testing.T) {
	cin := Contract{
		Revision: 41,
	}

	m := protocol.NewContractAmendment()

	c, cf := buildContractFormationFromContractAmendment(cin, &m)

	wantRevision := uint16(42)
	if c.Revision != wantRevision {
		t.Errorf("got %v, want %v", c.Revision, wantRevision)
	}

	if cf.ContractRevision != wantRevision {
		t.Errorf("got %v, want %v", cf.ContractRevision, wantRevision)
	}
}

func TestBuildResultFromVoteResult(t *testing.T) {
	vote := Vote{
		AssetType:    "RRE",
		AssetID:      "home",
		VoteType:     'X',
		RefTxnIDHash: "abc",
		VoteOptions:  []byte("ABCDEFGHIJKLMNOP"),
	}

	result := NewBallotResult()
	result['A'] = 10
	result['B'] = 2
	result['C'] = 1
	result['D'] = 1
	result['E'] = 1
	result['F'] = 1
	result['G'] = 1
	result['H'] = 1
	result['I'] = 1
	result['J'] = 1
	result['K'] = 1
	result['L'] = 1
	result['M'] = 1
	result['N'] = 1
	result['O'] = 1
	result['P'] = 1

	vote.Result = &result

	r := buildResultFromVoteResult(vote)

	wantResult := protocol.NewResult()
	wantResult.AssetType = []byte(vote.AssetType)
	wantResult.AssetID = []byte(vote.AssetID)
	wantResult.VoteType = vote.VoteType
	wantResult.VoteTxnID = []byte(vote.RefTxnIDHash)
	wantResult.Result = []byte{
		65, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0,
	}

	if !reflect.DeepEqual(r, wantResult) {
		t.Fatalf("got\n%#+v\nwant\n%#+v", r, wantResult)
	}
}

func TestBuildConfiscationFromOrder(t *testing.T) {
	order := protocol.NewOrder()
	order.AssetType = []byte("SHC")
	order.AssetID = []byte("74mnew8naetym9sofsf5rgo958m07b54")
	order.Message = []byte("wat")

	txhash := "4acfc13eed7b08b5b4f0e4ac6673df247af0950513167d9cfaa7af739b241bd6"
	h, _ := chainhash.NewHashFromStr(txhash)

	c := buildConfiscationFromOrder(&order, *h)

	want := protocol.NewConfiscation()
	want.AssetID = order.AssetID
	want.AssetType = order.AssetType
	want.Message = order.Message
	want.RefTxnID = []byte{
		74, 207, 193, 62, 237, 123, 8, 181,
		180, 240, 228, 172, 102, 115, 223, 36,
		122, 240, 149, 5, 19, 22, 125, 156,
		250, 167, 175, 115, 155, 36, 27, 214,
	}

	if !reflect.DeepEqual(c, want) {
		t.Fatalf("got\n%#+v\nwant\n%#+v", c, want)
	}
}

func TestHashToBytes(t *testing.T) {
	txhash := "4acfc13eed7b08b5b4f0e4ac6673df247af0950513167d9cfaa7af739b241bd6"
	h, _ := chainhash.NewHashFromStr(txhash)

	b := hashToBytes(*h)

	want := []byte{
		74, 207, 193, 62, 237, 123, 8, 181,
		180, 240, 228, 172, 102, 115, 223, 36,
		122, 240, 149, 5, 19, 22, 125, 156,
		250, 167, 175, 115, 155, 36, 27, 214,
	}

	if !reflect.DeepEqual(b, want) {
		t.Fatalf("got\n%x\nwant\n%x", b, want)
	}
}

func TestBuildAssetCreationFromAssetModification(t *testing.T) {
	m := protocol.NewAssetModification()
	m.AssetRevision = 41

	ac := buildAssetCreationFromAssetModification(&m)

	wantRevision := uint16(42)
	if ac.AssetRevision != wantRevision {
		t.Errorf("got %v, want %v", ac.AssetRevision, wantRevision)
	}
}
