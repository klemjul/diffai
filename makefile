.PHONY: lint format test install

lint: 
	golangci-lint run ./...

format:
	go fmt ./...

test:
	go test ./... -cover

install:
	go mod download

clean: 
	go mod tidy