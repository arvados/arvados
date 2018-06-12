# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

APP_NAME?=arvados-workbench2

# GIT_TAG is the last tagged stable release (i.e. 1.2.0)
GIT_TAG?=$(shell git describe --abbrev=0)

# TS_GIT is the timestamp in the current directory (i.e. 1528815021).
# Note that it will only change if files change.
TS_GIT?=$(shell git log -n1 --first-parent "--format=format:%ct" .)

# DATE_FROM_TS_GIT is the human(ish)-readable version of TS_GIT
# 1528815021 -> 20180612145021
DATE_FROM_TS_GIT?=$(shell date -ud @$(TS_GIT) +%Y%m%d%H%M%S)

# NIGHTLY_VERSION uses all the above to produce X.Y.Z.timestamp
# something in the lines of 1.2.0.20180612145021, this will be the package version
NIGHTLY_VERSION?=$(GIT_TAG).$(DATE_FROM_TS_GIT)

DESCRIPTION=Arvados Workbench2 - Arvados is a free and open source platform for big data science.
MAINTAINER=Ward Vandewege <wvandewege@veritasgenetics.com>

# DEST_DIR will have the build package copied.
DEST_DIR=/var/www/arvados-workbench2/workbench2/

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

build: yarn-install test
	yarn build

# use FPM to create DEB and RPM with a version (usually triggered from CI to make a release)
packages-with-version: build
	fpm \
	 -s dir \
	 -t deb \
	 -n "$(APP_NAME)" \
	 -v "$(VERSION)" \
	 --maintainer="$(MAINTAINER)" \
	 --description="$(DESCRIPTION)" \
	 --deb-no-default-config-files \
	$(WORKSPACE)/build/=DEST_DIR
	fpm \
	 -s dir \
	 -t rpm \
	 -n "$(APP_NAME)" \
	 -v "$(VERSION)" \
	 --maintainer="$(MAINTAINER)" \
	 --description="$(DESCRIPTION)" \
	 $(WORKSPACE)/build/=DEST_DIR

# use FPM to create DEB and RPM
packages: build
	fpm \
	 -s dir \
	 -t deb \
	 -n "$(APP_NAME)" \
	 -v "$(NIGHTLY_VERSION)" \
	 --maintainer="$(MAINTAINER)" \
	 --description="$(DESCRIPTION)" \
	 --deb-no-default-config-files \
	$(WORKSPACE)/build/=DEST_DIR
	fpm \
	 -s dir \
	 -t rpm \
	 -n "$(APP_NAME)" \
	 -v "$(NIGHTLY_VERSION)" \
	 --maintainer="$(MAINTAINER)" \
	 --description="$(DESCRIPTION)" \
	 $(WORKSPACE)/build/=DEST_DIR
