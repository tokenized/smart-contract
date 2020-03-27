package authority

import (
	"context"
	"encoding/hex"

	"github.com/tokenized/smart-contract/pkg/bitcoin"

	"github.com/tokenized/specification/dist/golang/actions"
	"github.com/tokenized/specification/dist/golang/protocol"

	"github.com/pkg/errors"
)

var (
	ErrNotApproved = errors.New("Not Approved")
)

// ApproveOrder requests a signature from the authority oracle to approve an enforcement order and
//   puts it in the order action.
func (o *Oracle) ApproveOrder(ctx context.Context, baseURL, contract string, order *actions.Order,
	isTest bool) error {

	order.AuthorityName = o.OracleEntity.Name
	order.AuthorityPublicKey = o.OracleKey.Bytes()

	request := struct {
		Contract string `json:"contract" validate:"required"`
		Order    string `json:"order" validate:"required"`
	}{
		Contract: contract,
	}

	b, err := protocol.Serialize(order, isTest)
	if err != nil {
		return errors.Wrap(err, "serialize order")
	}
	request.Order = hex.EncodeToString(b)

	var response struct {
		Data struct {
			Approved     bool   `json:"approved"`
			SigAlgorithm uint32 `json:"algorithm"`
			Sig          string `json:"signature"`
		}
	}

	if err := post(o.BaseURL+"/enforcement/order", request, &response); err != nil {
		return errors.Wrap(err, "http post")
	}

	if !response.Data.Approved {
		return ErrNotApproved
	}

	sig, err := bitcoin.SignatureFromStr(response.Data.Sig)
	if err != nil {
		return errors.Wrap(err, "parse signature")
	}

	order.SignatureAlgorithm = response.Data.SigAlgorithm
	order.OrderSignature = sig.Bytes()

	return nil
}
