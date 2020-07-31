package identity

import (
	"context"

	"github.com/tokenized/pkg/bitcoin"

	"github.com/tokenized/specification/dist/golang/actions"
	"github.com/tokenized/specification/dist/golang/protocol"

	"github.com/pkg/errors"
)

type ApprovedEntityPublicKey struct {
	SigAlgorithm uint32            `json:"algorithm"`
	Signature    bitcoin.Signature `json:"signature"`
	BlockHeight  uint32            `json:"block_height"`
	PublicKey    bitcoin.PublicKey `json:"public_key"`
}

// ApproveEntityPublicKey requests a signature from the identity oracle to verify the ownership of
//   a public key by a specified entity.
func (o *HTTPClient) ApproveEntityPublicKey(ctx context.Context, entity actions.EntityField,
	xpub bitcoin.ExtendedKey, index uint32) (ApprovedEntityPublicKey, error) {

	key, err := xpub.ChildKey(index)
	if err != nil {
		return ApprovedEntityPublicKey{}, errors.Wrap(err, "generate public key")
	}

	request := struct {
		XPub   bitcoin.ExtendedKey `json:"xpub" validate:"required"`
		Index  uint32              `json:"index" validate:"required"`
		Entity actions.EntityField `json:"entity" validate:"required"`
	}{
		XPub:   xpub,
		Index:  index,
		Entity: entity,
	}

	var response struct {
		Data struct {
			Approved     bool              `json:"approved"`
			SigAlgorithm uint32            `json:"algorithm"`
			Signature    bitcoin.Signature `json:"signature"`
			BlockHeight  uint32            `json:"block_height"`
		}
	}

	if err := post(o.URL+"/identity/verifyPubKey", request, &response); err != nil {
		return ApprovedEntityPublicKey{}, errors.Wrap(err, "http post")
	}

	if !response.Data.Approved {
		return ApprovedEntityPublicKey{}, ErrNotApproved
	}

	result := ApprovedEntityPublicKey{
		SigAlgorithm: response.Data.SigAlgorithm,
		Signature:    response.Data.Signature,
		BlockHeight:  response.Data.BlockHeight,
		PublicKey:    key.PublicKey(),
	}

	return result, nil
}

// ValidateReceive checks the validity of an identity oracle signature for a receive.
func (o *HTTPClient) ValidateEntityPublicKey(ctx context.Context, blocks BlockHashes,
	entity *actions.EntityField, data ApprovedEntityPublicKey) error {

	if data.SigAlgorithm != 1 {
		return errors.New("Unsupported signature algorithm")
	}

	// Get block hash for tip - 4
	blockHash, err := blocks.Hash(ctx, int(data.BlockHeight))
	if err != nil {
		return errors.Wrap(err, "block hash")
	}

	sigHash, err := protocol.EntityPubKeyOracleSigHash(ctx, entity, data.PublicKey, blockHash, 1)
	if err != nil {
		return errors.Wrap(err, "generate signature")
	}

	if !data.Signature.Verify(sigHash, o.PublicKey) {
		return errors.Wrap(ErrInvalidSignature, "validate signature")
	}

	return nil
}
