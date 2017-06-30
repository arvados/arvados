# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

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
	build/run-tests.sh ${TEST_FLAGS}
packages:
	build/run-build-packages-all-targets.sh ${PACKAGES_FLAGS}
test-packages:
	build/run-build-packages-all-targets.sh --test-packages ${PACKAGES_FLAGS}
