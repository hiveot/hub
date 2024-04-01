package authz

// AuthzStore is the store for user roles and authorization ACL
// This loads the roles into memory on startup and writes on change.
type AuthzStore struct {
}

// Load the store from the given path
func (store *AuthzStore) Load(path string) {

}

// Create a new instance of the authorization store
func NewAuthzStore() *AuthzStore {
	store := &AuthzStore{}
	return store
}
