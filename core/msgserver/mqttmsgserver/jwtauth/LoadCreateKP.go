package jwtauth

import (
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/base64"
	"github.com/hiveot/hub/lib/certs"
	"log/slog"
)

// LoadCreateUserKP loads a user keypair, or creates one if it doesn't exist
//
//	kpPath is file of key or "" to just create it
//	writeChanges if a file is given and key is generated
//
// This returns the public/private key pair with a public key string, or an error.
func LoadCreateUserKP(kpPath string, writeChanges bool) (kp *ecdsa.PrivateKey, kpPub string, err error) {
	// attempt to load
	if kpPath != "" {
		kp, _ = certs.LoadKeysFromPEM(kpPath)
	}
	// load fail, create and save
	if kp == nil {
		slog.Info("LoadCreateUserKP Keys not found. Creating new keys",
			slog.String("kpPath", kpPath),
			slog.Bool("writeChanges", writeChanges))
		kp, kpPub = certs.CreateECDSAKeys()
		if writeChanges {
			err = certs.SaveKeysToPEM(kp, kpPath)
		}
	} else {
		x509Pub, err := x509.MarshalPKIXPublicKey(&kp.PublicKey)
		if err != nil {
			return nil, "", err
		}
		kpPub = base64.StdEncoding.EncodeToString(x509Pub)
	}
	return kp, kpPub, err
}
