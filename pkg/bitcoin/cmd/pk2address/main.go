package main

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"os"

	"github.com/tokenized/smart-contract/pkg/bitcoin"
)

func main() {
	// Converts a serialized compressed public key in hex format into a mainnet bitcoin address.
	//
	// example
	//   pk2address 0250863ad64a87ae8a2fe83c1af1a8403cb53f53e486d8511dad8a04887e5b2352
	var data string

	if len(os.Args) > 1 {
		data = os.Args[1]
	} else {
		// no data so trying stdin
		s := bufio.NewScanner(os.Stdin)
		for s.Scan() {
			data = s.Text()
		}
	}

	pubKey, err := hex.DecodeString(data)
	if err != nil {
		panic(err)
	}

	b, err := bitcoin.AddressPKHFromBytes(bitcoin.Hash160(pubKey))
	if err != nil {
		panic(err)
	}

	fmt.Printf("%s\n", b.String(bitcoin.MainNet))
}
