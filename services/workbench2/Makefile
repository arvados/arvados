# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

# Use bash, and run all lines in each recipe as one shell command
SHELL := /bin/bash
.ONESHELL:

GOPATH?=~/go
APP_NAME?=arvados-workbench2

# Cypress test file that can be passed to the integration-test target
SPECFILE?=ALL

# VERSION uses all the above to produce X.Y.Z.timestamp
# something in the lines of 1.2.0.20180612145021, this will be the package version
VERSION?=$(shell ./version-at-commit.sh HEAD)
# We don't use BUILD_NUMBER at the moment, but it needs to be defined
BUILD_NUMBER?=0
GIT_COMMIT?=$(shell git rev-parse --short HEAD)

GOPATH=$(shell go env GOPATH)
export WORKSPACE?=$(shell pwd)

export CYPRESS_FAIL_FAST_ENABLED?=false

ARVADOS_DIRECTORY?=$(shell env -C $(WORKSPACE) git rev-parse --show-toplevel)

ifndef ci
	TI=-ti
else
	TI=
endif

.PHONY: help clean* *-install test build integration-tests-in-docker

help:
	@echo >&2
	@echo >&2 "There is no default make target here.  Did you mean 'make test'?"
	@echo >&2
	@echo >&2 "More info:"
	@echo >&2 "  Installing              --> http://doc.arvados.org/install"
	@echo >&2 "  Developing/contributing --> https://dev.arvados.org"
	@echo >&2 "  Project home            --> https://arvados.org"
	@echo >&2
	@false

clean-node-modules:
	rm -rf $(WORKSPACE)/node_modules

clean: clean-node-modules

arvados-server-install: check-arvados-directory
	cd $(ARVADOS_DIRECTORY)
	go mod download
	cd cmd/arvados-server
	echo GOPATH is $(GOPATH)
	GOFLAGS=-buildvcs=false go install
	cd -
	ls -l $(GOPATH)/bin/arvados-server
	$(GOPATH)/bin/arvados-server install -type test

yarn-install:
	yarn install

cypress-install: yarn-install
	yarn run cypress install && yarn run cypress verify

unit-tests: yarn-install
	yarn test

integration-tests: cypress-install check-docker-directory
ifeq ($(SPECFILE), ALL)
	$(WORKSPACE)/tools/run-integration-tests.sh $(INTERACTIVE) -a $(ARVADOS_DIRECTORY)
else
	$(WORKSPACE)/tools/run-integration-tests.sh $(INTERACTIVE) -a $(ARVADOS_DIRECTORY) -- --spec $(SPECFILE)
endif

integration-tests-in-docker: workbench2-build-image check-arvados-directory
	docker run $(TI) --rm \
		--env ARVADOS_DIRECTORY=/usr/src/arvados \
		--env GIT_DISCOVERY_ACROSS_FILESYSTEM=1 \
		--mount type=bind,src=$(WORKSPACE),dst=/usr/src/arvados/services/workbench2 \
		--mount type=bind,src=$(ARVADOS_DIRECTORY),dst=/usr/src/arvados \
		-w /usr/src/arvados/services/workbench2 \
		workbench2-build \
		make arvados-server-install integration-tests SPECFILE=$(SPECFILE)

unit-tests-in-docker: workbench2-build-image check-arvados-directory
	docker run $(TI) --rm \
		--env ARVADOS_DIRECTORY=/usr/src/arvados \
		--env GIT_DISCOVERY_ACROSS_FILESYSTEM=1 \
		--mount type=bind,src=$(WORKSPACE),dst=/usr/src/arvados/services/workbench2 \
		--mount type=bind,src=$(ARVADOS_DIRECTORY),dst=/usr/src/arvados \
		-w /usr/src/arvados/services/workbench2 \
		workbench2-build \
		make arvados-server-install unit-tests

tests-in-docker: workbench2-build-image check-arvados-directory
	docker run $(TI) --rm \
		--env ARVADOS_DIRECTORY=/usr/src/arvados \
		--env GIT_DISCOVERY_ACROSS_FILESYSTEM=1 \
		--env ci="${ci}" \
		--mount type=bind,src=$(WORKSPACE),dst=/usr/src/arvados/services/workbench2 \
		--mount type=bind,src=$(ARVADOS_DIRECTORY),dst=/usr/src/arvados \
		-w /usr/src/arvados/services/workbench2 \
		workbench2-build \
		make test

test: unit-tests integration-tests

build: yarn-install
	VERSION=$(VERSION) BUILD_NUMBER=$(BUILD_NUMBER) GIT_COMMIT=$(GIT_COMMIT) yarn build

check-arvados-directory:
	@if test "${ARVADOS_DIRECTORY}" == "unset"; then echo "the environment variable ARVADOS_DIRECTORY must be set to the path of an arvados git checkout"; exit 1; fi
	@if ! test -d "${ARVADOS_DIRECTORY}"; then echo "the environment variable ARVADOS_DIRECTORY does not point at a directory"; exit 1; fi

workbench2-build-image:
	docker inspect workbench2-build &> /dev/null || \
		docker build -t workbench2-build -f docker/Dockerfile ${ARVADOS_DIRECTORY}
