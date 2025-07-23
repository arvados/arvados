# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

context("Http Request")


test_that("execute raises exception if http verb is not valid", {

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
                equals(paste0("?filters=%5B%5B%22color%22%2C%22%3D%22%2C%22red",
                              "%22%5D%5D&limit=20&offset=50")))
})

test_that("createQuery generates and empty string when queryParams is an empty list", {

    http <- HttpRequest$new()
    expect_that(http$createQuery(list()), equals(""))
})

test_that("exec calls httr functions correctly", {
    httrNamespace <- getNamespace("httr")

    # Monkeypatch httr functions and assert that they are called later
    add_headersCalled <- FALSE
    unlockBinding("add_headers", httrNamespace)
    newAddHeaders <- function(h)
    {
        add_headersCalled <<- TRUE
        list()
    }
    httrNamespace$add_headers <- newAddHeaders
    lockBinding("add_headers", httrNamespace)

    expectedConfig <- list()
    retryCalled <- FALSE
    unlockBinding("RETRY", httrNamespace)
    newRETRY <- function(verb, url, body, config, times)
    {
        retryCalled <<- TRUE
        expectedConfig <<- config
    }
    httrNamespace$RETRY <- newRETRY
    lockBinding("RETRY", httrNamespace)

    Sys.setenv("ARVADOS_API_HOST_INSECURE" = TRUE)
    http <- HttpRequest$new()
    http$exec("GET", "url")

    expect_true(add_headersCalled)
    expect_true(retryCalled)
    expect_that(expectedConfig$options, equals(list(ssl_verifypeer = 0L)))
})

test_that("getConnection calls curl functions correctly", {
    curlNamespace <- getNamespace("curl")

    # Monkeypatch curl functions and assert that they are called later
    curlCalled <- FALSE
    unlockBinding("curl", curlNamespace)
    newCurl <- function(url, open, handle) curlCalled <<- TRUE
    curlNamespace$curl <- newCurl
    lockBinding("curl", curlNamespace)

    new_handleCalled <- FALSE
    unlockBinding("new_handle", curlNamespace)
    newHandleFun <- function()
    {
        new_handleCalled <<- TRUE
        list()
    }
    curlNamespace$new_handle <- newHandleFun
    lockBinding("new_handle", curlNamespace)

    handle_setheadersCalled <- FALSE
    unlockBinding("handle_setheaders", curlNamespace)
    newHandleSetHeaders <- function(h, .list) handle_setheadersCalled <<- TRUE
    curlNamespace$handle_setheaders <- newHandleSetHeaders
    lockBinding("handle_setheaders", curlNamespace)

    handle_setoptCalled <- FALSE
    unlockBinding("handle_setopt", curlNamespace)
    newHandleSetOpt <- function(h, ssl_verifypeer) handle_setoptCalled <<- TRUE
    curlNamespace$handle_setopt <- newHandleSetOpt
    lockBinding("handle_setopt", curlNamespace)


    Sys.setenv("ARVADOS_API_HOST_INSECURE" = TRUE)
    http <- HttpRequest$new()
    http$getConnection("location", list(), "r")

    expect_true(new_handleCalled)
    expect_true(handle_setheadersCalled)
    expect_true(handle_setoptCalled)
    expect_true(curlCalled)
})
