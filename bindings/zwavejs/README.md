# ZwaveJS binding for HiveOT

This binding connects to a ZWave USB-Stick controller, and publishes events to the HiveOT message bus.

## Status

This binding is functional. 
Building as a single binary is very challenging. This currently prevents upgrading zwave-js to the latest version.

### TODO
1. Upgrade to zwavejs-latest. Must support single binary. (pkg, deno or bun)
2. Report heal network status/progress of a node
3. API for reading device network layout
1. ZWave stick reconnect support (Recover after serial port removal) [done]
1. Track health of nodes; dropped messages, etc.
    * timeouts; dropped messages
    * list of neighbours [done]
1. Improve mapping of zwave-js VID names to hiveot property/event/action vocabulary. 
1. Dimming duration is currently not supported


### Issues

Some observed unexpected behavior
1. no-one is sending a "notification" event
2. AEON Labs ZW096 is sending "metadata updated" when only a value is changed
3. AEON Labs ZW141 (nano shutter v2) has no events (CC Binary|Multilevel Sensor|Central Scene)
4. EAON Labs DSB09 (home energy meter) sends value as 'metadata updated' and 'value added' instead of 'value updated'

## Mapping zwave VID to TD property, event and action affordances

Values of ZWaveJS nodes are identified by so-called value ID's or VID's.

For an overview see: https://zwave-js.github.io/node-zwave-js/#/api/valueid?id=valueid

A VID is an object with 4 fields, two are optional:
* commandClass - numeric identifier of the [command class](https://zwave-js.github.io/node-zwave-js/#/api/CCs/index) [required]
* property - the identifier of the property
* endpoint - in case multiple resources exist in the same device. Default is 0. See also: https://community.silabs.com/s/article/z-wave-multi-channel-end-points
* propertyKey - optional sub-address multiple values of a node. 

This binding uses the VID to construct a key for properties, events and actions:
  key = {commandClass}-{property}-{endpoint}[-{propertyKey}] 

Where propertyKey is omitted if not applicable. The endpoint is always provided and set to 0 for the default default.


# Build and install

To build a zwavejs executable for x64 or armv7, just run 'make zwavejs' from the hiveot source code.

This runs 'npm i' on both the ./libjs and the ./bindings/zwavejs directories followed by zwavejs/build.sh. If something goes wrong cd into these directories and run npm i to discover the problem. Once npm i runs fine, zwavejs can be rebuild using the bindings/zwavejs/build.sh script. The bundler will likely report a few warnings that can be ignored.

The 'zwavejs' binary output can be found in the bindings/zwavejs/dist directory which in turn is copied into the dist/plugins directory.

Notes:
1. The whole build and bundling is quite fragile. Some package versions can break things. Especially 'pkg', 'undici', 'serialport' and 'zwave-js' are a problem.
   Don't upgrade these unless you know what you're doing.  
   Package version known to work with node-v22:
* eventsource-3.0.5
* serialport-12.0.0   (version 13 breaks bundling)
* zwave-js-13.10.1    (version 14 breaks bundling)
* @yao-pkg/pkg-6.3.2  (build missing files)
* esbuild-0.25.1      (build missing files)
* undici-6.21         (v7 causes node:sqlite bundle error )

2. This uses 'pkg' to build an executable binary containing nodejs. I had a hard time to get it all working and bundled. zwave-js-ui has been a great help.

3. I'd love to use deno for running, dev and building an executable but it doesn't support serial port on linux. Node's serial port uses a node api that isn't supported on Deno.


## Prerequisites

* npm 
* Node v22.x  (or use nvm - node version manager)
* make

### Installing node v22 on raspberry pi 3 (armv7, 64bit)

As of Early 2025, the default image of Raspbian for pi3 (ARMv7, 64bit) comes with Debian GNU/Linux 12 (bookworm).
This distribution comes with nodejs-v18.19 and npm 9.2.0.

The easiest way to upgrade to node v22 is to use nvm (node version manager). The instructions can be found at: https://nodejs.org/en/download.
```
$ curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.40.1/install.sh | bash
(close and re-open a terminal)
$ nvm install 22
$ nvm alias default 22
```

The build.sh script builds the output for the current architecture. Cross compiling is not supported.

## Building with esbuild

This first step is just for testing the build process using esbuild. If this already fails then no use using pkg or the 'postject' node20+ injector. Note: one reason to take this step is to allow packages with es modules (axios) to work. pkg seems to not build correctly with code generated with tsc && tsc-alias.

To build the binding into a single js file:

> esbuild src/main.ts --bundle --platform=node --target=node20 --preserve-symlinks --outfile=build/zwavejs-esbuild.js

Note: to run this, 2 folders are needed in the project root: (yes not ideal but this is just for verification)

* ln -s node_modules/@serialport/bindings-cpp/prebuilds .
* ln -s node_modules/@zwave-js/config/config .

Then run from the project root with:
> ZWAVEJS_EXTERNAL_CONFIG=./dist/cache node build/zwavejs-esbuild.js --clientID testsvc --home ~/bin/hiveot/

* note1: --clientID is the client this runs under, eg 'testsvc' during testing. Default will be the binding name zwavejs.

## build a single executable using pkg and esbuild.js from zwave-js-ui

NOTE: This uses the esbuild.js script from zwave-js-ui, which does some filename mangling to
get the externals to work. Not sure how it works but I also really don't care.

Build with:
> yarn pkg    or   ./build.sh

run it:
> dist/zwavejs --clientID testsvc --home ~/bin/hiveot

TODO: Clean up this build mess. Switch back to JS.

## Installation

Installation needs the executable, an authentication token, and the CA certificate of the Hub.

The binding executable should be copied to the hiveot plugins folder, for example: ~/bin/hiveot/bin/plugins. The authentication token is generated by the launcher, or can be generated manually using the hubcli. The token file has the same name as the executable with the .token extension. The CA certificate is generated by the hubcli on startup and placed in certs/caCert.pem.

In addition, the user its running must have access to the serial port, eg the dialout group.

## Run

Before running the binding make sure the hub gateway is running. 'hubcli ls' lists the running processes.

The binding can be launched from the hubcli:
> bin/hubcli start zwavejs

To manually run the binding:
> ~/bin/hiveot/plugins/zwavejs

The binding connects to the Hub gateway using service discovery, or uses the hubURL configuration from the config/zwavejs.yaml configuration file.

On startup if a configuration file doesn't yet exist it generates a default file into hiveot/config/zwavejs.yaml. This configuration file contains all sorts of interesting settings such as the serviceID, the ZWave S2 keys, zwave controller serial port, and a few other settings. Most importantly in a conventional setup the gateway and serial port are auto-discovered, so things will just run out of the box.

If you have an existing zwave network with S2 keys then copy these keys in their hex format into the respective fields in the configuration file should let it adopt the network, in theory. This has not yet been tested.

To autostart the binding add it to the autostart section of the launcher.yaml configuration file in the config folder.

## Multiple Instances

* Note: This section is fluid.

Each binding instance on the network must have a unique serviceID. This allows for multiple zwave controllers in different areas. To run multiple instances with different service IDs rename the executable. For example to zwavejs-1, zwavejs-2, etc. The launcher generates keys and tokens for each separately. Each instance will publish the discovered Things and events under their own serviceID.


## Testing

OMG the pain!
You'd hope that intellij allows you to just run it, but no such luck.

Run the 'debugtsx' from the intellij debug  