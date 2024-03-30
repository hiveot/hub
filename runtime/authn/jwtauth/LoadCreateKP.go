package jwtauth

import (
	"crypto/ecdsa"
	"github.com/hiveot/hub/lib/keys"
	"log/slog"
)

// LoadCreateUserKP loads a user keypair, or creates one if it doesn't exist
//
//	pemPath is file of key or "" to just create it
//	writeChanges if a file is given and key is generated
//
// This returns the public/private key pair with a public key string, or an error.
func LoadCreateUserKP(pemPath string, writeChanges bool) (privKey *ecdsa.PrivateKey, pubPEM string, err error) {
	k := keys.NewKey(keys.KeyTypeECDSA)

	// attempt to load
	if pemPath != "" {
		err = k.ImportPrivateFromFile(pemPath)

		if err != nil {
			// if load fails then keep the created keys
			slog.Info("LoadCreateUserKP Keys not found. Creating new keys",
				slog.String("pemPath", pemPath),
				slog.Bool("writeChanges", writeChanges))

			// load failed, new keys are used
			if writeChanges {
				err = k.ExportPrivateToFile(pemPath)
			}
		}
	}

	privKey = k.PrivateKey().(*ecdsa.PrivateKey)
	pubPEM = k.ExportPublic()
	return privKey, pubPEM, err
}
