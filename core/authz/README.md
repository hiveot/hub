# authz service

## Objective

Provide authorization to access resources based on the client role and group. 


## Status

This service is being reworked for use with NATS JetStream

## Summary

This core service provides authorization for users and Things using a group role based access model.

The use-cases for using groups are:
1. Avoid the need to subscribe to individual related Things. Instead subscribe to the group that is formed based on the similarity.
2. Control access to certain Things to the group. For example, a security group can contain motion sensors that non-security users should not have access to. 
  - configure which Things are captured in the group
  - configure which users can access events in the group
3. differentiate who can control and configure Things using roles 
4. manage retention of events. For example, track environmental sensors for years while security sensor are tracked for only a couple of months.  This assumes that the need for retention aligns with the purpose of the group.

The authz service manages the groups, the members of each group and their role in the group.  


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
* clientID is the ID of the user, service or Thing.
* role is the client's role as described in the previous paragraph.


Example:
```yaml
all:
  admin: manager

temperature:
  user1: viewer
  things.binding1:thing1.event.>: thing  # only events  from thing1
  things.binding1.thing2: thing          # all events and actions of thing 2
```
