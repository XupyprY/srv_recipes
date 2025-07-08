.PHONY: help

help: ## Display this help screen
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z0-9_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

swagger: ### Launch OpenAPI page
	swagger generate spec -o ./swagger.json
	swagger serve -F swaggesr ./swagger.json
.PHONY: swagger

run: rundb ### Start Rest API server
	# MONGO_URI="mongodb://admin:password@localhost:27017/demo?authSource=admin" MONGO_DATABASE=demo go run main.go
	MONGO_URI="mongodb://admin:password@localhost:27017/demo?authSource=admin" MONGO_DATABASE=demo go run main.go
.PHONY: run

build: ### Build program
	go build main.go
.PHONY: build

rundb: ### Start MongoDB
	# docker run -d --name mongo -v ${PWD}/data:/data/db -e MONGO_INITDB_ROOT_USERNAME=admin -e MONGO_INITDB_ROOT_PASSWORD=password -p 27017:27017 mongo:8-noble
	docker compose up -d
.PHONY: rundb

initdb:
	# docker exec -i mongo mongoimport --username admin --password password --authenticationDatabase admin --db demo --collection recipes2 --file /docker-entrypoint-initdb.d/recipes.json --jsonArray
	docker exec -i mongo mongoimport --username admin --password password --authenticationDatabase admin --db demo --collection recipes --jsonArray < recipes.json
.PHONY: initdb

testdb:
	docker exec -it mongo mongosh -u admin -p password
	show dbs
	use demo
	show collections
	db.recipes.find().pretty()
.PHONY: testdb

redislab: ### Start relis IU tool
	docker run -d --name redisinsight --link redis -p 8001:8001 redislabs/redisinsight