# HiveOT Packages

HiveOT Packages providing the core services along with a starter set of protocol bindings and client services.

* cmd - commandline interface
    * hubcli - hub commandling interface
    * genvocab - generate vocabulary
    * genapi - generate api from TDD

* runtime - hub digitwin runtime
	* authn - authentication
	* authz - authorization
	* certs - management of CA and server certificate
	* digitwin - digital twin inbox, outbox and directory 
* services - hub core srevices
	* history - collects and serves a history of Thing events
	* idprov - provisioning service
	* launcher - service for starting and stopping of plugins
	* state - state and configuration storage for services and clients
* bindings - protocol bindings and other services
	* hiveoview - basic web client providing a directory and dashboard viewer of hiveot Things
	* ipnet - ip network scanner
	* isy99x - protocol binding for legacy ISY 99/994 gateway
	* owserver - protocol binding for OWServer 1-wire gateway
	* zwavejs - protocol binding for zwave devices using zwavejs
