package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/tokenized/smart-contract/pkg/bitcoin"
)

func main() {
	// Outputs the double sha256 of the specified text in hex format.
	//
	// example
	//   shasha256 test
	var data string

	if len(os.Args) > 1 {
		data = strings.Join(os.Args[1:], " ")
	} else {
		// no data so trying stdin
		s := bufio.NewScanner(os.Stdin)
		for s.Scan() {
			data = s.Text()
		}
	}

	b := bitcoin.DoubleSha256([]byte(data))

	fmt.Printf("%x\n", b)
}
