package service

import (
	"errors"
	"fmt"
	"github.com/hiveot/hub/api/go/hub"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/core/authn/service/unpwstore"
	"github.com/nats-io/jwt/v2"
	"github.com/nats-io/nkeys"
	"golang.org/x/exp/rand"
	"golang.org/x/exp/slog"
	"strings"
	"time"
)

// AuthnService provides the capabilities to manage and use authentication services
// This implements the IAuthnService interface
type AuthnService struct {
	// client used to receive requests via the messaging server
	// the account key used for issuing of JWT user tokens
	accountKP nkeys.KeyPair
	// password storage
	pwStore unpwstore.IUnpwStore
}

// AddDevice adds a device
//func (svc *AuthnService) AddDevice(clientID string, name string) (token string, err error) {
//
//	exists := svc.pwStore.Exists(clientID)
//	if exists {
//		return "", fmt.Errorf("device with clientID '%s' already exists", clientID)
//	}
//	err = svc.pwStore.SetName(clientID, name)
//	return err
//}

// AddService adds a service
//func (svc *AuthnService) AddService(clientID string, name string) (token string, err error) {
//
//	exists := svc.pwStore.Exists(clientID)
//	if exists {
//		return "", fmt.Errorf("service with clientID '%s' already exists", clientID)
//	}
//	err = svc.pwStore.SetName(clientID, name)
//	return err
//}

// AddUser adds a new user and returns a generated password
func (svc *AuthnService) AddUser(clientID string, password string) (err error) {

	exists := svc.pwStore.Exists(clientID)
	if exists {
		return fmt.Errorf("user with clientID '%s' already exists", clientID)
	}
	err = svc.pwStore.SetPassword(clientID, password)
	return err
}

// CreateUserToken create a new user token signed by the account
func (svc *AuthnService) CreateUserToken(userID string, userName string, validity uint) (string, error) {

	// when valid, provide the tokens
	userKP, _ := nkeys.CreateUser()
	userPub, _ := userKP.PublicKey()
	userSeed, _ := userKP.Seed()
	userClaims := jwt.NewUserClaims(userPub)
	userClaims.ID = userID
	userClaims.Name = userName
	userClaims.IssuedAt = time.Now().Unix()
	userClaims.Expires = time.Now().Add(time.Duration(validity) * time.Second).Unix()

	// default size
	userClaims.Limits.Data = 1024 * 1024 * 1024 // max data this client can ... do?
	// users can publish actions
	userClaims.Permissions.Pub.Allow.Add("groups.*.*.action.>")
	// users can subscribe to group events
	userClaims.Permissions.Sub.Allow.Add("groups.*.*.event.>")
	// users can receive replies in their inbox
	userClaims.Permissions.Sub.Allow.Add("_INBOX.>")

	userJWT, err := userClaims.Encode(svc.accountKP)
	creds, _ := jwt.FormatUserConfig(userJWT, userSeed)
	return string(creds), err
}

// GetProfile returns the user's profile
func (svc *AuthnService) GetProfile(clientID string) (profile hub.ClientProfile, err error) {
	//upa.profileStore[profile.LoginID] = profile
	entry, err := svc.pwStore.GetEntry(clientID)
	if err == nil {
		updatedStr := time.Unix(entry.Updated, 0).Format(vocab.ISO8601Format)
		profile.ClientID = entry.LoginID
		profile.Name = entry.UserName
		profile.Updated = updatedStr
	}
	return profile, err

}

// GeneratePassword with upper, lower, numbers and special characters
func (svc *AuthnService) GeneratePassword(length int, useSpecial bool) (password string) {
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

// ListClients provide a list of users and their info
func (svc *AuthnService) ListClients() (profiles []hub.ClientProfile, err error) {
	pwEntries, err := svc.pwStore.List()
	profiles = make([]hub.ClientProfile, len(pwEntries))
	for i, entry := range pwEntries {
		updatedStr := time.Unix(entry.Updated, 0).Format(vocab.ISO8601Format)
		profile := hub.ClientProfile{
			ClientID: entry.LoginID,
			Name:     entry.UserName,
			Updated:  updatedStr,
		}
		profiles[i] = profile
	}
	return profiles, err
}

// Login to authenticate a user
// This returns a short-lived auth token that can be used to connect to the message server
// The token can be refreshed to extend it without requiring a login password.
func (svc *AuthnService) Login(clientID string, password string) (authToken string, err error) {
	entry, err := svc.pwStore.VerifyPassword(clientID, password)
	if err != nil {
		return "", err
	}
	// when valid, provide the tokens
	authToken, err = svc.CreateUserToken(clientID, entry.UserName, hub.DefaultUserTokenValiditySec)

	//newToken, err := svc.jwtAuthn.CreateTokens(clientID)
	return authToken, err
}

// Logout invalidates the token
func (svc *AuthnService) Logout(clientID string, token string) (err error) {
	// TODO
	//svc.jwtAuthn.InvalidateToken(clientID, refreshToken)
	return nil
}

// Refresh an authentication token
// This returns a refreshed token that can be used to connect to the messaging server
// the old token must be a valid jwt token
func (svc *AuthnService) Refresh(clientID string, oldToken string) (newToken string, err error) {

	// verify the token
	entry, err := svc.ValidateToken(clientID, oldToken)
	if err != nil {
		return "", err
	}
	newToken, err = svc.CreateUserToken(clientID, entry.UserName, hub.DefaultUserTokenValiditySec)
	return newToken, err
}

// RemoveClient removes a user and disables login
// Existing tokens are immediately expired (tbd)
func (svc *AuthnService) RemoveClient(clientID string) (err error) {
	err = svc.pwStore.Remove(clientID)
	return err
}

// ResetPassword sets the client password
func (svc *AuthnService) ResetPassword(clientID, newPassword string) error {
	return svc.pwStore.SetPassword(clientID, newPassword)
}

// Start the service, open the password store and start listening for requests on the service topic
func (svc *AuthnService) Start() error {

	//authKey, err := svc.config.GetAuthKey()
	//if err != nil {
	//	return err
	//}

	err := svc.pwStore.Open()
	if err != nil {
		return err
	}
	//err = svc.hc.ConnectWithJWT(svc.config.ServerURL, []byte(authKey), svc.caCert)
	return err
}

func (svc *AuthnService) Stop() error {
	slog.Info("stopping service")
	svc.pwStore.Close()
	return nil
}

// UpdateName updates a user's name
func (svc *AuthnService) UpdateName(clientID string, name string) (err error) {
	exists := svc.pwStore.Exists(clientID)
	if !exists {
		return fmt.Errorf("user with loginID '%s' does not exist", clientID)
	}
	err = svc.pwStore.SetName(clientID, name)
	return err
}

// UpdatePassword changes the client password
func (svc *AuthnService) UpdatePassword(clientID, newPassword string) error {
	return svc.pwStore.SetPassword(clientID, newPassword)
}

// ValidateToken checks if the given token belongs the the user ID and is valid
func (svc *AuthnService) ValidateToken(clientID string, token string) (entry unpwstore.PasswordEntry, err error) {
	entry, err = svc.pwStore.GetEntry(clientID)
	_ = entry
	if err != nil {
		return entry, err
	}
	claims, err := jwt.Decode(token)
	if err != nil {
		return entry, errors.New("Invalid token of client " + clientID)
	}
	if claims.ClaimType() != jwt.UserClaim {
		return entry, errors.New("Token is not a user token of client " + clientID)
	}
	cd := claims.Claims()
	if cd.ID != clientID {
		slog.Warn("Refresh attempt on token from different user",
			"token ID", cd.ID, "token name", cd.Name, "clientID", clientID)
		return entry, errors.New("Token is from a different client, not" + clientID)
	}
	vr := jwt.ValidationResults{}
	cd.Validate(&vr)
	if !vr.IsEmpty() {
		err = errors.New("Invalid token: " + vr.Errors()[0].Error())
		return entry, err
	}
	return entry, nil
}

// NewManageAuthnService creates new instance of the service
// Call 'Start' to start the service and 'Stop' to end it.
//
//	cfg contains the service configuration
//	hc is the client connected to the messaging service to receive authn requests
func NewManageAuthnService(
	pwStore unpwstore.IUnpwStore, accountKP nkeys.KeyPair) *AuthnService {

	//pwStore := unpwstore.NewPasswordFileStore(cfg.PasswordFile)
	//jwtAuthn := jwtauthn.NewJWTAuthn(
	//	signingKey, uint(cfg.AccessTokenValiditySec), uint(cfg.RefreshTokenValiditySec))

	svc := &AuthnService{
		hc:        hc,
		pwStore:   pwStore,
		accountKP: accountKP,
		//jwtAuthn:   jwtAuthn,
	}
	return svc
}
