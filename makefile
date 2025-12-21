.PHONY: lint format test install

lint: 
	golangci-lint run ./...

format:
	go fmt ./...

test:
	go test ./... -coverprofile=coverage.txt

install:
	go mod download

clean: 
	go mod tidy