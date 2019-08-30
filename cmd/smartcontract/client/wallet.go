package client

import (
	"bytes"
	"context"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/tokenized/smart-contract/pkg/bitcoin"
	"github.com/tokenized/smart-contract/pkg/json"
	"github.com/tokenized/smart-contract/pkg/logger"
	"github.com/tokenized/smart-contract/pkg/wire"

	"github.com/pkg/errors"
)

type Wallet struct {
	Key     bitcoin.Key
	Address bitcoin.RawAddress
	outputs []Output
	path    string
}

type Output struct {
	OutPoint    wire.OutPoint
	PkScript    []byte
	Value       uint64
	SpentByTxId *bitcoin.Hash32
}

func BitcoinsFromSatoshis(satoshis uint64) float32 {
	return float32(satoshis) / 100000000.0
}

var emptyHash bitcoin.Hash32

func (wallet *Wallet) Balance() uint64 {
	result := uint64(0)
	for _, output := range wallet.outputs {
		if emptyHash.Equal(output.SpentByTxId) {
			result += output.Value
		}
	}
	return result
}

func (wallet *Wallet) UnspentOutputs() []*Output {
	result := make([]*Output, 0, len(wallet.outputs))
	for i, output := range wallet.outputs {
		if emptyHash.Equal(output.SpentByTxId) {
			result = append(result, &wallet.outputs[i])
		}
	}
	return result
}

func (wallet *Wallet) Spend(outpoint *wire.OutPoint, spentByTxId *bitcoin.Hash32) (uint64, bool) {
	for i, output := range wallet.outputs {
		if !bytes.Equal(output.OutPoint.Hash[:], outpoint.Hash[:]) ||
			output.OutPoint.Index != outpoint.Index {
			continue
		}

		if !emptyHash.Equal(output.SpentByTxId) {
			return output.Value, false // Already spent
		}

		wallet.outputs[i].SpentByTxId = spentByTxId
		return output.Value, true
	}
	return 0, false
}

func (wallet *Wallet) Unspend(outpoint *wire.OutPoint, spentByTxId *bitcoin.Hash32) (uint64, bool) {
	for i, output := range wallet.outputs {
		if !bytes.Equal(output.OutPoint.Hash[:], outpoint.Hash[:]) ||
			output.OutPoint.Index != outpoint.Index {
			continue
		}

		if emptyHash.Equal(output.SpentByTxId) {
			return output.Value, false // Not spent
		}

		wallet.outputs[i].SpentByTxId = &emptyHash
		return output.Value, true
	}
	return 0, false
}

func (wallet *Wallet) AddUTXO(txid *bitcoin.Hash32, index uint32, script []byte, value uint64) bool {
	for _, output := range wallet.outputs {
		if bytes.Equal(txid[:], output.OutPoint.Hash[:]) &&
			index == output.OutPoint.Index {
			return false
		}
	}

	newOutput := Output{
		OutPoint:    wire.OutPoint{Hash: *txid, Index: index},
		PkScript:    script,
		Value:       uint64(value),
		SpentByTxId: &bitcoin.Hash32{},
	}
	wallet.outputs = append(wallet.outputs, newOutput)
	return true
}

func (wallet *Wallet) RemoveUTXO(txid *bitcoin.Hash32, index uint32, script []byte, value uint64) bool {
	for i, output := range wallet.outputs {
		if txid.Equal(&output.OutPoint.Hash) &&
			index == output.OutPoint.Index {
			wallet.outputs = append(wallet.outputs[:i], wallet.outputs[i+1:]...)
			return true
		}
	}

	return false
}

func (wallet *Wallet) Load(ctx context.Context, wifKey, path string, net bitcoin.Network) error {
	// Private Key
	var err error
	wallet.Key, err = bitcoin.DecodeKeyString(wifKey)
	if !bitcoin.DecodeNetMatches(wallet.Key.Network(), net) {
		return errors.New("Incorrect network encoding")
	}

	// Pub Key Hash Address
	wallet.Address, err = bitcoin.NewRawAddressPKH(bitcoin.Hash160(wallet.Key.PublicKey().Bytes()))
	if err != nil {
		return err
	}

	// Load Outputs
	wallet.path = path
	utxoFilePath := filepath.Join(filepath.FromSlash(path), "outputs.json")
	data, err := ioutil.ReadFile(utxoFilePath)
	if err == nil {
		if err := json.Unmarshal(data, &wallet.outputs); err != nil {
			return errors.Wrap(err, "Failed to unmarshal wallet outputs")
		}
	} else if !os.IsNotExist(err) {
		wallet.outputs = nil
		return errors.Wrap(err, "Failed to read wallet outputs")
	}

	unspentCount := 0
	for _, output := range wallet.outputs {
		if emptyHash.Equal(output.SpentByTxId) {
			logger.Info(ctx, "Loaded unspent output %.08f : %s", BitcoinsFromSatoshis(output.Value),
				output.OutPoint.String())
			unspentCount++
		}
	}

	logger.Info(ctx, "Loaded wallet with %d outputs, %d unspent, and balance of %.08f",
		len(wallet.outputs), unspentCount, BitcoinsFromSatoshis(wallet.Balance()))

	logger.Info(ctx, "Wallet address : %x", wallet.Address.Bytes())
	return nil
}

func (wallet *Wallet) Save(ctx context.Context) error {
	utxoFilePath := filepath.Join(filepath.FromSlash(wallet.path), "outputs.json")
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
