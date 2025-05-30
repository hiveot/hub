// Package authz with the agent request handler for using service 'User'
// This builds a service consumer that send a service request.
// DO NOT EDIT. This file is auto generated by td2go. Any changes will be overwritten.
// Generated 01 Apr 2025 23:15 PDT. 
package authz

import "github.com/hiveot/hub/messaging"


// UserSetPermissions client method - Set Permissions.
// Set the roles that can use a Thing or service
func UserSetPermissions(hc *messaging.Consumer, permissions ThingPermissions)(err error){
    
    err = hc.Rpc("invokeaction", UserDThingID, UserSetPermissionsMethod, &permissions, nil)
    return
}
