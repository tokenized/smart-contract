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
	"strings"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcutil"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/tokenized/specification/dist/golang/protocol"
)

var cmdSign = &cobra.Command{
	Use:   "sign <typeCode> <jsonFile> <contract PKH> <receiverIndex> <blockhash> <wifkey>",
	Short: "Provide oracle signature",
	RunE: func(c *cobra.Command, args []string) error {
		if len(args) != 6 {
			return errors.New("Missing json file parameter")
		}

		return transferSign(c, args)
	},
}

func transferSign(c *cobra.Command, args []string) error {
	actionType := strings.ToUpper(args[0])

	// Create struct
	opReturn := protocol.TypeMapping(actionType)
	if opReturn == nil {
		fmt.Printf("Unsupported action type : %s\n", actionType)
		return nil
	}

	// Read json file
	path := filepath.FromSlash(args[1])
	data, err := ioutil.ReadFile(path)
	if err != nil {
		fmt.Printf("Failed to read json file : %s\n", err)
		return nil
	}

	// Put json data into opReturn struct
	if err := json.Unmarshal(data, opReturn); err != nil {
		fmt.Printf("Failed to unmarshal %s json file : %s\n", actionType, err)
		return nil
	}

	// Contract key
	hash := make([]byte, 20)
	n, err := hex.Decode(hash, []byte(args[2]))
	if err != nil {
		fmt.Printf("Invalid hash : %s\n", err)
		return nil
	}
	if n != 20 {
		fmt.Printf("Invalid hash size : %d\n", n)
		return nil
	}
	contractPKH := protocol.PublicKeyHashFromBytes(hash)

	receiverIndex, err := strconv.Atoi(args[3])
	if err != nil {
		fmt.Printf("Invalid receiver index : %s\n", err)
		return nil
	}

	// Block hash
	hash = make([]byte, 32)
	n, err = hex.Decode(hash, []byte(args[4]))
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

	wif, err := btcutil.DecodeWIF(args[5])
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
