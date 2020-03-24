package inspector

import (
	"bytes"

	"github.com/tokenized/smart-contract/pkg/bitcoin"
	"github.com/tokenized/smart-contract/pkg/wire"
)

type Input struct {
	Address bitcoin.RawAddress
	UTXO    bitcoin.UTXO
}

type Output struct {
	Address bitcoin.RawAddress
	UTXO    bitcoin.UTXO
}

// UTXOs is a wrapper for a []UTXO.
type UTXOs []bitcoin.UTXO

// Value returns the total value of the set of UTXO's.
func (u UTXOs) Value() uint64 {
	v := uint64(0)

	for _, utxo := range u {
		v += utxo.Value
	}

	return v
}

// ForAddress returns UTXOs that match the given Address.
func (u UTXOs) ForAddress(address bitcoin.RawAddress) (UTXOs, error) {
	filtered := UTXOs{}

	for _, utxo := range u {
		utxoAddress, err := utxo.Address()
		if err != nil {
			continue
		}

		if !address.Equal(utxoAddress) {
			continue
		}

		filtered = append(filtered, utxo)
	}

	return filtered, nil
}

func (in *Input) Write(buf *bytes.Buffer) error {
	if err := in.UTXO.Write(buf); err != nil {
		return err
	}

	return nil
}

func (in *Input) Read(version uint8, buf *bytes.Reader) error {
	if version == 0 {
		// Read full tx
		msg := wire.MsgTx{}
		if err := msg.Deserialize(buf); err != nil {
			return err
		}
	}

	if err := in.UTXO.Read(buf); err != nil {
		return err
	}

	// Calculate address
	var err error
	in.Address, err = in.UTXO.Address()
	if err != nil {
		return err
	}

	return nil
}
