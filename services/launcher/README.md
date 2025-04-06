# Launcher

## Status

This service is functional and can start and stop plugins.

## Objectives

Manage running of Hub plugins and monitor their status. 

## Features

1. Start hub runtime if configured
1. List, Start and Stop available plugins 
1. Generate a client auth certificate for plugins before start
1. Set logging output to log files for each plugin
1. Plugin autostart on startup. Use launcher.yaml config file.
2. Send events when plugins are started and stopped

Future:
1. Support multiple concurrent launchers 
1. Track launched services by PID 
1. Detect services that were manually started
1. Track memory and CPU usage.
1. Auto restart service if exit with error
1. Restart service if resources (CPU, Memory) exceed configured thresholds
1. Send event when resource usage exceeds limits
2. TBD: support distributed services with launchers running on remote systems 
1. TBD: Share status with instances running on distributed computers
2. TBD: Transfer eligible services to best available runtimes 

## Summary

The launcher is a service for starting, stopping and monitoring Hub plugins follow this workflow:
```
1. Launcher starts the hub runtime as per config
   * launcher clientID is that of the binary name 
   * hub runtime creates the admin and launcher key-token credentials if they don't exist
2. Launcher connects to the hub with launcher key/token credentials
   * Connect to the auth clients service to be able to create plugin key/tokens 
3. Launcher starts each plugin in order of config
   * Create the plugin key/token files if they don't exist
   * Start the plugin
      * plugin loads key and token from files
      * plugin connects to hub runtime
      * plugin registers authz for roles allowed to use it
      * plugin waits for SIGTERM signal to stop
   * publish event
```

### Auto Restart (future)
If a plugin stops unintentionally it is automatically restarted by the launcher. If restart fails, a backoff time delays the attempt to start again. This backoff time is slowly increased until a maximum of 1 hour.

### Stopping a Plugin
To stop a plugin the launcher simply terminates the process the plugin runs in, and disables its enabled status.


### Resource monitoring (future)
While running, the launcher keeps track of the CPU and memory usage of the service. This is available upon request.

### Limitations

* The launcher will not recognize plugins started on their own. Plugins might not function properly when started twice.


## Launcher Configuration

The launcher uses the launcher.yaml configuration file for determining which plugins to start. See the included sample file for details.

Configuration worth mentioning here are:
1. runtime, specifies the runtime to launch on startup before any plugins. This is enabled by default. 
1. createPluginCred, default True, creates an authentication token file before starting the plugin. 
2. logLevel, default 'warn', change the logging level for launcher.
3. logPlugins, default True, create a logfile for each plugin in the logs directory
4. provideServerURL, provide the plugins with the server URL --serverURL commandline, using the same URL as the launcher uses. This is only intended for agents that use a protocol that doesn't require discovery.

## Multiple Devices

In a multi-device setup, each device has a launcher that can start and stop services on that device. This introduces several challenges:
1. How to differentiate the launchers themselves
2. How to differentiate duplicate services - service type vs 'id' vs instance
   * auth with service ID
   * group using service type: sensors, actuators 
   * use instance to locate/address the service
   * 
   * use @type to indicate service type: launcher, hub, authn, authz, state, zwave, insteon
   *   or service vs device?
   *   tie into the vocabulary
3. Should launcher pass 'their' server URL to the services they launch? - eg no disco - yes
   * launcher controls which server to use
   * disco is only needed by the launcher
   * if service doesn't support protocol then it does its own thing
4. Should multiple instances be able to use the same auth?  - yes 

Option 1: add an instance ID to service ID. Support multiple service instances using their instance ID - eg a connectionID. 
Option 2: use different plugin names, eg launcher-{hostname}
Option 3: merge with the runtime and each device has its own runtime forming a hiveot mesh
	? should each device have a runtime? - no

Issue: where to store data
Issue: how to migrating a service from one host to another?
Issue: how to present (hiveoview) multiple devices



## Usage

After installation, the admin user can start the launcher manually or have it start automatically using systemd and the hiveot.service file.

See "launcher -h" for commandline options

