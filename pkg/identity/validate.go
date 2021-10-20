package identity

import (
	"bytes"
	"context"

	"github.com/tokenized/pkg/bitcoin"
	"github.com/tokenized/specification/dist/golang/actions"
	"github.com/tokenized/specification/dist/golang/protocol"

	"github.com/pkg/errors"
)

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
	blockHash, err := blocks.BlockHash(ctx, int(data.BlockHeight))
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

	if signature.Verify(*sigHash, oraclePublicKey) {
		return nil // valid approval signature
	}

	sigHash, err = protocol.ContractAdminIdentityOracleSigHash(ctx, admin, entity, *blockHash,
		data.Expiration, 0)
	if err != nil {
		return errors.Wrap(err, "generate signature hash")
	}

	if signature.Verify(*sigHash, oraclePublicKey) {
		// Signature is valid, but it is confirming not approved.
		return ErrNotApproved
	}

	// Neither signature verified so it is just invalid.
	return errors.Wrap(ErrInvalidSignature, "validate signature")
}

// ValidateReceive checks the validity of an identity oracle signature for a receive.
func (o *HTTPClient) ValidateEntityPublicKey(ctx context.Context, blocks BlockHashes,
	entity *actions.EntityField, data ApprovedEntityPublicKey) error {

	if data.SigAlgorithm != 1 {
		return errors.New("Unsupported signature algorithm")
	}

	// Get block hash for tip - 4
	blockHash, err := blocks.BlockHash(ctx, int(data.BlockHeight))
	if err != nil {
		return errors.Wrap(err, "block hash")
	}

	sigHash, err := protocol.EntityPubKeyOracleSigHash(ctx, entity, data.PublicKey, *blockHash, 1)
	if err != nil {
		return errors.Wrap(err, "generate signature")
	}

	if !data.Signature.Verify(*sigHash, o.PublicKey) {
		return errors.Wrap(ErrInvalidSignature, "validate signature")
	}

	return nil
}

// ValidateReceive checks the validity of an identity oracle signature for a receive.
func ValidateReceive(ctx context.Context, publicKey bitcoin.PublicKey, blocks BlockHashes,
	contract, asset string, receiver *actions.AssetReceiverField) error {

	if receiver.OracleSigAlgorithm != 1 {
		return errors.New("Unsupported signature algorithm")
	}

	contractAddress, err := bitcoin.DecodeAddress(contract)
	if err != nil {
		return errors.Wrap(err, "decode contract address")
	}
	contractRawAddress := bitcoin.NewRawAddressFromAddress(contractAddress)

	_, assetCode, err := protocol.DecodeAssetID(asset)
	if err != nil {
		return errors.Wrap(err, "decode asset id")
	}

	// Get block hash for tip - 4
	blockHash, err := blocks.BlockHash(ctx, int(receiver.OracleSigBlockHeight))
	if err != nil {
		return errors.Wrap(err, "block hash")
	}

	receiveAddress, err := bitcoin.DecodeRawAddress(receiver.Address)
	if err != nil {
		return errors.Wrap(err, "decode address")
	}

	signature, err := bitcoin.SignatureFromBytes(receiver.OracleConfirmationSig)
	if err != nil {
		return errors.Wrap(ErrInvalidSignature, "parse signature")
	}

	// Check for approved signature
	sigHash, err := protocol.TransferOracleSigHash(ctx, contractRawAddress, assetCode.Bytes(),
		receiveAddress, *blockHash, receiver.OracleSigExpiry, 1)
	if err != nil {
		return errors.Wrap(err, "signature hash")
	}

	if signature.Verify(*sigHash, publicKey) {
		return nil
	}

	// Check for not approved signature
	sigHash, err = protocol.TransferOracleSigHash(ctx, contractRawAddress, assetCode.Bytes(),
		receiveAddress, *blockHash, receiver.OracleSigExpiry, 0)
	if err != nil {
		return errors.Wrap(err, "signature hash")
	}

	if signature.Verify(*sigHash, publicKey) {
		// Signature is valid, but it is confirming the receive was not approved.
		return ErrNotApproved
	}

	// Neither signature verified so it is just invalid.
	return errors.Wrap(ErrInvalidSignature, "validate signature")
}

// ValidateReceiveHash checks the validity of an identity oracle signature for a receive.
func ValidateReceiveHash(ctx context.Context, publicKey bitcoin.PublicKey,
	blockHash bitcoin.Hash32, contract, asset string, receiver *actions.AssetReceiverField) error {

	if receiver.OracleSigAlgorithm != 1 {
		return errors.New("Unsupported signature algorithm")
	}

	contractAddress, err := bitcoin.DecodeAddress(contract)
	if err != nil {
		return errors.Wrap(err, "decode contract address")
	}
	contractRawAddress := bitcoin.NewRawAddressFromAddress(contractAddress)

	_, assetCode, err := protocol.DecodeAssetID(asset)
	if err != nil {
		return errors.Wrap(err, "decode asset id")
	}

	receiveAddress, err := bitcoin.DecodeRawAddress(receiver.Address)
	if err != nil {
		return errors.Wrap(err, "decode address")
	}

	signature, err := bitcoin.SignatureFromBytes(receiver.OracleConfirmationSig)
	if err != nil {
		return errors.Wrap(ErrInvalidSignature, "parse signature")
	}

	// Check for approved signature
	sigHash, err := protocol.TransferOracleSigHash(ctx, contractRawAddress, assetCode.Bytes(),
		receiveAddress, blockHash, receiver.OracleSigExpiry, 1)
	if err != nil {
		return errors.Wrap(err, "signature hash")
	}

	if signature.Verify(*sigHash, publicKey) {
		return nil
	}

	// Check for not approved signature
	sigHash, err = protocol.TransferOracleSigHash(ctx, contractRawAddress, assetCode.Bytes(),
		receiveAddress, blockHash, receiver.OracleSigExpiry, 0)
	if err != nil {
		return errors.Wrap(err, "signature hash")
	}

	if signature.Verify(*sigHash, publicKey) {
		// Signature is valid, but it is confirming the receive was not approved.
		return ErrNotApproved
	}

	// Neither signature verified so it is just invalid.
	return errors.Wrap(ErrInvalidSignature, "validate signature")
}
