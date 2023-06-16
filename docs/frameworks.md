# HiveOT Streams 

Requirements
- WoT compatible, event and action based
- lightweight; runs on embedded hardware from raspberry pi to cloud
- secure
  - authentication of sources and sinks 
  - data source identification (where does the data come from)
  - authorization - group role based data access
- integration
  - cloud data distribution - hook up sources and sink from all over
  - easy custom sources and sinks
- event processors
  - filtering
  - routing
  - alerting
- storage
- media support


Topic candidates:
1. ekuiper - choose when needing processor rules and storage
2. nats jetstream - choose for its ecosystem
3. benthos - 
4. mqtt - lightweight
  

# Existing Golang Frameworks

Pros go above the requirements, cons do not meet the requirements.

## ekuiper - https://github.com/lf-edge/ekuiper
* 1K github stars; 4 active contributors; backed by EMQ??
* active since 2020
* focus on edge
* MQTT throughput 12K messages/sec on raspberri pi
* well documented
* fairly new

Pros:
+ processors/etl
+ rules engine - rules pipeline 
+ storage support
+ tensorflow lite integration (gRPC based)
+ emqx for cloud hive 

Cons:
! no source authentication
! no authorization
- requires mqtt as data source??
- no media streaming
- no event storage & query 

## benthos - https://github.com/benthosdev/benthos

summary: probably overkill for hiveot but allows for growth

* 6K github stars; 2 active contributors
* active since 2016, ramping up in 2018
* highly configurable
* focus on cloud
* very well documented

Pros:
+ massive number of cloud sources and sinks; extensible
+ lots of processors
+ storage support built-in
+ metrics support and cloud integration
+ storage: sql_insert

Cons:
! no source authentication 
! no processing authorization
- lots of configuration
- its all about the cloud, 
- no media streaming

## watermill - 

* event driven framework

Pros:
+ lightweight
+ resiliency support: retrying, throttling
+ metrics support
+ router ack support (delivery confirmation)
+ pubsub integration: kafka, nats, google cloud, amqp, sql

Cons:
- no authorization


## Nats Jetstream
Message oriented middleware

* pubsub messaging streaming server (subjects == topics)
* based on services and streams
* clients in many many languages
* durable subscriptions
* payload agnostic (json, bson, etc)
* well documented; videos
* looks complex

Pros:
+ authorization support; dynamic/topic base permissions using signing keys; subject namespace; 
+ storage with replay; retention policies
+ filtering on topics
+ observability/metrics

Con:
- no mqtt5 support
- no processors
- filtering on topics, not message content 
- designed for scaling in the cloud

## EMQX open source
Mqtt broker

* build to scale (not relevant here)
Pros:
+ rule engine

Con:
- no persistence
- not for edge (unlike enterprise)