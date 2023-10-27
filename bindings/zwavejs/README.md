# ZwaveJS binding for HiveOT

This binding connects to a ZWave USB-Stick controller and publishes Wot TD documents and events to the HiveOT message
bus.

## Status

This binding is alpha. It is functional but breaking changes can still happen.

Binding TODOs:

1. Try to reconnect when gateway connection drops
1. Use a logging handler to choose level and write to file
1. Move lib to a typescript library folder reusable by other bindings
1. Use tinygo to reduce the generated wasm file.
   This is currently blocked as x/net package isn't supported by tinygo.

ZWave handling TODOs:

1. Update/improve mapping of zwave-js VID names to hiveot property/event/action vocabulary.
1. Include DataSchema in controller configurations for properties that aren't in the zwave-js vids.
1. Dimming duration is currently not supported
1. Load vid override map from file (determines vid prop/event/action and @type)
2. Handle arrays for property/event/action values

## Build

This needs:

* golang v1.18+
* yarn
* nodejs v18+
* typescript compiler v4.9+ (tsc, included in snap's node)
* make tools

Simply run make dist and watch the magic unfold (...hopefully, this is a javascript build environment after all).

The end result, if all goes according to plan, is a single binary: **dist/zwavebinding**

## Installation

Installation only needs 2 things, the executable and a client certificate used to authenticate with the hub.

The binding executable should be copied to the hiveot bindings folder, for example: ~/bin/hiveot/bin/bindings

Generate a client certificate:
> bin/hubcli createcervicecert zwavejs

This displays a x509 certificate and a private key. Copy these to a pem file in the hiveot/certs folder:

* certificate: hiveot/certs/zwavejsCert.pem
* private key: hiveot/certs/zwavejsKey.pem

NOTE: the client certificate identification is still somewhat fluid. The certificate CN must match the binding's
publisherID, by default zwavejs, but this could change to include the hostname in the future to allow for multiple
instances.
In that case, for a host called 'pc1', the binding ID would be zwavejs-pc1 and the certificate name '
zwavejs-pc1Cert.pem'.

On first run, the binding generates a zwavejs.yaml configuration file in the hiveot/config folder. The 'publisherID'
field contains the binding's ID that must match the certificate.

## Run

Before running the binding make sure the hub gateway is running. 'hubcli ls' lists the running processes.

The binding can be launched from the hubcli:
> bin/hubcli start zwavejs

To manually run the binding:
> ~/bin/hiveot/bin/bindings/zwavebinding

The binding connects to the Hub gateway using the default TLS websocket port 9884. The port can be changed in the
config/zwavejs.yaml configuration file.

On startup if a configuration file doesn't yet exist it is generates into hiveot/config/zwavejs.yaml.
This configuration file contains all sorts of interesting settings such as the publisherID, the ZWave S2 keys, zwave
serial port, and a few other settings. Most importantly in a conventional setup the gateway and serial port are
auto-discovered, so things will just run out of the box.

If you have an existing zwave network with S2 keys then copy these keys into the respective fields in the configuration
file should let it adopt the network, in theory. This has
not yet been tested.

To autostart the binding add it to the autostart section of the launcher.yaml configuration file in the config folder.

## Multiple Instances

* Note: This section is fluid.

Each binding instance on the network must have a unique publisherID. This allows for multiple zwave controllers in
different areas.
Each instance will publish the discovered Things and events under their own publisherID.

The gotcha is that bindings identify themselves to the hub with their publisherID using their client certificate
for authentication. The certificate must therefore be generated with the publisherID-hostname name.

The publisherID can be set in the zwavejs.yaml configuration file. This is useful when moving hosts without having to
generate a new certificate.

Currently the publisherID check is not enforced by the gateway but this might change in the future.