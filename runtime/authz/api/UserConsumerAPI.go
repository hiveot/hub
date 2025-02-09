// Package authz with the agent request handler for using service 'User'
// This builds a service consumer that send a service request.
// DO NOT EDIT. This file is auto generated by td2go. Any changes will be overwritten.
// Generated 31 Jan 2025 21:43 PST. 
package authz

import "github.com/hiveot/hub/transports/messaging"


// UserSetPermissions client method - Set Permissions.
// Set the roles that can use a Thing or service
func UserSetPermissions(hc *messaging.Consumer, permissions ThingPermissions)(err error){
    
    err = hc.Rpc("invokeaction", UserDThingID, UserSetPermissionsMethod, &permissions, nil)
    return
}
