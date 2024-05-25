# HiveOT Packages

HiveOT Packages providing the core services along with a starter set of protocol bindings and client services.

* cmd - commandline interface
    * hubcli - hub commandling interface

* core - hub core services
	* auth - management of authentication and authorization for use by messaging servers and communication protocols
	* certs - management of CA and server certificate
	* digitwin - digital twin repository
	* history - collects and serves a history of Thing events
	* idprov - provisioning service
	* launcher - service for starting and stopping of plugins
	* msgserver
		* mqttmsgserver - embedded mqtt messaging server for mqtt core
		* natsmsgserver - embedded nats messaging server for nats core
	* state - state and configuration storage for services and clients

* bindings - protocol bindings and other services
	* hiveoview - basic web client providing a directory and dashboard viewer of hiveot Things
	* ipnet - ip network scanner
	* isy99x - protocol binding for legacy ISY 99/994 gateway
	* owserver - protocol binding for OWServer 1-wire gateway
	* zwavejs - protocol binding for zwave devices using zwavejs
