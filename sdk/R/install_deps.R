# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

options(repos=structure(c(CRAN="http://cran.wustl.edu/")))
if (!requireNamespace("devtools")) {
  install.packages("devtools")
}
if (!requireNamespace("roxygen2")) {
  install.packages("roxygen2")
}
if (!requireNamespace("knitr")) {
  install.packages("knitr")
}
if (!requireNamespace("markdown")) {
  install.packages("markdown")
}
if (!requireNamespace("XML")) {
  # XML 3.99-0.4 depends on R >= 4.0.0, but we run tests on debian
  # stable (10) with R 3.5.2 so we install an older version from
  # source.
  install.packages("https://cran.r-project.org/src/contrib/Archive/XML/XML_3.99-0.3.tar.gz", repos=NULL, type="source")
}

devtools::install_dev_deps()
