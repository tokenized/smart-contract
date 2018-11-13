package rebuilder

/*
import (
	"context"
	"errors"

	"github.com/tokenized/smart-contract/internal/app/logger"
	"github.com/tokenized/smart-contract/internal/app/network"
	"github.com/tokenized/smart-contract/pkg/protocol"
	"github.com/tokenized/smart-contract/pkg/wire"

	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcutil"
)

type ContractBuilder struct {
	ContractService contract.ContractService
	Network         network.NetworkInterface
	Parser          contract.Parser
	UTXOSetBuilder  contract.UTXOSetBuilder
}

func NewContractBuilder(contractService contract.ContractService,
	network network.NetworkInterface,
	txService contract.Parser) ContractBuilder {

	builder := contract.NewUTXOSetBuilder(network)

	return ContractBuilder{
		ContractService: contractService,
		Network:         network,
		Parser:          txService,
		UTXOSetBuilder:  builder,
	}
}

func (c ContractBuilder) Rebuild(ctx context.Context,
	address btcutil.Address) (*Contract, error) {
	logger := logger.NewLoggerFromContext(ctx).Sugar()

	txs, err := c.getContractTX(ctx, address)
	if err != nil {
		return nil, err
	}

	if len(txs) == 0 {
		return nil, nil
	}

	// create the contract from the first message
	resp := ContractResponse{}

	tx, err := c.findFormationTX(ctx, txs)
	if err != nil {
		return nil, err
	}

	m, err := findTokenizedProtocol(ctx, tx)
	if err != nil {
		return nil, err
	}

	co := m.(*protocol.ContractOffer)

	utxos, err := c.UTXOSetBuilder.Build(tx)
	if err != nil {
		return nil, err
	}

	issuer, err := utxos[0].PublicAddress(&chaincfg.MainNetParams)
	if err != nil {
		return nil, err
	}

	receivers, err := c.Parser.GetOutputs(ctx, tx)
	if err != nil {
		return nil, err
	}
	receiver := receivers[0]

	contract := NewContract(tx, receiver.Address, issuer, nil, co)

	for _, tx := range txs {
		m, err := findTokenizedProtocol(ctx, tx)
		if err != nil {
			return nil, err
		}

		if m == nil {
			continue
		}

		// has this TX been seen before?
		if contract.KnownTX(ctx, tx) {
			// skipping, this TX has been seen
			continue
		}

		receivers, err := c.Parser.GetOutputs(ctx, tx)
		if err != nil {
			logger.Error(err)
			continue
			// return nil, err
		}

		if len(receivers) == 0 {
			logger.Infof("No contract receiver")
			continue
			// return nil, errors.New("No contract receiver")
		}

		utxos, err := c.UTXOSetBuilder.Build(tx)
		if err != nil {
			return nil, err
		}

		senders := []btcutil.Address{}
		for _, utxo := range utxos {
			a, err := utxo.PublicAddress(&chaincfg.MainNetParams)
			if err != nil {
				logger.Error(err)
				break
				// return nil, err
			}

			senders = append(senders, a)
		}

		if len(senders) == 0 {
			continue
		}

		cs := c.ContractService

		cr, err := cs.Apply(ctx, tx, senders, receivers, *contract, m)
		if err != nil {
			logger.Error(err)
			continue
			// return nil, err
		}

		contract = &cr.Contract

		resp = *cr
	}

	return &resp.Contract, nil
}

func (c ContractBuilder) getContractTX(ctx context.Context,
	address btcutil.Address) ([]*wire.MsgTx, error) {

	// get a list of transactions for the address from the API
	list, err := c.Network.ListTransactions(ctx, address)
	if err != nil {
		return nil, err
	}

	// find tx's that were sent to the contract address
	txs, err := c.findTXs(ctx, list, address)
	if err != nil {
		return nil, err
	}

	reverse(txs)

	return txs, nil
}

func (c ContractBuilder) findTXs(ctx context.Context,
	list []btcjson.ListTransactionsResult,
	address btcutil.Address) ([]*wire.MsgTx, error) {

	txs := []*wire.MsgTx{}

	for _, t := range list {
		hash, err := chainhash.NewHashFromStr(t.TxID)
		if err != nil {
			return nil, err
		}

		tx, err := c.Network.GetTX(ctx, hash)
		if err != nil {
			return nil, err
		}

		txs = append(txs, tx)
	}

	return txs, nil
}

func (c ContractBuilder) findFormationTX(ctx context.Context,
	list []*wire.MsgTx) (*wire.MsgTx, error) {

	for _, t := range list {
		m, err := findTokenizedProtocol(ctx, t)
		if err != nil {
			continue
		}

		if m == nil {
			continue
		}

		switch m.(type) {
		case *protocol.ContractOffer:
			return t, nil
		}
	}

	return nil, errors.New("Cannot find a CO")
}
*/
