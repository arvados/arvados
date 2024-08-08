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
# it can be overwritten when invoking make as in make packages VERSION=1.2.0
VERSION?=$(shell ./version-at-commit.sh HEAD)
# We don't use BUILD_NUMBER at the moment, but it needs to be defined
BUILD_NUMBER?=0
GIT_COMMIT?=$(shell git rev-parse --short HEAD)

# ITERATION is the package iteration, intended for manual change if anything non-code related
# changes in the package. (i.e. example config files externally added
ITERATION?=1

TARGETS?=rocky8 debian11 debian12 ubuntu2004 ubuntu2204

DESCRIPTION=Arvados Workbench2 - Arvados is a free and open source platform for big data science.
MAINTAINER=Arvados Package Maintainers <packaging@arvados.org>

# DEST_DIR will have the build package copied.
DEST_DIR=/var/www/$(APP_NAME)/workbench2/

# Debian package file
DEB_FILE=$(APP_NAME)_$(VERSION)-$(ITERATION)_amd64.deb

# redHat package file
RPM_FILE=$(APP_NAME)-$(VERSION)-$(ITERATION).x86_64.rpm

GOPATH=$(shell go env GOPATH)
export WORKSPACE?=$(shell pwd)

ARVADOS_DIRECTORY?=$(shell env -C $(WORKSPACE) git rev-parse --show-toplevel)

ifndef ci
	TI=-ti
else
	TI=
endif

.PHONY: help clean* yarn-install test build packages packages-with-version integration-tests-in-docker

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

clean-deb:
	rm -f $(WORKSPACE)/*.deb

clean-rpm:
	rm -f $(WORKSPACE)/*.rpm

clean-node-modules:
	rm -rf $(WORKSPACE)/node_modules

clean: clean-rpm clean-deb clean-node-modules

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

unit-tests: yarn-install
	yarn test

integration-tests: yarn-install check-arvados-directory
	yarn run cypress install
ifeq ($(SPECFILE), ALL)
	$(WORKSPACE)/tools/run-integration-tests.sh -a $(ARVADOS_DIRECTORY)
else
	$(WORKSPACE)/tools/run-integration-tests.sh -a $(ARVADOS_DIRECTORY) -- --spec $(SPECFILE)
endif

integration-tests-in-docker: workbench2-build-image check-arvados-directory
	docker run $(TI) --rm \
		--env ARVADOS_DIRECTORY=/usr/src/arvados \
		--env GIT_DISCOVERY_ACROSS_FILESYSTEM=1 \
		-v $(WORKSPACE):/usr/src/arvados/services/workbench2 \
		-v $(ARVADOS_DIRECTORY):/usr/src/arvados \
		-w /usr/src/arvados/services/workbench2 \
		workbench2-build \
		make arvados-server-install integration-tests SPECFILE=$(SPECFILE)

unit-tests-in-docker: workbench2-build-image check-arvados-directory
	docker run $(TI) --rm \
		--env ARVADOS_DIRECTORY=/usr/src/arvados \
		--env GIT_DISCOVERY_ACROSS_FILESYSTEM=1 \
		-v $(WORKSPACE):/usr/src/arvados/services/workbench2 \
		-v $(ARVADOS_DIRECTORY):/usr/src/arvados \
		-w /usr/src/arvados/services/workbench2 \
		workbench2-build \
		make arvados-server-install unit-tests

tests-in-docker: workbench2-build-image check-arvados-directory
	docker run $(TI) --rm \
		--env ARVADOS_DIRECTORY=/usr/src/arvados \
		--env GIT_DISCOVERY_ACROSS_FILESYSTEM=1 \
		--env ci="${ci}" \
		-v $(WORKSPACE):/usr/src/arvados/services/workbench2 \
		-v$(ARVADOS_DIRECTORY):/usr/src/arvados \
		-w /usr/src/arvados/services/workbench2 \
		workbench2-build \
		make test

test: unit-tests integration-tests

build: yarn-install
	VERSION=$(VERSION) BUILD_NUMBER=$(BUILD_NUMBER) GIT_COMMIT=$(GIT_COMMIT) yarn build

$(DEB_FILE): build
	fpm \
	 -s dir \
	 -t deb \
	 -n "$(APP_NAME)" \
	 -v "$(VERSION)" \
	 --iteration "$(ITERATION)" \
	 --vendor="The Arvados Authors" \
	 --maintainer="$(MAINTAINER)" \
	 --url="https://arvados.org" \
	 --license="GNU Affero General Public License, version 3.0" \
	 --description="$(DESCRIPTION)" \
	 --config-files="etc/arvados/$(APP_NAME)/workbench2.example.json" \
	$(WORKSPACE)/build/=$(DEST_DIR) \
	etc/arvados/workbench2/workbench2.example.json=/etc/arvados/$(APP_NAME)/workbench2.example.json

$(RPM_FILE): build
	fpm \
	 -s dir \
	 -t rpm \
	 -n "$(APP_NAME)" \
	 -v "$(VERSION)" \
	 --iteration "$(ITERATION)" \
	 --vendor="The Arvados Authors" \
	 --maintainer="$(MAINTAINER)" \
	 --url="https://arvados.org" \
	 --license="GNU Affero General Public License, version 3.0" \
	 --description="$(DESCRIPTION)" \
	 --config-files="etc/arvados/$(APP_NAME)/workbench2.example.json" \
	 $(WORKSPACE)/build/=$(DEST_DIR) \
	etc/arvados/workbench2/workbench2.example.json=/etc/arvados/$(APP_NAME)/workbench2.example.json

copy: $(DEB_FILE) $(RPM_FILE)
	for target in $(TARGETS); do \
		mkdir -p "packages/$$target" && \
		case "$$target" in \
			centos*|rocky*) cp -p "$(RPM_FILE)" "packages/$$target" ;; \
			debian*|ubuntu*) cp -p "$(DEB_FILE)" "packages/$$target" ;; \
			*) echo "Unknown copy target $$target"; exit 1 ;; \
		esac ; \
	done ; \
	rm -f "$(DEB_FILE)" "$(RPM_FILE)"

# use FPM to create DEB and RPM
packages: copy

check-arvados-directory:
	@if test "${ARVADOS_DIRECTORY}" == "unset"; then echo "the environment variable ARVADOS_DIRECTORY must be set to the path of an arvados git checkout"; exit 1; fi
	@if ! test -d "${ARVADOS_DIRECTORY}"; then echo "the environment variable ARVADOS_DIRECTORY does not point at a directory"; exit 1; fi

packages-in-docker: check-arvados-directory workbench2-build-image
	docker run -t --rm --env ci="true" \
		--env ARVADOS_DIRECTORY=/tmp/arvados \
		--env APP_NAME=${APP_NAME} \
		--env VERSION="${VERSION}" \
		--env ITERATION=${ITERATION} \
		--env TARGETS="${TARGETS}" \
		--env MAINTAINER="${MAINTAINER}" \
		--env DESCRIPTION="${DESCRIPTION}" \
		--env GIT_DISCOVERY_ACROSS_FILESYSTEM=1 \
		-w "/tmp/workbench2" \
		-v ${WORKSPACE}:/tmp/workbench2 \
		-v ${ARVADOS_DIRECTORY}:/tmp/arvados \
		workbench2-build:latest \
		sh -c 'git config --global --add safe.directory /tmp/workbench2 && make packages'

workbench2-build-image:
	docker inspect workbench2-build &> /dev/null || \
		docker build -t workbench2-build -f docker/Dockerfile ${ARVADOS_DIRECTORY}
