module github.com/hiveot/hub

go 1.20

require (
	github.com/alexedwards/argon2id v0.0.0-20230305115115-4b3c3280a736
	github.com/eclipse/paho.golang v0.11.0
	github.com/fsnotify/fsnotify v1.6.0
	github.com/golang-jwt/jwt/v5 v5.0.0
	github.com/gorilla/mux v1.8.0
	github.com/grandcat/zeroconf v1.0.0
	github.com/lmittmann/tint v0.3.4
	github.com/mochi-mqtt/server/v2 v2.3.0
	github.com/nats-io/jwt/v2 v2.5.0
	github.com/nats-io/nats-server/v2 v2.9.22
	github.com/nats-io/nats.go v1.28.0
	github.com/nats-io/nkeys v0.4.4
	github.com/rs/cors v1.9.0
	github.com/samber/lo v1.38.1
	github.com/stretchr/testify v1.8.4
	github.com/struCoder/pidusage v0.2.1
	github.com/urfave/cli/v2 v2.25.7
	golang.org/x/crypto v0.13.0
	golang.org/x/exp v0.0.0-20230905200255-921286631fa9
	golang.org/x/net v0.15.0
	gopkg.in/square/go-jose.v2 v2.6.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/cenkalti/backoff v2.2.1+incompatible // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.2 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/gorilla/websocket v1.5.0 // indirect
	github.com/klauspost/compress v1.16.7 // indirect
	github.com/kr/pretty v0.1.0 // indirect
	github.com/mattn/go-colorable v0.1.12 // indirect
	github.com/mattn/go-isatty v0.0.14 // indirect
	github.com/miekg/dns v1.1.55 // indirect
	github.com/minio/highwayhash v1.0.2 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/rs/xid v1.4.0 // indirect
	github.com/rs/zerolog v1.28.0 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/xrash/smetrics v0.0.0-20201216005158-039620a65673 // indirect
	golang.org/x/mod v0.12.0 // indirect
	golang.org/x/sync v0.3.0 // indirect
	golang.org/x/sys v0.12.0 // indirect
	golang.org/x/time v0.3.0 // indirect
	golang.org/x/tools v0.13.0 // indirect
	gopkg.in/check.v1 v1.0.0-20180628173108-788fd7840127 // indirect
)

replace github.com/nats-io/nats-server/v2 => ../nats-server

replace github.com/nats-io/nats.go => ../nats.go
