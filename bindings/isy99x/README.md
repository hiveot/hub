# ISY99x Protocol Binding

ISY99 is a legacy gateway for the Insteon protocol. It is not longer produced and replaced by the ISY994. The ISY994 which also works with this publisher, albeit limited to what the ISY99 can do.

## Status

This binding is bare bones functional POC. Breaking changes are to be expected.

Limitations:
1. no remote configuration
2. no actions yet
3. only support for switches and dimmer nodes
4. uses polling instead of subscription to ISY events
5. no support for ISY scenes and programs
6. no support for ISY ZWave or UPB  
7. the vocabulary mapping of property/event/action names is inconsistent. This needs to be fixed in hiveot itself first. 

Above restrictions can be fixed with some development and a working ISY/PLM for testing. Unfortunately

## Dependencies

This binding requires the use of an ISY99x INSTEON gateway device and a PLM device modem for connecting to the power net.

## Configuration

Edit isy99.yaml with the ISY99 gateway address and login name/password. The gateway can also be configured through the gateway node 'gatewayAddress' configuration.

See config files in ./test as examples
