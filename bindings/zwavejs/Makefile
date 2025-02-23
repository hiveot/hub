# Makefile to build the zwave protocol binding
DIST_FOLDER=./dist
BUILD_FOLDER=./build
INSTALL_HOME=~/bin/hiveot
PLUGINS_HOME=$(INSTALL_HOME)/plugins
.DEFAULT_GOAL := help

.FORCE:

all: dist

js: .FORCE  ## Compile typescript
	npx tsc

dist: .FORCE ## Build binary app using pkg and esbuild
	./build.sh

clean: ## Clear build and cache
	rm -rf node_modules
	rm -rf dist/*
	rm -rf build/*
	npm i

help: ## Show this help
	@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

install:   ## Install the plugin into $(PLUGINS_HOME)
	mkdir -p $(PLUGINS_HOME)
	cp $(DIST_FOLDER)/* $(PLUGINS_HOME)

run:  js ## Compile and run the app
	node ${BUILD_FOLDER}/main.js  ~/bin/hiveot/certs

test: ## Run tests
	echo "TODO"


upgrade: ## Upgrade nodejs packages (use with care)
	echo "TODO"
