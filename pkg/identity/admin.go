package identity

import (
	"bytes"
	"context"

	"github.com/tokenized/pkg/bitcoin"
	"github.com/tokenized/specification/dist/golang/actions"
	"github.com/tokenized/specification/dist/golang/protocol"

	"github.com/pkg/errors"
)

// AdminIdentityCertificate requests a admin identity certification for a contract offer.
func (o *HTTPClient) AdminIdentityCertificate(ctx context.Context, issuer actions.EntityField,
	contract bitcoin.RawAddress, xpubs bitcoin.ExtendedKeys, index uint32,
	requiredSigners int) (*actions.AdminIdentityCertificateField, error) {

	for _, xpub := range xpubs {
		if xpub.IsPrivate() {
			return nil, errors.New("private keys not allowed")
		}
	}

	request := struct {
		XPubs    bitcoin.ExtendedKeys `json:"xpubs" validate:"required"`
		Index    uint32               `json:"index" validate:"required"`
		Issuer   actions.EntityField  `json:"issuer"`
		Contract bitcoin.RawAddress   `json:"entity_contract"`
	}{
		XPubs:    xpubs,
		Index:    index,
		Issuer:   issuer,
		Contract: contract,
	}

	var response struct {
		Data struct {
			Approved    bool              `json:"approved"`
			Description string            `json:"description"`
			Signature   bitcoin.Signature `json:"signature"`
			BlockHeight uint32            `json:"block_height"`
			Expiration  uint64            `json:"expiration"`
		}
	}

	if err := post(ctx, o.URL+"/identity/verifyAdmin", request, &response); err != nil {
		return nil, errors.Wrap(err, "http post")
	}

	result := &actions.AdminIdentityCertificateField{
		EntityContract: o.ContractAddress.Bytes(),
		Signature:      response.Data.Signature.Bytes(),
		BlockHeight:    response.Data.BlockHeight,
		Expiration:     response.Data.Expiration,
	}

	if !response.Data.Approved {
		return result, errors.Wrap(ErrNotApproved, response.Data.Description)
	}

	return result, nil
}

// ValidateAdminIdentityCertificate checks the validity of an admin identity certificate.
func ValidateAdminIdentityCertificate(ctx context.Context,
	oracleAddress bitcoin.RawAddress, oraclePublicKey bitcoin.PublicKey, blocks BlockHashes,
	admin bitcoin.RawAddress, issuer actions.EntityField, contract bitcoin.RawAddress,
	data actions.AdminIdentityCertificateField) error {

	if !bytes.Equal(data.EntityContract, oracleAddress.Bytes()) {
		return errors.New("Wrong oracle entity contract")
	}

	signature, err := bitcoin.SignatureFromBytes(data.Signature)
	if err != nil {
		return errors.Wrap(err, "parse signature")
	}

	// Get block hash for tip - 4
	blockHash, err := blocks.Hash(ctx, int(data.BlockHeight))
	if err != nil {
		return errors.Wrap(err, "block hash")
	}

	var entity interface{}
	if contract.IsEmpty() {
		entity = issuer
	} else {
		entity = contract
	}

	sigHash, err := protocol.ContractAdminIdentityOracleSigHash(ctx, admin, entity, *blockHash,
		data.Expiration, 1)
	if err != nil {
		return errors.Wrap(err, "generate signature hash")
	}

	if signature.Verify(sigHash, oraclePublicKey) {
		return nil // valid approval signature
	}

	sigHash, err = protocol.ContractAdminIdentityOracleSigHash(ctx, admin, entity, *blockHash,
		data.Expiration, 0)
	if err != nil {
		return errors.Wrap(err, "generate signature hash")
	}

	if signature.Verify(sigHash, oraclePublicKey) {
		// Signature is valid, but it is confirming not approved.
		return ErrNotApproved
	}

	// Neither signature verified so it is just invalid.
	return errors.Wrap(ErrInvalidSignature, "validate signature")
}
