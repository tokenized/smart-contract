package identity

import (
	"context"
	"crypto/sha256"
	"encoding/binary"

	"github.com/tokenized/pkg/bitcoin"

	"github.com/tokenized/specification/dist/golang/actions"

	"github.com/google/uuid"
	"github.com/pkg/errors"
)

// RegisterUser checks if a user for this entity exists with the identity oracle and if not
//   registers a new user id.
func (o *HTTPClient) RegisterUser(ctx context.Context, entity actions.EntityField,
	xpubs []bitcoin.ExtendedKeys) (uuid.UUID, error) {

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
				UserID uuid.UUID `json:"user_id"`
			}
		}

		if err := post(o.URL+"/oracle/user", request, &response); err != nil {
			if errors.Cause(err) == ErrNotFound {
				continue
			}
			return uuid.UUID{}, errors.Wrap(err, "http post")
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
		PublicKey: o.ClientKey.PublicKey(),
	}

	// Look for 200 OK status with data
	var response struct {
		Data struct {
			Status string    `json:"status"`
			UserID uuid.UUID `json:"user_id"`
		}
	}

	if err := post(o.URL+"/oracle/register", request, &response); err != nil {
		return uuid.UUID{}, errors.Wrap(err, "http post")
	}

	o.ClientID = response.Data.UserID
	return o.ClientID, nil
}

// RegisterXPub checks if the xpub is already added to the identity user and if not adds it to the
//   identity oracle.
func (o *HTTPClient) RegisterXPub(ctx context.Context, path string, xpubs bitcoin.ExtendedKeys,
	requiredSigners int) error {

	if len(o.ClientID) == 0 {
		return errors.New("User not registered")
	}

	// Add xpub to user using identity oracle endpoint.
	request := struct {
		UserID          uuid.UUID            `json:"user_id"`
		XPubs           bitcoin.ExtendedKeys `json:"xpubs"`
		RequiredSigners int                  `json:"required_signers"`
		Signature       bitcoin.Signature    `json:"signature"` // hex signature of user id and xpub with users public key
	}{
		UserID:          o.ClientID,
		XPubs:           xpubs,
		RequiredSigners: requiredSigners,
	}

	s := sha256.New()
	s.Write(request.UserID[:])
	s.Write(request.XPubs.Bytes())
	if err := binary.Write(s, binary.LittleEndian, uint32(request.RequiredSigners)); err != nil {
		return errors.Wrap(err, "hash signers")
	}
	hash := sha256.Sum256(s.Sum(nil))

	var err error
	request.Signature, err = o.ClientKey.Sign(hash[:])
	if err != nil {
		return errors.Wrap(err, "sign")
	}

	if err := post(o.URL+"/oracle/addXPub", request, nil); err != nil {
		return errors.Wrap(err, "http post")
	}

	return nil
}
