# Launcher

## Status

This service is functional and can launch plugins.

## Objectives

Manage running of Hub plugins and monitor their status. 

## Features

1. Start runtime core if configured
1. List, Start and Stop available plugins including the message bus core
1. Generate a client auth certificate for plugins before start
1. Set logging output to log files for each plugin
1. Plugin autostart on startup. Use launcher.yaml config file.

Future:
1. Track launched services by PID 
1. Detect services that were manually started
1. Track memory and CPU usage.
1. Auto restart service if exit with error
1. Restart service if resources (CPU, Memory) exceed configured thresholds
1. Send event when services are started and stopped
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

### Auto Restart
If a plugin stops unintentionally it is automatically restarted by the launcher. If restart fails, a backoff time delays the attempt to start again. This backoff time is slowly increased until a maximum of 1 hour.

### Stopping a Plugin
To stop a plugin the launcher simply terminates the process the plugin runs in, and disables its enabled status.


### Resource monitoring
While running, the launcher keeps track of the CPU and memory usage of the service. This is available upon request.

### Limitations

* The launcher will not recognize plugins started on their own. Plugins might not function properly when started twice.


## Launcher Configuration

The launcher uses the launcher.yaml configuration file for determining which plugins to start. See the included sample file for details.


## Usage

After installation, the admin user can start the launcher manually or have it start from systemd using the hivehub.service file.

See "launcher -h" for commandline options

In test situations it might be necessary to start and stop services manually without the launcher. In this case make sure they are stopped in the launcher first using the hubcli.