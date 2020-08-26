package listeners

import (
	"bytes"
	"context"
	"time"

	"github.com/tokenized/pkg/bitcoin"
	"github.com/tokenized/pkg/logger"
	"github.com/tokenized/smart-contract/internal/contract"
	"github.com/tokenized/smart-contract/internal/platform/db"
	"github.com/tokenized/smart-contract/internal/platform/node"
	"github.com/tokenized/smart-contract/pkg/wallet"

	"github.com/pkg/errors"
)

// ContractIsStarted returns true if the contract has been started.
func (server *Server) ContractIsStarted(ctx context.Context, ca bitcoin.RawAddress) (bool, error) {
	// Check if contract exists
	c, err := contract.Retrieve(ctx, server.MasterDB, ca, server.Config.IsTest)
	if err == nil && c != nil {
		node.Log(ctx, "Contract found : %s",
			bitcoin.NewAddressFromRawAddress(ca, server.Config.Net).String())
		return true, nil
	}
	if err != contract.ErrNotFound {
		node.LogWarn(ctx, "Error retrieving contract : %s", err)
		return true, nil // Don't remove contract key
	}

	return false, nil
}

// AddContractKey adds a new contract key to those being monitored.
func (server *Server) AddContractKey(ctx context.Context, key *wallet.Key) error {
	server.walletLock.Lock()

	rawAddress, err := key.Key.RawAddress()
	if err != nil {
		server.walletLock.Unlock()
		return err
	}

	node.Log(ctx, "Adding key : %s",
		bitcoin.NewAddressFromRawAddress(rawAddress, server.Config.Net).String())

	server.contractAddresses = append(server.contractAddresses, rawAddress)
	server.txFilter.AddPubKey(ctx, key.Key.PublicKey().Bytes())

	server.walletLock.Unlock()

	if err := server.SaveWallet(ctx); err != nil {
		return err
	}
	return nil
}

// RemoveContract removes a contract key from those being monitored if it hasn't been
// used yet.
func (server *Server) RemoveContract(ctx context.Context, ca bitcoin.RawAddress,
	publicKey bitcoin.PublicKey) error {

	server.walletLock.Lock()
	defer server.walletLock.Unlock()

	node.Log(ctx, "Removing key : %s",
		bitcoin.NewAddressFromRawAddress(ca, server.Config.Net).String())
	server.wallet.RemoveAddress(ca)
	if err := server.SaveWallet(ctx); err != nil {
		return err
	}

	for i, caddress := range server.contractAddresses {
		if ca.Equal(caddress) {
			server.contractAddresses = append(server.contractAddresses[:i],
				server.contractAddresses[i+1:]...)
			break
		}
	}

	server.txFilter.RemovePubKey(ctx, publicKey.Bytes())
	return nil
}

func (server *Server) SaveWallet(ctx context.Context) error {
	node.Log(ctx, "Saving wallet")

	var buf bytes.Buffer
	start := time.Now()
	if err := server.wallet.Serialize(&buf); err != nil {
		return errors.Wrap(err, "serialize wallet")
	}
	logger.Elapsed(ctx, start, "Serialize wallet")

	defer logger.Elapsed(ctx, time.Now(), "Put wallet")
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
