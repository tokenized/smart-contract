package contract

import (
	"context"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/tokenized/smart-contract/pkg/protocol"
	"github.com/tokenized/smart-contract/pkg/wire"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcutil"
)

// Contract represents a Smart Contract.
type Contract struct {
	ID                          string           `json:"id"`
	CreatedAt                   int64            `json:"created_at"`
	IssuerAddress               string           `json:"issuer_address"`
	OperatorAddress             string           `json:"operator_address"`
	Revision                    uint16           `json:"revision"`
	ContractName                string           `json:"name"`
	ContractFileHash            string           `json:"hash"`
	GoverningLaw                string           `json:"law"`
	Jurisdiction                string           `json:"jurisdiction"`
	ContractExpiration          uint64           `json:"contract_expiration"`
	URI                         string           `json:"uri"`
	IssuerID                    string           `json:"issuer_id"`
	IssuerType                  string           `json:"issuer_type"`
	ContractOperatorID          string           `json:"tokenizer_id"`
	AuthorizationFlags          []byte           `json:"authorization_flags"`
	VotingSystem                string           `json:"voting_system"`
	InitiativeThreshold         float32          `json:"initiative_threshold"`
	InitiativeThresholdCurrency string           `json:"initiative_threshold_currency"`
	Qty                         uint64           `json:"qty"`
	Assets                      map[string]Asset `json:"assets"`
	Votes                       map[string]Vote  `json:"votes"`
	Hashes                      []string         `json:"hashes"`
}

func NewContract(tx *wire.MsgTx,
	contractAddress btcutil.Address,
	issuerAddress btcutil.Address,
	operatorAddress btcutil.Address) *Contract {

	c := Contract{
		ID:            contractAddress.EncodeAddress(),
		CreatedAt:     time.Now().UnixNano(),
		IssuerAddress: issuerAddress.EncodeAddress(),
		Hashes:        []string{},
		Assets:        map[string]Asset{},
		Votes:         map[string]Vote{},
	}

	if operatorAddress != nil {
		c.OperatorAddress = operatorAddress.EncodeAddress()
	}

	return &c
}

func EditContract(c *Contract, cf *protocol.ContractFormation) *Contract {
	newContract := c

	newContract.ContractName = string(cf.ContractName)
	newContract.ContractFileHash = fmt.Sprintf("%x", cf.ContractFileHash)
	newContract.GoverningLaw = string(cf.GoverningLaw)
	newContract.Jurisdiction = string(cf.Jurisdiction)
	newContract.ContractExpiration = cf.ContractExpiration
	newContract.URI = string(cf.URI)
	newContract.Revision = cf.ContractRevision
	newContract.IssuerID = string(cf.IssuerID)
	newContract.IssuerType = string(cf.IssuerType)
	newContract.ContractOperatorID = string(cf.ContractOperatorID)
	newContract.AuthorizationFlags = cf.AuthorizationFlags
	newContract.VotingSystem = string(cf.VotingSystem)
	newContract.InitiativeThreshold = cf.InitiativeThreshold
	newContract.InitiativeThresholdCurrency = string(cf.InitiativeThresholdCurrency)
	newContract.Qty = cf.RestrictedQty

	if cf.VotingSystem == 0x0 {
		c.VotingSystem = ""
	}

	if cf.IssuerType == 0x0 {
		c.IssuerType = ""
	}

	return newContract
}

// Address returns the contract ID as an Address.
func (c Contract) Address() (btcutil.Address, error) {
	return btcutil.DecodeAddress(c.ID, &chaincfg.MainNetParams)
}

// Flags converts the AuthorizationFlags as a uint16.
func (c Contract) Flags() uint16 {
	if len(c.AuthorizationFlags) != 2 {
		return 0
	}

	return binary.BigEndian.Uint16(c.AuthorizationFlags)
}

func (c Contract) IsIssuer(address string) bool {
	return c.IssuerAddress == address
}

func (c Contract) IsOperator(address string) bool {
	return c.OperatorAddress == address
}

func (c Contract) IsOwner(address string) bool {
	for _, asset := range c.Assets {
		if _, ok := asset.Holdings[address]; ok {
			return true
		}
	}

	return false
}

func (c Contract) CanVote(v Vote, b Ballot) uint8 {
	if !c.IsOwner(b.Address) {
		return protocol.RejectionCodeUnknownAddress
	}

	if !v.IsOpen(time.Now()) {
		return protocol.RejectionCodeVoteClosed
	}

	return protocol.RejectionCodeOK
}

func (c Contract) getUsers() []string {
	users := []string{}

	for _, asset := range c.Assets {
		for address, holding := range asset.Holdings {
			if holding.Balance == 0 {
				continue
			}

			users = append(users, address)
		}
	}

	return users
}

func (c Contract) KnownTX(ctx context.Context, tx *wire.MsgTx) bool {
	txHash := tx.TxHash().String()

	for _, hash := range c.Hashes {
		if hash == txHash {
			return true
		}
	}

	return false
}
