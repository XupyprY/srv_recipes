.PHONY: help

help: ## Display this help screen
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z0-9_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

swagger: ### Launch OpenAPI page
	swagger generate spec -o ./swagger.json
	swagger serve -F swaggesr ./swagger.json
.PHONY: swagger

run: ### Start Rest API server
	go run main.go
.PHONY: run

build: ### Build program
	go build main.go
.PHONY: build
