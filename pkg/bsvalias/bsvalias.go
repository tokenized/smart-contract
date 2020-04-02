package bsvalias

import (
	"context"
	"strings"

	"github.com/tokenized/smart-contract/pkg/bitcoin"

	"github.com/pkg/errors"
)

var (
	ErrInvalidHandle = errors.New("Invalid handle") // handle is badly formatted
	ErrNotCapable    = errors.New("Not capable")    // host site doesn't support a function
)

type Identity struct {
	Handle   string
	Site     Site
	Alias    string
	Hostname string
}

func NewIdentity(ctx context.Context, handle string) (Identity, error) {
	result := Identity{
		Handle: handle,
	}

	fields := strings.Split(handle, "@")
	if len(fields) != 2 {
		return result, errors.Wrap(ErrInvalidHandle, "split @ not 2")
	}

	result.Alias = fields[0]
	result.Hostname = fields[1]

	var err error
	result.Site, err = GetSite(ctx, result.Hostname)
	if err != nil {
		return result, errors.Wrap(err, "get site")
	}

	return result, nil
}

func (i *Identity) GetPublicKey(ctx context.Context) (bitcoin.PublicKey, error) {
	if len(i.Site.Capabilities.PKI) == 0 {
		return bitcoin.PublicKey{}, errors.Wrap(ErrNotCapable, "pki")
	}

	var response struct {
		Version   string `json:"bsvalias"`
		Handle    string `json:"handle"`
		PublicKey string `json:"pubkey"`
	}

	url := strings.ReplaceAll(i.Site.Capabilities.PKI, "{alias}", i.Alias)
	url = strings.ReplaceAll(url, "{domain.tld}", i.Hostname)
	if err := get(url, &response); err != nil {
		return bitcoin.PublicKey{}, errors.Wrap(err, "http get")
	}

	result, err := bitcoin.PublicKeyFromStr(response.PublicKey)
	if err != nil {
		return bitcoin.PublicKey{}, errors.Wrap(err, "parse public key")
	}

	return result, nil
}
