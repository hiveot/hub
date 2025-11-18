# Building and installing hiveot as user 'hiveot'.

Use this method during development to easily build and upgrade hiveot from source.

Prerequisites:

1. An x86 or arm based Linux system. Ubuntu, Debian, Raspberrian
2. Golang 1.22 or newer (with GOPATH set)
3. GCC Make any 2020+ version

## Step 1: setup the 'hiveot' user

This step is only needed if you build to run on the target device.

1. Create a non-system user 'hiveot' and give it rights to run the sudo command.  
   Note: this works for ubuntu/debian. Any other system your milage may vary.

```sh
sudo useradd -m -G sudo -s /bin/bash hiveot
sudo passwd hiveot
```

Switch to the user (and verify the password)

```sh
su -l hiveot
```

## Step 2: setup environment and tools

To build hiveot golang-v22+, git, make, nvm (node version manager) or node-v22 are needed.
Most debian/ubuntu or other linux systems already have git, make and go installed. Node is more work and easiest done with nvm.

Note that hiveot doesn't need any of these to run. The binaries are self-contained.

### Check Golang version 1.22 or higher

Verify a recent version of golang is installed, eg v1.22 or up

> go version

Installing and upgrading golang is out of scope of these instructions.

### Setup Node v22

Install node version manager:

```sh
curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.39.0/install.sh | bash
```

re-login or source .bashrc to make nvm work

```sh
source ~/.bashrc
```

Install node v22

```sh
nvm install 22
```

Verify

```sh
node --version
```

`
This should show "v22."

## Step 3: download the source and build (running as user hiveot)

Prerequisites: git, make, nvm (node version manager) or node-v22

### Build

```sh
mkdir src && cd src  (or whatever your source directory is)
git clone https://github.com/hiveot/hivehub
cd hub
make all
```

have a coffee... especially on a raspberry pi

If something goes wrong, build parts separately in order of dependency:

```sh
make api
make runtime
make hubcli
make services
make bindings
```

## Fix for raspberry-pi kernel 6.6 hangup

Raspberry pi boots fine but as soon as zwave-js opens the serial port it deadlocks the device.

There is a discussion on this: https://github.com/zwave-js/zwave-js-ui/discussions/3880

Workaround, downgrade the kernel to 6.1-77

> sudo rpi-update af3ab905d42

Potential fix instead of kernel downgrade:
Some people report that configuring the firmware in /boot/firmware/config.txt to say:

> dtoverlay=dwc2,dr_mode=host

would fix it, but this pi3 already had this line. Perhaps it needs to sit above the [cm5] section?

## Install To User (Easiest)

Installing to user is great for running a test setup to develop new bindings or just see how things work. The hub is installed into the user's bin directory and runs under that user's permissions.

For example, for a user named 'hiveot' the installation home is '/home/hiveot/bin/hiveot'.

```sh
make install
ls ~/bin/hiveot
```

Expect: "bin certs config logs plugins run stores"

This copies the distribution files to ~/bin/hiveot. The method can also be used to upgrade an existing installation. Executables are always replaced but only new configuration files are installed. Existing configuration remains untouched to prevent wrecking your working setup.

The ~/bin/hiveot directory can be copied to other systems as long as the architecture (x64, arm) is the same.
Golang, make, node are not needed to run hiveot.

### First Run (setup credentials)

The first run of the runtime creates the credentials in the certs directory. The runtime is typically started by the launcher, if enabled in launcher.yaml.

To change what plugins to run automatically edit config/launcher.yaml and (un)comment the services in the autostart section.

(in future this will be configured through the admin ui)

```sh
cd ~/bin/hiveot
bin/launcher &
```

If all goes well then it should be able to list the running services using hubcli:

```sh
bin/hubcli ls
fg                   # back to the foreground
```

Next shut it down with ctrl-c.

### Launch Using Systemd

Hiveot comes with a default systemd init script that starts the launcher as the hiveot user and group. It can easily be changed to accomodate a different user.

In the src/hub directory:

```sh
sudo cp init/hiveot.service /etc/systemd/system
sudo systemctl daemon-reload
```

6. Test
7.

```sh
sudo service hiveot start
sudo service hiveot status
```

Check it runs:

```sh
bin/hubcli ls
```

To enable startup on boot

```sh
sudo systemctl enable hiveot.service
```

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
sudo vi /etc/hiveot/hivehub.yaml    - and edit the config, log, plugin folders
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

# Distributed System Installation (manually)

Hiveot is designed so its services and bindings can run on multiple devices in a distributed configuration.

By installing the hiveot launcher on multiple devices, it can manage running services on these devices.
This is currently a manual process.

## Future

The (future) intent is to be able to use the launcher on every device using the same init script and simply enable the service to run remotely.
The launcher credentials should be copied from the primary device along with the (public) CA certificate.

How this could work (todo):

- each launcher has a unique ID 'launcher-{hostname}'.
- launchers are aware of each other by reading the directory and looking for launcher services.
- a launcher can list, start and stop services managed by other launchers
- a service can be run on any device that has a launcher running
- administrators can configure services to run on any launcher managed device (of the same architecture)
- support for failover if a device becomes unreachable

This is a simplistic version of a distributed system for IoT.

Manual setup rough description below.

## Runtime on device-1 and service {serviceName} on device-2.

First install the runtime as per above. Start the runtime and check its running.

> bin/hubcli ls

This has created a self-signed CA certificate that is needed by services to run on another device.

1. If the computer architecture on both devices is the same then simply copy the bin/hiveot directory to the second device.
   > scp bin/hiveot device-2:bin
2. Edit config/launcher.yaml and
   - disable launching the runtime
   - enable the service to run on device 2 in the autostart section.
3. Get a service token on device 1 and install on device-2
   > bin/hubcli addsvc {serviceName}
   > scp certs/{serviceName}.token device-2:bin/hiveot/certs
4. Run the launcher on device-2
   > bin/launcher
