package operator

import (
	"context"

	"github.com/tokenized/pkg/bitcoin"
	"github.com/tokenized/pkg/wire"

	"github.com/google/uuid"
)

type Factory interface {
	NewClient(contractAddress bitcoin.RawAddress, url string, publicKey bitcoin.PublicKey) (Client, error)
}

type Client interface {
	// FetchContractAddress requests a hosted smart contract agent address from the operator.
	// Returns contract address, contract fee, and master address.
	// The master address is optional to use.
	FetchContractAddress(context.Context) (bitcoin.RawAddress, uint64, bitcoin.RawAddress, error)

	// SignContractOffer adds a signed input and an output to a contract offer transaction.
	// The input will be added as the second input so it is the contract's "operator" input.
	// Then an output to retrieve the value in the input will be added.
	// These have to be accounted for in the tx fee before calling this function because, since
	// the input will be signed, no other changes can be made to the tx other than signing inputs
	// without invalidating the operator input's signature.
	SignContractOffer(ctx context.Context, tx *wire.MsgTx) (*wire.MsgTx, *bitcoin.UTXO, error)

	// GetContractAddress returns the oracle's contract address.
	GetContractAddress() bitcoin.RawAddress

	// GetURL returns the oracle's URL.
	GetURL() string

	// GetPublicKey returns the oracle's public key.
	GetPublicKey() bitcoin.PublicKey

	// SetClientID sets the client's ID and authorization key.
	SetClientID(id uuid.UUID, key bitcoin.Key)

	// SetClientKey sets the client's authorization key.
	SetClientKey(key bitcoin.Key)
}
