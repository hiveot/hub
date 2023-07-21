# Makefile to build and test the HiveOT Hub
DIST_FOLDER=../dist
.DEFAULT_GOAL := help


.FORCE:

hubcore: ## Build the core binary
	go build -o $(DIST_FOLDER)/bin/$@ cmd/$@/main.go

help: ## Show this help
	@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'
