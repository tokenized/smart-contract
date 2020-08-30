package identity

import (
	"bytes"
	"context"
	"math/rand"

	"github.com/tokenized/pkg/bitcoin"
	"github.com/tokenized/specification/dist/golang/actions"
	"github.com/tokenized/specification/dist/golang/protocol"

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
func (f *MockFactory) SetupOracle(contractAddress bitcoin.RawAddress, url string,
	key bitcoin.Key, blockHashes BlockHashes) {
	// Find setup mock oracle
	f.clients = append(f.clients, &MockClient{
		ContractAddress: contractAddress,
		URL:             url,
		Key:             key,
		blockHashes:     blockHashes,
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

	blockHashes BlockHashes
	users       []*MockUser
}

type MockUser struct {
	id    uuid.UUID
	xpubs []*MockXPub
}

type MockXPub struct {
	path            string
	xpubs           bitcoin.ExtendedKeys
	requiredSigners int
}

// RegisterUser registers a user with the identity oracle.
func (c *MockClient) RegisterUser(ctx context.Context, entity actions.EntityField,
	xpubs []bitcoin.ExtendedKeys) (uuid.UUID, error) {

	for _, xps := range xpubs {
		for _, xpub := range xps {
			if xpub.IsPrivate() {
				return uuid.UUID{}, errors.New("private keys not allowed")
			}
		}
	}

	for _, user := range c.users {
		for _, uxpub := range user.xpubs {
			for _, xpub := range xpubs {
				if uxpub.xpubs.Equal(xpub) {
					return user.id, nil // Already registered
				}
			}
		}
	}

	id := uuid.New()
	c.users = append(c.users, &MockUser{
		id: id,
	})
	return id, nil
}

// RegisterXPub registers an xpub under a user with an identity oracle.
func (c *MockClient) RegisterXPub(ctx context.Context, path string, xpubs bitcoin.ExtendedKeys,
	requiredSigners int) error {

	for _, xpub := range xpubs {
		if xpub.IsPrivate() {
			return errors.New("private keys not allowed")
		}
	}

	for _, user := range c.users {
		if bytes.Equal(user.id[:], c.ClientID[:]) {
			for _, xpub := range user.xpubs {
				if xpub.xpubs.Equal(xpubs) {
					return nil // Already registered
				}
			}

			user.xpubs = append(user.xpubs, &MockXPub{
				path:            path,
				xpubs:           xpubs,
				requiredSigners: requiredSigners,
			})

			return nil
		}
	}

	return errors.New("User not found")
}

// ApproveReceive requests an approval signature for a receiver from an identity oracle.
func (c *MockClient) ApproveReceive(ctx context.Context, contract, asset string, oracleIndex int,
	quantity uint64, xpubs bitcoin.ExtendedKeys, index uint32,
	requiredSigners int) (*actions.AssetReceiverField, bitcoin.Hash32, error) {

	for _, xpub := range xpubs {
		if xpub.IsPrivate() {
			return nil, bitcoin.Hash32{}, errors.New("private keys not allowed")
		}
	}

	// Find xpub
	found := false
	for _, user := range c.users {
		if !bytes.Equal(user.id[:], c.ClientID[:]) {
			continue
		}

		for _, xps := range user.xpubs {
			if xps.xpubs.Equal(xpubs) && xps.requiredSigners == requiredSigners {
				found = true
				break
			}
		}
	}

	if !found {
		return nil, bitcoin.Hash32{}, errors.New("User xpub not found")
	}

	contractAddress, err := bitcoin.DecodeAddress(contract)
	if err != nil {
		return nil, bitcoin.Hash32{}, errors.Wrap(err, "decode contract address")
	}
	contractRawAddress := bitcoin.NewRawAddressFromAddress(contractAddress)

	_, assetCode, err := protocol.DecodeAssetID(asset)
	if err != nil {
		return nil, bitcoin.Hash32{}, errors.Wrap(err, "decode asset id")
	}

	// Get random block hash
	height := rand.Intn(5000)
	blockHash, err := c.blockHashes.Hash(ctx, height)
	if err != nil {
		return nil, bitcoin.Hash32{}, errors.Wrap(err, "get sig block hash")
	}

	// Generate address at index
	addressKey, err := xpubs.ChildKeys(index)
	if err != nil {
		return nil, bitcoin.Hash32{}, errors.Wrap(err, "generate address key")
	}

	receiveAddress, err := addressKey.RawAddress(requiredSigners)
	if err != nil {
		return nil, bitcoin.Hash32{}, errors.Wrap(err, "generate address")
	}

	sigHash, err := protocol.TransferOracleSigHash(ctx, contractRawAddress, assetCode.Bytes(),
		receiveAddress, *blockHash, 0, 1)
	if err != nil {
		return nil, bitcoin.Hash32{}, errors.Wrap(err, "generate sig hash")
	}

	sig, err := c.Key.Sign(sigHash)
	if err != nil {
		return nil, bitcoin.Hash32{}, errors.Wrap(err, "sign")
	}

	result := &actions.AssetReceiverField{
		Address:               receiveAddress.Bytes(),
		Quantity:              quantity,
		OracleSigAlgorithm:    1,
		OracleIndex:           uint32(oracleIndex),
		OracleConfirmationSig: sig.Bytes(),
		OracleSigBlockHeight:  uint32(height),
		OracleSigExpiry:       0,
	}

	return result, *blockHash, nil
}

// ApproveEntityPublicKey requests a signature from the identity oracle to verify the ownership of
// a public key by a specified entity.
func (c *MockClient) ApproveEntityPublicKey(ctx context.Context, entity actions.EntityField,
	xpub bitcoin.ExtendedKey, index uint32) (ApprovedEntityPublicKey, error) {

	if xpub.IsPrivate() {
		return ApprovedEntityPublicKey{}, errors.New("private keys not allowed")
	}

	key, err := xpub.ChildKey(index)
	if err != nil {
		return ApprovedEntityPublicKey{}, errors.Wrap(err, "generate public key")
	}

	// Get random block hash
	height := rand.Intn(5000)
	blockHash, err := c.blockHashes.Hash(ctx, height)
	if err != nil {
		return ApprovedEntityPublicKey{}, errors.Wrap(err, "get sig block hash")
	}

	sigHash, err := protocol.EntityPubKeyOracleSigHash(ctx, &entity, key.PublicKey(), *blockHash, 1)
	if err != nil {
		return ApprovedEntityPublicKey{}, errors.Wrap(err, "generate sig hash")
	}

	sig, err := c.Key.Sign(sigHash)
	if err != nil {
		return ApprovedEntityPublicKey{}, errors.Wrap(err, "sign")
	}

	result := ApprovedEntityPublicKey{
		SigAlgorithm: 1,
		Signature:    sig,
		BlockHeight:  uint32(height),
		PublicKey:    key.PublicKey(),
	}

	return result, nil
}

// AdminIdentityCertificate requests a admin identity certification for a contract offer.
func (c *MockClient) AdminIdentityCertificate(ctx context.Context, issuer actions.EntityField,
	entityContract bitcoin.RawAddress, xpubs bitcoin.ExtendedKeys, index uint32,
	requiredSigners int) (*actions.AdminIdentityCertificateField, error) {

	for _, xpub := range xpubs {
		if xpub.IsPrivate() {
			return nil, errors.New("private keys not allowed")
		}
	}

	adminKey, err := xpubs.ChildKeys(index)
	if err != nil {
		return nil, errors.Wrap(err, "generate address key")
	}

	adminAddress, err := adminKey.RawAddress(requiredSigners)
	if err != nil {
		return nil, errors.Wrap(err, "generate address")
	}

	// Get random block hash
	height := rand.Intn(5000)
	blockHash, err := c.blockHashes.Hash(ctx, height)
	if err != nil {
		return nil, errors.Wrap(err, "get sig block hash")
	}

	var entity interface{}
	if entityContract.IsEmpty() {
		entity = issuer
	} else {
		entity = entityContract
	}

	sigHash, err := protocol.ContractAdminIdentityOracleSigHash(ctx, adminAddress, entity,
		*blockHash, 0, 1)
	if err != nil {
		return nil, errors.Wrap(err, "generate sig hash")
	}

	sig, err := c.Key.Sign(sigHash)
	if err != nil {
		return nil, errors.Wrap(err, "sign")
	}

	result := &actions.AdminIdentityCertificateField{
		EntityContract: c.ContractAddress.Bytes(),
		Signature:      sig.Bytes(),
		BlockHeight:    uint32(height),
		Expiration:     0,
	}

	return result, nil
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

// RandBlockHashes generates randomized block hashes for identity oracle checks.
type RandBlockHashes struct {
	hashes map[int]bitcoin.Hash32
}

func NewRandBlockHashes() *RandBlockHashes {
	return &RandBlockHashes{
		hashes: make(map[int]bitcoin.Hash32),
	}
}

func (bh *RandBlockHashes) Hash(ctx context.Context, height int) (*bitcoin.Hash32, error) {

	h, exists := bh.hashes[height]
	if exists {
		return &h, nil
	}

	var result bitcoin.Hash32
	rand.Read(result[:])
	bh.hashes[height] = result
	return &result, nil
}
