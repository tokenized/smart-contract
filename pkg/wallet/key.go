package wallet

import (
	"bytes"
	"encoding/binary"
	"errors"

	"github.com/tokenized/smart-contract/pkg/bitcoin"
	"github.com/tokenized/smart-contract/pkg/wire"
)

type Key struct {
	Address bitcoin.Address
	Key     bitcoin.Key
}

func NewKey(key bitcoin.Key) *Key {
	result := Key{
		Key: key,
	}

	s256, ok := key.(*bitcoin.KeyS256)
	if !ok {
		return nil
	}
	result.Address, _ = bitcoin.NewAddressPKH(bitcoin.Hash160(s256.PublicKey().Bytes()), key.Network())
	return &result
}

func (rk *Key) Read(buf *bytes.Buffer, net wire.BitcoinNet) error {
	var length uint8
	if err := binary.Read(buf, binary.LittleEndian, &length); err != nil {
		return err
	}

	data := make([]byte, length)
	if _, err := buf.Read(data); err != nil {
		return err
	}

	var err error
	rk.Key, err = bitcoin.DecodeKeyBytes(data, net)
	if err != nil {
		return err
	}

	s256, ok := rk.Key.(*bitcoin.KeyS256)
	if !ok {
		return errors.New("Key is not S256")
	}
	rk.Address, err = bitcoin.NewAddressPKH(bitcoin.Hash160(s256.PublicKey().Bytes()), net)
	return err
}

func (rk *Key) Write(buf *bytes.Buffer) error {
	b := rk.Key.Bytes()
	binary.Write(buf, binary.LittleEndian, uint8(len(b)))
	_, err := buf.Write(b)
	return err
}
