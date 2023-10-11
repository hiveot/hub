# ISY99x publisher

ISY99 is a gateway for the Insteon protocol. It is end of life but they are still around. It is replaced by the ISY944 which also works with this publisher, albeit limitated to what the ISY99 can do.

## Dependencies

This publisher does not have any further dependencies, other than listed in the go-iotdomain README.md


## Configuration

Edit isy99.yaml with the ISY99 gateway address and login name/password. The gateway can also be configured through the gateway node 'gatewayAddress' configuration.

See config files in ./test as examples

## Usage

Configure the publisher as described above and run it as described in the iotdomain-go library.
This will automatically discover insteon devices on the gateway, publish their discover and their current value. Switches can be controlled with a $set command.
