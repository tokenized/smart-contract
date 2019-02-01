package config

/**
 * Config Kit
 *
 * What is my purpose?
 * - You store app wide settings
 * - You tell me about settings I want
 */

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcutil"
)

// Config holds all configuration for the running service.
type Config struct {
	ContractProviderID string
	Version            string
	Fee                Fee
}

// NewConfig returns a new Config populated from environment variables.
func NewConfig(operatorName, version, feeAddr, feeAmount string) (*Config, error) {
	c := Config{
		ContractProviderID: operatorName,
		Version:            version,
	}

	// Operator fee address
	feeAddress, err := btcutil.DecodeAddress(feeAddr, &chaincfg.MainNetParams)
	if err != nil {
		return nil, err
	}

	// Fee Value per TXN
	fee, err := strconv.ParseInt(feeAmount, 10, 64)
	if err != nil {
		return nil, err
	}

	c.Fee = Fee{
		Address: feeAddress,
		Value:   uint64(fee),
	}

	if c.Fee.Value == 0 {
		return nil, errors.New("Fee is set to 0 sats")
	}

	return &c, nil
}

// String returns a custom string representation.
//
// This is important so we don't log sensitive config values.
func (c Config) String() string {
	pairs := map[string]string{
		"ContractProviderID": c.ContractProviderID,
		"Version":            c.Version,
		"Fee":                fmt.Sprintf("%+v", c.Fee),
	}

	parts := []string{}

	for k, v := range pairs {
		parts = append(parts, fmt.Sprintf("%v:%v", k, v))
	}

	return fmt.Sprintf("{%v}", strings.Join(parts, " "))
}
