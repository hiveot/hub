# HiveOT Packages 

HiveOT Packages provides the core functionality with protocol bindings and applications.

* cmd - commandline interface
  * hubcli - hub commandling interface
  * natscore - running of the nats core
  * mqttcore - running of the mqtt core

* core - hub core services 
  * auth   - management of authentication and authorization for the messaging server
  * certs  - management of CA and server certificate
  * config - core configuration
  * launcher      - service for starting and stopping of plugins 
  * mqttmsgserver - embedded mqtt messaging server for mqtt core
  * natsmsgserver - embedded nats messaging server for nats core

* plugins - protocol bindings and other services
  * directory - collects and serves the TD documents of discovered Things
  * history   - collects and serves a history of Thing events
  * hiveoview - web client dashboard for viewing hiveot Things
  * owserver  - binding for OWServer 1-wire gateway
  * zwavejs   - binding for zwave devices using zwavejs


