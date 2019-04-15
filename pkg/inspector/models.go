package inspector

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcutil"
	"github.com/tokenized/smart-contract/pkg/wire"
)

type Input struct {
	Address btcutil.Address
	Index   uint32
	Value   int64
	UTXO    UTXO
	FullTx  *wire.MsgTx
}

type Output struct {
	Address btcutil.Address
	Index   uint32
	Value   int64
	UTXO    UTXO
}

type UTXO struct {
	Hash     chainhash.Hash
	PkScript []byte
	Index    uint32
	Value    int64
}

func (u UTXO) ID() string {
	return fmt.Sprintf("%v:%v", u.Hash, u.Index)
}

func (u UTXO) PublicAddress(params *chaincfg.Params) (btcutil.Address, error) {
	if len(u.PkScript) != 25 &&
		u.PkScript[0] != txscript.OP_DUP &&
		u.PkScript[1] != txscript.OP_HASH160 {

		return nil, fmt.Errorf("Invalid pkScript %s : %v", u.Hash, u.PkScript)
	}

	return btcutil.NewAddressPubKeyHash(u.PkScript[3:23], params)
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
func (u UTXOs) ForAddress(address btcutil.Address) (UTXOs, error) {
	filtered := UTXOs{}

	s := address.String()

	for _, utxo := range u {
		publicAddress, err := utxo.PublicAddress(&chaincfg.MainNetParams)
		if err != nil {
			continue
		}

		if publicAddress.String() != s {
			continue
		}

		filtered = append(filtered, utxo)
	}

	return filtered, nil
}

func (in *Input) Write(buf *bytes.Buffer) error {
	if err := in.FullTx.Serialize(buf); err != nil {
		return err
	}

	if err := binary.Write(buf, binary.BigEndian, &in.Index); err != nil {
		return err
	}

	if err := binary.Write(buf, binary.BigEndian, &in.Value); err != nil {
		return err
	}

	if err := in.UTXO.Write(buf); err != nil {
		return err
	}

	return nil
}

func (in *Input) Read(buf *bytes.Buffer, netParams *chaincfg.Params) error {
	msg := wire.MsgTx{}
	if err := msg.Deserialize(buf); err != nil {
		return err
	}
	in.FullTx = &msg

	if err := binary.Read(buf, binary.BigEndian, &in.Index); err != nil {
		return err
	}

	if err := binary.Read(buf, binary.BigEndian, &in.Value); err != nil {
		return err
	}

	if err := in.UTXO.Read(buf); err != nil {
		return err
	}

	// Calculate address
	var err error
	in.Address, err = in.UTXO.PublicAddress(netParams)
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
	if err := binary.Write(buf, binary.BigEndian, &scriptSize); err != nil {
		return err
	}

	if _, err := buf.Write(utxo.PkScript); err != nil {
		return err
	}

	if err := binary.Write(buf, binary.BigEndian, &utxo.Index); err != nil {
		return err
	}

	if err := binary.Write(buf, binary.BigEndian, &utxo.Value); err != nil {
		return err
	}

	return nil
}

func (utxo *UTXO) Read(buf *bytes.Buffer) error {
	if _, err := buf.Read(utxo.Hash[:]); err != nil {
		return err
	}

	var scriptSize uint32
	if err := binary.Read(buf, binary.BigEndian, &scriptSize); err != nil {
		return err
	}

	utxo.PkScript = make([]byte, int(scriptSize))
	if _, err := buf.Read(utxo.PkScript); err != nil {
		return err
	}

	if err := binary.Read(buf, binary.BigEndian, &utxo.Index); err != nil {
		return err
	}

	if err := binary.Read(buf, binary.BigEndian, &utxo.Value); err != nil {
		return err
	}

	return nil
}
