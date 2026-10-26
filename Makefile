.PHONY: all frontend backend build build-windows build-mac build-linux test clean run

BINARY_NAME=tourna-desk

all: build

frontend:
	cd web && npm install && npm run build
	mkdir -p cmd/server/dist
	rm -rf cmd/server/dist/*
	cp -r web/dist/* cmd/server/dist/

backend:
	CGO_ENABLED=0 go build -ldflags="-s -w" -o $(BINARY_NAME) ./cmd/server

build: frontend backend

build-windows: frontend
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o $(BINARY_NAME).exe ./cmd/server

build-mac: frontend
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o $(BINARY_NAME) ./cmd/server

build-linux: frontend
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o $(BINARY_NAME) ./cmd/server

test:
	go test -v ./...

run: build
	./$(BINARY_NAME)

clean:
	rm -f $(BINARY_NAME) $(BINARY_NAME).exe $(BINARY_NAME)-darwin-arm64 $(BINARY_NAME)-linux-amd64
	rm -rf web/dist cmd/server/dist/assets
