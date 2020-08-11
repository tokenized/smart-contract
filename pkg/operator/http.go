package operator

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"time"

	"github.com/tokenized/pkg/bitcoin"
	"github.com/tokenized/pkg/wire"

	"github.com/google/uuid"
	"github.com/pkg/errors"
)

var (
	ErrNotFound = errors.New("Not Found")
)

// HTTPFactory implements the factory interface that creates http clients.
type HTTPFactory struct{}

// HTTPClient implements the client interface to perform HTTP requests to contract operators.
type HTTPClient struct {
	// Service information
	ContractAddress bitcoin.RawAddress // Address of contract entity.
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

	if err := get(result.URL+"/id", &response); err != nil {
		return nil, errors.Wrap(err, "http get")
	}

	result.ContractAddress = response.Data.ContractAddress
	result.PublicKey = response.Data.PublicKey

	return result, nil
}

// NewHTTPClient creates an HTTP oracle client from specified data.
func NewHTTPClient(contractAddress bitcoin.RawAddress, url string, publicKey bitcoin.PublicKey) (*HTTPClient, error) {
	return &HTTPClient{
		ContractAddress: contractAddress,
		URL:             url,
		PublicKey:       publicKey,
	}, nil
}

// FetchContractAddress fetches a new contract address from the contract operator.
// Returns contract address, contract fee, and master address.
// The master address is optional to use.
func (c *HTTPClient) FetchContractAddress(ctx context.Context) (bitcoin.RawAddress, uint64,
	bitcoin.RawAddress, error) {

	var response struct {
		EntityContract bitcoin.RawAddress `json:"entity_contract,omitempty"`
		Address        bitcoin.RawAddress `json:"address,omitempty"`
		MasterAddress  bitcoin.RawAddress `json:"master_address,omitempty"`
		ContractFee    uint64             `json:"contract_fee,omitempty"`
		Error          string             `json:"error,omitempty"`
		Signature      bitcoin.Signature  `json:"signature,omitempty"`
	}

	if err := get(c.URL+"/new_contract", &response); err != nil {
		return bitcoin.RawAddress{}, 0, bitcoin.RawAddress{}, errors.Wrap(err, "http get")
	}

	if len(response.Error) > 0 {
		return bitcoin.RawAddress{}, 0, bitcoin.RawAddress{}, errors.New(response.Error)
	}

	// Validate signature
	s := sha256.New()
	if _, err := s.Write(response.Address.Bytes()); err != nil {
		return bitcoin.RawAddress{}, 0, bitcoin.RawAddress{},
			errors.Wrap(err, "hash contract address")
	}
	if err := binary.Write(s, binary.LittleEndian, response.ContractFee); err != nil {
		return bitcoin.RawAddress{}, 0, bitcoin.RawAddress{}, errors.Wrap(err, "hash contract fee")
	}
	if _, err := s.Write(response.MasterAddress.Bytes()); err != nil {
		return bitcoin.RawAddress{}, 0, bitcoin.RawAddress{},
			errors.Wrap(err, "hash contract address")
	}
	h := sha256.Sum256(s.Sum(nil))

	if !response.Signature.Verify(h[:], c.PublicKey) {
		return bitcoin.RawAddress{}, 0, bitcoin.RawAddress{},
			errors.New("Invalid operator signature")
	}

	return response.Address, response.ContractFee, response.MasterAddress, nil
}

// SignContractOffer adds a signed input to a contract offer transaction.
func (c *HTTPClient) SignContractOffer(ctx context.Context, tx *wire.MsgTx) (*wire.MsgTx, *bitcoin.UTXO, error) {

	request := struct {
		Tx *wire.MsgTx `json:"tx"`
	}{
		Tx: tx,
	}

	var response struct {
		Tx   *wire.MsgTx   `json:"tx"`
		UTXO *bitcoin.UTXO `json:"utxo"`
	}

	if err := post(c.URL+"/sign_contract", request, &response); err != nil {
		return nil, nil, errors.Wrap(err, "http get")
	}

	return response.Tx, response.UTXO, nil
}

// GetContractAddress returns the oracle's contract address.
func (c *HTTPClient) GetContractAddress() bitcoin.RawAddress {
	return c.ContractAddress
}

// GetURL returns the oracle's URL.
func (c *HTTPClient) GetURL() string {
	return c.URL
}

// GetPublicKey returns the oracle's public key.
func (c *HTTPClient) GetPublicKey() bitcoin.PublicKey {
	return c.PublicKey
}

// SetClientID sets the client's ID and authorization key.
func (c *HTTPClient) SetClientID(id uuid.UUID, key bitcoin.Key) {
	c.ClientID = id
	c.ClientKey = key
}

// SetClientKey sets the client's authorization key.
func (c *HTTPClient) SetClientKey(key bitcoin.Key) {
	c.ClientKey = key
}

// post sends an HTTP POST request.
func post(url string, request, response interface{}) error {
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

	httpResponse, err := client.Post(url, "application/json", bytes.NewReader(b))
	if err != nil {
		return err
	}

	if httpResponse.StatusCode < 200 || httpResponse.StatusCode > 299 {
		if httpResponse.StatusCode == 404 {
			return errors.Wrap(ErrNotFound, httpResponse.Status)
		}
		if httpResponse.Body != nil {
			message, err := ioutil.ReadAll(httpResponse.Body)
			if err == nil && len(message) > 0 {
				return fmt.Errorf("%v %s : %s", httpResponse.StatusCode, httpResponse.Status,
					string(message))
			}
		}
		return fmt.Errorf("%v %s", httpResponse.StatusCode, httpResponse.Status)
	}

	defer httpResponse.Body.Close()

	if response != nil {
		if err := json.NewDecoder(httpResponse.Body).Decode(response); err != nil {
			return errors.Wrap(err, "decode response")
		}
	}

	return nil
}

// get sends an HTTP GET request.
func get(url string, response interface{}) error {
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

	httpResponse, err := client.Get(url)
	if err != nil {
		return err
	}

	if httpResponse.StatusCode < 200 || httpResponse.StatusCode > 299 {
		return fmt.Errorf("%v %s", httpResponse.StatusCode, httpResponse.Status)
	}

	defer httpResponse.Body.Close()

	if response != nil {
		if err := json.NewDecoder(httpResponse.Body).Decode(response); err != nil {
			return errors.Wrap(err, "decode response")
		}
	}

	return nil
}
