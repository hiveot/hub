package authcli

import (
	"fmt"
	"strings"
	"time"

	"github.com/hiveot/hivekit/go/consumer"
	authn "github.com/hiveot/hub/runtime/authn/api"
	"github.com/urfave/cli/v2"
	"golang.org/x/exp/rand"
)

// AuthSetPasswordCommand sets a client's password
func AuthSetPasswordCommand(hc **consumer.Consumer) *cli.Command {
	return &cli.Command{
		Name:      "setpass",
		Usage:     "Set password. (careful, no confirmation)",
		ArgsUsage: "<login> <password>",
		Category:  "auth",
		Action: func(cCtx *cli.Context) error {
			newPassword := ""
			if cCtx.NArg() != 2 {
				err := fmt.Errorf("expected 2 arguments")
				return err
			}
			loginID := cCtx.Args().Get(0)
			newPassword = cCtx.Args().Get(1)
			err := HandleSetPassword(*hc, loginID, newPassword)
			return err
		},
	}
}

// HandleSetPassword resets or replaces a password
//
//	loginID is the ID or email of the user
//	newPassword can be empty to auto-generate a password
func HandleSetPassword(hc *consumer.Consumer, loginID string, newPassword string) error {
	if newPassword == "" {
		newPassword = GeneratePassword(9, true)
	}
	err := authn.AdminSetClientPassword(hc, loginID, newPassword)

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
