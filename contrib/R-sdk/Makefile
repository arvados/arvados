# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

# NOTE: `R CMD check` (and by extension, the Arvados test suite)
# will carp at you if you use GNU extensions.
# <https://cran.r-project.org/doc/manuals/R-exts.html#Writing-portable-packages>

ALL=R/Arvados.R man
SDK_VERSION!=awk '($$1 == "Version:"){v=$$2} END {print v}' DESCRIPTION

all: $(ALL)

.PHONY: api
api: R/Arvados.R
R/Arvados.R: arvados-v1-discovery.json generateApi.R
	Rscript --vanilla generateApi.R

# Used by arvados/doc/Rakefile.
# Check whether we can load libraries necessary to build the package.
.PHONY: can_run
can_run:
	Rscript --vanilla -e "library(jsonlite); library(roxygen2);"

.PHONY: clean
clean:
	rm -rf $(ALL) "ArvadosR_$(SDK_VERSION).tar.gz"

.PHONY: install
install:
	R CMD INSTALL .

man: R/Arvados.R R/*.R
	Rscript --vanilla -e "library(roxygen2); roxygen2::roxygenize(clean=TRUE)"

.PHONY: package
package: "ArvadosR_$(SDK_VERSION).tar.gz"
"ArvadosR_$(SDK_VERSION).tar.gz": $(ALL) [A-Z]* *.R tests/*.R tests/testthat/*.R tests/testthat/fakes/*.R
	R CMD build .

.PHONY: test
test: $(ALL)
	Rscript --vanilla run_test.R
