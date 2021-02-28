.PHONY: test bins clean
PROJECT_ROOT = github.com/mikhailshilkov/temporal-bench

# default target
default: test

PROGS = temporal-bench \

TEST_ARG ?= -race -v -timeout 5m

NAMESPACE ?= benchtest
NAMESPACE_RETENTION ?= 1
FRONTEND_ADDRESS ?= 127.0.0.1:7233
NUM_DECISION_POLLERS ?= 10

DOCKER_IMAGE ?= temporal-bench
DOCKER_IMAGE_TAG := $(shell whoami | tr -d " ")-local

# Automatically gather all srcs
ALL_SRC := $(shell find . -name "*.go")

# all directories with *_test.go files in them
# TEST_DIRS=./workflows \
#	./activities \

bins/temporal-bench: ./cmd/worker/*.go
	go build -o bins/temporal-bench ./cmd/worker/*.go

bins: bins/temporal-bench

test: bins
	@rm -f test
	@rm -f test.log
#	@echo $(TEST_DIRS)
#	@for dir in $(TEST_DIRS); do \
#		go test -coverprofile=$@ "$$dir" | tee -a test.log; \
#	done;

update-sdk:
	go get -u go.temporal.io/api@master
	go get -u go.temporal.io/sdk@master
	go mod tidy

run: bins
	NAMESPACE=$(NAMESPACE) \
	NAMESPACE_RETENTION=$(NAMESPACE_RETENTION) \
	FRONTEND_ADDRESS=$(FRONTEND_ADDRESS) \
	NUM_DECISION_POLLERS=$(NUM_DECISION_POLLERS) \
	bins/temporal-bench

docker-image:
	docker build -t $(DOCKER_IMAGE):$(DOCKER_IMAGE_TAG) .

clean:
	rm -rf bins
