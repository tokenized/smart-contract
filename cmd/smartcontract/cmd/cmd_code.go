package cmd

import (
	"crypto/sha256"
	"fmt"

	"github.com/spf13/cobra"
)

var cmdCode = &cobra.Command{
	Use:   "code",
	Short: "Run random code",
	RunE: func(c *cobra.Command, args []string) error {

		hash := sha256.Sum256([]byte("secret"))
		key := make([]byte, 0)
		key = append(key, hash[:16]...)
		fmt.Printf("%x = SHA256(\"secret\")\n", hash)

		hash = sha256.Sum256(hash[:])
		key = append(key, hash[:16]...)
		fmt.Printf("CHC[1] = %x\n", hash)

		hash = sha256.Sum256(hash[:])
		key = append(key, hash[:16]...)
		fmt.Printf("CHC[2] = %x\n", hash)

		fmt.Printf("Cipher Key = %x\n", key)
		return nil
	},
}

func init() {
}
