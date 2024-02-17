module github.com/hiveot/hub

go 1.21

toolchain go1.21.1

require (
	github.com/alexedwards/argon2id v1.0.0
	github.com/araddon/dateparse v0.0.0-20210429162001-6b43995a97de
	github.com/cockroachdb/pebble v0.0.0-20231007004400-803507a71849
	github.com/eclipse/paho.golang v0.12.0
	github.com/fsnotify/fsnotify v1.7.0
	github.com/go-chi/chi/v5 v5.0.11
	github.com/golang-jwt/jwt/v5 v5.2.0
	github.com/google/uuid v1.5.0
	github.com/gorilla/mux v1.8.1
	github.com/grandcat/zeroconf v1.0.0
	github.com/lmittmann/tint v1.0.3
	github.com/mochi-mqtt/server/v2 v2.4.4
	github.com/mostlygeek/arp v0.0.0-20170424181311-541a2129847a
	github.com/nats-io/jwt/v2 v2.5.3
	github.com/nats-io/nats-server/v2 v2.10.7
	github.com/nats-io/nats.go v1.31.0
	github.com/nats-io/nkeys v0.4.6
	github.com/pkg/errors v0.9.1
	github.com/rs/cors v1.10.1
	github.com/samber/lo v1.39.0
	github.com/stretchr/testify v1.8.4
	github.com/struCoder/pidusage v0.2.1
	github.com/thanhpk/randstr v1.0.6
	github.com/tidwall/btree v1.7.0
	github.com/urfave/cli/v2 v2.26.0
	go.etcd.io/bbolt v1.3.8
	golang.org/x/crypto v0.17.0
	golang.org/x/exp v0.0.0-20231219180239-dc181d75b848
	golang.org/x/net v0.19.0
	golang.org/x/sys v0.15.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/DataDog/zstd v1.5.5 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cenkalti/backoff v2.2.1+incompatible // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/cockroachdb/errors v1.11.1 // indirect
	github.com/cockroachdb/logtags v0.0.0-20230118201751-21c54148d20b // indirect
	github.com/cockroachdb/redact v1.1.5 // indirect
	github.com/cockroachdb/tokenbucket v0.0.0-20230807174530-cc333fc44b06 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.3 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/getsentry/sentry-go v0.25.0 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/google/go-cmp v0.6.0 // indirect
	github.com/gorilla/websocket v1.5.1 // indirect
	github.com/huin/goupnp v1.3.0 // indirect
	github.com/klauspost/compress v1.17.4 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/matttproud/golang_protobuf_extensions/v2 v2.0.0 // indirect
	github.com/miekg/dns v1.1.57 // indirect
	github.com/minio/highwayhash v1.0.2 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/prometheus/client_golang v1.17.0 // indirect
	github.com/prometheus/client_model v0.5.0 // indirect
	github.com/prometheus/common v0.45.0 // indirect
	github.com/prometheus/procfs v0.12.0 // indirect
	github.com/rogpeppe/go-internal v1.12.0 // indirect
	github.com/rs/xid v1.5.0 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/xrash/smetrics v0.0.0-20231213231151-1d8dd44e695e // indirect
	golang.org/x/mod v0.14.0 // indirect
	golang.org/x/sync v0.6.0 // indirect
	golang.org/x/text v0.14.0 // indirect
	golang.org/x/time v0.5.0 // indirect
	golang.org/x/tools v0.16.1 // indirect
	google.golang.org/protobuf v1.32.0 // indirect
)

replace github.com/nats-io/nats-server/v2 => ../nats-server

replace github.com/nats-io/nats.go => ../nats.go
