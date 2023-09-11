package mqttmsgserver

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"github.com/hiveot/hub/api/go/auth"
	"github.com/hiveot/hub/api/go/msgserver"
	"github.com/hiveot/hub/lib/certs"
	"time"
)

func (srv *MqttMsgServer) ApplyAuth(clients []msgserver.ClientAuthInfo) error {
	return fmt.Errorf("not yet implemented")
}

// CreateKP creates a keypair for use in connecting or signing.
// This returns the key pair and its public key string.
func (srv *MqttMsgServer) CreateKP() (interface{}, string) {
	kp := certs.CreateECDSAKeys()

	x509EncodedPub, _ := x509.MarshalPKIXPublicKey(&kp.PublicKey)

	pemEncodedPub := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: x509EncodedPub})

	return kp, string(pemEncodedPub)
}

func (srv *MqttMsgServer) CreateToken(clientID string) (token string, err error) {

	if clientID == "" {
		err = fmt.Errorf("CreateToken: Missing clientID")
		return "", err
	}
	clientAuth, found := srv.authClients[clientID]
	if !found {
		err = fmt.Errorf("CreateToken: Unknown client")
		return "", err
	}
	if clientAuth.PubKey == "" {
		err = fmt.Errorf("CreateToken: client has no public key")
		return "", err
	}

	// see also: https://golang-jwt.github.io/jwt/usage/create/
	// TODO: use validity period from profile
	validity := auth.DefaultUserTokenValidityDays
	if clientAuth.ClientType == auth.ClientTypeDevice {
		validity = auth.DefaultDeviceTokenValidityDays
	} else if clientAuth.ClientType == auth.ClientTypeService {
		validity = auth.DefaultServiceTokenValidityDays
	}
	expiryTime := time.Now().Add(time.Duration(validity) * time.Hour * 24)

	// Create the JWT claims, which includes the username, clientType and expiry time
	claims := jwt.MapClaims{
		"alg":  "ES256",
		"type": "JWT",
		"aud":  clientAuth.ClientType,          //
		"sub":  clientAuth.PubKey,              // public key of client (same as nats)
		"iss":  srv.Config.ServerKey.PublicKey, // public key of issuer
		"exp":  expiryTime.Unix(),              // expiry time. Seconds since epoch
		"iat":  time.Now().Unix(),              // issued at. Seconds since epoch

		// custom claim fields
		"clientID": clientID,
	}

	// Declare the token with the algorithm used for signing, and the claims
	claimsToken := jwt.NewWithClaims(jwt.SigningMethodES256, claims)

	authToken, err := claimsToken.SignedString(srv.Config.ServerKey)
	if err != nil {
		return "", err
	}
	return authToken, nil
}

// getClientAuth returns the client auth info for the given ID
func (srv *MqttMsgServer) getClientAuth(clientID string) (msgserver.ClientAuthInfo, error) {
	clientAuth, found := srv.authClients[clientID]
	if !found {
		return clientAuth, fmt.Errorf("client %s not known", clientID)
	}
	return clientAuth, nil
}

//func (srv *MqttMsgServer) CreateJWTToken(clientID string, pubKey string) (newToken string, err error) {
//	return "", fmt.Errorf("not yet implemented")
//}

func (srv *MqttMsgServer) SetRolePermissions(
	rolePerms map[string][]msgserver.RolePermission) {
	srv.rolePermissions = rolePerms
}

func (srv *MqttMsgServer) SetServicePermissions(
	serviceID string, capability string, roles []string) {
}

func (srv *MqttMsgServer) ValidateJWTToken(
	clientID string, pubKey string, tokenString string, signedNonce string, nonce string) error {
	return fmt.Errorf("not yet implemented")
}

func (srv *MqttMsgServer) ValidatePassword(loginID string, password string) error {
	return fmt.Errorf("not yet implemented")
}

func (srv *MqttMsgServer) ValidateToken(
	clientID string, pubKey string, oldToken string, signedNonce string, nonce string) (err error) {
	return fmt.Errorf("not yet implemented")
}
