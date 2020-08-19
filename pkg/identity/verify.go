package identity

import (
	"context"

	"github.com/pkg/errors"
	"github.com/tokenized/pkg/bitcoin"
	"github.com/tokenized/specification/dist/golang/actions"
)

// AdminIdentityCertificate requests a admin identity certification for a contract offer.
func (o *HTTPClient) AdminIdentityCertificate(ctx context.Context, issuer actions.EntityField,
	entityContract bitcoin.RawAddress, xpubs bitcoin.ExtendedKeys, index uint32,
	requiredSigners int) (*actions.AdminIdentityCertificateField, bitcoin.Hash32, error) {

	request := struct {
		XPubs    bitcoin.ExtendedKeys `json:"xpubs" validate:"required"`
		Index    uint32               `json:"index" validate:"required"`
		Issuer   actions.EntityField  `json:"issuer"`
		Contract bitcoin.RawAddress   `json:"entity_contract"`
	}{
		XPubs:    xpubs,
		Index:    index,
		Issuer:   issuer,
		Contract: entityContract,
	}

	var response struct {
		Data struct {
			Approved    bool              `json:"approved"`
			Description string            `json:"description"`
			Signature   bitcoin.Signature `json:"signature"`
			BlockHeight uint32            `json:"block_height"`
			BlockHash   bitcoin.Hash32    `json:"block_hash"`
			Expiration  uint64            `json:"expiration"`
		}
	}

	if err := post(ctx, o.URL+"/identity/verifyAdmin", request, &response); err != nil {
		return nil, bitcoin.Hash32{}, errors.Wrap(err, "http post")
	}

	result := &actions.AdminIdentityCertificateField{
		EntityContract: o.ContractAddress.Bytes(),
		Signature:      response.Data.Signature.Bytes(),
		BlockHeight:    response.Data.BlockHeight,
		Expiration:     response.Data.Expiration,
	}

	if !response.Data.Approved {
		return result, response.Data.BlockHash, errors.Wrap(ErrNotApproved,
			response.Data.Description)
	}

	return result, response.Data.BlockHash, nil
}
