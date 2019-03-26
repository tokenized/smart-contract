package node

import (
	"context"

	"github.com/btcsuite/btcutil"
	"github.com/tokenized/smart-contract/internal/platform/protomux"
	"github.com/tokenized/smart-contract/pkg/inspector"
	"github.com/tokenized/smart-contract/pkg/wire"
)

type ResponseWriter struct {
	Inputs  []inspector.UTXO
	Outputs []Output
	Config  *Config
	Mux     protomux.Handler
}

// AddChangeOutput is a helper to add a change output
func (w *ResponseWriter) AddChangeOutput(ctx context.Context, addr btcutil.Address) error {
	out := outputValue(ctx, w.Config, addr, 0, true)
	w.Outputs = append(w.Outputs, *out)
	return nil
}

// AddOutput is a helper to add a payment output
func (w *ResponseWriter) AddOutput(ctx context.Context, addr btcutil.Address, value uint64) error {
	out := outputValue(ctx, w.Config, addr, value, false)
	w.Outputs = append(w.Outputs, *out)
	return nil
}

// AddFee attaches the fee as the next output, if configured
func (w *ResponseWriter) AddContractFee(ctx context.Context, value uint64) error {
	if fee := outputFee(ctx, w.Config, value); fee != nil {
		w.Outputs = append(w.Outputs, *fee)
	}
	return nil
}

// SetUTXOs is an optional function that allows explicit UTXOs to be spent in the response
// be sure to remember any remaining UTXOs so they can be spent later.
func (w *ResponseWriter) SetUTXOs(ctx context.Context, utxos []inspector.UTXO) error {
	w.Inputs = utxos
	return nil
}

// Respond sends the prepared response to the protocol mux
func (w *ResponseWriter) Respond(ctx context.Context, tx *wire.MsgTx) error {
	return w.Mux.Respond(ctx, tx)
}

// Output is an output address for a response
type Output struct {
	Address btcutil.Address
	Value   uint64
	Change  bool
}

// outputValue returns a payment output ensuring the value is always above the dust limit
func outputValue(ctx context.Context, config *Config, addr btcutil.Address, value uint64, change bool) *Output {
	if value < config.DustLimit {
		value = config.DustLimit
	}

	out := &Output{
		Address: addr,
		Value:   value,
		Change:  change,
	}

	return out
}

// outputFee prepares a special fee output based on node configuration
func outputFee(ctx context.Context, config *Config, value uint64) *Output {
	if config.FeeValue > 0 {
		feeAddr, _ := btcutil.DecodeAddress(config.FeeAddress, &config.ChainParams)
		return &Output{
			Address: feeAddr,
			Value:   value,
		}
	}

	return nil
}
