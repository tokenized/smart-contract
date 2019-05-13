package client

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/tokenized/smart-contract/pkg/logger"
	"github.com/tokenized/smart-contract/pkg/wire"

	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcutil"
	"github.com/pkg/errors"
	"golang.org/x/crypto/ripemd160"
)

type Wallet struct {
	Key           *btcec.PrivateKey
	PublicKeyHash []byte
	outputs       []Output
	path          string
}

type Output struct {
	OutPoint    wire.OutPoint
	PkScript    []byte
	Value       uint64
	SpentByTxId chainhash.Hash
}

func BitcoinsFromSatoshis(satoshis uint64) float32 {
	return float32(satoshis) / 100000000.0
}

func (wallet *Wallet) Balance() uint64 {
	var emptyHash chainhash.Hash
	result := uint64(0)
	for _, output := range wallet.outputs {
		if output.SpentByTxId == emptyHash {
			result += output.Value
		}
	}
	return result
}

func (wallet *Wallet) UnspentOutputs() []*Output {
	var emptyHash chainhash.Hash
	result := make([]*Output, 0, len(wallet.outputs))
	for i, output := range wallet.outputs {
		if output.SpentByTxId == emptyHash {
			result = append(result, &wallet.outputs[i])
		}
	}
	return result
}

func (wallet *Wallet) Spend(outpoint *wire.OutPoint, spentByTxId chainhash.Hash) (uint64, bool) {
	var emptyHash chainhash.Hash
	for i, output := range wallet.outputs {
		if !bytes.Equal(output.OutPoint.Hash[:], outpoint.Hash[:]) ||
			output.OutPoint.Index != outpoint.Index {
			continue
		}

		if output.SpentByTxId != emptyHash {
			return output.Value, false // Already spent
		}

		wallet.outputs[i].SpentByTxId = spentByTxId
		return output.Value, true
	}
	return 0, false
}

func (wallet *Wallet) AddUTXO(txId chainhash.Hash, index uint32, script []byte, value uint64) bool {
	for _, output := range wallet.outputs {
		if bytes.Equal(txId[:], output.OutPoint.Hash[:]) &&
			index == output.OutPoint.Index {
			return false
		}
	}

	newOutput := Output{
		OutPoint: wire.OutPoint{Hash: txId, Index: index},
		PkScript: script,
		Value:    uint64(value),
	}
	wallet.outputs = append(wallet.outputs, newOutput)
	return true
}

func (wallet *Wallet) Address(net *chaincfg.Params) (btcutil.Address, error) {
	return btcutil.NewAddressPubKeyHash(wallet.PublicKeyHash, net)
}

func (wallet *Wallet) Load(ctx context.Context, wifKey, path string, net *chaincfg.Params) error {
	wif, err := btcutil.DecodeWIF(wifKey)
	if err != nil {
		return errors.Wrap(err, "Failed to decode wallet key")
	}

	// Private Key
	wallet.Key = wif.PrivKey

	// Pub Key Hash
	hash256 := sha256.New()
	hash160 := ripemd160.New()
	hash256.Write(wallet.Key.PubKey().SerializeCompressed())
	hash160.Write(hash256.Sum(nil))
	wallet.PublicKeyHash = hash160.Sum(nil)

	// Load Outputs
	wallet.path = path
	utxoFilePath := filepath.FromSlash(path + "/outputs.json")
	data, err := ioutil.ReadFile(utxoFilePath)
	if err == nil {
		if err := json.Unmarshal(data, &wallet.outputs); err != nil {
			return errors.Wrap(err, "Failed to unmarshal wallet outputs")
		}
	} else if !os.IsNotExist(err) {
		wallet.outputs = nil
		return errors.Wrap(err, "Failed to read wallet outputs")
	}

	var emptyHash chainhash.Hash
	unspentCount := 0
	for _, output := range wallet.outputs {
		if output.SpentByTxId == emptyHash {
			logger.Info(ctx, "Loaded unspent UTXO %.08f : %s", BitcoinsFromSatoshis(output.Value),
				output.OutPoint.Hash)
			unspentCount++
		}
	}

	logger.Info(ctx, "Loaded wallet with %d outputs, %d unspent, and balance of %.08f",
		len(wallet.outputs), unspentCount, BitcoinsFromSatoshis(wallet.Balance()))

	address, err := wallet.Address(net)
	if err != nil {
		return errors.Wrap(err, "Failed to get wallet address")
	}
	logger.Info(ctx, "Wallet address : %s", address)
	return nil
}

func (wallet *Wallet) Save(ctx context.Context) error {
	utxoFilePath := filepath.FromSlash(wallet.path + "/outputs.json")
	data, err := json.Marshal(&wallet.outputs)
	if err != nil {
		return errors.Wrap(err, "Failed to marshal wallet outputs")
	}

	err = ioutil.WriteFile(utxoFilePath, data, 0644)
	if err != nil {
		return errors.Wrap(err, "Failed to write wallet outputs")
	}

	logger.Info(ctx, "Saved wallet with %d outputs and balance of %.08f", len(wallet.outputs),
		BitcoinsFromSatoshis(wallet.Balance()))
	return nil
}
