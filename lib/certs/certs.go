// Package certs with functions to load CA and client certificates for use
// by the protocol binding in the Consumed Thing factory or other clients.
package certs

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"github.com/hiveot/hub/lib/keys"
	"log/slog"
	"os"
)

const DefaultCaCertFile = "caCert.pem"
const DefaultCaKeyFile = "caKey.pem"

// Certificate Organization Unit for client certificate based authorization
const (
	//OUAdmin lets a client approve things provisioning (postOOB), add and remove users
	// Provision API permissions: GetDirectory, ProvisionRequest, GetStatus, PostOOB
	OUAdmin = "admin"

	// OUNone is the default OU with no API access permissions
	OUNone = "unauth"

	// OUUser for consumers with mutual authentication
	OUUser = "user"

	// OUIoTDevice for IoT devices with mutual authentication
	OUIoTDevice = "device"

	// OUService for Hub services with mutual authentication
	// By default, services have access to other services
	// Provision API permissions: Any
	OUService = "service"
)

// LoadX509CertFromPEM loads the x509 certificate from a PEM file format.
//
// Intended to load the CA certificate to validate server and broker.
//
//	pemPath is the full path to the X509 PEM file.
func LoadX509CertFromPEM(pemPath string) (cert *x509.Certificate, err error) {
	pemEncoded, err := os.ReadFile(pemPath)
	if err != nil {
		return nil, err
	}
	return X509CertFromPEM(string(pemEncoded))
}

// LoadTLSCertFromPEM loads the TLS certificate from PEM formatted file.
// TLS certificates are a container for both X509 certificate and private key.
//
// Intended to load the certificate and key for servers, or for clients such as IoT devices
// that use client certificate authentication. The idprov service issues this type of
// certificate during IoT device provisioning.
//
// This is simply a wrapper around tls.LoadX509KeyPair. See also SaveTLSCertToPEM.
//
// If loading fails, this returns nil as certificate pointer
func LoadTLSCertFromPEM(certPEMPath, keyPEMPath string) (cert *tls.Certificate, err error) {
	// FYI, not all browsers support certificates with ed25519 keys, so this file contains a ecdsa key
	tlsCert, err := tls.LoadX509KeyPair(certPEMPath, keyPEMPath)
	if err != nil {
		return nil, err
	}
	return &tlsCert, err
}

// PublicKeyFromCert extracts an ECDSA public key from x509 certificate
// Returns nil if certificate doesn't hold a ECDSA public key
func PublicKeyFromCert(cert *x509.Certificate) *ecdsa.PublicKey {
	var pubKey *ecdsa.PublicKey
	switch pub := cert.PublicKey.(type) {
	case *ecdsa.PublicKey:
		pubKey = pub
	}
	return pubKey
}

// SaveTLSCertToPEM saves the x509 certificate and private key to separate files in PEM format
//
// Intended for saving a certificate received from provisioning or created for testing.
//
//	cert is the obtained TLS certificate whose parts to save
//	certPEMPath the file to save the X509 certificate to in PEM format
//	keyPEMPath the file to save the private key to in PEM format
func SaveTLSCertToPEM(cert *tls.Certificate, certPEMPath, keyPEMPath string) error {
	//slog.Info("Saving TLS cert to " + certPEMPath)
	b := pem.Block{Type: "CERTIFICATE", Bytes: cert.Certificate[0]}
	certPEM := pem.EncodeToMemory(&b)
	// remove existing cert since perm 0444 doesn't allow overwriting it
	_ = os.Remove(certPEMPath)
	_ = os.Remove(keyPEMPath)
	err := os.WriteFile(certPEMPath, certPEM, 0444)
	if err != nil {
		slog.Error("Failed writing server cert to file", "err", err)
		return err
	}
	ecdsaPK, found := cert.PrivateKey.(*ecdsa.PrivateKey)
	if found {
		k := keys.NewEcdsaKeyFromPrivate(ecdsaPK)
		err = k.ExportPrivateToFile(keyPEMPath)
		return err
	}
	ed25519PK, found := cert.PrivateKey.(ed25519.PrivateKey)
	if found {
		//raw, err := x509.MarshalPKCS8PrivateKey(ed25519PK)
		//block := &pem.Block{
		//	Type:  "PRIVATE KEY",
		//	Bytes: raw,
		//}
		//err = os.WriteFile(keyPEMPath, pem.EncodeToMemory(block), 0600)
		k := keys.NewEd25519KeyFromPrivate(ed25519PK)
		err = k.ExportPrivateToFile(keyPEMPath)
		return err
	}

	////cert.PrivateKey.
	//privKey := cert.PrivateKey
	//seed := privKey.Seed()
	//pemEnc = pem.EncodeToMemory(
	//	&pem.Block{Type: "PRIVATE KEY",
	//		Bytes: seed,
	//	})
	//return string(pemEnc)

	//k, err := keys.NewKeyFromEnc(ed25519PK)
	if err == nil {
		//err = k.ExportPrivateToFile(keyPEMPath)
	}
	return err
}

// SaveX509CertToPEM saves the x509 certificate to file in PEM format.
// Clients that receive a client certificate from provisioning can use this
// to save the provided certificate to file.
func SaveX509CertToPEM(cert *x509.Certificate, pemPath string) error {
	b := pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw}
	certPEM := pem.EncodeToMemory(&b)
	// remove existing cert since perm 0444 doesn't allow overwriting it
	_ = os.Remove(pemPath)
	err := os.WriteFile(pemPath, certPEM, 0444)
	return err
}

// X509CertFromPEM converts a X509 certificate in PEM format to an X509 instance
func X509CertFromPEM(certPEM string) (*x509.Certificate, error) {
	caCertBlock, _ := pem.Decode([]byte(certPEM))
	if caCertBlock == nil {
		return nil, errors.New("pem.Decode failed")
	}
	caCert, err := x509.ParseCertificate(caCertBlock.Bytes)
	return caCert, err
}

// X509CertToPEM converts the x509 certificate to PEM format
func X509CertToPEM(cert *x509.Certificate) string {
	b := pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw}
	certPEM := pem.EncodeToMemory(&b)
	return string(certPEM)
}

// X509CertToTLS combines a x509 certificate and private key into a TLS certificate
func X509CertToTLS(cert *x509.Certificate, privKey keys.IHiveKey) *tls.Certificate {
	// A TLS certificate is a wrapper around x509 with private key
	tlsCert := &tls.Certificate{}
	tlsCert.Certificate = append(tlsCert.Certificate, cert.Raw)
	tlsCert.PrivateKey = privKey.PrivateKey()

	return tlsCert
}
