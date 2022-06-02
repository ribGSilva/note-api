SHELL := /bin/bash

# =====================================
# Variable

VERSION := 1.0
PROJECT_NAME := note-api

# =====================================
# Develop

app=api

debug:
	go install github.com/go-delve/delve/cmd/dlv@latest
	go install github.com/cosmtrek/air@latest
	air -c .air.$(app).debug.toml

run:
	go install github.com/cosmtrek/air@latest
	air -c .air.$(app).toml

tests:
	go test ./app/$(app)/tests

tidy:
	go mod tidy
	go mod vendor

# =====================================
# Swagger

swagger:
	go install github.com/swaggo/swag/cmd/swag@latest
	swag init \
        --parseInternal \
        --parseDependency \
        --parseDepth 3 \
        --output app/api/docs \
        --dir app/api/

# =====================================
# Enviroment

env-build:
	-docker network create -d bridge $(PROJECT_NAME)
	-docker run -ti -p 3306:3306 --name mysql --network="$(PROJECT_NAME)" -d -e MYSQL_ROOT_PASSWORD=admin mysql
	-docker run -d -ti -p 6379:6379 --network="$(PROJECT_NAME)" --name redis redis

env-setup:
	-docker exec mysql mysql -u root -padmin -e "create database if not exists note;"
	-go run ./app/cmd/main.go schema create

env-up:
	-docker start mysql
	-docker start redis

env-down:
	-docker stop mysql
	-docker stop redis

env-clear:
	-docker rm mysql
	-docker rm redis
	-docker network rm $(PROJECT_NAME)

# =====================================
# Docker

docker-up: docker-build docker-run docker-logs

docker-reload: docker-stop docker-remove docker-build docker-run docker-logs

docker-down: docker-stop docker-remove docker-clear

docker-build:
	docker build -t $(PROJECT_NAME):$(VERSION) -f zarf/docker/api.Dockerfile .

docker-run:
	docker run -it -d -p 8080:8080 -p 4000:4000 --network="$(PROJECT_NAME)" \
 		-e MONGO_CONNECTION_URL="root:admin@tcp(mysql:3306)/note" \
 		-e REDIS_ADDRESS="redis:6379" \
 		--name $(PROJECT_NAME) $(PROJECT_NAME):$(VERSION)

docker-stop:
	docker stop $(PROJECT_NAME)

docker-stats:
	docker stats $(PROJECT_NAME)

docker-logs:
	docker logs -f --tail 20 $(PROJECT_NAME)

docker-remove:
	docker rm $(PROJECT_NAME)

docker-clear:
	docker rmi $(PROJECT_NAME):$(VERSION)
