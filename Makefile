# Makefile to build and test the HiveOT Hub
DIST_FOLDER=./dist
BIN_FOLDER=./dist/bin
PLUGINS_FOLDER=./dist/plugins
INSTALL_HOME=~/bin/hiveot
GENAPI=go run cmd/tdd2api/main.go
.DEFAULT_GOAL := help

.FORCE: 

all: runtime services bindings hubcli   ## Build Core, Bindings and hubcli

# --- Runtime services

runtime: .FORCE
	$(GENAPI) ggo runtime/authn/tdd
	$(GENAPI) ggo runtime/authz/tdd
	$(GENAPI) ggo runtime/digitwin/tdd
	go build -o $(BIN_FOLDER)/$@ runtime/cmd/main.go

services: launcher state idprov certs history

launcher: .FORCE
	go build -o $(BIN_FOLDER)/$@ services/$@/cmd/main.go
	mkdir -p $(DIST_FOLDER)/config
	cp services/$@/config/*.yaml $(DIST_FOLDER)/config

certs: .FORCE
	go build -o $(PLUGINS_FOLDER)/$@ services/$@/cmd/main.go

history: .FORCE
	go build -o $(PLUGINS_FOLDER)/$@ services/$@/cmd/main.go

idprov: .FORCE
	go build -o $(PLUGINS_FOLDER)/$@ services/$@/cmd/main.go

state: .FORCE
	go build -o $(PLUGINS_FOLDER)/$@ services/$@/cmd/main.go

# --- protocol bindings

bindings:  hiveoview ipnet isy99x owserver zwavejs   ## Build the protocol bindings

hiveoview: .FORCE ## build the SSR web viewer binding
	go build -o $(PLUGINS_FOLDER)/$@  bindings/$@/cmd/main.go

ipnet: .FORCE ## Build the ip network scanner protocol binding
	go build -o $(PLUGINS_FOLDER)/$@  bindings/$@/cmd/main.go
	cp bindings/$@/config/*.yaml $(DIST_FOLDER)/config

isy99x: .FORCE ## Build the ISY99x INSTEON protocol binding
	go build -o $(PLUGINS_FOLDER)/$@  bindings/$@/cmd/main.go
	cp bindings/$@/config/*.yaml $(DIST_FOLDER)/config

owserver: .FORCE ## Build the 1-wire owserver protocol binding
	go build -o $(PLUGINS_FOLDER)/$@  bindings/$@/cmd/main.go
	cp bindings/$@/config/*.yaml $(DIST_FOLDER)/config

zwavejs: .FORCE ## Build the zwave-js protocol binding
	cd bindings/$@ && make dist
	cp bindings/$@/dist/$@ $(PLUGINS_FOLDER)


# --- user interfaces

hubcli: .FORCE ## Build Hub CLI
	go build -o $(BIN_FOLDER)/$@ cmd/$@/main.go



clean: ## Clean distribution files
	go clean -cache -testcache -modcache
	cd bindings/zwavejs && make clean
	rm -rf $(DIST_FOLDER)
	mkdir -p $(BIN_FOLDER)
	mkdir -p $(DIST_FOLDER)/plugins
	mkdir -p $(DIST_FOLDER)/certs
	mkdir -p $(DIST_FOLDER)/config
	mkdir -p $(DIST_FOLDER)/logs
	mkdir -p $(DIST_FOLDER)/run
	go mod tidy
	go get all

help: ## Show this help
	@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'


install:  ## core plugins ## build and install the services
	mkdir -p $(INSTALL_HOME)/bin
	mkdir -p $(INSTALL_HOME)/plugins
	mkdir -p $(INSTALL_HOME)/certs
	mkdir -p $(INSTALL_HOME)/config
	mkdir -p $(INSTALL_HOME)/logs
	mkdir -p $(INSTALL_HOME)/stores
	mkdir -p $(INSTALL_HOME)/run
	cp -af $(BIN_FOLDER)/* $(INSTALL_HOME)/bin
	cp -af $(PLUGINS_FOLDER)/* $(INSTALL_HOME)/plugins
	cp -n $(DIST_FOLDER)/config/*.yaml $(INSTALL_HOME)/config/

test: runtime services bindings  ## Run tests (stop on first error, don't run parallel)
	go test -race -failfast -p 1 ./...

upgrade:
	go get -u all
	go mod tidy
