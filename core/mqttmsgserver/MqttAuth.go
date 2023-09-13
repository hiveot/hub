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

// ApplyAuth apply update user authentication and authorization settings
func (srv *MqttMsgServer) ApplyAuth(clients []msgserver.ClientAuthInfo) error {
	authClients := map[string]msgserver.ClientAuthInfo{}
	for _, clientInfo := range clients {
		authClients[clientInfo.ClientID] = clientInfo
	}
	srv.authMux.Lock()
	srv.authClients = authClients
	srv.authMux.Unlock()
	return nil
}

// CreateKP creates a keypair for use in connecting or signing.
// This returns the key pair and its public key string.
func (srv *MqttMsgServer) CreateKP() (interface{}, string) {
	kp := certs.CreateECDSAKeys()

	x509EncodedPub, _ := x509.MarshalPKIXPublicKey(&kp.PublicKey)

	pemEncodedPub := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: x509EncodedPub})

	return kp, string(pemEncodedPub)
}

// CreateToken creates a new JWT authtoken for a client.
// The client must have a public key on file.
func (srv *MqttMsgServer) CreateToken(clientID string) (token string, err error) {

	if clientID == "" {
		err = fmt.Errorf("CreateToken: Missing clientID")
		return "", err
	}
	clientAuth, err := srv.GetClientAuth(clientID)
	if err != nil {
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

// GetClientAuth returns the client auth info for the given ID
// This returns an error if the client is not found
func (srv *MqttMsgServer) GetClientAuth(clientID string) (msgserver.ClientAuthInfo, error) {
	srv.authMux.RLock()
	clientAuth, found := srv.authClients[clientID]
	srv.authMux.RUnlock()
	if !found {
		return clientAuth, fmt.Errorf("client %s not known", clientID)
	}
	return clientAuth, nil
}

func (srv *MqttMsgServer) SetRolePermissions(
	rolePerms map[string][]msgserver.RolePermission) {
	srv.authMux.Lock()
	srv.rolePermissions = rolePerms
	srv.authMux.Unlock()
}

func (srv *MqttMsgServer) SetServicePermissions(
	serviceID string, capability string, roles []string) {
}

//func (srv *MqttMsgServer) ValidateJWTToken(
//	clientID string, pubKey string, tokenString string, signedNonce string, nonce string) error {
//	return fmt.Errorf("not yet implemented")
//}

func (srv *MqttMsgServer) ValidatePassword(loginID string, password string) error {
	return fmt.Errorf("not yet implemented")
}

func (srv *MqttMsgServer) ValidateToken(
	clientID string, pubKey string, token string, signedNonce string, nonce string) (err error) {

	claims := jwt.MapClaims{}
	//claims := jwt.RegisteredClaims{}
	jwtToken, err := jwt.ParseWithClaims(token, &claims,
		func(token *jwt.Token) (interface{}, error) {
			return &srv.Config.ServerKey.PublicKey, nil
		})
	if err != nil || jwtToken == nil || !jwtToken.Valid {
		return fmt.Errorf("invalid JWT token: %s", err)
	}
	// StandardClaims.Valid ignores missing 0 dates so check ourselves
	//err = jwtToken.Claims.Valid()
	//now := time.Now().Unix()

	sub, _ := claims.GetSubject() //public key
	aud, _ := claims.GetAudience()
	if sub != pubKey {
		return fmt.Errorf("token public key doesn't match the key for client '%s'", clientID)
	}
	cid := claims["clientID"]
	if cid != clientID {
		return fmt.Errorf("token client '%s' doesn't match clientID '%s'", cid, clientID)
	}
	_ = aud
	_ = sub
	//_ = claims.Issuer // signer
	//_ = srv.Config.ServerKey

	//if err == nil && !jwtToken.Claims.VerifyExpiresAt(now, true) {
	//	delta := time.Unix(now, 0).Sub(time.Unix(claims.ExpiresAt, 0))
	//	err = fmt.Errorf("token of user '%s' is expired by %v", delta, userID)
	//}
	//if err == nil && !claims.VerifyIssuedAt(now, true) {
	//	err = fmt.Errorf("token of user '%s' used before issued", userID)
	//}
	//if err == nil && !claims.VerifyNotBefore(now, true) {
	//	err = fmt.Errorf("token of user '%s' is not valid yet", userID)
	//}

	if err != nil {
		return err
	}

	// check if token is for the given user
	//if claims.Subject != pubKey {
	//	return fmt.Errorf("token is issued to '%s', not to '%s'", claims.Subject, clientID)
	//}
	return nil
}
