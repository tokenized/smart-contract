package identity

import (
	"context"

	"github.com/tokenized/pkg/bitcoin"

	"github.com/tokenized/specification/dist/golang/actions"
	"github.com/tokenized/specification/dist/golang/protocol"

	"github.com/pkg/errors"
)

var (
	ErrNotApproved      = errors.New("Not Approved")
	ErrInvalidSignature = errors.New("Invalid Signature")
)

// ApproveReceive requests a signature from the identity oracle to approve receipt of a token.
func (o *Oracle) ApproveReceive(ctx context.Context, contract, asset string, oracleIndex int,
	quantity uint64, xpubs bitcoin.ExtendedKeys, index uint32, requiredSigners int) (*actions.AssetReceiverField, bitcoin.Hash32, error) {

	keys, err := xpubs.ChildKeys(index)
	if err != nil {
		return nil, bitcoin.Hash32{}, errors.Wrap(err, "generate key")
	}

	address, err := keys.RawAddress(requiredSigners)
	if err != nil {
		return nil, bitcoin.Hash32{}, errors.Wrap(err, "generate address")
	}

	request := struct {
		XPubs    bitcoin.ExtendedKeys `json:"xpubs"`
		Index    uint32               `json:"index"`
		Contract string               `json:"contract"`
		AssetID  string               `json:"asset_id"`
		Quantity uint64               `json:"quantity"`
	}{
		XPubs:    xpubs,
		Index:    index,
		Contract: contract,
		AssetID:  asset,
		Quantity: quantity,
	}

	var response struct {
		Data struct {
			Approved     bool              `json:"approved"`
			Description  string            `json:"description"`
			SigAlgorithm uint32            `json:"algorithm"`
			Signature    bitcoin.Signature `json:"signature"`
			BlockHeight  uint32            `json:"block_height"`
			BlockHash    bitcoin.Hash32    `json:"block_hash"`
			Expiration   uint64            `json:"expiration"`
		}
	}

	if err := post(o.URL+"/transfer/approve", request, &response); err != nil {
		return nil, bitcoin.Hash32{}, errors.Wrap(err, "http post")
	}

	result := &actions.AssetReceiverField{
		Address:               address.Bytes(),
		Quantity:              quantity,
		OracleSigAlgorithm:    response.Data.SigAlgorithm,
		OracleIndex:           uint32(oracleIndex),
		OracleConfirmationSig: response.Data.Signature.Bytes(),
		OracleSigBlockHeight:  response.Data.BlockHeight,
		OracleSigExpiry:       response.Data.Expiration,
	}

	if !response.Data.Approved {
		return result, response.Data.BlockHash, errors.Wrap(ErrNotApproved, response.Data.Description)
	}

	return result, response.Data.BlockHash, nil
}

// ValidateReceive checks the validity of an identity oracle signature for a receive.
func (o *Oracle) ValidateReceive(ctx context.Context, blocks BlockHashes, contract, asset string,
	receiver *actions.AssetReceiverField) error {

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
	blockHash, err := blocks.Hash(ctx, int(receiver.OracleSigBlockHeight))
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
		receiveAddress, blockHash, receiver.OracleSigExpiry, 1)
	if err != nil {
		return errors.Wrap(err, "signature hash")
	}

	if signature.Verify(sigHash, o.PublicKey) {
		return nil
	}

	// Check for not approved signature
	sigHash, err = protocol.TransferOracleSigHash(ctx, contractRawAddress, assetCode.Bytes(),
		receiveAddress, blockHash, receiver.OracleSigExpiry, 0)
	if err != nil {
		return errors.Wrap(err, "signature hash")
	}

	if signature.Verify(sigHash, o.PublicKey) {
		// Signature is valid, but it is confirming the receive was not approved.
		return ErrNotApproved
	}

	// Neither signature verified so it is just invalid.
	return errors.Wrap(ErrInvalidSignature, "validate signature")
}

// ValidateReceiveHash checks the validity of an identity oracle signature for a receive.
func (o *Oracle) ValidateReceiveHash(ctx context.Context, blockHash bitcoin.Hash32,
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
		receiveAddress, &blockHash, receiver.OracleSigExpiry, 1)
	if err != nil {
		return errors.Wrap(err, "signature hash")
	}

	if signature.Verify(sigHash, o.PublicKey) {
		return nil
	}

	// Check for not approved signature
	sigHash, err = protocol.TransferOracleSigHash(ctx, contractRawAddress, assetCode.Bytes(),
		receiveAddress, &blockHash, receiver.OracleSigExpiry, 0)
	if err != nil {
		return errors.Wrap(err, "signature hash")
	}

	if signature.Verify(sigHash, o.PublicKey) {
		// Signature is valid, but it is confirming the receive was not approved.
		return ErrNotApproved
	}

	// Neither signature verified so it is just invalid.
	return errors.Wrap(ErrInvalidSignature, "validate signature")
}
