package service

import (
	"crypto/ecdsa"
	"fmt"
	"github.com/hiveot/hub/core/authn"
	"github.com/hiveot/hub/core/authn/config"
	"github.com/hiveot/hub/core/authn/service/jwtauthn"
	"github.com/hiveot/hub/core/authn/service/unpwstore"
	"golang.org/x/exp/rand"
	"golang.org/x/exp/slog"
	"strings"
	"time"

	"github.com/hiveot/hub/lib/certsclient"
)

// AuthnService provides the capabilities to manage and use authentication services
// This implements the IAuthnService interface
type AuthnService struct {
	config config.AuthnConfig
	// key used for signing of JWT tokens
	signingKey *ecdsa.PrivateKey
	// password storage
	pwStore unpwstore.IUnpwStore
	//
	jwtAuthn *jwtauthn.JWTAuthn
}

// AddUser adds a new user and returns a generated password
func (svc *AuthnService) AddUser(clientID string, newPassword string) (password string, err error) {

	exists := svc.pwStore.Exists(clientID)
	if exists {
		return "", fmt.Errorf("user with clientID '%s' already exists", clientID)
	}
	if newPassword == "" {
		newPassword = svc.GeneratePassword(0, false)
	}
	err = svc.pwStore.SetPassword(clientID, newPassword)
	return newPassword, err
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

// ListUsers provide a list of users and their info
func (svc *AuthnService) ListUsers() (profiles []authn.UserProfile, err error) {
	pwEntries, err := svc.pwStore.List()
	profiles = make([]authn.UserProfile, len(pwEntries))
	for i, entry := range pwEntries {
		profile := authn.UserProfile{
			LoginID: entry.LoginID,
			Name:    entry.UserName,
			Updated: entry.Updated,
		}
		profiles[i] = profile
	}
	return profiles, err
}

// GetProfile returns the user's profile
// User must be authenticated first
func (svc *AuthnService) GetProfile(clientID string) (profile authn.UserProfile, err error) {
	//upa.profileStore[profile.LoginID] = profile
	entry, err := svc.pwStore.GetEntry(clientID)
	if err == nil {
		profile.LoginID = entry.LoginID
		profile.Name = entry.UserName
		profile.Updated = entry.Updated
	}
	return profile, err

}

// Login to authenticate a user
// This returns a short lived auth token for use with the HTTP api,
// and a medium lived refresh token used to obtain a new auth token.
func (svc *AuthnService) Login(clientID string, password string) (
	authToken, refreshToken string, err error) {
	err = svc.pwStore.VerifyPassword(clientID, password)
	if err != nil {
		return "", "", fmt.Errorf("invalid login as '%s'", clientID)
	}
	// when valid, provide the tokens
	at, rt, err := svc.jwtAuthn.CreateTokens(clientID)
	return at, rt, err
}

// Logout invalidates the refresh token
func (svc *AuthnService) Logout(clientID string, refreshToken string) (err error) {

	svc.jwtAuthn.InvalidateToken(clientID, refreshToken)
	return nil
}

// Refresh an authentication token
// refreshToken must be a valid refresh token obtained at login
// This returns a short lived auth token and medium lived refresh token
func (svc *AuthnService) Refresh(clientID string, refreshToken string) (
	newAuthToken, newRefreshToken string, err error) {

	at, rt, err := svc.jwtAuthn.RefreshTokens(clientID, refreshToken)
	return at, rt, err
}

// SetPassword changes the client password
func (svc *AuthnService) SetPassword(clientID, newPassword string) error {
	return svc.pwStore.SetPassword(clientID, newPassword)
}

// SetProfile replaces the user profile
func (svc *AuthnService) SetProfile(profile authn.UserProfile) error {
	return svc.pwStore.SetName(profile.LoginID, profile.Name)
}
func (svc *AuthnService) Start() error {
	slog.Info("starting authn service ", "passwordfile", svc.config.PasswordFile)
	return svc.pwStore.Open()
}
func (svc *AuthnService) Stop() error {
	slog.Info("stopping service")
	svc.pwStore.Close()
	return nil
}

// RemoveUser removes a user and disables login
// Existing tokens are immediately expired (tbd)
func (svc *AuthnService) RemoveUser(loginID string) (err error) {
	err = svc.pwStore.Remove(loginID)
	return err
}

// ResetPassword reset a user's password and returns a new temporary password
func (svc *AuthnService) ResetPassword(clientID string, newPassword string) (password string, err error) {
	if newPassword == "" {
		newPassword = svc.GeneratePassword(8, false)
	}
	err = svc.pwStore.SetPassword(clientID, newPassword)
	return newPassword, err
}

// UpdateUser updates a user's name
func (svc *AuthnService) UpdateUser(clientID string, name string) (err error) {
	exists := svc.pwStore.Exists(clientID)
	if !exists {
		return fmt.Errorf("user with loginID '%s' does not exist", clientID)
	}
	err = svc.pwStore.SetName(clientID, name)
	return err
}

// NewAuthnService creates new instance of the service.
// Call Connect before using the service.
func NewAuthnService(cfg config.AuthnConfig) *AuthnService {
	signingKey := certsclient.CreateECDSAKeys()
	pwStore := unpwstore.NewPasswordFileStore(cfg.PasswordFile)
	jwtAuthn := jwtauthn.NewJWTAuthn(
		signingKey, uint(cfg.AccessTokenValiditySec), uint(cfg.RefreshTokenValiditySec))

	svc := &AuthnService{
		config:     cfg,
		pwStore:    pwStore,
		signingKey: signingKey,
		jwtAuthn:   jwtAuthn,
	}
	return svc
}
