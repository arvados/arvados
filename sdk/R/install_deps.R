# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

options(repos=structure(c(CRAN="https://cloud.r-project.org/")))
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
  install.packages("XML")
}

devtools::install_dev_deps()
