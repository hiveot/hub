# Hive-Of-Things Hub

HiveOT stands for the "Hive of Things". It consists of one or more 'Hubs' to collect and share IoT data with its users.

The Hub for the *Hive-of-Things* provides a secure [runtime](runtime/README-runtime.md) to view and control IoT devices. The Hub securely mediates between IoT device 'Things', services, and users using a hub-and-spokes architecture. Users interact with Things via the Hub's digital twin, without connecting directly to the IoT devices or services. The Hub is based on the [W3C WoT TD 1.1 specification](https://www.w3.org/TR/wot-thing-description11/). Multiple communication protocols are supported for IoT devices and users.

![System Overview](docs/hub-overview.jpg)

## Project Status

Status July 2025: The Hub runtime, services and bindings have been reworked to support the Web-of-things (WoT) Thing Description (TD) specification. It is currently in Alpha.  

Medium term roadmap:
1. Launcher support for distributed environment [todo]
1. Support lets-encrypt CA and server certificate [todo]
2. Support mqtt transport protocol. [tbd as client or server?]
1. Websockets sub-protocol binding [functional but spec is in development]
1. Support for TD Forms sections. [contentious, as it doesn't serve a purpose in this setup]
1. Support for WoT discovery profile [done]
1. Revisit the vocabulary to integrate or adopt existing vocabularies where possible
1. improve/fix security;
   2. detect/notify of bad agents or consumers
   3. Manage token expiry
   3. role based access to Things
   4. rate limiting
2. hiveoview dashboard improvements
   3. notifications
3. Various services and bindings
   3. weather service integration: open-meteo [in progress]
   3. weather service integration: environment canada  (better forecast)
   2. ups binding (using [nut](https://networkupstools.org/))
4. Android integration/location tracking

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

## Build and Install

See [docs/INSTALL.md](docs/INSTALL.md)

## Configuration

All Hub services will run out of the box with their default configuration. Some services use an optional yaml based configuration file found in the config folder. 

Most important configs:
* launcher.yaml  section 'autostart' lists the services to run at startup

A typical service or protocol binding publishes its configuration options with its TD to allow centralized configuration by administrators. This is up to each service to support.  

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


# Contributing

Contributions to HiveOT projects are always welcome. There are many areas where help is needed, especially with documentation, testing and building bindings for IoT and other devices. See [CONTRIBUTING](CONTRIBUTING.md) for guidelines.

# Credits

This project builds on the Web of Things (WoT) standardization by the W3C.org standards organization. For more information https://www.w3.org/WoT/

This project is inspired by the Mozilla Thing draft API [published here](https://iot.mozilla.org/wot/#web-thing-description). However, the Mozilla API is intended to be implemented by Things and is not intended for Things to register themselves. The HiveOT Hub will therefore deviate where necessary.

Many thanks go to JetBrains for sponsoring the HiveOT open source project with development tools.  
