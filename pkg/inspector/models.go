package inspector

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/pkg/errors"
	"github.com/tokenized/smart-contract/pkg/bitcoin"
	"github.com/tokenized/smart-contract/pkg/wire"
)

type Input struct {
	Address bitcoin.Address
	Index   uint32
	Value   int64
	UTXO    UTXO
	FullTx  *wire.MsgTx
}

type Output struct {
	Address bitcoin.Address
	Index   uint32
	Value   int64
	UTXO    UTXO
}

type UTXO struct {
	Hash     chainhash.Hash
	Index    uint32
	PkScript []byte
	Value    int64
}

func (u UTXO) ID() string {
	return fmt.Sprintf("%v:%v", u.Hash, u.Index)
}

func (u UTXO) PublicAddress() (bitcoin.Address, error) {
	return bitcoin.AddressFromLockingScript(u.PkScript)
}

func (u UTXO) Address() (bitcoin.Address, error) {
	return bitcoin.AddressFromLockingScript(u.PkScript)
}

// UTXOs is a wrapper for a []UTXO.
type UTXOs []UTXO

// Value returns the total value of the set of UTXO's.
func (u UTXOs) Value() int64 {
	v := int64(0)

	for _, utxo := range u {
		v += utxo.Value
	}

	return v
}

// ForAddress returns UTXOs that match the given Address.
func (u UTXOs) ForAddress(address bitcoin.Address) (UTXOs, error) {
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
	if in.FullTx == nil {
		return errors.New("No input tx")
	}
	if err := in.FullTx.Serialize(buf); err != nil {
		return err
	}

	if err := binary.Write(buf, binary.LittleEndian, &in.Index); err != nil {
		return err
	}

	if err := binary.Write(buf, binary.LittleEndian, &in.Value); err != nil {
		return err
	}

	if err := in.UTXO.Write(buf); err != nil {
		return err
	}

	return nil
}

func (in *Input) Read(buf *bytes.Buffer) error {
	msg := wire.MsgTx{}
	if err := msg.Deserialize(buf); err != nil {
		return err
	}
	in.FullTx = &msg

	if err := binary.Read(buf, binary.LittleEndian, &in.Index); err != nil {
		return err
	}

	if err := binary.Read(buf, binary.LittleEndian, &in.Value); err != nil {
		return err
	}

	if err := in.UTXO.Read(buf); err != nil {
		return err
	}

	// Calculate address
	var err error
	in.Address, err = in.UTXO.PublicAddress()
	if err != nil {
		return err
	}

	return nil
}

func (utxo *UTXO) Write(buf *bytes.Buffer) error {
	if _, err := buf.Write(utxo.Hash[:]); err != nil {
		return err
	}

	scriptSize := uint32(len(utxo.PkScript))
	if err := binary.Write(buf, binary.LittleEndian, &scriptSize); err != nil {
		return err
	}

	if _, err := buf.Write(utxo.PkScript); err != nil {
		return err
	}

	if err := binary.Write(buf, binary.LittleEndian, &utxo.Index); err != nil {
		return err
	}

	if err := binary.Write(buf, binary.LittleEndian, &utxo.Value); err != nil {
		return err
	}

	return nil
}

func (utxo *UTXO) Read(buf *bytes.Buffer) error {
	if _, err := buf.Read(utxo.Hash[:]); err != nil {
		return err
	}

	var scriptSize uint32
	if err := binary.Read(buf, binary.LittleEndian, &scriptSize); err != nil {
		return err
	}

	utxo.PkScript = make([]byte, int(scriptSize))
	if _, err := buf.Read(utxo.PkScript); err != nil {
		return err
	}

	if err := binary.Read(buf, binary.LittleEndian, &utxo.Index); err != nil {
		return err
	}

	if err := binary.Read(buf, binary.LittleEndian, &utxo.Value); err != nil {
		return err
	}

	return nil
}
