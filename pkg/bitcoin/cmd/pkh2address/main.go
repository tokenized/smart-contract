package main

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"os"

	"github.com/tokenized/smart-contract/pkg/bitcoin"
)

func main() {
	// Converts a public key hash in hex format into a mainnet bitcoin address.
	//
	// example
	//   pkh2address 4974a24418c676add75fc291fccf3e2253ceb21d
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

	pkh, err := hex.DecodeString(data)
	if err != nil {
		panic(err)
	}

	b, err := bitcoin.AddressPKHFromBytes(pkh)
	if err != nil {
		panic(err)
	}

	fmt.Printf("%s\n", b.String(bitcoin.MainNet))
}
