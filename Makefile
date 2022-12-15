# Image URL to use all building/pushing image targets
IMG ?= generator:latest

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: build
build: fmt vet ## Build manager binary.
	go build -o bin/generator main.go

.PHONY: test
test:
	go test ./... -v -count=1

.PHONY: docker-build
docker-build: test ## Build docker image with the manager.
	docker build -t ${IMG} .

.PHONY: podman-build
podman-build: test ## Build docker image with the manager.
	podman build -t ${IMG} .