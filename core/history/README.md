# Thing Value History 

## Objective

Provide historical reading of thing events and actions using the bucket store.


## Status

This service is functional.

To be completed:
* Add pubsub subscription for events and actions to update the history through pubsub.
* Configuration of event retention.
* Configuration to determine which bucket store to use. (currently Pebble is hard coded)


## Summary

The History service provides capture and retrieval of past events and actions. 

Data ingress at a continuous rate of 1000 events per minute is readily supported on small to medium sized systems with 500MB of RAM or more and plenty of disk space. More sensors can be supported if storage space, memory and CPU are available. For smaller environments this can be dialed down further with some experimentation.

Basic data queryies are provided through the API for the purpose to display and compare historical information.

Data retention rules must be set to start storing events. To run out of the box, a default rule set is included with the history.yaml configuration file. The rules can be modified through the retention API. Changes to the configuration file require a restart of the service.

**Limitations:**

* The history store is designed to use the bucket store and is thus limited by the storage capabilities and query capabilities of the bucket store API.

* Complex analytics of historical information is out of scope for the history store. The recommended approach is to periodically export a batch of new data to a time series database such as Prometheus, InfluxDB, or others that support clustering and can be integrated with 3rd party analytics engines. This can best be accomplished by adding an export service.

* There are no tools yet to monitor the bucket store health, and track its resource usage.


## Backend Storage

This service uses the bucketstore for the storage backend. The bucketstore supports several embedded backend implementations that run out of the box without the need for any setup and configuration.

Extending the bucket store with external databases such as Mongodb, SQLite, PostgresSQL and possibly others is under consideration.

The bucketstore API provides a cursor with key-ranged seek capability which can be used for time-based queries. All bucket store implementations support this range query through cursors. 

Currently, the Pebble bucket store is the default for the history store. It provides a good balance between storage size and resource usage for smaller systems. Pebble should be able to handle a TB of data or even more.

More testing is needed to determine the actual limitations.

### Data Size

Data size of event samples depends strongly on the type of sensor, actuator or service that captures the data. Below some example cases and the estimated memory to get an idea of the required space.

Since the store uses a bucket per thingID, the thingID itself does not add significant size. The key is the msec timestamp since epoc, approx 15 characters.

The following estimates are based on a sample size of 100 bytes uncompressed (key:20, event name:10, value: 60, json: 10). These are worst case numbers as deduplication and compression can reduce the sizes significantly.

Case 1: sensors with a 1 minute average sample interval. 

* A single sensor -> 500K samples => 50MB/year (uncompressed)
* A small system with 10 sensors -> 5M samples => 500MB/year
* A medium system with 100 sensors -> 50M samples => 5GB/year
* A larger system with 1000 sensors -> 500M samples => 50GB/year

Case 2: sensors with a 1 second average sample interval.
* A single sensor -> 32M samples => 3.2GB/year (uncompressed)
* A small system with 10 sensors -> 320M samples => 32GB/year
* A larger system with 1000 sensors -> 32000 M samples => 3.2TB/year

In reality these numbers will be lower depending on the chosen store.

Case 3: image timelapse snapshot with 5 minute interval
An image is 720i compressed, around 100K/image. 

* A single image -> 100K snapshots/year => 10 GB/year
* A system with 10 cameras -> 1000K snapshots/year => 100 GB/year
* A larger system with 100 cameras -> 10M snapshots/year => 1 TB/year
