package inspector

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/tokenized/pkg/bitcoin"
	"github.com/tokenized/pkg/wire"
)

// LoadFixture loads a fixture from disk and panics
func LoadFixture(name string) []byte {
	b, err := LoadFixtureWithError(name)
	if err != nil {
		panic(err)
	}

	return b
}

// LoadFixtureWithError loads a fixture from disk returns an error
func LoadFixtureWithError(name string) ([]byte, error) {
	filename := fmt.Sprintf("fixtures/%s", name)

	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	return b, nil
}

func loadFixtureTX(name string) wire.MsgTx {
	payload := LoadFixture(name)

	// strip and trailing spaces and newlines etc that editors might add
	data := strings.Trim(string(payload), "\n ")

	// setup the tx
	b, err := hex.DecodeString(data)
	if err != nil {
		panic("Failed to decode payload")
	}

	return decodeTX(b)
}

func newHash(hash string) *bitcoin.Hash32 {
	h, err := bitcoin.NewHash32FromStr(hash)
	if err != nil {
		panic(err)
	}

	return h
}

func decodeAddress(address string) bitcoin.RawAddress {
	a, err := bitcoin.DecodeAddress(address)
	if err != nil {
		panic(err)
	}

	return bitcoin.NewRawAddressFromAddress(a)
}

func decodeTX(b []byte) wire.MsgTx {
	tx := wire.MsgTx{}
	buf := bytes.NewReader(b)

	if err := tx.Deserialize(buf); err != nil {
		panic("Failed to deserialize TX")
	}

	return tx
}
