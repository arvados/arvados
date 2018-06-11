# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

APP_NAME?=arvados-workbench2
#Get version from the latest tag plus timsetamp
GIT_TAG?=$(shell git describe --abbrev=0)
TS_GIT?=$(shell git log -n1 --first-parent "--format=format:%ct" .)
DATE_FROM_TS_GIT?=$(shell date -ud @$(TS_GIT) +%Y%m%d%H%M%S)
CI_VERSION?="$(GIT_TAG).$(DATE_FROM_TS_GIT)"	
export WORKSPACE?=$(shell pwd)

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
clean:
	@rm -f $(WORKSPACE)/*.deb
	@rm -f $(WORKSPACE)/*.rpm
test:
	@yarn install
	@yarn test	--no-watchAll --bail --ci

build:
	@yarn install
	@yarn build

package-version: build
	# Build deb and rpm packages using fpm from dist passing the destination folder for the deploy to be /var/www/arvados-workbench2/
	@fpm -s dir -t deb  -n "$(APP_NAME)" -v "$(VERSION)" "--maintainer=Ward Vandewege <ward@curoverse.com>" --description "workbench2 Package" --deb-no-default-config-files $(WORKSPACE)/build/=/var/www/arvados-workbench2/workbench2/
	@fpm -s dir -t rpm  -n "$(APP_NAME)" -v "$(VERSION)" "--maintainer=Ward Vandewege <ward@curoverse.com>" --description "workbench2 Package" $(WORKSPACE)/build/=/var/www/arvados-workbench2/workbench2/

package-no-version: build
	# Build deb and rpm packages using fpm from dist passing the destination folder for the deploy to be /var/www/arvados-workbench2/
	@fpm -s dir -t deb  -n "$(APP_NAME)" -v "$(CI_VERSION)" "--maintainer=Ward Vandewege <ward@curoverse.com>" --description "workbench2 Package" --deb-no-default-config-files $(WORKSPACE)/build/=/var/www/arvados-workbench2/workbench2/
	@fpm -s dir -t rpm  -n "$(APP_NAME)" -v "$(CI_VERSION)" "--maintainer=Ward Vandewege <ward@curoverse.com>" --description "workbench2 Package" $(WORKSPACE)/build/=/var/www/arvados-workbench2/workbench2/
