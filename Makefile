# Makefile to build and test the HiveOT Hub
DIST_FOLDER=./dist
BIN_FOLDER=./dist/bin
PLUGINS_FOLDER=./dist/bin/plugins
INSTALL_HOME=~/bin/hiveot
.DEFAULT_GOAL := help

.FORCE: 

all: core plugins hubcli  ## Build APIs, CLI, Hub services

core: natscore

natscore: ## Build the hub core binary using nats messaging
	go build -o $(DIST_FOLDER)/bin/$@ core/cmd/$@/main.go

plugins: certs directory launcher owserver provisioning zwavejs

certs: .FORCE ## Build the certificate management service
	go build -o $(PLUGINS_FOLDER)/$@ $@/cmd/main.go

directory: .FORCE ## Build the directory service
	go build -o $(PLUGINS_FOLDER)/$@ $@/cmd/main.go

hubcli: .FORCE ## Build Hub CLI
	go build -o $(BIN_FOLDER)/$@ cmd/$@/main.go

launcher: .FORCE ## Build the hub plugin launcher
	go build -o $(BIN_FOLDER)/$@ plugins/$@/cmd/main.go
	cp plugins/$@/config/*.yaml $(DIST_FOLDER)/config

owserver: .FORCE ## Build the Thing directory store
	go build -o $(PLUGINS_FOLDER)/$@  plugins/$@/cmd/main.go

provisioning: .FORCE ## Build Hub provisioning service
	go build -o $(PLUGINS_FOLDER)/$@ plugins/$@/cmd/main.go

zwavejs: .FORCE ## Build Hub provisioning service
	go build -o $(PLUGINS_FOLDER)/$@ plugins/$@/cmd/main.go


clean: ## Clean distribution files
	go clean -cache -testcache -modcache
	rm -rf $(DIST_FOLDER)
	mkdir -p $(BIN_FOLDER)
	mkdir -p $(BIN_FOLDER)/plugins
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
	mkdir -p $(INSTALL_HOME)/bin/plugins
	mkdir -p $(INSTALL_HOME)/certs
	mkdir -p $(INSTALL_HOME)/config
	mkdir -p $(INSTALL_HOME)/logs
	mkdir -p $(INSTALL_HOME)/stores
	mkdir -p $(INSTALL_HOME)/run
	cp -ar $(BIN_FOLDER)/* $(INSTALL_HOME)/bin
	cp -n $(DIST_FOLDER)/config/*.yaml $(INSTALL_HOME)/config/

test: hub  ## Run tests (stop on first error, don't run parallel)
	go test -race -failfast -p 1 ./...

upgrade:
	go get -u all
	go mod tidy
