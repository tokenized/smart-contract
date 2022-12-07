package listeners

import (
	"bytes"
	"context"
	"time"

	"github.com/tokenized/logger"
	"github.com/tokenized/pkg/bitcoin"
	"github.com/tokenized/smart-contract/internal/contract"
	"github.com/tokenized/smart-contract/internal/platform/db"
	"github.com/tokenized/smart-contract/internal/platform/node"
	"github.com/tokenized/smart-contract/pkg/wallet"

	"github.com/pkg/errors"
)

// ContractIsStarted returns true if the contract has been started.
func (server *Server) ContractIsStarted(ctx context.Context, ra bitcoin.RawAddress) (bool, error) {
	// Check if contract exists
	c, err := contract.Retrieve(ctx, server.MasterDB, ra, server.Config.IsTest)
	if err == nil && c != nil {
		node.Log(ctx, "Contract found : %s", bitcoin.NewAddressFromRawAddress(ra,
			server.Config.Net))
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

	node.Log(ctx, "Adding key : %s",
		bitcoin.NewAddressFromRawAddress(key.Address, server.Config.Net))

	server.contractLockingScripts = append(server.contractLockingScripts, key.LockingScript)

	if server.SpyNode != nil {
		hashes, err := key.Address.Hashes()
		if err != nil {
			server.walletLock.Unlock()
			return err
		}

		for _, hash := range hashes {
			server.SpyNode.SubscribePushDatas(ctx, [][]byte{hash[:]})
		}
	}

	server.walletLock.Unlock()

	if err := server.SaveWallet(ctx); err != nil {
		return err
	}
	return nil
}

// RemoveContract removes a contract key from those being monitored if it hasn't been
// used yet.
func (server *Server) RemoveContract(ctx context.Context, lockingScript bitcoin.Script,
	publicKey bitcoin.PublicKey) error {

	rawAddress, err := bitcoin.RawAddressFromLockingScript(lockingScript)
	if err != nil {
		server.walletLock.Unlock()
		return err
	}

	server.walletLock.Lock()
	defer server.walletLock.Unlock()

	node.Log(ctx, "Removing key : %s", bitcoin.NewAddressFromRawAddress(rawAddress,
		server.Config.Net))
	server.wallet.RemoveAddress(rawAddress)
	if err := server.SaveWallet(ctx); err != nil {
		return err
	}

	for i, contractLockingScript := range server.contractLockingScripts {
		if lockingScript.Equal(contractLockingScript) {
			server.contractLockingScripts = append(server.contractLockingScripts[:i],
				server.contractLockingScripts[i+1:]...)
			break
		}
	}

	if server.SpyNode != nil {
		rawAddress, err := publicKey.RawAddress()
		if err != nil {
			return err
		}

		hashes, err := rawAddress.Hashes()
		if err != nil {
			return err
		}

		for _, hash := range hashes {
			server.SpyNode.UnsubscribePushDatas(ctx, [][]byte{hash[:]})
		}
	}

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

	server.contractLockingScripts = make([]bitcoin.Script, len(keys))
	for i, key := range keys {
		server.contractLockingScripts[i] = key.LockingScript
	}

	return nil
}
