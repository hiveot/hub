module github.com/hiveot/hub

go 1.24.0

// can't use go.work. See https://github.com/golang/go/issues/50750
replace github.com/hiveot/hivekit/go => ../hivekit/go

require (
	aidanwoods.dev/go-paseto v1.6.0
	github.com/alexedwards/argon2id v1.0.0
	github.com/araddon/dateparse v0.0.0-20210429162001-6b43995a97de
	github.com/cockroachdb/pebble v1.1.5
	github.com/dchest/uniuri v1.2.0
	github.com/eclipse/paho.golang v0.23.0
	github.com/fsnotify/fsnotify v1.9.0
	github.com/go-chi/chi/v5 v5.2.5
	github.com/golang-jwt/jwt/v5 v5.3.1
	github.com/gorilla/websocket v1.5.3
	github.com/grandcat/zeroconf v1.0.1-0.20230119201135-e4f60f8407b1
	github.com/hiveot/hivekit/go v0.0.0-20260205212135-65c0126cb783
	github.com/huin/goupnp v1.3.0
	github.com/json-iterator/go v1.1.12
	github.com/lmittmann/tint v1.1.3
	github.com/mostlygeek/arp v0.0.0-20170424181311-541a2129847a
	github.com/rs/cors v1.11.1
	github.com/samber/lo v1.52.0
	github.com/stretchr/testify v1.11.1
	github.com/struCoder/pidusage v0.2.1
	github.com/teris-io/shortid v0.0.0-20220617161101-71ec9f2aa569
	github.com/thanhpk/randstr v1.0.6
	github.com/tidwall/btree v1.8.1
	github.com/tmaxmax/go-sse v0.11.0
	github.com/urfave/cli/v2 v2.27.7
	golang.org/x/crypto v0.47.0
	golang.org/x/exp v0.0.0-20260112195511-716be5621a96
	golang.org/x/net v0.49.0
	golang.org/x/sys v0.40.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	aidanwoods.dev/go-result v0.3.1 // indirect
	github.com/DataDog/zstd v1.5.7 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cenkalti/backoff v2.2.1+incompatible // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/cockroachdb/errors v1.12.0 // indirect
	github.com/cockroachdb/fifo v0.0.0-20240816210425-c5d0cb0b6fc0 // indirect
	github.com/cockroachdb/logtags v0.0.0-20241215232642-bb51bb14a506 // indirect
	github.com/cockroachdb/redact v1.1.6 // indirect
	github.com/cockroachdb/tokenbucket v0.0.0-20250429170803-42689b6311bb // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.7 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/getsentry/sentry-go v0.42.0 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/snappy v1.0.0 // indirect
	github.com/klauspost/compress v1.18.3 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/miekg/dns v1.1.72 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/prometheus/client_golang v1.23.2 // indirect
	github.com/prometheus/client_model v0.6.2 // indirect
	github.com/prometheus/common v0.67.5 // indirect
	github.com/prometheus/procfs v0.19.2 // indirect
	github.com/rogpeppe/go-internal v1.14.1 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/xrash/smetrics v0.0.0-20250705151800-55b8f293f342 // indirect
	go.yaml.in/yaml/v2 v2.4.3 // indirect
	golang.org/x/mod v0.32.0 // indirect
	golang.org/x/sync v0.19.0 // indirect
	golang.org/x/text v0.33.0 // indirect
	golang.org/x/tools v0.41.0 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)
