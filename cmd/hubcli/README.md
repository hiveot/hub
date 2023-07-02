# Hub CLI

## Status

In development. Migrating to a messaging based Hub.

## Summary

The Hub CLI provides a commandline interface for managing Hub and service configuration.

Usage: hubcli -h


## Core Commands

The first thing to do after installation is to initialize the hub configuration.  

> hubcli init --home ~/bin/hiveot

Where ~/bin/hiveot is the root installation folder.

This generates any missing configuration needed to run the Hub. This command is idempotent. If the files already exist they will remain unchanged.



