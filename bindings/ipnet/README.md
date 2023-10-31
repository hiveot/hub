# IPNet protocol binding 

The IPNet binding discovers devices on the network and tracks their uptime.

## Status

In development. Migration from an older version.
This initially only finds IP and MAC addresses of devices on the network.

Limitations:
* TODO: remote configuration
* TODO: port scan configuration
* TODO: SNMP scanning
* TODO: use of arp to detect devices on the fly 
* TODO: use dns-sd to detect devices on the fly

## Configuration

See ipnet.yaml for the configuration and the discovery output

IPNet scans the subnets of the local interfaces that are not loopback and publishes discovered devices.

Devices are identified by their MAC address, or by their IP address if they are not on the local subnet. MAC addresses
are not visible outside the local subnet as routers block this information.

The configuration also enables providing the name, make and model of known devices for easy identification.
Non-local devices can be added to the configuration by their IP address. This will be scanned after the local subnet 
is scanned.

Port scan is optional and is disabled by default. This is slower but provides more information on the open ports of the
device.

SNMP scan is optional and is disabled by default. Planned to configure this per device.

ARP scan is optional and disabled by default.

## Published data

Example, the discovery of raspberry pi on 10.3.3.20 with MAC AA:BB:CC:DD:EE:FF
> On topic discovery/inet/[id]
> {
>    id: "AA:BB:CC:DD:EE:FF",                       Device's ID under which it is published, MAC or IP
>    mac: "AA:BB:CC:DD:EE:FF",                      Device's network MAC address, if on local subnet
>    ip4: "10.3.3.20",                              IPv4 network address if available
>    ip6: "",                                       IPv6 network address if available
>    latency: 0.004000,                             Network latency in seconds
>    firstSeen: "2018-01-01T00:00:00-07:00",        ISO time with local timezone
>    lastSeen: "2018-01-01T00:00:00-07:00",         ISO time with local timezone
>    name: "",                                      Name for this device if configured below
>    make: "",                                      Make for this device if configured below
>    model: "",                                     Model for this device if configured below
>    ports: [{port: 123, name: "servicename", protocol: "tcp"}, ...]  Services discovered when 'scanports' is true
> }

## Dependencies

Application dependencies
* nmap

Go package dependencies
* github.com/mostlygeek/arp
* github.com/lair-framework/go-nmap
