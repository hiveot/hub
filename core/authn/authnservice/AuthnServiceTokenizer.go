package authnservice

import (
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"github.com/hiveot/hub/api/go/authn"
	"github.com/hiveot/hub/lib/certs"
	"golang.org/x/exp/slog"
	"time"
)

// AuthnServiceTokenizer is the built-in tokenizer to generate and validate JWT authentication tokens.
// This implements the IAuthnTokenizer interface
type AuthnServiceTokenizer struct {
	signingKey    ecdsa.PrivateKey
	signingKeyPub string
}

// CreateToken creates a signed authentication JWT token
func (svc *AuthnServiceTokenizer) CreateToken(
	clientID string, clientType string, pubKey string, validitySec int) (newToken string, err error) {

	// see also: https://golang-jwt.github.io/jwt/usage/create/
	expiryTime := time.Now().Add(time.Duration(validitySec) * time.Second)
	if clientID == "" {
		err = fmt.Errorf("CreateJWTTokens: Missing clientID")
		return
	}

	// Create the JWT claims, which includes the username and expiry time
	claims := jwt.MapClaims{
		"alg":  "ES256",
		"type": "JWT",
		"aud":  clientType,        //
		"sub":  pubKey,            // public key of client
		"iss":  svc.signingKeyPub, // public key of issuer
		"exp":  expiryTime.Unix(), // expiry time. Seconds since epoch
		"iat":  time.Now().Unix(), // issued at. Seconds since epoch

		// custom claim fields
		"clientID": clientID,
	}

	// Declare the token with the algorithm used for signing, and the claims
	claimsToken := jwt.NewWithClaims(jwt.SigningMethodES256, claims)

	authnToken, err := claimsToken.SignedString(svc.signingKey)
	if err != nil {
		return "", err
	}
	return authnToken, nil
}

// ValidateToken is the built-in token validator, used when no external tokenizer is given.
// This validates:
//   - if the token is a JWT token
//   - if the token's clientID matches the given clientID
//   - if the claims issuer is the public signing key
//   - if the token is signed using es256, es384 or es512
//     is signed by our signing key
//   - if the token contains the client public key (subject)
//   - if the token audience has a client type of: user, device or service
//   - if the token isn't expired
//   - if signedNonce is provided, verify against the client public key from token
//
// if signedNonce is provided then nonce is required.
func (svc *AuthnServiceTokenizer) ValidateToken(
	clientID string, tokenString string, signedNonce string, nonce string) (
	err error) {

	// see also: https://golang-jwt.github.io/jwt/usage/parse/
	//claims := jwt.RegisteredClaims{}
	claims := jwt.MapClaims{}

	// TODO: TBD if "kid" (key identifier) can be used, in case of multiple signing keys

	parser := jwt.NewParser(
		jwt.WithIssuedAt(),
		jwt.WithIssuer(svc.signingKeyPub),
		jwt.WithValidMethods([]string{"es256", "es384", "es512"}),
	)
	// parse with claims checks the token is:
	// - has a valid signature (eg token isn't tampered with)
	// - issued by our signingKey using es256 (or es384,es512)
	// - is not expired
	jwtToken, err := parser.ParseWithClaims(tokenString, &claims,
		func(token *jwt.Token) (interface{}, error) {
			// verify signature using public key
			return svc.signingKeyPub, nil
		})

	if err != nil || jwtToken == nil || !jwtToken.Valid {
		return fmt.Errorf("invalid JWT token of client %s: %w", clientID, err)
	}

	claimedClientID := claims["clientID"].(string)
	if claimedClientID != clientID {
		slog.Warn("Token from different client",
			slog.String("token ID", claimedClientID),
			slog.String("clientID", clientID))
		err = fmt.Errorf("token from different user")
		return err
	}

	clientType, _ := claims.GetAudience()
	validClientType := len(clientType) > 0 &&
		(clientType[0] == authn.ClientTypeDevice ||
			clientType[0] == authn.ClientTypeService ||
			clientType[0] == authn.ClientTypeUser)

	clientPubKey, _ := claims.GetSubject()
	if len(clientType) == 0 || !validClientType || clientPubKey == "" {
		return fmt.Errorf("missing client type (aud) or public key (sub) for client %s", clientID)
	}

	// verify the nonce signature
	// TODO: not sure if this is the right way. Where is the client public key
	// supposed to come from? (we use subject here)
	if signedNonce != "" {
		sig, err := base64.RawURLEncoding.DecodeString(signedNonce)
		if err != nil {
			// Allow fallback to normal base64.
			sig, err = base64.StdEncoding.DecodeString(signedNonce)
			if err != nil {
				return fmt.Errorf("signature not valid base64: %w", err)
			}
		}
		// Verify that the signature is signed by the public key in the token
		pubKey, err := certs.PublicKeyFromPEM(clientPubKey)
		ecdsa.VerifyASN1(pubKey, []byte(nonce), sig)
		//ed25519.Verify(pubKey, []byte(nonce), sig)
	}

	return nil
}

// NewAuthnServiceTokenizer provides the default built-in JWT tokenizer for authentication.
//
//	signingKP used for signing and verifying JWT tokens
func NewAuthnServiceTokenizer(signingKey ecdsa.PrivateKey) *AuthnServiceTokenizer {
	signingKeyPub, _ := x509.MarshalPKIXPublicKey(signingKey.PublicKey)
	signingKeyStr := base64.StdEncoding.EncodeToString(signingKeyPub)

	tokenizer := &AuthnServiceTokenizer{
		signingKey:    signingKey,
		signingKeyPub: signingKeyStr,
	}
	return tokenizer
}
