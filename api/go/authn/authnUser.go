// Package authn with types and interfaces for using this service with agent 'authn'
// DO NOT EDIT. This file is auto generated by tdd2api. Any changes will be overwritten.
// Generated 21 Jan 25 13:17 PST. 
package authn

import "errors"
import "github.com/hiveot/hub/transports/messaging"
import "github.com/hiveot/hub/transports/tputils"
import "github.com/hiveot/hub/transports"

// UserAgentID is the account ID of the agent managing the Thing.
const UserAgentID = "authn"

// UserServiceID is the thingID of the device/service as used by agents.
// Agents use this to publish events and subscribe to actions
const UserServiceID = "user"

// UserDThingID is the Digitwin thingID as used by consumers. Digitwin adds the dtw:{agent} prefix to the serviceID
// Consumers use this to publish actions and subscribe to events
const UserDThingID = "dtw:authn:user"

// Thing names
const (
    UserActionGetProfile = "getProfile"
    UserActionLogin = "login"
    UserActionLogout = "logout"
    UserActionRefreshToken = "refreshToken"
    UserActionUpdateName = "updateName"
    UserActionUpdatePassword = "updatePassword"
    UserActionUpdatePubKey = "updatePubKey"
)

//--- Schema definitions of Thing 'dtw:authn:user' ---

// ClientProfile defines a Client Profile data schema of the authn agent.
//
// This contains client information of device agents, services and consumers
type ClientProfile struct {
    
    // ClientID with Client ID
    ClientID string `json:"clientID,omitempty"`
    
    // ClientType with Client Type
    ClientType ClientType `json:"clientType,omitempty"`
    
    // Disabled with Disabled
    //
    // This client account has been disabled
    Disabled bool `json:"disabled,omitempty"`
    
    // DisplayName with Display Name
    DisplayName string `json:"displayName,omitempty"`
    
    // PubKey with Public Key
    PubKey string `json:"pubKey,omitempty"`
    
    // Updated with Updated timestamp in msec since epoch
    Updated int64 `json:"updated,omitempty"`
}

// ClientType enumerator
//
// identifies the client's category
type ClientType string
const (
    
    // ClientTypeAgent for Agent
    //
    // Agents represent one or more devices
    ClientTypeAgent ClientType = "agent"
    
    // ClientTypeService for Service
    //
    // Service enrich information
    ClientTypeService ClientType = "service"
    
    // ClientTypeConsumer for Consumer
    //
    // Consumers are end-users of information
    ClientTypeConsumer ClientType = "consumer"
)

//--- Argument and Response struct for action of Thing 'dtw:authn:user' ---

const UserGetProfileMethod = "getProfile"

const UserLoginMethod = "login"

// UserLoginArgs defines the arguments of the login function
// Login - Login with password
type UserLoginArgs struct {
    
    // ClientID with Login ID
    ClientID string `json:"clientID,omitempty"`
    
    // Password with Password
    Password string `json:"password,omitempty"`
}

const UserLogoutMethod = "logout"

const UserRefreshTokenMethod = "refreshToken"

const UserUpdateNameMethod = "updateName"

const UserUpdatePasswordMethod = "updatePassword"

const UserUpdatePubKeyMethod = "updatePubKey"


// UserGetProfile client method - Get Client Profile.
func UserGetProfile(hc *messaging.Consumer)(resp ClientProfile, err error){
    
    err = hc.Rpc("invokeaction", UserDThingID, UserGetProfileMethod, nil, &resp)
    return
}

// UserLogin client method - Login.
// Login with password
func UserLogin(hc *messaging.Consumer, clientID string, password string)(token string, err error){
    var args = UserLoginArgs{clientID, password}
    err = hc.Rpc("invokeaction", UserDThingID, UserLoginMethod, &args, &token)
    return
}

// UserLogout client method - Logout.
// Logout from all devices
func UserLogout(hc *messaging.Consumer)(err error){
    
    err = hc.Rpc("invokeaction", UserDThingID, UserLogoutMethod, nil, nil)
    return
}

// UserRefreshToken client method - Request a new auth token for the current client.
func UserRefreshToken(hc *messaging.Consumer, oldToken string)(newToken string, err error){
    
    err = hc.Rpc("invokeaction", UserDThingID, UserRefreshTokenMethod, &oldToken, &newToken)
    return
}

// UserUpdateName client method - Request changing the display name of the current client.
func UserUpdateName(hc *messaging.Consumer, newName string)(err error){
    
    err = hc.Rpc("invokeaction", UserDThingID, UserUpdateNameMethod, &newName, nil)
    return
}

// UserUpdatePassword client method - Update Password.
// Request changing the password of the current client
func UserUpdatePassword(hc *messaging.Consumer, password string)(err error){
    
    err = hc.Rpc("invokeaction", UserDThingID, UserUpdatePasswordMethod, &password, nil)
    return
}

// UserUpdatePubKey client method - Update Public Key.
// Request changing the public key on file of the current client.
func UserUpdatePubKey(hc *messaging.Consumer, publicKeyPEM string)(err error){
    
    err = hc.Rpc("invokeaction", UserDThingID, UserUpdatePubKeyMethod, &publicKeyPEM, nil)
    return
}


// IUserService defines the interface of the 'User' service
//
// This defines a method for each of the actions in the TD. 
// 
type IUserService interface {

   // GetProfile Get Client Profile
   GetProfile(senderID string) (resp ClientProfile, err error)

   // Login Login
   // Login with password
   Login(senderID string, args UserLoginArgs) (token string, err error)

   // Logout Logout
   // Logout from all devices
   Logout(senderID string) error

   // RefreshToken Request a new auth token for the current client
   RefreshToken(senderID string, oldToken string) (newToken string, err error)

   // UpdateName Request changing the display name of the current client
   UpdateName(senderID string, newName string) error

   // UpdatePassword Update Password
   // Request changing the password of the current client
   UpdatePassword(senderID string, password string) error

   // UpdatePubKey Update Public Key
   // Request changing the public key on file of the current client.
   UpdatePubKey(senderID string, publicKeyPEM string) error
}

// NewHandleUserRequest returns an agent handler for Thing 'dtw:authn:user' requests.
//
// This unmarshalls the request payload into an args struct and passes it to the service
// that implements the corresponding interface method.
// 
// This returns the marshalled response data or an error.
func NewHandleUserRequest(svc IUserService)(func(msg *transports.RequestMessage, c transports.IConnection) *transports.ResponseMessage) {
    return func(msg *transports.RequestMessage, c transports.IConnection) *transports.ResponseMessage {
        var output any
        var err error
        switch msg.Name {
            case "logout":
                if err == nil {
                  err = svc.Logout(msg.SenderID)
                } else {
                  err = errors.New("bad function argument: "+err.Error())
                }
                break
            case "refreshToken":
                var args string
                err = tputils.DecodeAsObject(msg.Input, &args)
                if err == nil {
                  output, err = svc.RefreshToken(msg.SenderID, args)
                } else {
                  err = errors.New("bad function argument: "+err.Error())
                }
                break
            case "updateName":
                var args string
                err = tputils.DecodeAsObject(msg.Input, &args)
                if err == nil {
                  err = svc.UpdateName(msg.SenderID, args)
                } else {
                  err = errors.New("bad function argument: "+err.Error())
                }
                break
            case "updatePassword":
                var args string
                err = tputils.DecodeAsObject(msg.Input, &args)
                if err == nil {
                  err = svc.UpdatePassword(msg.SenderID, args)
                } else {
                  err = errors.New("bad function argument: "+err.Error())
                }
                break
            case "updatePubKey":
                var args string
                err = tputils.DecodeAsObject(msg.Input, &args)
                if err == nil {
                  err = svc.UpdatePubKey(msg.SenderID, args)
                } else {
                  err = errors.New("bad function argument: "+err.Error())
                }
                break
            case "getProfile":
                if err == nil {
                  output, err = svc.GetProfile(msg.SenderID)
                } else {
                  err = errors.New("bad function argument: "+err.Error())
                }
                break
            case "login":
                args := UserLoginArgs{}
                err = tputils.DecodeAsObject(msg.Input, &args)
                if err == nil {
                  output, err = svc.Login(msg.SenderID, args)
                } else {
                  err = errors.New("bad function argument: "+err.Error())
                }
                break
            default:
            	err = errors.New("Unknown Method '"+msg.Name+"' of service '"+msg.ThingID+"'")
        }
        return msg.CreateResponse(output,err)
    }
}

// UserTD contains the raw TD of this service for publication to the Hub
const UserTD = `{"actions":{"getProfile":{"@type":"hiveot:function","title":"Get Client Profile","output":{"readOnly":false,"type":"ClientProfile"},"safe":true},"login":{"@type":"hiveot:function","description":"Login with password","title":"Login","input":{"readOnly":false,"type":"object","properties":{"clientID":{"title":"Login ID","readOnly":false,"type":"string"},"password":{"title":"Password","readOnly":false,"type":"string"}}},"output":{"title":"Token","readOnly":false,"type":"string"}},"logout":{"@type":"hiveot:function","description":"Logout from all devices","title":"Logout","idempotent":true},"refreshToken":{"@type":"hiveot:function","title":"Request a new auth token for the current client","input":{"title":"Old Token","readOnly":false,"type":"string"},"output":{"title":"New Token","readOnly":false,"type":"string"}},"updateName":{"@type":"hiveot:function","title":"Request changing the display name of the current client","idempotent":true,"input":{"title":"New Name","readOnly":false,"type":"string"}},"updatePassword":{"@type":"hiveot:function","description":"Request changing the password of the current client","title":"Update Password","idempotent":true,"input":{"title":"Password","readOnly":false,"type":"string"}},"updatePubKey":{"@type":"hiveot:function","description":"Request changing the public key on file of the current client.","title":"Update Public Key","idempotent":true,"input":{"title":"Public Key PEM","description":"Public Key in PEM format","readOnly":false,"type":"string"}}},"@context":["https://www.w3.org/2022/wot/td/v1.1",{"hiveot":"https://www.hiveot.net/vocab/v0.1"}],"@type":"Service","created":"2024-06-04T17:00:00.000Z","deny":["none"],"description":"HiveOT runtime service for users","events":{},"id":"user","modified":"2024-06-04T17:00:00.000Z","properties":{},"schemaDefinitions":{"ClientProfile":{"title":"Client Profile","description":"This contains client information of device agents, services and consumers","readOnly":false,"type":"object","properties":{"clientID":{"title":"Client ID","readOnly":false,"type":"string"},"clientType":{"title":"Client Type","readOnly":false,"type":"ClientType"},"disabled":{"title":"Disabled","description":"This client account has been disabled","readOnly":false,"type":"bool"},"displayName":{"title":"Display Name","readOnly":false,"type":"string"},"pubKey":{"title":"Public Key","readOnly":false,"type":"string"},"updated":{"title":"Updated timestamp in msec since epoch","readOnly":false,"type":"int64"}}},"ClientType":{"title":"Client Type","description":"identifies the client's category","oneOf":[{"title":"Agent","description":"Agents represent one or more devices","const":"agent","readOnly":false},{"title":"Service","description":"Service enrich information","const":"service","readOnly":false},{"title":"Consumer","description":"Consumers are end-users of information","const":"consumer","readOnly":false}],"readOnly":false,"type":"string"}},"security":["bearer"],"securityDefinitions":{"bearer":{"scheme":""}},"title":"Authentication User Service","support":"https://www.github.com/hiveot/hub"}`