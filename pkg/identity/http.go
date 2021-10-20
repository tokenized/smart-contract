package identity

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/tokenized/pkg/bitcoin"
	"github.com/tokenized/pkg/json"
	"github.com/tokenized/pkg/logger"
	"github.com/tokenized/specification/dist/golang/actions"

	"github.com/google/uuid"
	"github.com/pkg/errors"
)

// HTTPFactory implements the factory interface that creates http clients.
type HTTPFactory struct{}

// HTTPClient implements the client interface to perform HTTP requests to identity oracles.
type HTTPClient struct {
	// Oracle information
	ContractAddress bitcoin.RawAddress // Address of oracle's contract entity.
	URL             string
	PublicKey       bitcoin.PublicKey

	// Client information
	ClientID  uuid.UUID   // User ID of client
	ClientKey bitcoin.Key // Key used to authorize/encrypt with oracle

	// TODO Implement retry functionality --ce
	// MaxRetries int
	// RetryDelay int
}

// NewHTTPFactory creates a new http factory.
func NewHTTPFactory() *HTTPFactory {
	return &HTTPFactory{}
}

// NewClient creates a new http client.
func (f *HTTPFactory) NewClient(contractAddress bitcoin.RawAddress, url string,
	publicKey bitcoin.PublicKey) (Client, error) {
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		url = "https://" + url
	}
	return NewHTTPClient(contractAddress, url, publicKey)
}

// GetHTTPClient fetches an HTTP oracle client's data from the URL.
func GetHTTPClient(ctx context.Context, baseURL string) (*HTTPClient, error) {
	result := &HTTPClient{
		URL: baseURL,
	}

	var response struct {
		Data struct {
			ContractAddress bitcoin.RawAddress `json:"contract_address"`
			PublicKey       bitcoin.PublicKey  `json:"public_key"`
		}
	}

	if err := get(ctx, result.URL+"/oracle/id", &response); err != nil {
		return nil, errors.Wrap(err, "http get")
	}

	result.ContractAddress = response.Data.ContractAddress
	result.PublicKey = response.Data.PublicKey

	return result, nil
}

// NewHTTPClient creates an HTTP oracle client from specified data.
func NewHTTPClient(contractAddress bitcoin.RawAddress, url string,
	publicKey bitcoin.PublicKey) (*HTTPClient, error) {
	return &HTTPClient{
		ContractAddress: contractAddress,
		URL:             url,
		PublicKey:       publicKey,
	}, nil
}

// GetContractAddress returns the oracle's contract address.
func (o *HTTPClient) GetContractAddress() bitcoin.RawAddress {
	return o.ContractAddress
}

// GetURL returns the oracle's service URL.
func (o *HTTPClient) GetURL() string {
	return o.URL
}

// GetPublicKey returns the oracle's public key.
func (o *HTTPClient) GetPublicKey() bitcoin.PublicKey {
	return o.PublicKey
}

// SetClientID sets the client's ID and authorization key.
func (o *HTTPClient) SetClientID(id uuid.UUID, key bitcoin.Key) {
	o.ClientID = id
	o.ClientKey = key
}

// SetClientKey sets the client's authorization key.
func (o *HTTPClient) SetClientKey(key bitcoin.Key) {
	o.ClientKey = key
}

// RegisterUser checks if a user for this entity exists with the identity oracle and if not
// registers a new user id.
func (o *HTTPClient) RegisterUser(ctx context.Context, entity actions.EntityField,
	xpubs []bitcoin.ExtendedKeys) (*uuid.UUID, error) {

	for _, xps := range xpubs {
		for _, xpub := range xps {
			if xpub.IsPrivate() {
				return nil, errors.New("private keys not allowed")
			}
		}
	}

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

		if err := post(ctx, o.URL+"/oracle/user", request, &response); err != nil {
			if errors.Cause(err) == ErrNotFound {
				continue
			}
			return nil, errors.Wrap(err, "http post")
		}

		o.ClientID = response.Data.UserID
		return &o.ClientID, nil
	}

	// Call endpoint to register user and get ID.
	request := struct {
		Entity    actions.EntityField `json:"entity"`     // hex protobuf
		PublicKey bitcoin.PublicKey   `json:"public_key"` // hex compressed
		Signature bitcoin.Signature   `json:"signature"`
	}{
		Entity:    entity,
		PublicKey: o.ClientKey.PublicKey(),
	}

	// Sign entity
	s := sha256.New()
	if err := request.Entity.WriteDeterministic(s); err != nil {
		return nil, errors.Wrap(err, "write entity")
	}
	hash := bitcoin.Hash32(sha256.Sum256(s.Sum(nil)))

	var err error
	request.Signature, err = o.ClientKey.Sign(hash)
	if err != nil {
		return nil, errors.Wrap(err, "sign")
	}

	// Look for 200 OK status with data
	var response struct {
		Data struct {
			Status string    `json:"status"`
			UserID uuid.UUID `json:"user_id"`
		}
	}

	if err := post(ctx, o.URL+"/oracle/register", request, &response); err != nil {
		return nil, errors.Wrap(err, "http post")
	}

	o.ClientID = response.Data.UserID
	return &o.ClientID, nil
}

// RegisterXPub checks if the xpub is already added to the identity user and if not adds it to the
//   identity oracle.
func (o *HTTPClient) RegisterXPub(ctx context.Context, path string, xpubs bitcoin.ExtendedKeys,
	requiredSigners int) error {

	if len(o.ClientID) == 0 {
		return errors.New("User not registered")
	}

	for _, xpub := range xpubs {
		if xpub.IsPrivate() {
			return errors.New("private keys not allowed")
		}
	}

	// Add xpub to user using identity oracle endpoint.
	request := struct {
		UserID          uuid.UUID            `json:"user_id"`
		XPubs           bitcoin.ExtendedKeys `json:"xpubs"`
		RequiredSigners int                  `json:"required_signers"`
		Signature       bitcoin.Signature    `json:"signature"`
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
	hash := bitcoin.Hash32(sha256.Sum256(s.Sum(nil)))

	var err error
	request.Signature, err = o.ClientKey.Sign(hash)
	if err != nil {
		return errors.Wrap(err, "sign")
	}

	if err := post(ctx, o.URL+"/oracle/addXPub", request, nil); err != nil {
		return errors.Wrap(err, "http post")
	}

	return nil
}

// UpdateIdentity updates the user's identity information with the identity oracle.
func (o *HTTPClient) UpdateIdentity(ctx context.Context, entity actions.EntityField) error {

	if len(o.ClientID) == 0 {
		return errors.New("User not registered")
	}

	// Add xpub to user using identity oracle endpoint.
	request := struct {
		UserID    uuid.UUID           `json:"user_id"`
		Entity    actions.EntityField `json:"entity"`
		Signature bitcoin.Signature   `json:"signature"`
	}{
		UserID: o.ClientID,
		Entity: entity,
	}

	// Sign entity
	s := sha256.New()
	if _, err := s.Write(request.UserID[:]); err != nil {
		return errors.Wrap(err, "write user id")
	}
	if err := request.Entity.WriteDeterministic(s); err != nil {
		return errors.Wrap(err, "write entity")
	}
	hash := bitcoin.Hash32(sha256.Sum256(s.Sum(nil)))

	var err error
	request.Signature, err = o.ClientKey.Sign(hash)
	if err != nil {
		return errors.Wrap(err, "sign")
	}

	if err := post(ctx, o.URL+"/oracle/updateIdentity", request, nil); err != nil {
		return errors.Wrap(err, "http post")
	}

	return nil
}

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

// ApproveEntityPublicKey requests a signature from the identity oracle to verify the ownership of
//   a public key by a specified entity.
func (o *HTTPClient) ApproveEntityPublicKey(ctx context.Context, entity actions.EntityField,
	xpub bitcoin.ExtendedKey, index uint32) (*ApprovedEntityPublicKey, error) {

	if xpub.IsPrivate() {
		return nil, errors.New("private keys not allowed")
	}

	key, err := xpub.ChildKey(index)
	if err != nil {
		return nil, errors.Wrap(err, "generate public key")
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

	if err := post(ctx, o.URL+"/identity/verifyPubKey", request, &response); err != nil {
		return nil, errors.Wrap(err, "http post")
	}

	if !response.Data.Approved {
		return nil, ErrNotApproved
	}

	result := &ApprovedEntityPublicKey{
		SigAlgorithm: response.Data.SigAlgorithm,
		Signature:    response.Data.Signature,
		BlockHeight:  response.Data.BlockHeight,
		PublicKey:    key.PublicKey(),
	}

	return result, nil
}

// ApproveReceive requests a signature from the identity oracle to approve receipt of a token.
// quantity is simply placed in the result data structure and not used in the certificate.
func (o *HTTPClient) ApproveReceive(ctx context.Context, contract, asset string, oracleIndex int,
	quantity uint64, xpubs bitcoin.ExtendedKeys, index uint32,
	requiredSigners int) (*actions.AssetReceiverField, bitcoin.Hash32, error) {

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
	}{
		XPubs:    xpubs,
		Index:    index,
		Contract: contract,
		AssetID:  asset,
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

	if err := post(ctx, o.URL+"/transfer/approve", request, &response); err != nil {
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

// post sends an HTTP POST request.
func post(ctx context.Context, url string, request, response interface{}) error {
	var transport = &http.Transport{
		Dial: (&net.Dialer{
			Timeout: 5 * time.Second,
		}).Dial,
		TLSHandshakeTimeout: 5 * time.Second,
	}

	var client = &http.Client{
		Timeout:   time.Second * 10,
		Transport: transport,
	}

	b, err := json.Marshal(request)
	if err != nil {
		return errors.Wrap(err, "marshal request")
	}

	logger.Verbose(ctx, "POST URL : %s\nRequest : %s", url, string(b))

	httpResponse, err := client.Post(url, "application/json", bytes.NewReader(b))
	if err != nil {
		return err
	}

	if httpResponse.StatusCode < 200 || httpResponse.StatusCode > 299 {
		switch httpResponse.StatusCode {
		case http.StatusNotFound:
			return errors.Wrap(ErrNotFound, httpResponse.Status)
		case http.StatusUnauthorized:
			return errors.Wrap(ErrUnauthorized, httpResponse.Status)
		}

		return fmt.Errorf("%d %s", httpResponse.StatusCode, httpResponse.Status)
	}

	defer httpResponse.Body.Close()

	if response != nil {
		b, err := ioutil.ReadAll(httpResponse.Body)
		if err != nil {
			return errors.Wrap(err, "read response")
		}
		if err := json.Unmarshal(b, response); err != nil {
			return errors.Wrap(err, fmt.Sprintf("decode response : \n%s\n", string(b)))
		}
	}

	return nil
}

// get sends an HTTP GET request.
func get(ctx context.Context, url string, response interface{}) error {
	var transport = &http.Transport{
		Dial: (&net.Dialer{
			Timeout: 5 * time.Second,
		}).Dial,
		TLSHandshakeTimeout: 5 * time.Second,
	}

	var client = &http.Client{
		Timeout:   time.Second * 10,
		Transport: transport,
	}

	logger.Verbose(ctx, "GET URL : %s", url)

	httpResponse, err := client.Get(url)
	if err != nil {
		return err
	}

	if httpResponse.StatusCode < 200 || httpResponse.StatusCode > 299 {
		switch httpResponse.StatusCode {
		case http.StatusNotFound:
			return errors.Wrap(ErrNotFound, httpResponse.Status)
		case http.StatusUnauthorized:
			return errors.Wrap(ErrUnauthorized, httpResponse.Status)
		}
		return fmt.Errorf("%v %s", httpResponse.StatusCode, httpResponse.Status)
	}

	defer httpResponse.Body.Close()

	if response != nil {
		b, err := ioutil.ReadAll(httpResponse.Body)
		if err != nil {
			return errors.Wrap(err, "read response")
		}
		if err := json.Unmarshal(b, response); err != nil {
			return errors.Wrap(err, fmt.Sprintf("decode response : \n%s\n", string(b)))
		}
	}

	return nil
}
