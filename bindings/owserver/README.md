# OWServer Binding

HiveOT protocol binding for the EDS OWServer-v2 1-wire gateway.

## Objective

Make EDS OWServer 1-wire devices available as WoT Things.

## Status

This binding status is alpha and breaking changes should be expected.

Limitations:

- No remote configuration
- Use of polling
- Limited support for 1-wire node types. Mostly sensors.
- Use of the HA7Net gateway is not supported

## Summary

The OWServer binding discovers and connects to 1-wire gateways to read information from connected 1-wire devices.

This binding:

- is implemented in golang.
- auto discovers OWServer V2 gateways on the local network using port 30303
- uses the owserver REST API to retrieve information.
- publishes TD documents for connected devices onto the Hub
- publishes updates sensor values periodically and on change onto the Hub
- has configuration to:
  - use mdns auto discovery or set an owserver address
  - set credentials to access the 1-wire gateway
  - the poll interval
  - the interval values are republished regardless if they have not changed

## Dependencies

1. A working EDS OWServer gateway on the local network.
2. This binding works with the [HiveOT Hub](https://github.com/hiveot/hivehub).

## Usage

This binding plugin runs out of the box. Build and install with the hub plugins into the installation hiveot/plugins folder and start it using the launcher.

Out of the box it will use MDNS discovery to locate an owserver gateway on the network. Once started, the binding is added to the directory, as is the discovered owserver gateway. It can be viewed using the cli or hiveoview web server.
