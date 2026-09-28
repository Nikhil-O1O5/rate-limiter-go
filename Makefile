build:
	@go build -o ./bin/app ./cmd/.
	@chmod +x ./bin/app

run: build
	@./bin/app

up:
	@docker compose up -d

down:
	@docker compose down

test:
	@go test -v ./...

test-race:
	@go clean -testcache
	@go test -race -v ./...
