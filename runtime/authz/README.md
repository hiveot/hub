# authz - HiveOT Hub Authorization

## Objective

Manage authorization of client requests based on their role.

## Status

This service is functional. This changes to include authorization roles in the agent published TD.

TODOs:
* Define authorization in the TD instead of some hidden mechanism
  * Extend vocabulary with an 'authz' field for each action and event
  * authz: [list of roles]
* Implementation of custom roles is not yet complete.

## Summary

This service provides the following capabilities:

1. Authorize requests made by agents, consumers and services
2. Retrieve roles
3. Custom role management (future)

Authorization is built on top of authentication and uses its client profile.

## Usage

This service is included in the hub runtime. 


## Setting Permissions

All permissions are role based. All clients, eg consumers, agents and services each have a role.

Permissions are split in permissions to publish actions and permissions to subscribe to events. 

Only agents and services offer Things that can be controlled through actions. Note that Service Things are also called capabilities. 

Only consumers and services are able to subscribe to Thing events. Agents represent IoT devices and are not consumers of events (that would make them a service).

An agent or service can set the default roles that are allowed to invoke actions on its Things or capabilities using the SetPermission method. This lets the service be used out of the box without configuration. Agents typically do not set permissions for its IoT devices in which case the default role permissions apply.

The authorization service can override the defaults through configuration. 

## Validating Permissions

In validating permissions, the authz service first checks whether the requested agent or service has any permissions set for the requested Thing. If set then the service permissions validation is used. If not set then the default role permissions apply.

This allows each service to control who can use it. It is even possible to apply this to IoT devices themselves and only make them available to certain roles.
  
A consumer can validate if it has permissions to invoke an action to control or configure a Thing. This can be useful to enable or disable actions in the user interface.

### Default Role Permissions

The default role permissions apply to authenticated clients only:
1. All clients can subscribe to Thing events.
2. Operators can invoke defined actions
3. Managers can in addition invoke the action to set properties
4. Agents (for devices) can not invoke actions, only publish events.
5. Services and administrators roles have no publish or subscribe restrictions.

### Service Permissions

Each service can set its own default permissions. If no service permissions are set, then the default rol permissions apply.

When an agent or service sets usage permissions then it defines the ThingID and the roles that are allowed or denied requesting actions of that Thing. 

Use denied without allowed to allow access to custom roles.
Use allowed without denied to not allow access other than the listed roles.

