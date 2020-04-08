package identity

import (
	"context"

	"github.com/tokenized/smart-contract/pkg/bitcoin"
	"github.com/tokenized/smart-contract/pkg/logger"

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
	quantity uint64, xpub bitcoin.ExtendedKeys, index uint32, requiredSigners int) (*actions.AssetReceiverField, error) {

	keys, err := xpub.ChildKeys(index)
	if err != nil {
		return nil, errors.Wrap(err, "generate key")
	}

	address, err := keys.RawAddress(requiredSigners)
	if err != nil {
		return nil, errors.Wrap(err, "generate address")
	}

	request := struct {
		XPub     bitcoin.ExtendedKeys `json:"xpub"`
		Index    uint32               `json:"index"`
		Contract string               `json:"contract"`
		AssetID  string               `json:"asset_id"`
		Quantity uint64               `json:"quantity"`
	}{
		XPub:     xpub,
		Index:    index,
		Contract: contract,
		AssetID:  asset,
		Quantity: quantity,
	}

	var response struct {
		Data struct {
			Approved     bool              `json:"approved"`
			SigAlgorithm uint32            `json:"algorithm"`
			Signature    bitcoin.Signature `json:"signature"`
			BlockHeight  uint32            `json:"block_height"`
		}
	}

	if err := post(o.BaseURL+"/transfer/approve", request, &response); err != nil {
		return nil, errors.Wrap(err, "http post")
	}

	if !response.Data.Approved {
		return nil, ErrNotApproved
	}

	result := &actions.AssetReceiverField{
		Address:               address.Bytes(),
		Quantity:              quantity,
		OracleSigAlgorithm:    response.Data.SigAlgorithm,
		OracleIndex:           uint32(oracleIndex),
		OracleConfirmationSig: response.Data.Signature.Bytes(),
		OracleSigBlockHeight:  response.Data.BlockHeight,
	}

	return result, nil
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

	sigHash, err := protocol.TransferOracleSigHash(ctx, contractRawAddress, assetCode.Bytes(),
		receiveAddress, receiver.Quantity, blockHash, 1)
	if err != nil {
		return errors.Wrap(err, "signature hash")
	}

	logger.Info(ctx, "Validate receive signature hash : %x", sigHash)

	signature, err := bitcoin.SignatureFromBytes(receiver.OracleConfirmationSig)
	if err != nil {
		return errors.Wrap(ErrInvalidSignature, "parse signature")
	}

	if !signature.Verify(sigHash, o.OracleKey) {
		return errors.Wrap(ErrInvalidSignature, "validate signature")
	}

	return nil
}
