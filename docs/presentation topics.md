1. HiveOT Introduction demo
   * Directory view
   * Dashboard view
   * CLI
2. General Overview
   * what is HiveOT 'Hive Of Things' 
     * objectives
     * concepts: hub, agents, things, digital twins, consumers, transport vs protocol bindings
       * Thing IDs vs digital twin Thing IDs
     * current status
   * How HiveOT helps improve security and ease of use
     * Premise: Things are insecure and not to be trusted
     * HiveOT Things dont run network servers
     * Hub provides centralized device and consumer management
     * Role Based Authorization for interaction
   * HiveOT Topology [Picture]
     * hub: runtime; services; 
     * agents and 'things': bindings
     * consumers: cli, ui 
3. Hub runtime [picture]
     * digital twin: 
       * Thing TD's (directory)
       * event flow (inbox)
       * action flow (outbox)
     * transport bindings: http/sse, mqtt, ...
     * authentication
     * authorization
4. Use of the TD
   * IoT device TDs
     * Forms and protocols are not used!?
   * Hub digital twin TDs
     * The digital Twin Thing ID: dtw:agent:thing
     * Injecting Forms 
   * Challenges and Work In Progress
     * Suspend past experiences. Past experience hinders with expectations and assumptions. The WoT standard is well thought out and should be followed from a clean slate. 
     * TD uses polymorphism and inheritance that Golang doesn't support.
     * Property affordance inherits from dataschema but the dataschema of each type is different. How can a PA inherit from a different schema depending on its type? Need a different PA for each type?
       * workaround: flatten the various dataschema types into a single type and let the 'type' field determine which fields can be used. No compiler protection here.
     * Confusion on IDs. Examples rarely show a URI prefix. Are they really URI's? Why, for what purpose? Does anyone care? How to guarantee ID uniqueness.
     * Identifying the source of a TD, eg the device/service agent (servient) sending it.
       * HiveOT Hub exposes the digital twin ID which has the "dtw:agentID" prefix. This is guaranteed to be unique on the Hub. When linking Hubs this can be expanded with the hub ID (tbd).
     * Forms per property, event, action add potentially unnecessary TD bloat. Not good for limited devices. (WIP)
       * Top level methods, eg 'getproperty' would suffice for interaction.
       * In a Hub scenario Forms are very similar as they all use the same transport.
     * How to standardize definitions for metadata messageID, timestamp, sender in the transport protocols? (WIP)
       * HiveOT uses the 'Thing Message envelope' when sending events to agents and consumers. How to remove the need for it, or define this in the TD?
     * How to send action delivery/progress updates to senders? (WIP)
       * Update: Ege/Vignesh suggest to use properties to reflect action state and observe properties to receive it.  
       * HiveOT currently uses a '$delivery' event message and the messageID to send progress updates to the sender. 
       * WIP: change to observable properties. 
     * Linking events to actions that caused them. (WIP)
       * HiveOT treats identical event, action and property names are reflections of the same Thing state. 
       * HiveOT sends events for changes to Actions and properties. These events are not defined with an event affordance as they are implied through the property and action affordances.  
       * WIP: use observable properties for property changes
       * WIP: define observable properties for actions that reflect state (RPC type actions don't have observable properties)
     * Enums have no title 
       * This makes then not suitable for presentation
     * IoT Ontology
       * Standard vocabulary for device types, property/event/action types and units of measurement
       * Use of @type "ht:" prefix 
5. Hub services (batteries included)
     * Certificates - server certificate management
     * Provisioning - provisioning of devices
     * Launcher - starting and stopping services
     * State - storage for agents, services and consumers 
     * History - event history
     * Hiveoview - web ui
6. Commandline utilities
	* hubcli - hub commandline interface for admins
	* tdd2api - generating golang API from a TD 
7. POC Protocol bindings 
   * zwavejs
   * owserver
   * isy99x
   * ...
8. Implementations
   * WoT library (golang)
     * TDD
       * Challenges in Golang
         * Polymorphism, Inheritance
         * Forms
       * Action, Event, Property affordance
       * Forms
     * ConsumedThing
   * Hub library (golang)
     * TLS server
     * TLS client
     * Key management
     * Discovery
     * Bucket storage
     * Test environment
     * Certificate handling
   * Constructing an IoT protocol binding (golang)
       * agent connection
       * constructing TDs
       * sending events
       * receiving actions
       * writing properties
9. Future
   1. Improve compatibility with WoT standards
   2. Interoperability with 3rd party WoT based clients
   3. Beta release
   4. Transport bindings for websockets, mqtt, and others
   5. Protocol bindings for zigbee, coap, weather, etc., etc.
   6. Client libraries for javascript, python
   7. Services for automation, security, data analysis, etc., etc.
   8. Inter-Hub hive bridge 
