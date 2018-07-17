# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

#Run script with $Rscript createDoc.R input.Rmd output.html

require(knitr) # required for knitting from rmd to md
require(markdown) # required for md to html

args <- commandArgs(TRUE)

if(length(args) != 2)
    stop("Please provide 2 arguments corresponding to input and output file!")

inputFile <- args[[1]] # .Rmd file
outputFile <- args[[2]] # .html file

# Create and fill temp .md file from existing .Rmd file
#tempMdFile <- tempfile("tempREADME", fileext = "md")
knitr::knit(inputFile, outputFile)
#knitr::knit(inputFile, tempMdFile)

# Generate HTML from temporary .md file
#markdown::markdownToHTML(tempMdFile, outputFile)
