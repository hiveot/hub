package authz

import (
	"log/slog"
)

// AuthzService is the authorization service for authorizing access to devices
type AuthzService struct {
	cfg *AuthzConfig
}

// CreateRole adds a new custom role
func (svc *AuthzService) CreateRole(role string) error {
	// FIXME:implement
	slog.Error("CreateRole is not yet implemented")
	return nil
}

// DeleteRole deletes a custom role
func (svc *AuthzService) DeleteRole(role string) error {
	// FIXME:implement
	slog.Error("DeleteRole is not yet implemented")
	return nil
}

// VerifyPermissions checks if the given client with the role can publish a message
func (svc *AuthzService) VerifyPermissions(clientID string, role string, message string) bool {
	// todo
	return true
}

// Start starts the authorization service
func (svc *AuthzService) Start() error {
	return nil
}

// Stop stops the authorization service
func (svc *AuthzService) Stop() {
}

// NewAuthzService creates a new instance of the authorization service with default rules
func NewAuthzService(cfg *AuthzConfig) *AuthzService {
	svc := &AuthzService{
		cfg: cfg,
	}
	return svc
}
