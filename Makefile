projectname?=clispot

default: help

.PHONY: help
help: ## show this help message
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'


.PHONY: build
build: server-build ## build the Go application
	@echo "--> Building Go application..."
	@go build -ldflags "-X main.version=$(shell git describe --abbrev=0 --tags) -X main.Debug=true" -o $(projectname)


.PHONY: install
install: build ## build and install clispot to /usr/local/bin
	@echo "--> Installing clispot to /usr/local/bin..."
	@sudo cp $(projectname) /usr/local/bin/
	@echo "--> Installation complete. Run 'clispot' to start."

.PHONY: run
run: build ## build and run clispot
	@./$(projectname)

.PHONY: bootstrap	
bootstrap: ## bootstrap go tools
	go generate -tags tools tools/tools.go

.PHONY: test
test: clean ## run go tests
	go test --cover -parallel=1 -v -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out | sort -rnk3

.PHONY: cover
cover: ## display go test coverage
	go test -v -race $(shell go list ./... | grep -v /vendor/) -v -coverprofile=coverage.out
	go tool cover -func=coverage.out

.PHONY: fmt
fmt: ## format go files
	gofumpt -w .
	gci write .

.PHONY: lint
lint: ## lint go files
	golangci-lint run -c .golangci.yml

.PHONY: pre-commit
pre-commit:	## run pre-commit hooks
	pre-commit run --all-files

.PHONY: hooks
hooks: ## install git commit-msg hook for commitlint (local)
	@chmod +x scripts/hooks/commit-msg
	@git config core.hooksPath scripts/hooks
	@echo "--> Git hooks installed (commit-msg)."


.PHONY: server-build
server-build: ## build python to single executable 
	@echo "building python to single executable"
	@GOHOSTOS=$$(go env GOHOSTOS); \
	GOHOSTARCH=$$(go env GOHOSTARCH); \
	GOOS=$$(go env GOOS); \
	GOARCH=$$(go env GOARCH); \
	if [ "$$GOOS" != "$$GOHOSTOS" ] || [ "$$GOARCH" != "$$GOHOSTARCH" ]; then \
		echo "Error: PyInstaller cannot cross-compile for $$GOOS/$$GOARCH on $$GOHOSTOS/$$GOHOSTARCH host"; \
		exit 1; \
	fi
	.venv/bin/pyinstaller main.spec 
	@mkdir -p backend/binaries
	@GOHOSTOS=$$(go env GOHOSTOS); \
	GOHOSTARCH=$$(go env GOHOSTARCH); \
	if [ "$$GOHOSTOS" = "windows" ] && [ "$$GOHOSTARCH" = "amd64" ]; then \
		cp dist/main.exe backend/binaries/python-windows-amd64.exe; \
	elif [ "$$GOHOSTOS" = "darwin" ] && [ "$$GOHOSTARCH" = "arm64" ]; then \
		cp dist/main backend/binaries/python-darwin-arm64; \
	elif [ "$$GOHOSTOS" = "linux" ] && [ "$$GOHOSTARCH" = "amd64" ]; then \
		cp dist/main backend/binaries/python-linux-amd64; \
	else \
		echo "Error: Unsupported host platform $$GOHOSTOS/$$GOHOSTARCH for backend build"; \
		exit 1; \
	fi

.PHONY: proto
proto: proto-python proto-go ## generate protobuf files for both python and go

.PHONY: proto-python
proto-python:
	@echo "Generating Python protobuf files..."
	@mkdir -p grpc_server/gen
	@touch grpc_server/gen/__init__.py

	.venv/bin/python -m grpc_tools.protoc \
		-Iproto \
		--python_out=grpc_server/gen \
		--grpc_python_out=grpc_server/gen \
		--pyi_out=grpc_server/gen \
		proto/music.proto

	@sed -i 's/^import music_pb2 as/from . import music_pb2 as/' grpc_server/gen/music_pb2_grpc.py

	@echo "Generated Python files successfully."

.PHONY: proto-go
proto-go: 
	@echo "Generating Go protobuf files..."
	@mkdir -p gen
	protoc -Iproto --go_out=gen --go_opt=module=github.com/kumneger0/clispot/core/gen --go-grpc_out=gen --go-grpc_opt=module=github.com/kumneger0/clispot/core/gen proto/music.proto
	@echo "Generated Go files successfully."

.PHONY: server-watch
server-watch: 
	@echo "Watching proto/music.proto for changes..."
	nodemon --watch proto/music.proto --exec "make proto"

.PHONY: server-run
server-run: ## run the python gRPC server
	@echo "Starting gRPC server..."
	nodemon --ext py --exec ".venv/bin/python grpc_server/main.py"

.PHONY: server-login
server-login: ## spin up http server for login
	@echo "Starting gRPC server..."
	.venv/bin/python grpc_server/main.py --login

.PHONY: server-ui
server-ui: ## open interactive gRPC web UI using grpcui
	@echo "Starting grpcui client (make sure the server is running first)..."
	grpcui -plaintext -proto proto/music.proto localhost:50051

.PHONY: server-sync
server-sync: ## sync python virtual environment dependencies
	@echo "Syncing virtual environment dependencies using uv..."
	uv sync

# ==============================================================================
# General Targets
# ==============================================================================

.PHONY: clean
clean: ## clean up both go and python generated files
	@echo "--> Cleaning up..."
	@rm -rf coverage.out dist/ $(projectname)
	@rm -f gen/*.go
	@rm -f grpc_server/gen/music_pb2.py grpc_server/gen/music_pb2.pyi grpc_server/gen/music_pb2_grpc.py
	@find grpc_server -type d -name "__pycache__" -exec rm -rf {} +
	@echo "--> Clean completed."