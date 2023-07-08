# authn service

## Objective

Provide authorization to access resources based on the client role and group. 


## Status

This service is being reworked for use with NATS

## Summary

This core service provides authorization for users and Things using a group role based access model.

The authz service manages a list of groups, the members of each group and their role in the group. 
The service is responsible for configuring NATS JetStream streams subject subscription to Things and access control to users.  

### Events

The authz services configures JetStream to receive events in group streams. Each group is defined as a stream. Users or services that are a member of the group can subscribe to the stream to receive events that are collected in the stream.

To collect events, the authz service creates a central $events stream that subscribes to all events on subject:
> "things.*.*.event.>"

For each group stream, a series of stream sources are defined with $events as the source stream and subject for the Thing that is a member of the group. Therefore for 100 things in a group, the stream will have 100 source stream instances, all with $events as the source and "things.{bindingID}.{thingID}.event.> as the subject. The reason for the extra $events stream is that JetStream currently does not support overlapping subscriptions in streams, which would prohibit adding a Thing to multiple groups. NATS 2.10 is required for this setup to work.

The bindingID is not required per-se but considered good practice in case duplicate thingIDs happen to occur in larger networks.
The thingID is required. Only events are captured in the stream groups. 

Users and services subscribe to JetStream streams to receive events from a group. 
>  "{groupID}"

The authz service has a listGroups action to allow a client to list the groups they are a member of. 

In summary, the above setup accomplishes the following:
1. Authorization to receive events only for Things that are in the same group(s) as the user
2. Archiving of events for each group with its own retention period
3. Users can retrieve historical events
4. Users only need to subscribe to a group to receive relevant events. No need to subscribe to individual things.

### Actions

Actions follow a similar pattern as events but in reverse direction. 
A single '$actions' stream captures all actions by subscribing to the subject:
> things.*.*.action.>

Bindings that listen to actions for their Things, subscribe to the actions stream. Each binding has a view on the action stream filtered on the "things.{bindingID}.>" subject. Bindings therefore only receive actions aimed at themselves. 

In case of intermittent connectivity, the Thing binding receives the actions that were missed from the stream on reconnect.

To prevent unauthorized publication of actions, each user is only authorized to publish in the group they are a member of and only if their role is operator or manager, and the thingID is a member of the group. Instead of using the subject prefix 'things', The groupID must be used. Publish: 
> {groupID}.{bindingID}.{thingID}.{actionID}.{clientID}

Authz subscribes to all actions of the groups and checks if the thingID is a member of that group. When passed, the action is republished with the "things" prefix, which is captured by the actions stream.
  
The 'all' group is built-in and automatically includes all Things. To allow a user to view all Things, the loginID is added to the all group with the 'viewer' role.


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
  urn:things:binding1:thing1: thing
  urn:things:binding1:thing2: thing
```
