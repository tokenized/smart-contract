package request

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcutil"
	"github.com/tokenized/smart-contract/internal/platform/config"
	"github.com/tokenized/smart-contract/internal/platform/logger"
	"github.com/tokenized/smart-contract/internal/platform/state/contract"
	"github.com/tokenized/smart-contract/pkg/protocol"
	"github.com/tokenized/smart-contract/pkg/txscript"
	"github.com/tokenized/smart-contract/pkg/wire"
	"go.uber.org/zap"
)

// newSilentContext creates a Context with a no-op Logger.
func newSilentContext() context.Context {
	ctx := logger.NewContext()
	l := zap.NewNop()

	return logger.ContextWithLogger(ctx, l)
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

// newTestConfig returns a basic test Config, isolated from env vars.
func newTestConfig() config.Config {
	c := config.Config{
		ContractProviderID: "Tokenized",
		Fee: config.Fee{
			Address: decodeAddress("19fhPw9rheNT9kT4BcLsNCyZhjo1QRivd8"),
			Value:   546,
		},
	}

	return c
}

func loadFixture(name string) []byte {
	b, err := loadFixtureWithError(name)
	if err != nil {
		panic(err)
	}

	return b
}

func fixtureName(name string) string {
	dir, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	return filepath.Join(dir, "fixtures", name)
}

func loadFixtureWithError(name string) ([]byte, error) {
	b, err := ioutil.ReadFile(fixtureName(name))
	if err != nil {
		return nil, err
	}

	return b, nil
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

	return decodeTX(b)
}

func decodeTX(b []byte) wire.MsgTx {
	tx := wire.MsgTx{}
	buf := bytes.NewReader(b)

	if err := tx.Deserialize(buf); err != nil {
		panic("Failed to deserialize TX")
	}

	return tx
}

// isCloseTo returns true if n is within +/- delta of v, false otherwise.
func isCloseTo(n, v, delta int64) bool {
	return isBetween(n, v-delta, v+delta)
}

// isBetween returns true if n is >= min and <= max.
func isBetween(n, min, max int64) bool {
	return n >= min && n <= max
}

func loadMessageFromTX(tx wire.MsgTx) protocol.OpReturnMessage {
	for _, txOut := range tx.TxOut {
		m, err := newOpReturnMessage(txOut)
		if err != nil {
			panic("Broken OP_RETURN message")
		}

		if m == nil {
			// this isn't something we are interested in
			continue
		}

		return m
	}

	panic("No tokenized OP_RETURN")
}

func newOpReturnMessage(txOut *wire.TxOut) (protocol.OpReturnMessage, error) {
	if txOut.PkScript[0] != txscript.OP_RETURN {
		return nil, errors.New("Payload is not an OP_RETURN")
	}

	return protocol.New(txOut.PkScript)
}

func zeroTimestamps(c *contract.Contract) {
	c.CreatedAt = 0

	for k, a := range c.Assets {
		a.CreatedAt = 0
		c.Assets[k] = a

		for hk, h := range a.Holdings {
			h.CreatedAt = 0
			c.Assets[k].Holdings[hk] = h
		}
	}
}
