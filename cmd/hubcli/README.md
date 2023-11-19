# Hub CLI

Commandline interface for managing Hub services.

## Status

The status of the Hub CLI Alpha.
It is functional but breaking changes might happen. CLI for services are still being added.

TODO:

- subscriptions don't persist after auto-reconnect.

## Summary

The Hub CLI provides a commandline interface for managing Hub and service configuration.

Usage: hubcli -h

## Setup

The first thing to do after installation is to initialize the hub configuration.

> hubcli init --home ~/bin/hiveot

Where ~/bin/hiveot is the root installation folder. By default the home folder is the parent of the hubcli binary.

## Startup

To start the hub run:
> hubcli start  [--home ~/bin/hiveot]

This will start the Launcher which in turn starts any configured services. The launcher is configured through the '
config/launcher.yaml' file. This file describes the core to use and the services to launch. See the included file for
more details. For example:

```yaml
# config/launcher.yaml

# core to launch before the plugins. The core binary must reside in the bin directory.
# comment out if a core is already running elsewhere.
# known cores are mqttcore or natscore
core: mqttcore

# start services in order
autostart:
  # core services
  - certs          # IoT certificate management
  - bucket         # simple state store for services and clients
  - directory      # Thing directory store
  - history        # Thing history store
  - provisioning   # provisioning of IoT devices

  # plugins for are additional services
  - dashboard      # dashboard for viewing IoT device info

  # protocol bindings are added in future
  - owserver       # Publish Things connected to a local OWServer V2 1-wire gateway
  # etc...
```

Each service or gateway entry can have additional specifications:

To list running services:
> bin/hubcli ls
>
> 