package identity

import "github.com/tokenized/pkg/bitcoin"

type HTTPFactory struct{}

func NewHTTPFactory() *HTTPFactory {
	return &HTTPFactory{}
}

func (f *HTTPFactory) NewClient(contractAddress bitcoin.RawAddress, url string,
	publicKey bitcoin.PublicKey) (Client, error) {
	return NewHTTPClient(contractAddress, url, publicKey)
}
