package identity

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/tokenized/pkg/bitcoin"
	"github.com/tokenized/pkg/json"
	"github.com/tokenized/pkg/logger"

	"github.com/google/uuid"
	"github.com/pkg/errors"
)

var (
	ErrNotFound = errors.New("Not Found")
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
func NewHTTPClient(contractAddress bitcoin.RawAddress, url string, publicKey bitcoin.PublicKey) (*HTTPClient, error) {
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
		if httpResponse.StatusCode == 404 {
			return errors.Wrap(ErrNotFound, httpResponse.Status)
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
