GOLANGCI_LINT_VERSION := "v1.31.0"
BENCH_COUNT ?= 5
REF_NAME ?= $(shell git symbolic-ref HEAD --short | tr / - 2>/dev/null)

# The head of Makefile determines location of dev-go to include standard targets.
GO ?= go
export GO111MODULE = on

ifneq "$(GOFLAGS)" ""
  $(info GOFLAGS: ${GOFLAGS})
endif

ifneq "$(wildcard ./vendor )" ""
  $(info Using vendor)
  modVendor =  -mod=vendor
  ifeq (,$(findstring -mod,$(GOFLAGS)))
      export GOFLAGS := ${GOFLAGS} ${modVendor}
  endif
  ifneq "$(wildcard ./vendor/github.com/bool64/dev)" ""
  	DEVGO_PATH := ./vendor/github.com/bool64/dev
  endif
endif

ifeq ($(DEVGO_PATH),)
	DEVGO_PATH := $(shell GO111MODULE=on $(GO) list ${modVendor} -f '{{.Dir}}' -m github.com/bool64/dev)
	ifeq ($(DEVGO_PATH),)
    	$(info Module github.com/bool64/dev not found, downloading.)
    	DEVGO_PATH := $(shell export GO111MODULE=on && $(GO) mod tidy && $(GO) list -f '{{.Dir}}' -m github.com/bool64/dev)
	endif
endif

-include $(DEVGO_PATH)/makefiles/main.mk
-include $(DEVGO_PATH)/makefiles/test-unit.mk
-include $(DEVGO_PATH)/makefiles/lint.mk
-include $(DEVGO_PATH)/makefiles/github-actions.mk

## Run tests
test: test-unit test-examples

test-examples:
	cd _examples && go test -race ./...

## Run benchmark, iterations count controlled by BENCH_COUNT, default 5.
bench:
	@$(GO) test -bench=. -count=$(BENCH_COUNT) -run=^a  ./... >bench-$(REF_NAME).txt
	@test -s $(GOPATH)/bin/benchstat || GO111MODULE=off GOFLAGS= GOBIN=$(GOPATH)/bin $(GO) get -u golang.org/x/perf/cmd/benchstat
	@test -e bench-master.txt && benchstat bench-master.txt bench-$(REF_NAME).txt || benchstat bench-$(REF_NAME).txt

## Run benchmark for app examples, iterations count controlled by BENCH_COUNT, default 5.
bench-examples:
	@cd _examples && $(GO) test -bench=. -count=$(BENCH_COUNT) -run=^a  ./... >../bench-examples-$(REF_NAME).txt
	@test -s $(GOPATH)/bin/benchstat || GO111MODULE=off GOFLAGS= GOBIN=$(GOPATH)/bin $(GO) get -u golang.org/x/perf/cmd/benchstat
	@test -s bench-examples-master.txt && benchstat bench-examples-master.txt bench-examples-$(REF_NAME).txt || benchstat bench-examples-$(REF_NAME).txt
