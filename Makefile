create_migration:
	migrate create -ext sql -dir migrations -seq init_table

sqlc:
	cd sqlc && sqlc generate --file sqlc.yaml

allow_direnv:
	direnv allow . || true

build:
	docker build -t url-shortener .

volume:
	docker volume create local-postgres

network:
	docker network create my-network || true

db:
	docker run -d --rm --name ${APP_URLSHORTENER_DBHOST} \
		--net my-network -p ${APP_URLSHORTENER_DBPORT}:${APP_URLSHORTENER_DBPORT} \
		-e POSTGRES_PASSWORD=${APP_URLSHORTENER_DBPASSWORD} \
		-v local-postgres:/var/lib/postgresql/data \
		postgres:latest && sleep 5 || true


format_env:
	find .envrc && sed 's/^export //' .envrc > .env || true

migrate_up:
	migrate -database 'postgres://${APP_URLSHORTENER_DBUSERNAME}:${APP_URLSHORTENER_DBPASSWORD}@localhost:${APP_URLSHORTENER_DBPORT}/${APP_URLSHORTENER_DBDEFAULTNAME}?sslmode=disable' -path migrations up

run: allow_direnv build volume network stop db format_env migrate_up
	docker run -d --rm --name url-shortener-app \
		--net my-network -p ${APP_GODUTCH_SERVERPORT}:${APP_GODUTCH_SERVERPORT} \
		--env-file .env url-shortener

stop_app:
	docker stop go-dutch-app || true

stop_db:
	docker stop go-dutch-db || true
	sleep 5

stop: stop_app stop_db

gqlgen:
	go tool github.com/99designs/gqlgen generate

run_local: allow_direnv sqlc gqlgen
	set -a && . .envrc && set +a &&go run server.go