package authority

import (
	"context"
	"encoding/hex"

	"github.com/tokenized/pkg/bitcoin"

	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
)

func GetOracle(ctx context.Context, baseURL string) (*Oracle, error) {
	result := &Oracle{
		BaseURL: baseURL,
	}

	var response struct {
		Data struct {
			Entity    string `json:"entity"`
			URL       string `json:"url"`
			PublicKey string `json:"public_key"`
		}
	}

	if err := get(result.BaseURL+"/oracle/id", &response); err != nil {
		return nil, errors.Wrap(err, "http get")
	}

	entityBytes, err := hex.DecodeString(response.Data.Entity)
	if err != nil {
		return nil, errors.Wrap(err, "decode entity hex")
	}

	if err := proto.Unmarshal(entityBytes, &result.OracleEntity); err != nil {
		return nil, errors.Wrap(err, "deserialize entity")
	}

	result.OracleKey, err = bitcoin.PublicKeyFromStr(response.Data.PublicKey)
	if err != nil {
		return nil, errors.Wrap(err, "decode public key")
	}

	return result, nil
}
