options(repos=structure(c(CRAN="http://cran.wustl.edu/")))
if (!requireNamespace("devtools")) {
  install.packages("devtools")
}
if (!requireNamespace("roxygen2")) {
  install.packages("roxygen2")
}

# These install from github so install known-good versions instead of
# letting any push to master break our build.
if (!requireNamespace("pkgload")) {
  devtools::install_github("r-lib/pkgload", ref="7a97de62adf1793c03e73095937e4655baad79c9")
}
if (!requireNamespace("pkgdown")) {
  devtools::install_github("r-lib/pkgdown", ref="897ffbc016549c11c4263cb5d1f6e9f5c99efb45")
}

devtools::install_dev_deps()
