# authn service

## Objective

Provide authorization to access resources based on the client role and group. 


## Status

This service is being reworked for use with NATS

## Summary

This Hub service provides authorization for users, services and IoT devices using a group role based access model. 

IoT devices have the device role that allows them to publish TD documents, events and subscribe to actions. Access is limited to the subject:
> pub/sub: "things.{publisherID}.>"

Services have a broad access to things and groups. Their access is defined in jwt tokens issued based on their purpose. The default access is defined as:
> pub/sub: ">"

Users don't subscribe directly to things, but to group streams instead. Each group has a stream defined that members of that group can access. Users can publish actions to the group:
> sub: "{groupID}.>"
> pub: "{groupID}.*.*.action.>"

The authorization service handles:
1. defining group streams for all defined groups
2. define subject mapping from thingID to group ID for all things in the group
3. define mapping of group actions to thing actions for all things in the group




* Clients can be users, services, and IoT devices. They must be authenticated using a valid certificate or access token.
* Groups contain resources and clients. Clients can access resources in the same group based on their role. A client only has a single role.
  * The 'all' group includes all resources without need to add them explicitly. Use with care. 
* role. Clients have a role in a group. The role determines the action the client is allowed on the resource ('Things'). Roles are:
  * viewer: allows read-only access to the resource attributes such as Thing properties and output values
  * operator: in addition to viewer, allows operating the resource inputs such as a Thing switch
  * manager: in addition to operator, allows changing the resource configuration
  * administrator: in addition to manager, can manage users to the group
  * thing: role is for use by IoT devices only and identifies it as the resource to access. Thing publishers are devices that have full access to the Things they publish. They are identified by their publisher ID in the device client certificate. 
  
The 'all' group is built-in and automatically includes all Things. To allow a user to view all Things, the loginID is added to the all group with the 'view' role.

In addition to manual groups, groups are automatically created for each Thing type. For example a group of temperature sensors. Clients in this group will automatically have access to new sensors of type temperature.

### Group Management

Things, users, groups and roles are defined in the ACL (access control list) store. The default store implementation is file based that is loaded in memory. The 'authz' commandline lets the administrator manage users, groups and roles in this file. 

To authorize a request, the authz library uses the ID of the client to determine the role for the requested resource(s). The role determines the permissions, which are:
* Read TD: Read the TD of a Thing.
* Configure: Thing: Permission to request an update of Thing properties.  
* Event: Permission to publish or subscribe to Thing events. 
* Action: Permission to publish or subscribe to Thing actions

The role permissions for these message actions are:

| Role / action | Read TD | Configure | Event | Action |
|---------------|---------|-----------|-------|--------|
| viewer        | read    | -         | read  | read   |
| operator      | read    | -         | read  | write  |
| manager       | read    | write     | read  | write  |
| admin         | read    | write     | read  | write  |
| thing         | write   | write     | write | write  |


## Configuration


### Groups File
Groups are stored in a groups.yaml file in the following format.
```
groupName:
  clientID: role
```
Where:
* groupName can be any name. The 'all' group is predefined and implies to contain all Thing IDs as client. Only end-user IDs need to be added. 
* clientID is either the user-IDs or Thing IDs.
* role is the client's role as described in the previous paragraph.


Example:
```yaml
all:
  admin: manager

temperature:
  user1: viewer
  urn:zone1:publisher1:thing1: thing
  urn:zone1:publisher1:thing2: thing
```
