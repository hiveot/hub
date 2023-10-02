package utils

import (
	"os"
	"path"
)

// MakeClientID implements the convention for naming plugin and device clients.
// This takes the current application binary followed by the hostname as in:
//
//	{binary}-{hostname}
//
// This convention is used by the launcher and provisioning service in generating
// authentication tokens.
func MakeClientID() string {
	binName := path.Base(os.Args[0])
	hostName, _ := os.Hostname()
	clientID := binName + "-" + hostName
	return clientID
}
