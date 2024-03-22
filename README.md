# Hive-Of-Things Hub

HiveOT stands for the "Hive of Things". It provides a framework and services to collect and share IoT data with users.

The Hub for the *Hive-of-Things* provides a secure core and plugins to view and operate IoT devices. The Hub securely mediates between IoT device 'Things', services, and users using a hub-and-spokes architecture. Users interact with Things via the Hub without connecting directly to the IoT devices or services. The Hub is based on the [W3C WoT TD 1.1 specification](https://www.w3.org/TR/wot-thing-description11/) and uses a NATS or MQTT message bus for secure communication.

## Project Status

Status: The status of the Hub is alpha development (Feb 2024)

Completed core services:

* embedded messaging server for MQTT and Nats
* authentication and authorization (core/auth)
* directory service (core/directory)
* history service (core/history)
* provisioning service (core/idprov)
* state storage service (core/state)
* launcher service (core/launcher)

Bindings

* 1-wire protocol binding using owserver-v2 gateway (bindings/owserver)
* insteon binding using isy99x gateway (bindings/isy99x)
* zwave protocol binding using zwavejs (bindings/zwavejs)
* web client using html/htmx and go templates (bindings/hiveoview) - in progress

Integrations

It is a bit early to look at integrations, but some interesting candidates are:
* plc4go (https://plc4x.apache.org/users/getting-started/plc4go.html)
* home assistant (https://www.home-assistant.io/)

## Audience

This project is aimed at software developers and system implementors that are working on secure IoT solutions. HiveOT users subscribe to the security mandate that IoT devices should be isolated from the internet and end-users should not have direct access to IoT devices. Instead, all access operates via the Hub.

## Objectives

The primary objective of HiveOT is to provide a solution to secure the 'internet of things'.

The state of security of IoT devices is appalling. Many of those devices become part of botnets once exposed to the internet. It is too easy to hack these devices and most of them do not support firmware updates to install security patches.

This security objective is supported by not allowing direct access to IoT devices and isolate them from the rest of the network. Instead, IoT devices discover and connect to a 'hub' to exchange information through publish and subscribe. Hub services offer 'capabilities' to clients via a 'gateway' proxy service. Capabilities based security ensures that capability can only be used for its intended purpose.

> The HiveOT mandate is: 'Things Do Not Run (TCP) Servers'.

When IoT devices don't run TCP servers they cannot be connected to. This removes a broad attack surface. Instead, IoT devices connect to the hub using standard protocols for provisioning, publishing events, and subscribing to actions.

The secondary objective is to simplify development of IoT devices for the web of things.

The HiveOT Hub supports this objective by handling authentication, authorization, logging, tracing, persistence, rate limiting, resiliency and user interface. The IoT device only has to send the TD document describing the things it has on board, submit events for changes, and accept actions by subscribing to the Hub.

The third objective is to follow the WoT and other open standard where possible.

Open standards improves interoperability with devices and 3rd party services. Protocol bindings provide this interop.

Provide a decentralized solution. Multiple Hubs can build a bigger hive without requiring a cloud service and can operate successfully on a private network.

HiveOT is based on the [W3C WoT TD 1.1 specification](https://www.w3.org/TR/wot-thing-description11/). See [docs/README-TD.md] for more information.

## About HiveOT

Security is big concern with today's IoT devices. The Internet of Things contains billions of devices that when not properly secured can be hacked. Unfortunately the reality is that the security of many of these devices leaves a lot to be desired. Many devices are vulnerable to attacks and are never upgraded with security patches. This problem is only going to get worse as more IoT devices are coming to market. Imagine a botnet of a billion devices on the Internet ready for use by unscrupulous actors.

This 'HiveOT Hub' provides capabilities to securely interact with IoT devices and consumers. This includes certificate management, authentication, authorization, provisioning, directory and history services.

HiveOT compatible IoT devices therefore do not need to implement these features. This improves security as IoT devices do not run Web servers and are not directly accessible. They can remain isolated from the wider network and only require an outgoing connection to the Hub. This in turn reduces required device resources such as memory and CPU (and cost). An additional benefit is that consumers receive a consistent user experience independent of the IoT device provider as all
interaction takes place via the Hub interface.

HiveOT follows the 'WoT' (Web of Things) open standard developed by the W3C organization, to define 'Things'. It aims to be compatible with this standard.

Integration with 3rd party IoT devices is supported through the use of protocol bindings. These protocol bindings translate between the 3rd device protocol and WoT defined messages.

The communication infrastructure of the Hub is provided through a message bus. The core initially supports the NATS and MQTT message bus protocols through embedded servers.

Since the Hub acts as the intermediary, it is responsible for features such as authentication, logging, resiliency, pub/sub and other protocol integration. The Hub can dynamically delegate some of these services to devices that are capable of doing so, potentially creating a decentralized solution that can scale as needed and recover from device failure. As a minimum the Hub manages service discovery acts as a proxy for capabilities.

Last but not least, the 'hive' can be expanded by connecting hubs to each other through a 'bridge'. The bridge lets the Hub owner share select IoT information with other hubs. (future feature)

## Build From Source

To build the hub and plugins from source, a Linux system with golang 1.21 or newer must be available for the target system. To build zwavejs nodejs is used. hivoview is golang based and does not require any additional build tools.

Prerequisites:

1. An x86 or arm based Linuxsystem. Ubuntu, Debian, Raspberrian
2. Golang 1.21 or newer (with GOPATH set)
3. GCC Make any 2020+ version
4. NATS server and NATS-go developement library v2.10+
5. nodejs v18+ (for building zwavejs)

### Build Hub And CLI

1. Download source code:

```sh
git clone git@github.com:hiveot/hub
cd hub
``` 

2. Build the hub

```sh
make all
```

After the build is successful, the distribution files can be found in the 'dist' folder that can be deployed to the installation directory.

```md
dist / bin - core application binaries (hubcli, message bus, launcher)
dist / plugins - plugin binaries
dist / certs - CA and server certificate; authentication keys and tokens
dist / config - core and plugin configuration files
dist / stores - plugin data storage directory
```

## Install To User (Easiest)

When installed to user, the hub is installed into a user's bin directory and runs under that user's permissions.

This is the simplest way to install and run hiveot.

For example, for a user named 'hub' the installation home is '/home/hub/bin/hiveot'.

Download the source (see above), run
> make all && make install

This copies the distribution files to ~/bin/hiveot. The method can also be used to upgrade an existing installation. Executables are always replaced but only new configuration files are installed. Existing configuration remains untouched to prevent wrecking your working setup.

### Uninstall:

To uninstall simply remove the ~/bin/hiveot folder.

## Install To System (tenative)

While it is a bit early to install hiveot as a system application, this is how it could work:

For systemd installation to run as user 'hiveot'. When changing the user and folders make sure to edit the init/hiveot.service file accordingly. From the dist folder run:

1. Create the folders and install the files

```sh
sudo mkdir -P /opt/hiveot/bin
sudo mkdir -P /etc/hiveot/conf.d/ 
sudo mkdir -P /etc/hiveot/certs/ 
sudo mkdir /var/log/hiveot/   
sudo mkdir /var/lib/hiveot   
sudo mkdir /run/hiveot/

# Install HiveOT 
# download and extract the binaries tarfile in a temp for and copy the files:
tar -xf hiveot.tgz
sudo cp config/* /etc/hiveot/conf.d
sudo vi /etc/hiveot/hub.yaml    - and edit the config, log, plugin folders
sudo cp -a bin/* /opt/hiveot
```

Add /opt/hiveot/bin to the path

2. Setup the system user and permissions

Create a user 'hiveot' and set permissions.

```sh
sudo adduser --system --no-create-home --home /opt/hiveot --shell /usr/sbin/nologin --group hiveot
sudo chown -R hiveot:hiveot /etc/hiveot
sudo chown -R hiveot:hiveot /var/log/hiveot
sudo chown -R hiveot:hiveot /var/lib/hiveot
```

## Docker Installation

This is planned for the future.

## Configuration

All Hub services will run out of the box with their default configuration. Service can use an optional yaml based configuration file found in the config folder.

### Generate a CA certificate

Before starting the hub, a CA certificate must be created. By default, the hub generates a self-signed CA certificate. It is possible to use a CA certificate from a 3rd party source, but this isn't needed when used on the local network.

Generate the CA certificate using the CLI:

```sh
cd ~/bin/hiveot        # when installed locally
bin/hubcli ca create    
```

### Service Autostart Configuration

To configure autostart of services and protocol binding plugins, edit the provided launcher.yaml and add the plugins to the autostart section.
> vi config/launcher.yaml

### Systemd Configuration

Automatic startup after boot is supported through a systemd service. This can be used when installed system wide or as a user.

```shell
vi init/hiveot.service    #  (edit user, group, paths)
sudo cp init/hivehub.service /etc/systemd/system
sudo systemctl daemon-reload
sudo systemctl enable hivehub
sudo systemctl start hivehub
```

Once running, the running services can be viewed using the hub cli:
> hubcli ls

To stop or start a service:
> hubcli stop {serviceName}

> hubcli start {serviceName}

# Contributing

Contributions to HiveOT projects are always welcome. There are many areas where help is needed, especially with documentation, testing and building bindings for IoT and other devices. See [CONTRIBUTING](CONTRIBUTING.md) for guidelines.

# Credits

This project builds on the Web of Things (WoT) standardization by the W3C.org standards organization. For more information https://www.w3.org/WoT/

This project is inspired by the Mozilla Thing draft API [published here](https://iot.mozilla.org/wot/#web-thing-description). However, the Mozilla API is intended to be implemented by Things and is not intended for Things to register themselves. The HiveOT Hub will therefore deviate where necessary.

Many thanks go to JetBrains for sponsoring the HiveOT open source project with development tools.  
