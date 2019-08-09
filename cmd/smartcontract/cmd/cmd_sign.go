package cmd

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strconv"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcutil"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/tokenized/smart-contract/pkg/bitcoin"
	"github.com/tokenized/specification/dist/golang/protocol"
)

var cmdSign = &cobra.Command{
	Use:   "sign <jsonFile> <contract address> <receiverIndex> <blockhash> <oracle wifkey>",
	Short: "Provide oracle signature",
	RunE: func(c *cobra.Command, args []string) error {
		if len(args) != 5 {
			return errors.New("Invalid parameter count")
		}

		return transferSign(c, args)
	},
}

func transferSign(c *cobra.Command, args []string) error {
	network := network(c)
	if len(network) == 0 {
		return nil
	}
	params := bitcoin.NewChainParams(network)

	// Create struct
	opReturn := protocol.TypeMapping(protocol.CodeTransfer)
	if opReturn == nil {
		fmt.Printf("Unsupported action type : %s\n", protocol.CodeTransfer)
		return nil
	}

	// Read json file
	path := filepath.FromSlash(args[0])
	data, err := ioutil.ReadFile(path)
	if err != nil {
		fmt.Printf("Failed to read json file : %s\n", err)
		return nil
	}

	// Put json data into opReturn struct
	if err := json.Unmarshal(data, opReturn); err != nil {
		fmt.Printf("Failed to unmarshal %s json file : %s\n", protocol.CodeTransfer, err)
		return nil
	}

	// Contract key
	hash := make([]byte, 20)
	contractAddress, err := btcutil.DecodeAddress(args[1], params)
	if err != nil {
		fmt.Printf("Invalid contract address : %s\n", err)
		return nil
	}
	contractPKH := protocol.PublicKeyHashFromBytes(contractAddress.ScriptAddress())

	receiverIndex, err := strconv.Atoi(args[2])
	if err != nil {
		fmt.Printf("Invalid receiver index : %s\n", err)
		return nil
	}

	// Block hash
	hash = make([]byte, 32)
	n, err := hex.Decode(hash, []byte(args[3]))
	if err != nil {
		fmt.Printf("Failed to parse block hash : %s\n", err)
		return nil
	}
	if n != 32 {
		fmt.Printf("Invalid block hash size : %d\n", n)
		return nil
	}
	// Reverse hash (make little endian)
	reverseHash := make([]byte, 32)
	for i, b := range hash {
		reverseHash[31-i] = b
	}
	blockHash, err := chainhash.NewHash(reverseHash)
	if err != nil {
		fmt.Printf("Invalid block hash : %s\n", err)
		return nil
	}

	wif, err := btcutil.DecodeWIF(args[4])
	if err != nil {
		fmt.Printf("Invalid WIF key : %s\n", err)
		return nil
	}

	transfer, ok := opReturn.(*protocol.Transfer)
	if !ok {
		fmt.Printf("Not a transfer\n")
		return nil
	}

	index := 0
	for _, asset := range transfer.Assets {
		for _, receiver := range asset.AssetReceivers {
			if index == receiverIndex {
				fmt.Printf("Signing for PKH quantity %d : %s\n", receiver.Quantity, receiver.Address.String())
				hash, err := protocol.TransferOracleSigHash(context.Background(), contractPKH, &asset.AssetCode,
					&receiver.Address, receiver.Quantity, blockHash)
				if err != nil {
					fmt.Printf("Failed to generate sig hash : %s\n", err)
					return nil
				}
				fmt.Printf("Hash : %x\n", hash)

				signature, err := wif.PrivKey.Sign(hash)
				if err != nil {
					fmt.Printf("Failed to sign sig hash : %s\n", err)
					return nil
				}

				fmt.Printf("Signature : %x\n", signature.Serialize())
				fmt.Printf("Signature b64 : %s\n", base64.StdEncoding.EncodeToString(signature.Serialize()))
				return nil
			}

			index++
		}
	}

	fmt.Printf("Failed to find receiver index")
	return nil
}

func init() {
}
