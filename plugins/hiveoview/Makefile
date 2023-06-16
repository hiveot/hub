# Makefile to build the zwave protocol binding
DIST_FOLDER=./dist
BUILD_FOLDER=./build
INSTALL_HOME=~/bin/hiveot
BINDINGS_HOME=$(INSTALL_HOME)/bin/bindings
.DEFAULT_GOAL := help

.FORCE:

all: dist

js: .FORCE  ## Compile typescript
	tsc

wasm: ## Compile go to wasm
	GOOS=js GOARCH=wasm go build -o ${BUILD_FOLDER}/hapi.wasm cmd/hapi/main.go

dist: wasm ## Build binary app using pkg and esbuild
	tsc
	pkg --compress GZip .

clean: ## Clear build and cache
	rm -rf node_modules
	rm -rf dist/*
	rm -rf build/*
	go clean
	go mod tidy
	yarn

help: ## Show this help
	@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

install:   ## Install the plugin into $(INSTALL_HOME)
	mkdir -p $(BINDINGS_HOME)
	cp $(DIST_FOLDER)/* $(BINDINGS_HOME)

run: wasm js ## Compile and run the app
	node ${BUILD_FOLDER}/main.js  ~/bin/hiveot/certs

test: ## Run tests
	GOOS=js GOARCH=wasm go test -failfast  -cover ./...

upgrade: ## Upgrade packages (use with care)
	go get -u all
	go mod tidy
