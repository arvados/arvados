# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

.onLoad <- function(libName, pkgName)
{
    minAllowedRVersion <- "3.3.0"
    currentRVersion <- getRversion()

    if(currentRVersion < minAllowedRVersion)
        print(paste0("Minimum R version required to run ", pkgName, " is ",
                     minAllowedRVersion, ". Your current version is ",
                     toString(currentRVersion), ". Please update R and try again."))
}
