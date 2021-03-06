VERSION="`git describe --abbrev=0 --tags`"
COMMIT="`git rev-list -1 --abbrev-commit HEAD`"

all: clean fmt test build

fmt:
	@echo "Formatting..."
	@goimports -l -w ./

install: build
	@echo "Installing..."
	sudo mkdir -p /etc/xlisp
	sudo cp ./bin/xlisp /usr/local/bin && sudo cp ./lib/core.xlisp /etc/xlisp/core.xlisp

clean:
	@echo "Cleaning up..."
	@rm -rf ./bin
	@go mod tidy -v

test:
	@echo "Running tests..."
	@go test -cover ./...

test-verbose:
	@echo "Running tests..."
	@go test -v -cover ./...

benchmark:
	@echo "Running benchmarks..."
	@go test -benchmem -run="none" -bench="Benchmark.*" -v ./...

build:
	mkdir -p ./bin
	@go build -ldflags="-X main.version=${VERSION} -X main.commit=${COMMIT}" -o ./bin/xlisp ./cmd/xlisp/
