package identity

import (
	"context"

	"github.com/tokenized/pkg/bitcoin"

	"github.com/tokenized/specification/dist/golang/actions"

	"github.com/pkg/errors"
)

// RegisterUser checks if a user for this entity exists with the identity oracle and if not
//   registers a new user id.
func (o *Oracle) RegisterUser(ctx context.Context, entity actions.EntityField,
	xpubs []bitcoin.ExtendedKeys) (string, error) {

	// Check for existing user for xpubs.
	for _, xpub := range xpubs {

		request := struct {
			XPubs bitcoin.ExtendedKeys `json:"xpubs"`
		}{
			XPubs: xpub,
		}

		// Look for 200 OK status with data
		var response struct {
			Data struct {
				UserID string `json:"user_id"`
			}
		}

		if err := post(o.BaseURL+"/oracle/user", request, &response); err != nil {
			if errors.Cause(err) == ErrNotFound {
				continue
			}
			return "", errors.Wrap(err, "http post")
		}

		o.ClientID = response.Data.UserID
		return o.ClientID, nil
	}

	// Call endpoint to register user and get ID.
	request := struct {
		Entity    actions.EntityField `json:"entity"`     // hex protobuf
		PublicKey bitcoin.PublicKey   `json:"public_key"` // hex compressed
	}{
		Entity:    entity,
		PublicKey: o.ClientAuthKey.PublicKey(),
	}

	// Look for 200 OK status with data
	var response struct {
		Data struct {
			Status string `json:"status"`
			UserID string `json:"user_id"`
		}
	}

	if err := post(o.BaseURL+"/oracle/register", request, &response); err != nil {
		return "", errors.Wrap(err, "http post")
	}

	o.ClientID = response.Data.UserID
	return o.ClientID, nil
}

// RegisterXPub checks if the xpub is already added to the identity user and if not adds it to the
//   identity oracle.
func (o *Oracle) RegisterXPub(ctx context.Context, path string, xpubs bitcoin.ExtendedKeys,
	requiredSigners int) error {

	if len(o.ClientID) == 0 {
		return errors.New("User not registered")
	}

	// Add xpub to user using identity oracle endpoint.
	request := struct {
		UserID          string               `json:"user_id"`
		XPubs           bitcoin.ExtendedKeys `json:"xpubs"`
		RequiredSigners int                  `json:"required_signers"`
		Signature       bitcoin.Signature    `json:"signature"` // hex signature of user id and xpub with users public key
	}{
		UserID:          o.ClientID,
		XPubs:           xpubs,
		RequiredSigners: requiredSigners,
	}

	hash := bitcoin.DoubleSha256([]byte(request.UserID + request.XPubs.String()))

	var err error
	request.Signature, err = o.ClientAuthKey.Sign(hash)
	if err != nil {
		return errors.Wrap(err, "sign")
	}

	if err := post(o.BaseURL+"/oracle/addXPub", request, nil); err != nil {
		return errors.Wrap(err, "http post")
	}

	return nil
}
