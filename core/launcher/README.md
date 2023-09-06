# Launcher

## Objectives

Manage Hub core and plugin services and monitor their status. 

## Features

1. List, Start and Stop available services
1. Set logging output to log files for each plugin
1. Support plugin autostart on startup. Use launcher.yaml config file.

Planned:
1. Track memory and CPU usage.
1. Auto restart service if exit with error
1. Restart service if resources (CPU, Memory) exceed configured thresholds
1. Send event when services are started and stopped
1. Send event when resource usage exceeds limits
 

## Summary

The launcher is used to start, stop and monitor the core and plugin services.

When starting a service, it is launched as a new process. Services terminate on the SIGTERM signal.

If a service stops unintentionally it is automatically restarted. If restart fails, a backoff time delays the attempt to start again. This backoff time is slowly increased until a maximum of 1 hour.

To stop a service the launcher simply terminates the process the service runs into and disables its enabled status.

While running, the launcher keeps track of the CPU and memory usage of the service. This is available upon request.

**Limitations:**

* The launcher will not recognize services started on their own. Services will not function properly when started twice.


## Launcher Configuration

The launcher uses the following configuration for launching services:
```
{app}/config/launcher.yaml  contains the launcher settings, including the folders to use.
```

See the example file for details.
