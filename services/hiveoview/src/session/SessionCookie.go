package session

import (
	"crypto/ed25519"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"aidanwoods.dev/go-paseto"
)

// SessionCookieID defines the ID of the cookie containing the user sessionID
const SessionCookieID = "session"

// GetSessionCookie retrieves the credentials from the browser cookie.
// If no valid cookie is found then the bearer token is checked, otherwise this returns an error.
// Intended for use by middleware. Any updates to the session will not be available in the cookie
// until the next request. In almost all cases use session from context as set by middleware.
// This returns the session cookie's auth token and clientID, or an error if not found.
func GetSessionCookie(r *http.Request, pubKey ed25519.PublicKey) (clientID string, authToken string, err error) {
	cookie, err := r.Cookie(SessionCookieID)

	if err != nil || cookie.Valid() != nil {
		//hasBearer := false
		// missing or invalid cookie
		// thou shall not pass. Even with a bearer token!
		//reqToken := r.Header.Get("Authorization")
		//hasBearer = strings.HasPrefix(strings.ToLower(reqToken), "bearer ")
		//if hasBearer {
		// //	the hub auth token
		//authToken = reqToken[len("bearer "):]
		//} else {
		// no cookie, no bearer. We are done here.
		slog.Debug("missing or invalid session cookie", "remoteAddr", r.RemoteAddr)
		return "", "", errors.New("no valid session cookie")
		//}
	} else {
		var pToken *paseto.Token
		pasetoParser := paseto.NewParserForValidNow()
		v4PubKey, _ := paseto.NewV4AsymmetricPublicKeyFromEd25519(pubKey)
		// validate the cookie value token, which is a paseto token signed
		// by the hiveoview key and make sure the bearer token is present
		pToken, err = pasetoParser.ParseV4Public(v4PubKey, cookie.Value, nil)
		if err == nil {
			clientID, err = pToken.GetSubject()
			if err == nil {
				authToken, err = pToken.GetString("authToken")
			}
		}
	}

	return clientID, authToken, err
}

// SetSessionCookie when the client has logged in successfully.
// This stores the user login and session token in a secured 'same-site' cookie.
// The session Token is a Paseto token containing the authentication token on the hub.
//
// The token is signed by the server and cannot be decoded by the client. This
// hides the clientID and server authentication in case the cookie is stolen.
// Note that the token can be used and updated from multiple client connections on the same device,
// e.g: each new browser tab will open a new session and update this cookie, extending its expiry.
//
// max-age in seconds, after which the cookie is deleted, use 0 to delete the cookie on browser exit.
//
// The session cookie is restricted to SameSite policy to reduce the risk of CSRF.
// The cookie has the HttpOnly flag set to disable JS access.
// The cookie path has the session prefix
//
// This generates a JWT token, signed by the service key, with claims.
func SetSessionCookie(w http.ResponseWriter,
	clientID string, authToken string, maxAge time.Duration, privKey ed25519.PrivateKey) error {

	slog.Debug("SetSessionCookie", "clientID", clientID)

	pToken := paseto.NewToken()
	pToken.SetIssuer("hiveoview")
	pToken.SetSubject(clientID)
	pToken.SetExpiration(time.Now().Add(maxAge))
	pToken.SetIssuedAt(time.Now())
	pToken.SetNotBefore(time.Now())
	// custom claims
	pToken.SetString("authToken", authToken)
	secretKey, err := paseto.NewV4AsymmetricSecretKeyFromEd25519(privKey)
	signedToken := pToken.V4Sign(secretKey, nil)

	if err != nil {
		return err
	}
	// TODO: encrypt cookie value

	c := &http.Cookie{
		Name:     SessionCookieID,
		Value:    signedToken,
		MaxAge:   int(maxAge.Seconds()),
		HttpOnly: true, // Cookie is not accessible via client-side java (XSS attack)
		//Secure:   true, // TODO: only use with https
		// With SSR, samesite strict should offer good CSRF protection
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
	}
	http.SetCookie(w, c)
	return err
}

// RemoveSessionCookie removes the cookie. Intended for logout.
func RemoveSessionCookie(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie(SessionCookieID)
	if err != nil {
		slog.Debug("No session cookie found", "url", r.URL.String())
		return
	}
	c.Value = ""
	c.MaxAge = -1
	c.Expires = time.Now().Add(-time.Hour)

	http.SetCookie(w, c)
}
