.PHONY: up down logs db run test test-integration fmt vet build

DB_TEST_URL ?= postgres://ecommerce:ecommerce@localhost:5432/ecommerce?sslmode=disable

up:        ## levanta postgres + api en docker
	docker compose up -d --build

down:      ## detiene todo (agrega -v a mano para borrar los datos)
	docker compose down

logs:
	docker compose logs -f api

db:        ## solo la base de datos (para correr la API con go run)
	docker compose up -d db

run:       ## API local contra la base en docker
	go run ./cmd/api

test:      ## pruebas unitarias (sin base de datos)
	go test ./... -race -count=1

test-integration: ## unitarias + integracion contra postgres (make db antes)
	TEST_DATABASE_URL="$(DB_TEST_URL)" go test ./... -race -count=1

fmt:
	gofmt -w .

vet:
	go vet ./...

build:
	CGO_ENABLED=0 go build -o bin/api ./cmd/api
