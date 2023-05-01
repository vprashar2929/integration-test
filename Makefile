DOCKER_REPO ?= quay.io/rh_ee_vprashar/rhobs-test
BRANCH := $(strip $(shell git rev-parse --abbrev-ref HEAD))
BUILD_DATE :=$(shell date -u +"%Y-%m-%d")
VERSION := $(strip $(shell [ -d .git ] && git describe --always --tags --dirty))
EXAMPLES := examples
MANIFESTS := ${EXAMPLES}/manifests

all: build
build: rhobs-test

.PHONY: rhobs-test
rhobs-test:
	CGO_ENABLED=0 go build ./cmd/rhobs-test

.PHONY: vendor
vendor: go.mod go.sum
		go mod tidy
		go mod vendor
.PHONY: go-fmt
go-fmt:
	@fmt_res=$$(gofmt -d -s $$(find . -type f -name '*.go' -not -path './vendor/*')); if [ -n "$$fmt_res" ]; then printf '\nGofmt found style issues. Please check the reported issues\nand fix them if necessary before submitting the code for review:\n\n%s' "$$fmt_res"; exit 1; fi

.PHONY: container-dev
container-dev: kind
	@docker build \
		--build-arg DOCKERFILE_PATH="/Dockerfile" \
		-t $(DOCKER_REPO):$(BRANCH)-$(BUILD_DATE)-$(VERSION) \
		.
	docker tag $(DOCKER_REPO):$(BRANCH)-$(BUILD_DATE)-$(VERSION) localhost:5001/rhobs-test:latest
	docker push localhost:5001/rhobs-test:latest

.PHONY: kind
kind:
	wget https://kind.sigs.k8s.io/examples/kind-with-registry.sh
	chmod 755 kind-with-registry.sh
	./kind-with-registry.sh

.PHONY: local
local: kind container-dev
	kubectl apply -f $(MANIFESTS)/test-deployment.yaml
	kubectl apply -f $(MANIFESTS)/rbac.yaml
	kubectl apply -f $(MANIFESTS)/rhobs-test-dev-job.yaml

.PHONY: clean
clean:
	rm -rf kind-with-registry.sh
	rm -rf ./rhobs-test
	kind delete cluster
	docker rm -f kind-control-plane
	docker rm -f kind-registry

.PHONY: container-build
container-build:
	git update-index --refresh
	docker buildx build \
		--platform linux/amd64,linux/arm64 \
		--build-arg DOCKERFILE_PATH="/Dockerfile" \
		-t $(DOCKER_REPO):$(BRANCH)-$(BUILD_DATE)-$(VERSION) \
		-t $(DOCKER_REPO):latest \
		.
.PHONY: container-build-push
container-build-push:
	git update-index --refresh
	@docker buildx build \
		--push \
		--platform linux/amd64,linux/arm64 \
		-t $(DOCKER_REPO):$(BRANCH)-$(BUILD_DATE)-$(VERSION) \
		-t $(DOCKER_REPO):latest \
		.
