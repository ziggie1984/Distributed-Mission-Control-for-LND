#? build: Build the project and create ec-debug binary
build: rpc
	go build -buildvcs=false -o ec-debug

#? rpc: Generate RPC code using buf
rpc:
	buf generate

#? install: Install the binary to $(HOME)/go/bin as ec
install: build
	cp ec-debug ec
	mkdir -p $(HOME)/go/bin
	mv ec $(HOME)/go/bin/


#? test: Run tests with verbose output
test:
	go test ./... -v

#? lint: Run golangci-lint
lint:
	golangci-lint run -v

#? fmt: Format the code
fmt:
	go fmt ./...

#? clean: Clean build caches and binaries
clean:
	go clean -cache -testcache -modcache
	rm -f ec-debug

#? all: Run all targets
all: clean fmt rpc build test lint

#? help: List all available make targets with their descriptions
help: Makefile
	@$(call print, "Listing commands:")
	@sed -n 's/^#?//p' $< | column -t -s ':' |  sort | sed -e 's/^/ /'
