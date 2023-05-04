include .bingo/Variables.mk

DOCKER_REPO ?= quay.io/rh_ee_vprashar/rhobs-test
BRANCH := $(strip $(shell git rev-parse --abbrev-ref HEAD))
BUILD_DATE :=$(shell date -u +"%Y-%m-%d")
VERSION := $(strip $(shell [ -d .git ] && git describe --always --tags --dirty))
EXAMPLES := examples
OCP_MANIFESTS := ${EXAMPLES}/manifests/openshift
DEV_MANIFESTS := ${EXAMPLES}/manifests/dev
XARGS ?= $(shell which gxargs 2>/dev/null || which xargs)
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
	kubectl apply -f $(DEV_MANIFESTS)/test-deployment-template.yaml
	kubectl apply -f $(DEV_MANIFESTS)/rbac-template.yaml
	kubectl apply -f $(DEV_MANIFESTS)/job-template.yaml

.PHONY: local-faulty
local-faulty: kind container-dev
	kubectl apply -f $(DEV_MANIFESTS)/test-deployment-faulty-template.yaml
	kubectl apply -f $(DEV_MANIFESTS)/rbac-template.yaml
	kubectl apply -f $(DEV_MANIFESTS)/job-template.yaml

.PHONY: clean
clean:
	find $(EXAMPLES) -type f ! -name '*.yaml' -delete
	find $(OCP_MANIFESTS) -type f ! -name '*.yaml' -delete
	find $(DEV_MANIFESTS) -type f ! -name '*.yaml' -delete

.PHONY: clean-local
clean-local:
	rm -rf kind-with-registry.sh
	rm -rf ./rhobs-test
	kind delete cluster
	docker ps -a -q | xargs docker rm -f

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
rbac-template: $(JSONNET) $(JSONNETFMT) $(GOJSONTOYAML)
	echo "Running rbac template"
	$(JSONNETFMT) -n 2 --max-blank-lines 2 --string-style s --comment-style s -i jsonnet/ocp-manifests.jsonnet
	$(JSONNET) -m $(OCP_MANIFESTS) jsonnet/ocp-manifests.jsonnet | $(XARGS) -I{} sh -c 'cat {} | $(GOJSONTOYAML) > {}.yaml' -- {}
	make clean
dev-template: $(JSONNET) $(JSONNETFMT) $(GOJSONTOYAML)
	echo "Running dev rbac templates"
	$(JSONNETFMT) -n 2 --max-blank-lines 2 --string-style s --comment-style s -i jsonnet/ocp-manifests.jsonnet
	$(JSONNET) -m $(DEV_MANIFESTS) jsonnet/dev-manifests.jsonnet | $(XARGS) -I{} sh -c 'cat {} | $(GOJSONTOYAML) > {}.yaml' -- {}
	make clean
.PHONY: manifests
manifests: rbac-template dev-template
