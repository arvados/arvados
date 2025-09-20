# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

# Use bash, and run all lines in each recipe as one shell command
SHELL := /bin/bash
.ONESHELL:

# IMPORTANT BACKGROUND: Because Workbench used to be a separate project, its
# build tools usually expect $WORKSPACE to be the Workbench source.
# $ARVADOS_DIRECTORY is the path to the full Arvados source tree.
# When you call other Arvados build tools, you'll probably need to set
# WORKSPACE=$(ARVADOS_DIRECTORY).
export WORKSPACE=$(PWD)
export ARVADOS_DIRECTORY=$(shell git rev-parse --show-toplevel)

APP_NAME?=arvados-workbench2
DOCKER_IMAGE=arvados/workbench
DOCKER_VOLUME=arvados-workbench

# Cypress test file that can be passed to the integration-test target
SPECFILE?=ALL

# VERSION uses all the above to produce X.Y.Z.timestamp
# something in the lines of 1.2.0.20180612145021, this will be the package version
VERSION?=$(shell ./version-at-commit.sh HEAD)
# We don't use BUILD_NUMBER at the moment, but it needs to be defined
BUILD_NUMBER?=0
GIT_COMMIT?=$(shell git rev-parse --short HEAD)

export CYPRESS_FAIL_FAST_ENABLED?=false

ifndef ci
	TI=-ti
else
	TI=
endif

.PHONY: help clean* *-install test build *-tests* workbench-docker-*

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

clean-docker-image:
	if docker image inspect "$(DOCKER_IMAGE)" >/dev/null; \
	then docker image rm "$(DOCKER_IMAGE)"; \
	else :; fi

clean-docker-volume:
	if docker volume inspect "$(DOCKER_VOLUME)" >/dev/null; \
	then docker volume rm "$(DOCKER_VOLUME)"; \
	else :; fi

clean: clean-docker-image clean-docker-volume clean-node-modules

yarn-install:
	yarn install

cypress-install: yarn-install
	yarn run cypress install && yarn run cypress verify

unit-tests: yarn-install
	yarn test

integration-tests: cypress-install
ifeq ($(SPECFILE), ALL)
	$(WORKSPACE)/tools/run-integration-tests.sh $(INTERACTIVE)
else
	$(WORKSPACE)/tools/run-integration-tests.sh $(INTERACTIVE) -- --spec $(SPECFILE)
endif

integration-tests-in-docker: workbench-docker-volume
	docker run $(TI) --rm \
		--mount="type=volume,src=$(DOCKER_VOLUME),dst=/home" \
		--mount="type=bind,src=$(ARVADOS_DIRECTORY),dst=/opt/arvados" \
		--workdir=/opt/arvados/services/workbench2 \
		"$(DOCKER_IMAGE)" \
		bash tools/run-integration-tests.sh

unit-tests-in-docker: workbench-docker-volume
	docker run $(TI) --rm \
		--mount="type=volume,src=$(DOCKER_VOLUME),dst=/home" \
		--mount="type=bind,src=$(ARVADOS_DIRECTORY),dst=/opt/arvados" \
		--workdir=/opt/arvados/services/workbench2 \
		"$(DOCKER_IMAGE)" \
		yarn test

tests-in-docker: unit-tests-in-docker integration-tests-in-docker

shell-in-docker: workbench-docker-image
	docker run $(TI) --rm \
		--mount="type=volume,src=$(DOCKER_VOLUME),dst=/home" \
		--mount="type=bind,src=$(ARVADOS_DIRECTORY),dst=/opt/arvados" \
		--workdir=/opt/arvados/services/workbench2 \
		"$(DOCKER_IMAGE)"

test: unit-tests integration-tests

build: yarn-install
	VERSION=$(VERSION) BUILD_NUMBER=$(BUILD_NUMBER) GIT_COMMIT=$(GIT_COMMIT) yarn build

workbench-docker-image: yarn-install
	if ! docker image inspect "$(DOCKER_IMAGE)" >/dev/null; then \
		env -C "$(ARVADOS_DIRECTORY)/tools/ansible" \
			WORKSPACE="$(ARVADOS_DIRECTORY)" \
			ansible-playbook \
			--extra-vars=arvados_build_playbook=install-dev-tools.yml \
			--inventory=files/development-docker-images.yml \
			--limit=arvados_workbench \
			build-docker-image.yml; \
	fi

workbench-docker-volume: workbench-docker-image
	if docker volume inspect "$(DOCKER_VOLUME)" >/dev/null; then exit; fi; \
	docker volume create "$(DOCKER_VOLUME)" || exit; \
	if ! docker run --interactive --rm \
		--mount="type=bind,src=$(ARVADOS_DIRECTORY),dst=/opt/arvados" \
		--mount="type=volume,src=$(DOCKER_VOLUME),dst=/mnt" \
		--user=root \
		"$(DOCKER_IMAGE)" \
		bash /opt/arvados/services/workbench2/tools/setup-docker-volume.sh </dev/null; \
	then docker volume rm "$(DOCKER_VOLUME)"; exit 73; fi
