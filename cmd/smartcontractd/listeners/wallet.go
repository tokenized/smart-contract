package listeners

import (
	"bytes"
	"context"

	"github.com/tokenized/pkg/bitcoin"
	"github.com/tokenized/smart-contract/internal/contract"
	"github.com/tokenized/smart-contract/internal/platform/db"
	"github.com/tokenized/smart-contract/internal/platform/node"
	"github.com/tokenized/smart-contract/pkg/wallet"

	"github.com/pkg/errors"
)

// AddContractKey adds a new contract key to those being monitored.
func (server *Server) AddContractKey(ctx context.Context, key *wallet.Key) error {
	server.walletLock.Lock()
	defer server.walletLock.Unlock()

	rawAddress, err := bitcoin.NewRawAddressPKH(bitcoin.Hash160(key.Key.PublicKey().Bytes()))
	if err != nil {
		return err
	}

	address, _ := bitcoin.NewAddressPKH(bitcoin.Hash160(key.Key.PublicKey().Bytes()),
		server.Config.Net)
	node.Log(ctx, "Adding key : %s", address.String())
	if err := server.SaveWallet(ctx); err != nil {
		return err
	}
	server.contractAddresses = append(server.contractAddresses, rawAddress)

	server.txFilter.AddPubKey(ctx, key.Key.PublicKey().Bytes())
	return nil
}

// RemoveContractKeyIfUnused removes a contract key from those being monitored if it hasn't been used yet.
func (server *Server) RemoveContractKeyIfUnused(ctx context.Context, k bitcoin.Key) error {
	server.walletLock.Lock()
	defer server.walletLock.Unlock()

	rawAddress, err := bitcoin.NewRawAddressPKH(bitcoin.Hash160(k.PublicKey().Bytes()))
	if err != nil {
		return err
	}
	newKey := wallet.Key{
		Address: rawAddress,
		Key:     k,
	}

	// Check if contract exists
	_, err = contract.Retrieve(ctx, server.MasterDB, rawAddress, server.Config.IsTest)
	if err == nil {
		node.LogWarn(ctx, "Contract found")
		return nil // Don't remove contract key
	}
	if err != contract.ErrNotFound {
		node.LogWarn(ctx, "Error retrieving contract : %s", err)
		return nil // Don't remove contract key
	}

	stringAddress := bitcoin.NewAddressFromRawAddress(rawAddress, server.Config.Net)
	node.Log(ctx, "Removing key : %s", stringAddress.String())
	server.wallet.Remove(&newKey)
	if err := server.SaveWallet(ctx); err != nil {
		return err
	}

	for i, caddress := range server.contractAddresses {
		if rawAddress.Equal(caddress) {
			server.contractAddresses = append(server.contractAddresses[:i], server.contractAddresses[i+1:]...)
			break
		}
	}

	server.txFilter.RemovePubKey(ctx, k.PublicKey().Bytes())
	return nil
}

func (server *Server) SaveWallet(ctx context.Context) error {
	node.Log(ctx, "Saving wallet")
	var buf bytes.Buffer

	if err := server.wallet.Serialize(&buf); err != nil {
		return errors.Wrap(err, "serialize wallet")
	}

	return server.MasterDB.Put(ctx, walletKey, buf.Bytes())
}

func (server *Server) LoadWallet(ctx context.Context) error {
	node.Log(ctx, "Loading wallet")

	data, err := server.MasterDB.Fetch(ctx, walletKey)
	if err != nil {
		if err == db.ErrNotFound {
			return nil // No keys yet
		}
		return errors.Wrap(err, "fetch wallet")
	}

	buf := bytes.NewReader(data)

	if err := server.wallet.Deserialize(buf); err != nil {
		return errors.Wrap(err, "deserialize wallet")
	}

	return server.SyncWallet(ctx)
}

func (server *Server) SyncWallet(ctx context.Context) error {
	node.Log(ctx, "Syncing wallet")

	// Refresh node for wallet.
	keys := server.wallet.ListAll()

	server.contractAddresses = make([]bitcoin.RawAddress, 0, len(keys))
	for _, key := range keys {
		// Contract address
		server.contractAddresses = append(server.contractAddresses, key.Address)

		// Tx Filter
		server.txFilter.AddPubKey(ctx, key.Key.PublicKey().Bytes())
	}

	return nil
}
