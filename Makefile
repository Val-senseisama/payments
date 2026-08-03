build:
	@go build -o bin/payments cmd/main.go

test:
	@go test -v ./...

run: build
	@./bin/payments

dev:
	@air

migrate:
	@migrate create -ext sql -dir cmd/migrations/sql -seq $(filter-out $@,$(MAKECMDGOALS))

migrate-up:
	@go run cmd/migrations/main.go up

migrate-down:
	@go run cmd/migrations/main.go down

%:
	@:
