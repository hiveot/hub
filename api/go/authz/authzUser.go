// Package authz with types and interfaces for using this service with agent 'authz'
// DO NOT EDIT. This file is auto generated by tdd2api. Any changes will be overwritten.
// Generated 09 Jun 24 10:18 PDT.
package authz

import "encoding/json"
import "errors"
import "github.com/hiveot/hub/lib/things"
import "github.com/hiveot/hub/lib/hubclient"

// UserAgentID is the connection ID of the agent managing the Thing.
const UserAgentID = "authz"

// UserServiceID is the internal thingID of the device/service as used by agents.
// Agents use this to publish events and subscribe to actions
const UserServiceID = "user"

// UserDThingID is the Digitwin thingID as used by agents. Digitwin adds the dtw:{agent} prefix to the serviceID
// Consumers use this to publish actions and subscribe to events
const UserDThingID = "dtw:authz:user"

//--- Schema definitions of Thing 'dtw:authz:user' ---

// ThingPermissions defines a Thing Permissions data schema of the authz agent.
//
// This defines the roles that have permissions to access this thing
// Used by agents and services to set the roles that can invoke actions on a service.
// These permissions are default recommendations made by the service provider.
// The authz service can override these defaults with another configuration.
type ThingPermissions struct {

	// AgentID with Agent ID
	//
	// The agent granting the permissions.
	AgentID string `json:"agentID,omitempty"`

	// Allow with Roles
	//
	// Roles allowed access to a Thing
	Allow []string `json:"allow,omitempty"`

	// Deny with
	Deny []string `json:"deny,omitempty"`

	// ThingID with Thing ID
	//
	// ThingID of the service whose permissions are set.
	// This is the ThingID as send by the agent, without the digitwin prefix.
	ThingID string `json:"thingID,omitempty"`
}

//--- Argument and Response struct for action of Thing 'dtw:authz:user' ---

const UserSetPermissionsMethod = "setPermissions"

// UserClient client for talking to the 'dtw:authz:user' service
type UserClient struct {
	dThingID string
	hc       hubclient.IHubClient
}

// SetPermissions client method - Set Permissions.
// Set the roles that can use a Thing or service
func (svc *UserClient) SetPermissions(permissions ThingPermissions) (err error) {
	err = svc.hc.Rpc(svc.dThingID, UserSetPermissionsMethod, &permissions, nil)
	return
}

// NewUserClient creates a new client for invoking Authorization Client Services methods.
func NewUserClient(hc hubclient.IHubClient) *UserClient {
	cl := UserClient{
		hc:       hc,
		dThingID: "dtw:authz:user",
	}
	return &cl
}

// IUserService defines the interface of the 'User' service
//
// This defines a method for each of the actions in the TD.
type IUserService interface {

	// SetPermissions Set Permissions
	// Set the roles that can use a Thing or service
	SetPermissions(senderID string, permissions ThingPermissions) error
}

// NewUserHandler returns a server handler for Thing 'dtw:authz:user' actions.
//
// This unmarshalls the request payload into an args struct and passes it to the service
// that implements the corresponding interface method.
//
// This returns the marshalled response data or an error.
func NewUserHandler(svc IUserService) func(*things.ThingMessage) hubclient.DeliveryStatus {
	return func(msg *things.ThingMessage) (stat hubclient.DeliveryStatus) {
		var err error
		var resp interface{}
		var senderID = msg.SenderID
		switch msg.Key {
		case "setPermissions":
			var args ThingPermissions
			err = msg.Unmarshal(&args)
			if err == nil {
				err = svc.SetPermissions(senderID, args)
			} else {
				err = errors.New("bad function argument: " + err.Error())
			}
			break
		default:
			err = errors.New("Unknown Method '" + msg.Key + "' of service '" + msg.ThingID + "'")
			stat.Failed(msg, err)
		}
		if resp != nil {
			stat.Reply, _ = json.Marshal(resp)
		}
		stat.Completed(msg, err)
		return stat
	}
}