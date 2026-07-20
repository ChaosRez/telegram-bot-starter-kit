.PHONY: build fmt fmt-check run test test-integration tidy vet verify

build:
	go build -o bin/telegram-bot ./cmd/bot

fmt:
	gofmt -w $$(find cmd internal -name '*.go')

fmt-check:
	test -z "$$(gofmt -l cmd internal)"

run:
	go run ./cmd/bot

test:
	go test -race -cover ./...

test-integration:
	go test -tags=integration ./internal/users

tidy:
	go mod tidy

vet:
	go vet ./...

verify: fmt-check vet test
