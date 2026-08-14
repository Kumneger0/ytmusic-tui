SERVER_URL ?= http://localhost:8080
project_name?=ytmusic-tui

default: help

.PHONY: help
help: ## show this help message
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'


.PHONY: build
build: ## build the Go application
	@echo "--> Building Go application..."
	@go build -ldflags "-X main.version=$(shell git describe --abbrev=0 --tags) -X main.Debug=true -X main.serverURL=$(SERVER_URL)" -o $(project_name)


.PHONY: install
install: build ## build and install ytmusic-tui to /usr/local/bin
	@echo "--> Installing ytmusic-tui to /usr/local/bin..."
	@sudo cp $(project_name) /usr/local/bin/
	@echo "--> Installation complete. Run 'ytmusic-tui' to start."

.PHONY: run
run: build ## build and run ytmusic-tui
	@./$(project_name)

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

.PHONY: proto
proto:
	@buf generate --template buf.gen.go.yaml
	@echo "compiled proto files successfully."

.PHONY: server-watch
server-watch: 
	@echo "Watching proto/music.proto for changes..."
	nodemon --watch proto/music.proto --exec "make proto"

.PHONY: server-run
server-run: ## run the python Connect RPC server
	@echo "Starting Connect RPC server..."
	@nodemon --watch grpc_server -e py --exec ".venv/bin/python -m grpc_server.main"

.PHONY: server-login
server-login: ## spin up http server for login
	@echo "Starting login flow..."
	.venv/bin/python grpc_server/main.py --login

.PHONY: server-sync
server-sync: ## sync python virtual environment dependencies
	@echo "Syncing virtual environment dependencies using uv..."
	uv sync


.PHONY: clean
clean: ## clean up both go and python generated files
	@echo "--> Cleaning up..."
	@rm -rf coverage.out dist/ $(project_name)
	@rm -rf gen/*.go gen/genconnect
	@rm -f grpc_server/gen/music_pb2.py grpc_server/gen/music_pb2.pyi grpc_server/gen/music_connect.py grpc_server/gen/music_pb2_grpc.py