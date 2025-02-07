# HiveOT Packages

HiveOT golang packages provide the core services along with a starter set of protocol bindings and client services.

* cmd - commandline interface
    * hubcli - hub commandling interface
    * tdd2api - generate vocabulary and API from a TDD

* runtime - hub digitwin runtime
	* authn - authentication
	* authz - authorization
	* certs - management of CA and server certificate
	* digitwin - digital twin inbox, outbox and directory
      * inbox: stores action requests and tracks delivery progress. The latest value can be queried by consumers.
      * outbox: stores events for delivery to consumers. The latest value of each event can be queried by consumers.
      * directory: stores the TD's published by agents. The latest TDD of each Thing can be queried by consumers. 
  
* services - hub core services
	* history - collects and serves a history of Thing events
	* idprov - provisioning service
	* launcher - service for starting and stopping of plugins and generating authentication tokens for these services.
	* state - state and configuration storage for services and clients
	* hiveoview - SSR web client providing a directory and dashboard viewer of hiveot Things
  
* bindings - example protocol bindings and other services
	* ipnet - ip network scanner
	* isy99x - protocol binding for legacy ISY 99/994 gateways
	* owserver - protocol binding for OWServer 1-wire gateways
	* zwavejs - protocol binding for zwave devices using zwavejs
