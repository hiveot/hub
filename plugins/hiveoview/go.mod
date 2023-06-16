module github.com/hiveot/bindings/hiveoview

go 1.19

require (
	capnproto.org/go/capnp/v3 v3.0.0-alpha.25
	github.com/hiveot/hub v0.0.0-20230207180321-215941d87c3c
	github.com/sirupsen/logrus v1.9.0
)

require (
	github.com/cenkalti/backoff v2.2.1+incompatible // indirect
	github.com/gobwas/httphead v0.1.0 // indirect
	github.com/gobwas/pool v0.2.1 // indirect
	github.com/gobwas/ws v1.2.0 // indirect
	github.com/grandcat/zeroconf v1.0.0 // indirect
	github.com/miekg/dns v1.1.52 // indirect
	golang.org/x/mod v0.9.0 // indirect
	golang.org/x/net v0.9.0 // indirect
	golang.org/x/sync v0.1.0 // indirect
	golang.org/x/sys v0.7.0 // indirect
	golang.org/x/tools v0.7.0 // indirect
	zenhack.net/go/util v0.0.0-20230305222753-27582c3a8381 // indirect
	zenhack.net/go/websocket-capnp v0.0.0-20230212023810-f179b8b2c72b // indirect
)

replace github.com/hiveot/hub => ../../hub

//replace zenhack.net/go/websocket-capnp => ../../../go-websocket-capnp
