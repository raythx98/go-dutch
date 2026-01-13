.PHONY: build run test clean create_migration sqlc gqlgen run_local up down logs

# Default migration name if not provided (make create_migration name=my_migration)
name ?= new_migration

build:
	go build -o bin/server server.go

run: build
	./bin/server

test:
	go test -v ./...

clean:
	rm -rf bin/

create_migration:
	migrate create -ext sql -dir migrations -seq $(name)

sqlc:
	cd sqlc && sqlc generate --file sqlc.yaml

gqlgen:
	go tool github.com/99designs/gqlgen generate

run_local: sqlc gqlgen
	set -a && . .envrc && set +a && go run server.go

# Docker Compose Helpers
up:
	docker compose up -d

down:
	docker compose down

logs:
	docker compose logs -f

remove_caddy:
	docker stop go-dutch-caddy
	docker rm go-dutch-caddy
	docker volume rm go-dutch_caddy_data go-dutch_caddy_config