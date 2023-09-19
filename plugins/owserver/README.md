# OWServer Binding

HiveOT 1-wire protocol binding for OWServer-v2 gateway.


## Objective

Convert EDS OWServer 1-wire devices to WoT Things.

## Status

This binding is being converted to using the messaging based hub.


## Summary

The OWServer binding discovers and connects to 1-wire gateways to read information from connected 1-wire devices. 

This binding:
* is implemented in golang.
* auto discovers OWServer V2 gateways on the local network using port 30303
* uses the owserver REST API to retrieve information.
* publishes TD documents for connected devices onto the Hub
* publishes updates sensor values periodically and on change onto the Hub
* has configuration to:
  * use mdns auto discovery or set an owserver address
  * set credentials to access the 1-wire gateway
  * the poll interval 
  * the interval values are republished regardless if they have not changed


## Dependencies

1. A working EDS OWServer gateway on the local network. 
2. This binding works with the [HiveOT Hub](https://github.com/hiveot/hub).


## Usage

This binding plugin runs out of the box. Build and install with the hub plugins into the installation bin/plugins folder and start it using the launcher.

Out of the box it will use MDNS discovery to locate an owserver gateway on the network. Once started, the binding is added to the directory, as is the discovered owserver gateway. It can be viewed using the cli or hiveoview web server.
