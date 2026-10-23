package authcli

import (
	"context"
	"fmt"
	"strings"
	"time"

	authnpkg "github.com/hiveot/hivekit/go/cells/authn/pkg"
	"github.com/hiveot/hivekit/go/cells/consumer"
	"github.com/urfave/cli/v3"
	"golang.org/x/exp/rand"
)

// AuthSetPasswordCommand sets a client's password
func AuthSetPasswordCommand(hc **consumer.Consumer) *cli.Command {
	return &cli.Command{
		Name:      "setpass",
		Usage:     "Set password. (careful, no confirmation)",
		ArgsUsage: "<login> <password>",
		Category:  "auth",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			newPassword := ""
			if cmd.NArg() != 2 {
				err := fmt.Errorf("expected 2 arguments")
				return err
			}
			loginID := cmd.Args().Get(0)
			newPassword = cmd.Args().Get(1)
			err := HandleSetPassword(*hc, loginID, newPassword)
			return err
		},
	}
}

// HandleSetPassword resets or replaces a password
//
//	loginID is the ID or email of the user
//	newPassword can be empty to auto-generate a password
func HandleSetPassword(co *consumer.Consumer, loginID string, newPassword string) error {
	if newPassword == "" {
		newPassword = GeneratePassword(9, true)
	}
	authnClient := authnpkg.NewAuthnAdminClient(co)
	err := authnClient.SetClientPassword(loginID, newPassword)

	if err != nil {
		fmt.Println("Error: " + err.Error())
	} else {
		fmt.Println("User " + loginID + " password has been updated")
	}
	return err
}

// GeneratePassword with upper, lower, numbers and special characters
func GeneratePassword(length int, useSpecial bool) (password string) {
	const charsLow = "abcdefghijklmnopqrstuvwxyz"
	const charsUpper = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	const charsSpecial = "!#$%&*+-./:=?@^_"
	const numbers = "0123456789"
	var pool = []rune(charsLow + numbers + charsUpper)

	if length < 2 {
		length = 8
	}
	if useSpecial {
		pool = append(pool, []rune(charsSpecial)...)
	}
	rand.Seed(uint64(time.Now().Unix()))
	//pwchars := make([]string, length)
	pwchars := strings.Builder{}

	for i := 0; i < length; i++ {
		pos := rand.Intn(len(pool))
		pwchars.WriteRune(pool[pos])
	}
	password = pwchars.String()
	return password
}
