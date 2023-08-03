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

The authz service manages the groups, the members of each group and their role in the group. The service is responsible for configuring NATS JetStream streams subject subscription to Things and access control to users.  

## Mapping Groups To Streams

Groups are simulated in nats using streams and ephemeral consumers. 

Bindings publish Thing events using the subject format: 
> things.{bindingID}.{thingID}.event.{eventType}.{instance}

These events are captured in a central '$events' stream.

When a group is created by the administrator, the associated stream is created using the group name. 

When a Thing is added to the group by the group manager, its subject is added as a stream source using $events as the source stream.

A group streams have a source defined for each Thing that is a member of the group. Each source is the combination of $events stream and the subject of the events captured in the stream. 

To access a group stream, the user creates an ephemeral consumer for the stream. Under the hood this uses the subject $JS.API.CONSUMER.CREATE.{groupName}. Only members of the group can publish to this subject. Similar for DELETE, INFO.

To read from a group stream, users also need permission for $JS.API.CONSUMER.MSG.NEXT.{groupName}.>


The authz service has a listGroups action to allow a client to list the groups they are a member of. 

In summary, the above setup accomplishes the following:
1. Authorization to receive events only for Things that are in the same group(s) as the user
2. Archiving of events for each group with its own retention period
3. Users can retrieve the latest value of each event
4Users can retrieve historical events
5Users only need to subscribe to a group to receive relevant events. No need to subscribe to individual things.

### Actions

Users can publish actions by writing them to the group stream on subject:
> things.{bindingID}.{thingID}.action.{actionID}.{clientID}

TBD: this requires that the published subject are constrained to to those defined with the stream. 

The $actions stream has a source for each group stream using subject "things.*.*.action.>". 

Bindings subscribe to the $actions stream using a durable consumer to receive requests. If multiple actions with the same ID are received after a reconnect, then only the last one should be applied by the binding.


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
