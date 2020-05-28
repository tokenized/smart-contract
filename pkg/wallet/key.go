package wallet

import (
	"bytes"
	"encoding/binary"

	"github.com/tokenized/pkg/bitcoin"
)

type Key struct {
	Address bitcoin.RawAddress
	Key     bitcoin.Key
}

func NewKey(key bitcoin.Key) *Key {
	result := Key{
		Key: key,
	}

	result.Address, _ = key.RawAddress()
	return &result
}

func (rk *Key) Read(buf *bytes.Reader, net bitcoin.Network) error {
	var length uint8
	if err := binary.Read(buf, binary.LittleEndian, &length); err != nil {
		return err
	}

	data := make([]byte, length)
	if _, err := buf.Read(data); err != nil {
		return err
	}

	var err error
	rk.Key, err = bitcoin.KeyFromBytes(data, net)
	if err != nil {
		return err
	}

	rk.Address, _ = rk.Key.RawAddress()
	return err
}

func (rk *Key) Write(buf *bytes.Buffer) error {
	b := rk.Key.Bytes()
	binary.Write(buf, binary.LittleEndian, uint8(len(b)))
	_, err := buf.Write(b)
	return err
}
