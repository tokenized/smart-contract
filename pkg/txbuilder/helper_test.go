package txbuilder

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/tokenized/smart-contract/pkg/wire"
	"go.uber.org/zap"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcutil"
	"github.com/tokenized/smart-contract/internal/platform/logger"
)

// newSilentContext creates a Context with a no-op Logger.
func newSilentContext() context.Context {
	ctx := logger.NewContext()
	l := zap.NewNop()

	return logger.ContextWithLogger(ctx, l)
}

func loadFixture(name string) []byte {
	filename := fmt.Sprintf("fixtures/%s", name)

	b, err := ioutil.ReadFile(filename)
	if err != nil {
		panic(err)
	}

	return b
}

func loadFixtureTX(name string) wire.MsgTx {
	payload := loadFixture(name)

	// strip and trailing spaces and newlines etc that editors might add
	data := strings.Trim(string(payload), "\n ")

	// setup the tx
	b, err := hex.DecodeString(data)
	if err != nil {
		panic("Failed to decode payload")
	}

	tx := wire.MsgTx{}
	buf := bytes.NewReader(b)

	if err := tx.Deserialize(buf); err != nil {
		panic("Failed to deserialize TX")
	}

	return tx
}

func dumpRaw(tx *wire.MsgTx) {
	// all signed and ready to go
	var buf bytes.Buffer
	tx.Serialize(&buf)

	out := fmt.Sprintf("%#x", buf.Bytes())
	fmt.Printf("%s\n", strings.Replace(out, "0x", "", 1))
}

func getRaw(tx *wire.MsgTx) string {
	// all signed and ready to go
	var buf bytes.Buffer
	tx.Serialize(&buf)

	out := fmt.Sprintf("%#x", buf.Bytes())
	return fmt.Sprintf("%s", strings.Replace(out, "0x", "", 1))
}

func decodeTX(payload string) wire.MsgTx {
	// strip and trailing spaces and newlines etc that editors might add
	data := strings.Trim(string(payload), "\n ")

	// setup the tx
	b, err := hex.DecodeString(data)
	if err != nil {
		panic("Failed to decode payload")
	}

	tx := wire.MsgTx{}
	buf := bytes.NewReader(b)

	if err := tx.Deserialize(buf); err != nil {
		panic("Failed to deserialize TX")
	}

	return tx
}

func newHash(hash string) chainhash.Hash {
	h, err := chainhash.NewHashFromStr(hash)
	if err != nil {
		panic(err)
	}

	return *h
}

func decodeAddress(address string) btcutil.Address {
	a, err := btcutil.DecodeAddress(address, &chaincfg.MainNetParams)
	if err != nil {
		panic(err)
	}

	return a
}
