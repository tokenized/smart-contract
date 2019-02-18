package node

import (
	"net"

	"github.com/tokenized/smart-contract/internal/platform/config"
	"github.com/tokenized/smart-contract/internal/platform/inspector"
	"github.com/tokenized/smart-contract/internal/platform/network"
	"github.com/tokenized/smart-contract/internal/platform/state"
	"github.com/tokenized/smart-contract/internal/platform/wallet"
	"github.com/tokenized/smart-contract/internal/request"
	"github.com/tokenized/smart-contract/internal/response"
	"github.com/tokenized/smart-contract/internal/validator"
	"github.com/tokenized/smart-contract/pkg/storage"
	"github.com/tokenized/smart-contract/pkg/wire"
)

const (
	MainNetBch wire.BitcoinNet = 0xe8f3e1e3
	TestNetBch wire.BitcoinNet = 0xf4f3e5f4
	RegTestBch wire.BitcoinNet = 0xfabfb5da
)

type Node struct {
	Config   config.Config
	Network  network.NetworkInterface
	State    state.StateInterface
	Wallet   wallet.Wallet
	conn     net.Conn
	messages chan wire.Message
	storage  storage.Storage
}

func NewNode(config config.Config,
	network network.NetworkInterface,
	wallet wallet.Wallet,
	storage storage.Storage) Node {

	contractState := state.NewStateService(storage)

	a := Node{
		Config:   config,
		Network:  network,
		Wallet:   wallet,
		messages: make(chan wire.Message),
		storage:  storage,
		State:    contractState,
	}

	return a
}

func (n Node) Start() error {
	inspector := inspector.NewInspectorService(n.Network)
	validator := validator.NewValidatorService(n.Config, n.Wallet, n.State)
	request := request.NewRequestService(n.Config, n.Wallet, n.State, inspector)
	response := response.NewResponseService(n.Config, n.Wallet, n.State, inspector)

	txHandler := NewTXHandler(n.Config,
		n.Network,
		n.Wallet,
		inspector,
		validator,
		request,
		response)

	n.Network.RegisterTxListener(txHandler)

	// blockHandler := contract.NewBlockHandler(n.Config, service)
	// network.RegisterBlockListener(blockHandler)

	return n.Network.Start()
}
