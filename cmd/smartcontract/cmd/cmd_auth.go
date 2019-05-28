package cmd

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/tokenized/specification/dist/golang/protocol"
)

var cmdAuth = &cobra.Command{
	Use:   "auth <fieldcount> <votecount> <permit> <issuerprop> <holderprop> <voteX> ...",
	Short: "Build auth flags from parameters.",
	Long:  "Build auth flags from parameters.\n\t<fieldcount> Number of fields to specify permissions for.\n\t<votecount> Number of voting systems.\n\t<permit> Issuer allowed to directly change.\n\t<issuerprop> Issuer can create proposal.\n\t<holderprop> Holders can create proposal.\n\t<voteX> Voting system X is allowed for proposal to change field.",
	RunE: func(c *cobra.Command, args []string) error {
		if len(args) < 2 {
			return errors.New("Not enough arguments")
		}

		fieldcount, err := strconv.Atoi(args[0])
		if err != nil {
			return err
		}
		fmt.Printf("%d fields\n", fieldcount)

		votecount, err := strconv.Atoi(args[1])
		if err != nil {
			return err
		}
		fmt.Printf("%d votes\n", votecount)

		result := ""
		permit := len(args) > 2 && strings.ToLower(args[2]) == "true"
		if permit {
			result += " permit"
		} else {
			result += " false"
		}
		issuer := len(args) > 3 && strings.ToLower(args[3]) == "true"
		if issuer {
			result += " issuer"
		} else {
			result += " false"
		}
		holder := len(args) > 4 && strings.ToLower(args[4]) == "true"
		if holder {
			result += " holder"
		} else {
			result += " false"
		}

		votes := make([]bool, votecount)
		for i := 5; i < votecount+5; i++ {
			votes[i-5] = len(args) > i && strings.ToLower(args[i]) == "true"
			if votes[i-5] {
				result += " true"
			} else {
				result += " false"
			}
		}

		fmt.Println(result)

		permissions := make([]protocol.Permission, fieldcount)
		for i, _ := range permissions {
			permissions[i].Permitted = permit              // Issuer can update field without proposal
			permissions[i].AdministrationProposal = issuer // Issuer can update field with a proposal
			permissions[i].HolderProposal = holder         // Holder's can initiate proposals to update field

			permissions[i].VotingSystemsAllowed = votes
		}

		authFlags, err := protocol.WriteAuthFlags(permissions)
		if err != nil {
			return err
		}
		fmt.Printf("%d auth flag bytes\n", len(authFlags))

		fmt.Printf("Auth flags : %s\n", base64.StdEncoding.EncodeToString(authFlags))
		return nil
	},
}

func init() {
}
