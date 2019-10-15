package cmd

import (
	"crypto/rand"
	"fmt"

	"github.com/btcsuite/btcutil/base58"
	"github.com/spf13/cobra"
)

var cmdCode = &cobra.Command{
	Use:   "code",
	Short: "Run random code",
	RunE: func(c *cobra.Command, args []string) error {

		rb := make([]byte, 24)
		rand.Read(rb)

		fmt.Printf("P2PKH Main 0x00 = %s\n", base58.Encode(append([]byte{0x00}, rb...)))
		fmt.Printf("P2PKH Test 0x6f = %s\n", base58.Encode(append([]byte{0x6f}, rb...)))
		fmt.Printf("P2SH Main 0x05 = %s\n", base58.Encode(append([]byte{0x05}, rb...)))
		fmt.Printf("P2SH Test 0xc4 = %s\n", base58.Encode(append([]byte{0xc4}, rb...)))
		fmt.Printf("\n")
		fmt.Printf("R Puzzle Main 0x7b = r = %s\n", base58.Encode(append([]byte{0x7b}, rb...)))
		fmt.Printf("R Puzzle Test 0x7d = s = %s\n", base58.Encode(append([]byte{0x7d}, rb...)))
		fmt.Printf("Multi PKH Main 0x76 = p = %s\n", base58.Encode(append([]byte{0x76}, rb...)))
		fmt.Printf("Multi PKH Test 0x78 = q = %s\n", base58.Encode(append([]byte{0x78}, rb...)))

		// for i := 0; i < 256; i++ {
		// 	for j := 0; j < 3; j++ {
		// 		rand.Read(rb)
		// 		data := append([]byte{byte(i)}, rb...)
		// 		fmt.Printf("0x%x = %s\n", i, base58.Encode(data))
		// 	}
		// }

		// for i := 0; i < 256; i++ {
		// 	rand.Read(rb)
		// 	data := append([]byte{0x25, byte(i)}, rb...)
		// 	fmt.Printf("0x%x = %s\n", 0x25, base58.Encode(data))
		// }

		// hash := sha256.Sum256([]byte("secret"))
		// key := make([]byte, 0)
		// key = append(key, hash[:16]...)
		// fmt.Printf("%x = SHA256(\"secret\")\n", hash)
		//
		// hash = sha256.Sum256(hash[:])
		// key = append(key, hash[:16]...)
		// fmt.Printf("CHC[1] = %x\n", hash)
		//
		// hash = sha256.Sum256(hash[:])
		// key = append(key, hash[:16]...)
		// fmt.Printf("CHC[2] = %x\n", hash)
		//
		// fmt.Printf("Cipher Key = %x\n", key)
		return nil
	},
}

func init() {
}
