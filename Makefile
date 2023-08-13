include .bingo/Variables.mk

IMG_REPO ?= quay.io/rh_ee_vprashar/integration-test
BRANCH := $(strip $(shell git rev-parse --abbrev-ref HEAD))
BUILD_DATE :=$(shell date -u +"%Y-%m-%d")
VERSION := $(strip $(shell [ -d .git ] && git describe --always --tags --dirty))
EXAMPLES := examples
DEV_MANIFESTS := $(EXAMPLES)/manifests/dev
XARGS ?= $(shell which gxargs 2>/dev/null || which xargs)
 # Setting GOENV
 GOOS := $(shell go env GOOS)
 GOARCH := $(shell go env GOARCH)
all: build
build: integration-test

.PHONY: integration-test
integration-test:
	CGO_ENABLED=0 go build ./cmd/integration-test

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
		-t $(IMG_REPO):$(BRANCH)-$(BUILD_DATE)-$(VERSION) \
		.
	docker tag $(IMG_REPO):$(BRANCH)-$(BUILD_DATE)-$(VERSION) localhost:5001/integration-test:latest
	docker push localhost:5001/integration-test:latest

.PHONY: kind
kind:
	wget https://kind.sigs.k8s.io/examples/kind-with-registry.sh
	chmod 755 kind-with-registry.sh
	./kind-with-registry.sh

.PHONY: test
test:
	go test ./...
	
.PHONY: local
local: kind container-dev
	kubectl apply -f $(DEV_MANIFESTS)/test-deployment.yaml
	kubectl apply -f $(DEV_MANIFESTS)/test-rbac.yaml
	kubectl apply -f $(DEV_MANIFESTS)/test-job.yaml

.PHONY: local-faulty
local-faulty: kind container-dev
	kubectl apply -f $(DEV_MANIFESTS)/test-deployment-faulty.yaml
	kubectl apply -f $(DEV_MANIFESTS)/test-rbac.yaml
	kubectl apply -f $(DEV_MANIFESTS)/test-job.yaml

.PHONY: clean
clean:
	find $(EXAMPLES) -type f ! -name '*.yaml' -delete
	find $(DEV_MANIFESTS) -type f ! -name '*.yaml' -delete

.PHONY: clean-local
clean-local:
	rm -f kind-with-registry.sh
	rm -f ./integration-test
	kind delete cluster
	docker ps -a -q | xargs docker rm -f

.PHONY: container-build
container-build:
	docker build \
		--platform linux/$(GOARCH) \
		-t $(IMG_REPO):$(BRANCH)-$(BUILD_DATE)-$(VERSION) \
		-t $(IMG_REPO):latest \
		.
.PHONY: container-build-push
container-build-push: container-build
	@docker push $(IMG_REPO):latest

dev-template: $(JSONNET) $(JSONNETFMT) $(GOJSONTOYAML)
	echo "Running dev templates"
	$(JSONNETFMT) -n 2 --max-blank-lines 2 --string-style s --comment-style s -i jsonnet/dev-manifests.jsonnet
	$(JSONNET) -m $(DEV_MANIFESTS) jsonnet/dev-manifests.jsonnet | $(XARGS) -I{} sh -c 'cat {} | $(GOJSONTOYAML) > {}.yaml' -- {}
	make clean

.PHONY: manifests
manifests: dev-template
