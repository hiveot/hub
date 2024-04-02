// Package rpc with message encoder/decoder for authorization management
// Intended for use by protocol bindings to encode/decode the request parameters and responses.
package rpc

// AuthRolesCapability defines the 'capability' address part used in sending messages
const AuthRolesCapability = "roles"

// CreateRoleReq defines the request to create a new custom role
const CreateRoleReq = "createRole"

type CreateRoleArgs struct {
	Role string `json:"role"`
}

// DeleteRoleReq defines the request to delete a custom role.
const DeleteRoleReq = "deleteRole"

type DeleteRoleArgs struct {
	Role string `json:"role"`
}
