package operator

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/tokenized/pkg/bitcoin"
)

type HTTPFactory struct{}

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

func NewHTTPFactory() *HTTPFactory {
	return &HTTPFactory{}
}

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

func (c *HTTPClient) FetchContractAddress(ctx context.Context) (bitcoin.RawAddress, uint64, error) {
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

	resp, err := client.Get(c.URL + "/new_contract")
	if err != nil {
		return bitcoin.RawAddress{}, 0, errors.Wrap(err, "http get")
	}

	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return bitcoin.RawAddress{}, 0, fmt.Errorf("%d %s", resp.StatusCode, resp.Status)
	}

	var newContract struct {
		EntityContract bitcoin.RawAddress `json:"entity_contract,omitempty"`
		Address        bitcoin.RawAddress `json:"address,omitempty"`
		ContractFee    uint64             `json:"contract_fee,omitempty"`
		Error          string             `json:"error,omitempty"`
		Signature      bitcoin.Signature  `json:"signature,omitempty"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&newContract); err != nil {
		return bitcoin.RawAddress{}, 0, errors.Wrap(err, "unmarshal response")
	}

	if len(newContract.Error) > 0 {
		return bitcoin.RawAddress{}, 0, errors.New(newContract.Error)
	}

	// Validate signature
	s := sha256.New()
	if _, err := s.Write(newContract.Address.Bytes()); err != nil {
		return bitcoin.RawAddress{}, 0, errors.Wrap(err, "hash contract address")
	}
	if err := binary.Write(s, binary.LittleEndian, newContract.ContractFee); err != nil {
		return bitcoin.RawAddress{}, 0, errors.Wrap(err, "hash contract fee")
	}
	h := sha256.Sum256(s.Sum(nil))

	if !newContract.Signature.Verify(h[:], c.PublicKey) {
		return bitcoin.RawAddress{}, 0, errors.New("Invalid operator signature")
	}

	return newContract.Address, newContract.ContractFee, nil
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
