.PHONY: test test-unit test-integration build run clean fmt vet

test: test-unit test-integration

test-unit:
	go test -count=1 ./internal/...

test-integration:
	go test -count=1 -timeout 60s ./test/integration/...

test-verbose:
	go test -count=1 -v ./internal/... ./test/integration/...

build:
	go build -o bin/juridico-watcher ./cmd/juridico-watcher

build-windows:
	GOOS=windows GOARCH=amd64 go build -o bin/juridico-watcher.exe ./cmd/juridico-watcher

build-linux:
	GOOS=linux GOARCH=amd64 go build -o bin/juridico-watcher ./cmd/juridico-watcher

run:
	go run ./cmd/juridico-watcher

clean:
	rm -rf bin/

fmt:
	go fmt ./...

vet:
	go vet ./...