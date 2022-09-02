package operator

import (
	"context"
	"math/rand"

	"github.com/tokenized/logger"
	"github.com/tokenized/pkg/bitcoin"
	"github.com/tokenized/pkg/txbuilder"
	"github.com/tokenized/pkg/wire"

	"github.com/google/uuid"
	"github.com/pkg/errors"
)

type MockFactory struct {
	clients []*MockClient
}

func NewMockFactory() *MockFactory {
	return &MockFactory{}
}

func (f *MockFactory) NewClient(contractAddress bitcoin.RawAddress, url string,
	publicKey bitcoin.PublicKey) (Client, error) {
	// Find setup mock oracle
	for _, client := range f.clients {
		if client.ContractAddress.Equal(contractAddress) {
			return client, nil
		}
	}

	return nil, errors.New("Client contract address not found")
}

// SetupOracle prepares a mock client in the factory. This must be called before calling NewClient.
func (f *MockFactory) SetupOracle(contractAddress bitcoin.RawAddress, url string, key bitcoin.Key,
	contractFee uint64) {
	// Find setup mock oracle
	f.clients = append(f.clients, &MockClient{
		ContractAddress: contractAddress,
		URL:             url,
		Key:             key,
		contractFee:     contractFee,
	})
}

type MockClient struct {
	// Oracle information
	ContractAddress bitcoin.RawAddress // Address of oracle's contract entity.
	URL             string
	Key             bitcoin.Key

	// Client information
	ClientID  uuid.UUID   // User ID of client
	ClientKey bitcoin.Key // Key used to authorize/encrypt with oracle

	contractFee uint64
	keys        []*bitcoin.Key
}

func (c *MockClient) FetchContractAddress(ctx context.Context) (bitcoin.RawAddress, uint64,
	bitcoin.RawAddress, error) {
	k, err := bitcoin.GenerateKey(bitcoin.MainNet)
	if err != nil {
		return bitcoin.RawAddress{}, c.contractFee, bitcoin.RawAddress{},
			errors.Wrap(err, "generate key")
	}

	ra, err := k.RawAddress()
	if err != nil {
		return bitcoin.RawAddress{}, c.contractFee, bitcoin.RawAddress{},
			errors.Wrap(err, "create address")
	}

	mk, err := bitcoin.GenerateKey(bitcoin.MainNet)
	if err != nil {
		return bitcoin.RawAddress{}, c.contractFee, bitcoin.RawAddress{},
			errors.Wrap(err, "generate key")
	}

	mra, err := mk.RawAddress()
	if err != nil {
		return bitcoin.RawAddress{}, c.contractFee, bitcoin.RawAddress{},
			errors.Wrap(err, "create address")
	}

	return ra, c.contractFee, mra, nil
}

// SignContractOffer adds a signed input to a contract offer transaction.
func (c *MockClient) SignContractOffer(ctx context.Context,
	tx *wire.MsgTx) (*wire.MsgTx, *bitcoin.UTXO, error) {

	tx = tx.Copy()

	serviceAddress, err := c.Key.RawAddress()
	if err != nil {
		return nil, nil, errors.Wrap(err, "address")
	}

	lockingScript, err := serviceAddress.LockingScript()
	if err != nil {
		return nil, nil, errors.Wrap(err, "locking script")
	}

	utxo := bitcoin.UTXO{
		Index:         uint32(rand.Intn(10)),
		Value:         txbuilder.DustLimitForLockingScript(lockingScript, 0.25),
		LockingScript: lockingScript,
	}
	rand.Read(utxo.Hash[:])

	// Add dust input from service key and output back to service key.
	inputIndex := 1 // contract operator input must be immediately after admin input
	input := wire.NewTxIn(wire.NewOutPoint(&utxo.Hash, utxo.Index), nil)

	if len(tx.TxIn) > 1 {
		after := make([]*wire.TxIn, len(tx.TxIn)-1)
		copy(after, tx.TxIn[1:])
		tx.TxIn = append(append(tx.TxIn[:1], input), after...)
	} else {
		tx.TxIn = append(tx.TxIn, input)
	}

	output := wire.NewTxOut(utxo.Value, lockingScript)
	tx.AddTxOut(output)

	// Sign input based on current tx. Note: The client can only add signatures after this or they
	// will invalidate this signature.
	input.UnlockingScript, err = txbuilder.P2PKHUnlockingScript(c.Key, tx, inputIndex,
		utxo.LockingScript, utxo.Value, txbuilder.SigHashAll+txbuilder.SigHashForkID,
		&txbuilder.SigHashCache{})
	if err != nil {
		return nil, nil, errors.Wrap(err, "sign")
	}

	logger.InfoWithFields(ctx, []logger.Field{
		logger.Stringer("hash", utxo.Hash),
		logger.Stringer("unlocking_script", input.UnlockingScript),
	}, "Added contract agent input")

	logger.InfoWithFields(ctx, []logger.Field{
		logger.Uint64("value", utxo.Value),
		logger.Stringer("script", lockingScript),
	}, "Adding contract agent output")

	logger.InfoWithFields(ctx, []logger.Field{
		logger.Stringer("public_key", c.Key.PublicKey()),
	}, "Signing contract agent input with key")

	return tx, &utxo, nil
}

// GetContractAddress returns the oracle's contract address.
func (c *MockClient) GetContractAddress() bitcoin.RawAddress {
	return c.ContractAddress
}

// GetURL returns the oracle's URL.
func (c *MockClient) GetURL() string {
	return c.URL
}

// GetPublicKey returns the oracle's public key.
func (c *MockClient) GetPublicKey() bitcoin.PublicKey {
	return c.Key.PublicKey()
}

// SetClientID sets the client's ID and authorization key.
func (c *MockClient) SetClientID(id uuid.UUID, key bitcoin.Key) {
	c.ClientID = id
	c.ClientKey = key
}

// SetClientKey sets the client's authorization key.
func (c *MockClient) SetClientKey(key bitcoin.Key) {
	c.ClientKey = key
}
