# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

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

test:
	@yarn install
	@yarn test	--no-watchAll

build:
	@yarn install
	@yarn build
