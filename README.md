# Hive-Of-Things Hub

HiveOT stands for the "Hive of Things". It consists of one or more 'Hubs' to collect and share IoT data with its users.

The Hub for the *Hive-of-Things* provides a secure [runtime](runtime/README-runtime.md) to view and control IoT devices. The Hub securely mediates between IoT device 'Things', services, and users using a hub-and-spokes architecture. Users interact with Things via the Hub's digital twin, without connecting directly to the IoT devices or services. The Hub is based on the [W3C WoT TD 1.1 specification](https://www.w3.org/TR/wot-thing-description11/). Multiple communication protocols are supported for IoT devices and users.

![System Overview](docs/hub-overview.jpg)

## Project Status

Status: The Hub core is currently being reworked to support the Web-of-things (WoT) Thing Description (TD) specification. (Jan 2025). 

TODO before alpha:
1. Support for Forms sections in TDD documents describing the protocols to interact with a Thing.
1. Revisit the vocabulary to integrate or adopt existing vocabularies where possible
1. Test/fix security; token expiry, bad agents or consumers, rate-limiting, role authz

Roadmap:
1. Add websockets sub-protocol binding (in progress)
1. Support for WoT discovery profile
1. Support client certificate authentication
1. Support mqtt transport protocol
2. Rework internal TD document format to improve WoT TD compatibility

Future:
1. HiveOT inter-hub bridging service
2. OAuth2 support


Integrations

It is a bit early to look at integrations, but some interesting candidates are:
* interoperability with WoT clients
* plc4go (https://plc4x.apache.org/users/getting-started/plc4go.html)
* home assistant (https://www.home-assistant.io/)

## Audience

This project is aimed at software developers and system implementors that are working on secure IoT solutions. HiveOT users subscribe to the security mandate that IoT devices should be isolated from the internet and end-users should not have direct access to IoT devices. Instead, all access operates via the Hub.

## About HiveOT

Security is big concern with today's IoT devices. The Internet of Things contains billions of devices that when not properly secured can be hacked. Unfortunately the reality is that the security of many of these devices leaves a lot to be desired. Many devices are vulnerable to attacks and are never upgraded with security patches. This problem is only going to get worse as more IoT devices are coming to market. Imagine a botnet of a billion devices on the Internet ready for use by unscrupulous actors.

The 'HiveOT Hub' provides capabilities to securely interact with IoT devices and consumers. This includes certificate management, authentication, authorization, provisioning, directory and history services.

HiveOT compatible IoT devices therefore do not need to implement these features. This improves security as IoT devices do not run Web servers and are not directly accessible. They can remain isolated from the wider network and only require an outgoing connection to the Hub. This in turn reduces required device resources such as memory and CPU (and cost). An additional benefit is that consumers receive a consistent user experience independent of the IoT device provider as all interaction takes place via the Hub interface.

HiveOT is based on the [W3C WoT TD 1.1 specification](https://www.w3.org/TR/wot-thing-description11/) for interaction between IoT devices and consumers. It aims to be compatible with this standard. See [docs/README-TD.md] for more information.

Integration with 3rd party IoT devices is supported through the use of IoT protocol bindings. These protocol bindings translate between the 3rd IoT protocol and WoT defined action and event messages.

The Hub supports multiple transport protocols, such as HTTPS/SSE, WebSocket, NATS, and MQTT message bus protocols through embedded servers. (under development)

Last but not least, the 'hive' can be expanded by connecting multiple hubs to each other through a 'bridge'. The bridge lets the Hub owner share select IoT information with other hubs. (future feature)

![Digital Twin Runtime Overview](docs/digitwin-overview.jpg)

## Build From Source

Building the Hub takes place in 2 parts: core and protocol bindings. 

Prerequisites:

1. An x86 or arm based Linux system. Ubuntu, Debian, Raspberrian
2. Golang 1.22 or newer (with GOPATH set)
3. GCC Make any 2020+ version
5. nodejs v18+ (for building zwavejs binding)

### Build Hub Core

1. Download source code:

```sh
git clone git@github.com:hiveot/hub
cd hub
``` 

2. Build the Hub core, Services and Protocol bindings

The quickest way:
> make all

If something goes wrong, build parts separately in order of dependency:
```sh
make runtime
make hubcli
make services
make bindings
```

After the build is successful, the distribution files can be found in the 'dist' folder that can be deployed to the installation directory.

```md
dist / bin - core application binaries (runtime, launcher and hubcli)
dist / plugins - plugin binaries
dist / certs - CA and server certificate; authentication keys and tokens
dist / config - core and plugin configuration files
dist / stores - plugin data storage directory
```

## Install To User (Easiest)

Installing to user is great for running a test setup to develop new bindings or just see how things work. The hub is installed into the user's bin directory and runs under that user's permissions.

For example, for a user named 'hub' the installation home is '/home/hub/bin/hiveot'.

To install to user after a successful build, run
> make install

This copies the distribution files to ~/bin/hiveot. The method can also be used to upgrade an existing installation. Executables are always replaced but only new configuration files are installed. Existing configuration remains untouched to prevent wrecking your working setup.

### Uninstall:

To uninstall simply remove the ~/bin/hiveot folder.

## Install To System (tenative)

While it is a bit early to install hiveot as a system application, this is how it could work:

For systemd installation to run as user 'hiveot'. When changing the user and folders make sure to edit the init/hiveot.service file accordingly. From the dist folder run:

1. Create the folders and install the files

```sh
sudo mkdir -P /opt/hiveot/bin
sudo mkdir -P /opt/hiveot/plugins
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
sudo cp -a plugins/* /opt/hiveot
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

This is considered for the future.

## Configuration

All Hub services will run out of the box with their default configuration. Some services use an optional yaml based configuration file found in the config folder. 

Most important configs:
* launcher.yaml  section 'autostart' lists the services to run at startup

A typical service or protocol binding publishes its configuration options with its TD to allow centralized configuration by managers or administrators. This is up to each service to support.  

### CA certificate

By default, the hub runtime generates a self-signed CA certificate and server certificate if none is found.

To use an externally generated CA certificate, install it into hiveot/certs/caCert.pem and caKey.pem and restart the launcher/runtime.

The CLI can be used to view the currently used CA and server certificate:

```sh
cd ~/bin/hiveot        # when installed locally
bin/hubcli vca    
```

To force generating a new self-signed CA certificate using the CLI:

```sh
cd ~/bin/hiveot        # when installed locally
bin/hubcli cca --force
```

A self-signed CA should be imported in the browser to avoid an error when opening the UI.

### Server certificate

On first use the runtime generates a self-signed server certificate from the CA and installs it in hiveot/certs/hubCert.pem and hiveot/certs/hubKey.pem. This certificate is used with the runtime protocol servers and the hiveoview UI server.

To use a 3rd party server certificate, replace the hubCert.pem and hubKey.pem and restart the runtime. Also replace caCert.pem with the one used to create the server certificate. 

This is shared with Things during provisioning and used by services to secure the server connection.  

Note that use of letsEncrypt is planned for the future which will automate this. This does require an internet connect to work though.


### Systemd Configuration

Automatic startup after boot is supported through a systemd service. This can be used when installed system-wide or as a user.

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
