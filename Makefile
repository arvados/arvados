# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

# Use bash, and run all lines in each recipe as one shell command
SHELL := /bin/bash
.ONESHELL:

APP_NAME?=arvados-workbench2

# GIT_TAG is the last tagged stable release (i.e. 1.2.0)
GIT_TAG?=$(shell git describe --abbrev=0)

# TS_GIT is the timestamp in the current directory (i.e. 1528815021).
# Note that it will only change if files change.
TS_GIT?=$(shell git log -n1 --first-parent "--format=format:%ct" .)

# DATE_FROM_TS_GIT is the human(ish)-readable version of TS_GIT
# 1528815021 -> 20180612145021
DATE_FROM_TS_GIT?=$(shell date -ud @$(TS_GIT) +%Y%m%d%H%M%S)

# VERSION uses all the above to produce X.Y.Z.timestamp
# something in the lines of 1.2.0.20180612145021, this will be the package version
# it can be overwritten when invoking make as in make packages VERSION=1.2.0
VERSION?=$(GIT_TAG).$(DATE_FROM_TS_GIT)

# ITERATION is the package iteration, intended for manual change if anything non-code related
# changes in the package. (i.e. example config files externally added
ITERATION?=1

DESCRIPTION=Arvados Workbench2 - Arvados is a free and open source platform for big data science.
MAINTAINER=Ward Vandewege <wvandewege@veritasgenetics.com>

# DEST_DIR will have the build package copied.
DEST_DIR=/var/www/arvados-workbench2/workbench2/

# Debian package file
DEB_FILE=$(APP_NAME)_$(VERSION)-$(ITERATION)_amd64.deb

# redHat package file
RPM_FILE=$(APP_NAME)-$(VERSION)-$(ITERATION).x86_64.rpm

export WORKSPACE?=$(shell pwd)

.PHONY: help clean* yarn-install test build packages packages-with-version 

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

yarn-install:
	yarn install

test: yarn-install
	yarn test	--no-watchAll --bail --ci

build: test
	yarn build

$(DEB_FILE): build
	fpm \
	 -s dir \
	 -t deb \
	 -n "$(APP_NAME)" \
	 -v "$(VERSION)" \
	 --iteration "$(ITERATION)" \
	 --maintainer="$(MAINTAINER)" \
	 --description="$(DESCRIPTION)" \
	 --config-files="etc/arvados/workbench2/workbench2.example.json" \
	$(WORKSPACE)/build/=$(DEST_DIR)

$(RPM_FILE): build
	fpm \
	 -s dir \
	 -t rpm \
	 -n "$(APP_NAME)" \
	 -v "$(VERSION)" \
	 --iteration "$(ITERATION)" \
	 --maintainer="$(MAINTAINER)" \
	 --description="$(DESCRIPTION)" \
	 --config-files="etc/arvados/workbench2/workbench2.example.json" \
	 $(WORKSPACE)/build/=$(DEST_DIR)

copy: $(DEB_FILE) $(RPM_FILE)
	for target in $(TARGETS); do \
		if [[ $$target =~ ^centos ]]; then
			cp -p $(RPM_FILE) packages/$$target ; \
		else
			cp -p $(DEB_FILE) packages/$$target ; \
		fi
	done
	rm -f $(RPM_FILE)
	rm -f $(DEB_FILE)

# use FPM to create DEB and RPM
packages: copy
