# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

context("Http Request")


test_that("execyte raises exception if http verb is not valid", {

    http <- HttpRequest$new()
    expect_that(http$exec("FAKE VERB", "url"),
               throws_error("Http verb is not valid."))
}) 

test_that("createQuery generates and encodes query portion of http", {

    http <- HttpRequest$new()
    queryParams <- list()
    queryParams$filters <- list(list("color", "=", "red"))
    queryParams$limit <- 20
    queryParams$offset <- 50
    expect_that(http$createQuery(queryParams),
                equals(paste0("/?filters=%5B%5B%22color%22%2C%22%3D%22%2C%22red",
                              "%22%5D%5D&limit=20&offset=50")))
}) 

test_that("createQuery generates and empty string when queryParams is an empty list", {

    http <- HttpRequest$new()
    expect_that(http$createQuery(list()), equals(""))
}) 
