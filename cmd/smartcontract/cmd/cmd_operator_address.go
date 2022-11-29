package cmd

import (
	"context"
	"fmt"

	"github.com/tokenized/pkg/bitcoin"
	"github.com/tokenized/smart-contract/pkg/operator"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var cmdOperatorAddress = &cobra.Command{
	Use:   "operator_address client_wif_key operator_url operator_public_key",
	Short: "Request a smart contract agent address from a contract operator service.",
	Args:  cobra.ExactArgs(3),
	RunE: func(c *cobra.Command, args []string) error {
		clientKey, err := bitcoin.KeyFromStr(args[0])
		if err != nil {
			return errors.Wrap(err, "client_wif_key")
		}

		url := args[1]

		publicKey, err := bitcoin.PublicKeyFromStr(args[2])
		if err != nil {
			return errors.Wrap(err, "operator_public_key")
		}

		client, err := operator.NewHTTPClient(bitcoin.RawAddress{}, url, publicKey, clientKey)
		if err != nil {
			return errors.Wrap(err, "HTTP client")
		}

		ctx := context.Background()
		net := bitcoin.MainNet
		contractAddress, contractFee, masterAddress, err := client.FetchContractAddress(ctx)
		if err != nil {
			return errors.Wrap(err, "fetch address")
		}

		fmt.Printf("Contract: %s\n", bitcoin.NewAddressFromRawAddress(contractAddress, net))
		fmt.Printf("Contract Fee: %d\n", contractFee)
		fmt.Printf("Master: %s\n", bitcoin.NewAddressFromRawAddress(masterAddress, net))

		return nil
	},
}
