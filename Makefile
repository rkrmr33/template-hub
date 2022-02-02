.DEFAULT_GOAL=def
VERSION := v0.0.1
GIT_COMMIT := $(shell git rev-parse HEAD)
BUILD_DATE := $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')

GO_PROTO_GEN=gogofast
OPENAPI_DEST=$(shell pwd)/assets/swagger.json
OUT_DIR := dist

ALL_TARGETS := linux-amd64 linux-arm64 darwin-amd64 darwin-arm64 windows-amd64
LOCAL_TAGET := $(shell go env GOOS)-$(shell go env GOARCH)

BIN_NAME=server
SRCS := $(shell echo main.go go.mod go.sum && go list -f '{{ join .Deps "/*.go\n" }}' . | grep 'rkrmr33/template-hub' | cut -c 33-)

ifndef GOBIN
ifndef GOPATH
$(error GOPATH is not set, please make sure you set your GOPATH correctly!)
endif
GOBIN=$(GOPATH)/bin
ifndef GOBIN
$(error GOBIN is not set, please make sure you set your GOBIN correctly!)
endif
endif

define install_dep
	@go mod vendor
	@echo building $(1)...
	@go build -o $(GOBIN)/$(1) $(2)
endef

define build_image
	@docker buildx build . -t $(1)
endef

# build all binaries in local settings
def:
	@make local

# docker images targets
.PHONY: pre-images
pre-images:
	@rm -rf ./vendor || true

.PHONY: images
images: pre-images frontiers-image

%-image:
	$(call build_image,$*)

### Generic ###
.PHONY: all
all: $(addprefix $(OUT_DIR)/$(BIN_NAME)-, $(ALL_TARGETS))

.PHONY: local
local: $(OUT_DIR)/$(BIN_NAME)-$(LOCAL_TAGET)
	@rm /usr/local/bin/$(BIN_NAME) 2>/dev/null || true
	@ln -s $(shell pwd)/$(OUT_DIR)/$(BIN_NAME)-$(LOCAL_TAGET) /usr/local/bin/$(BIN_NAME) || \
		echo you probably have some permission issue, try running: 'sudo chown -R $(whoami) /usr/local/bin'

%.sha256:
	@make $* BIN_NAME=$(BIN_NAME)
	@cd $(OUT_DIR) && tar -czvf $(notdir $*.tar.gz) $(notdir $*) && cd ..
	@openssl dgst -sha256 "$*.tar.gz" | awk '{ print $$2 }' > "$*".sha256
	
$(OUT_DIR)/$(BIN_NAME)-linux-amd64: GO_FLAGS='GOOS=linux GOARCH=amd64 CGO_ENABLED=0 GOPRIVATE=$(GO_PRIVATE)'
$(OUT_DIR)/$(BIN_NAME)-linux-arm64: GO_FLAGS='GOOS=linux GOARCH=arm64 CGO_ENABLED=0 GOPRIVATE=$(GO_PRIVATE)'
$(OUT_DIR)/$(BIN_NAME)-darwin-amd64: GO_FLAGS='GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 GOPRIVATE=$(GO_PRIVATE)'
$(OUT_DIR)/$(BIN_NAME)-darwin-arm64: GO_FLAGS='GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 GOPRIVATE=$(GO_PRIVATE)'
$(OUT_DIR)/$(BIN_NAME)-windows-amd64: GO_FLAGS='GOOS=windows GOARCH=amd64 CGO_ENABLED=0 GOPRIVATE=$(GO_PRIVATE)'

$(OUT_DIR)/$(BIN_NAME)-%: $(SRCS)
	@rm -rf ./vendor || true
	@go mod tidy
	@GO_FLAGS=$(GO_FLAGS) \
	BINARY_NAME=$(BIN_NAME) \
	VERSION=$(VERSION) \
	GIT_COMMIT=$(GIT_COMMIT) \
	BUILD_DATE=$(BUILD_DATE) \
	OUT_FILE=$(OUT_DIR)/$(BIN_NAME)-$* \
	MAIN=. \
	GCFLAGS="$(GCFLAGS)" \
	./hack/build.sh

.PHONY: install
install: manifests/install.yaml
	@kubectl apply -f manifests/install.yaml

.PHONY: uninstall
uninstall: manifests/install.yaml
	@kubectl delete -f manifests/install.yaml

.PHONY: gen
gen: ./vendor mocks-gen proto-gen manifests-gen

.PHONY: mocks-gen
mocks-gen: $(GOBIN)/mockery
	@echo generating mocks...
	@go generate ./...

.PHONY: manifests-gen
manifests-gen: $(GOBIN)/kustomize
	@echo updating installation manifests...
	@kustomize build ./manifests/base > ./manifests/install.yaml

.PHONY: proto-gen
proto-gen: /usr/local/bin/protoc /usr/local/bin/jq $(GOBIN)/protoc-gen-$(GO_PROTO_GEN) $(GOBIN)/protoc-gen-swagger $(GOBIN)/protoc-gen-grpc-gateway $(GOBIN)/swagger
	@echo generating protobuf...
	@GO_PROTO_GEN=$(GO_PROTO_GEN) \
	 OPENAPI_DEST=$(OPENAPI_DEST) \
	 VERSION=$(VERSION) \
	 ./hack/proto-gen.sh
	@echo done!
	@go mod tidy

.PHONY: pre-push
pre-push: gen test check-worktree images 

.PHONY: pre-release
pre-release: BIN_NAME=$(CLI_BIN_NAME)
pre-release: $(addsuffix .sha256, $(addprefix $(OUT_DIR)/$(CLI_BIN_NAME)-, $(ALL_TARGETS)))

.PHONY: release
release: pre-push pre-release
	@VERSION=$(VERSION) ./hack/release.sh

.PHONY: check-worktree
check-worktree:
	@./hack/check-worktree.sh

PHONY: lint
lint: $(GOBIN)/golangci-lint
	@echo linting go code...
	@rm -rf ./vendor || true
	@go mod tidy
	@GOGC=off golangci-lint run --fix --timeout 6m
	
.PHONY: test
test:
	@echo running tests...
	@rm -rf ./vendor || true
	@go mod tidy
	@./hack/test.sh

.PHONY: clean
clean: 
	@rm -rf $(OUT_DIR) *.out vendor || true

./vendor:
	@echo vendoring...
	@go mod vendor

### External Dependencies ###
$(GOBIN)/mockery:
	@mkdir $(OUT_DIR) || true
	@echo downloading mockery... $(GOBIN)/mockery
	@curl -L -o $(OUT_DIR)/mockery.tar.gz -- https://github.com/vektra/mockery/releases/download/v2.7.6/mockery_2.7.6_$(shell uname -s)_$(shell uname -m).tar.gz
	@tar zxvf $(OUT_DIR)/mockery.tar.gz mockery
	@chmod +x mockery
	@mkdir -p $(GOBIN)
	@mv mockery $(GOBIN)/mockery
	@mockery --version

$(GOBIN)/golangci-lint:
	@mkdir $(OUT_DIR) || true
	@echo downloading golangci-lint...
	@curl -L -o $(OUT_DIR)/golangci-lint.tar.gz -- https://github.com/golangci/golangci-lint/releases/download/v1.41.1/golangci-lint-1.41.1-$(shell uname -s)-$(shell uname -m).tar.gz
	@tar zxvf $(OUT_DIR)/golangci-lint.tar.gz golangci-lint
	@chmod +x golangci-lint
	@mkdir -p $(GOBIN)
	@mv golangci-lint $(GOBIN)/golangci-lint
	@golangci-lint --version

$(GOBIN)/kustomize:
	@mkdir $(OUT_DIR) || true
	@echo downloading kustomize...
	@curl -L -o $(OUT_DIR)/kustomize.tar.gz -- https://github.com/kubernetes-sigs/kustomize/releases/download/kustomize%2Fv4.2.0/kustomize_v4.2.0_$(shell go env GOOS)_$(shell go env GOARCH).tar.gz
	@tar zxvf $(OUT_DIR)/kustomize.tar.gz kustomize
	@chmod +x kustomize
	@mkdir -p $(GOBIN)
	@mv kustomize $(GOBIN)/kustomize
	@kustomize version

$(GOBIN)/protoc-gen-$(GO_PROTO_GEN):
	$(call install_dep,protoc-gen-$(GO_PROTO_GEN),./vendor/github.com/gogo/protobuf/protoc-gen-$(GO_PROTO_GEN))

$(GOBIN)/protoc-gen-gogo:
	$(call install_dep,protoc-gen-gogo,./vendor/github.com/gogo/protobuf/protoc-gen-gogo)

$(GOBIN)/protoc-gen-swagger:
	$(call install_dep,protoc-gen-swagger,./vendor/github.com/grpc-ecosystem/grpc-gateway/protoc-gen-swagger)

$(GOBIN)/protoc-gen-grpc-gateway:
	$(call install_dep,protoc-gen-grpc-gateway,./vendor/github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway)

$(GOBIN)/swagger:
	$(call install_dep,swagger,./vendor/github.com/go-swagger/go-swagger/cmd/swagger)

ifeq ($(shell go env GOOS), linux)
PROTOC_PKG=protoc-3.14.0-linux-x86_64.zip
endif
ifeq ($(shell go env GOOS), darwin)
PROTOC_PKG=protoc-3.14.0-osx-x86_64.zip
endif
/usr/local/bin/protoc:
	@echo downloading protoc...
	@curl -OL https://github.com/protocolbuffers/protobuf/releases/download/v3.14.0/$(PROTOC_PKG)
	@sudo unzip -o $(PROTOC_PKG) -d /usr/local bin/protoc
	@sudo unzip -o $(PROTOC_PKG) -d /usr/local 'include/*'
	@rm -f $(PROTOC_PKG)

/usr/local/bin/skaffold:
	@curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/latest/skaffold-$(shell go env GOOS)-$(shell go env GOARCH)
	@chmod +x skaffold
	@mv skaffold /usr/local/bin/skaffold

ifeq ($(shell go env GOOS), linux)
JQ_BIN=jq-linux64
endif
ifeq ($(shell go env GOOS), darwin)
JQ_BIN=jq-osx-amd64
endif
/usr/local/bin/jq:
	@echo downloading jq...
	@curl -OL https://github.com/stedolan/jq/releases/download/jq-1.6/$(JQ_BIN)
	@chmod +x $(JQ_BIN)
	@mv $(JQ_BIN) /usr/local/bin/jq