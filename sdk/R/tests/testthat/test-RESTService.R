# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

source("fakes/FakeArvados.R")
source("fakes/FakeHttpRequest.R")
source("fakes/FakeHttpParser.R")

context("REST service")

test_that("getWebDavHostName calls REST service properly", {

    expectedURL <- "https://host/discovery/v1/apis/arvados/v1/rest"
    serverResponse <- list(keepWebServiceUrl = "https://myWebDavServer.com")
    httpRequest <- FakeHttpRequest$new(expectedURL, serverResponse)

    REST <- RESTService$new("token", "host",
                            httpRequest, FakeHttpParser$new())

    REST$getWebDavHostName()

    expect_that(httpRequest$URLIsProperlyConfigured, is_true())
    expect_that(httpRequest$requestHeaderContainsAuthorizationField, is_true())
    expect_that(httpRequest$numberOfGETRequests, equals(1))
})

test_that("getWebDavHostName returns webDAV host name properly", {

    serverResponse <- list(keepWebServiceUrl = "https://myWebDavServer.com")
    httpRequest <- FakeHttpRequest$new(expectedURL = NULL, serverResponse)

    REST <- RESTService$new("token", "host",
                            httpRequest, FakeHttpParser$new())

    expect_that("https://myWebDavServer.com", equals(REST$getWebDavHostName()))
})

test_that("create calls REST service properly", {

    uuid <- "aaaaa-j7d0g-ccccccccccccccc"
    expectedURL <- "https://webDavHost/c=aaaaa-j7d0g-ccccccccccccccc/file"
    fakeHttp <- FakeHttpRequest$new(expectedURL)
    fakeHttpParser <- FakeHttpParser$new()

    REST <- RESTService$new("token", "https://host/",
                            fakeHttp, fakeHttpParser,
                            0, "https://webDavHost/")

    REST$create("file", uuid)

    expect_that(fakeHttp$URLIsProperlyConfigured, is_true())
    expect_that(fakeHttp$requestHeaderContainsAuthorizationField, is_true())
    expect_that(fakeHttp$numberOfPUTRequests, equals(1))
})

test_that("create raises exception if server response code is not between 200 and 300", {

    uuid <- "aaaaa-j7d0g-ccccccccccccccc"
    response <- list()
    response$status_code <- 404
    fakeHttp <- FakeHttpRequest$new(serverResponse = response)

    REST <- RESTService$new("token", "https://host/",
                            fakeHttp, HttpParser$new(),
                            0, "https://webDavHost/")

    expect_that(REST$create("file", uuid),
                throws_error("Server code: 404"))
})

test_that("delete calls REST service properly", {

    uuid <- "aaaaa-j7d0g-ccccccccccccccc"
    expectedURL <- "https://webDavHost/c=aaaaa-j7d0g-ccccccccccccccc/file"
    fakeHttp <- FakeHttpRequest$new(expectedURL)
    fakeHttpParser <- FakeHttpParser$new()

    REST <- RESTService$new("token", "https://host/",
                            fakeHttp, fakeHttpParser,
                            0, "https://webDavHost/")

    REST$delete("file", uuid)

    expect_that(fakeHttp$URLIsProperlyConfigured, is_true())
    expect_that(fakeHttp$requestHeaderContainsAuthorizationField, is_true())
    expect_that(fakeHttp$numberOfDELETERequests, equals(1))
})

test_that("delete raises exception if server response code is not between 200 and 300", {

    uuid <- "aaaaa-j7d0g-ccccccccccccccc"
    response <- list()
    response$status_code <- 404
    fakeHttp <- FakeHttpRequest$new(serverResponse = response)

    REST <- RESTService$new("token", "https://host/",
                            fakeHttp, HttpParser$new(),
                            0, "https://webDavHost/")

    expect_that(REST$delete("file", uuid),
                throws_error("Server code: 404"))
})

test_that("move calls REST service properly", {

    uuid <- "aaaaa-j7d0g-ccccccccccccccc"
    expectedURL <- "https://webDavHost/c=aaaaa-j7d0g-ccccccccccccccc/file"
    fakeHttp <- FakeHttpRequest$new(expectedURL)
    fakeHttpParser <- FakeHttpParser$new()

    REST <- RESTService$new("token", "https://host/",
                            fakeHttp, fakeHttpParser,
                            0, "https://webDavHost/")

    REST$move("file", "newDestination/file", uuid)

    expect_that(fakeHttp$URLIsProperlyConfigured, is_true())
    expect_that(fakeHttp$requestHeaderContainsAuthorizationField, is_true())
    expect_that(fakeHttp$requestHeaderContainsDestinationField, is_true())
    expect_that(fakeHttp$numberOfMOVERequests, equals(1))
})

test_that("move raises exception if server response code is not between 200 and 300", {

    uuid <- "aaaaa-j7d0g-ccccccccccccccc"
    response <- list()
    response$status_code <- 404
    fakeHttp <- FakeHttpRequest$new(serverResponse = response)

    REST <- RESTService$new("token", "https://host/",
                            fakeHttp, HttpParser$new(),
                            0, "https://webDavHost/")

    expect_that(REST$move("file", "newDestination/file", uuid),
                throws_error("Server code: 404"))
})

test_that("copy calls REST service properly", {

    uuid <- "aaaaa-j7d0g-ccccccccccccccc"
    expectedURL <- "https://webDavHost/c=aaaaa-j7d0g-ccccccccccccccc/file"
    fakeHttp <- FakeHttpRequest$new(expectedURL)
    fakeHttpParser <- FakeHttpParser$new()

    REST <- RESTService$new("token", "https://host/",
                            fakeHttp, fakeHttpParser,
                            0, "https://webDavHost/")

    REST$copy("file", "newDestination/file", uuid)

    expect_that(fakeHttp$URLIsProperlyConfigured, is_true())
    expect_that(fakeHttp$requestHeaderContainsAuthorizationField, is_true())
    expect_that(fakeHttp$requestHeaderContainsDestinationField, is_true())
    expect_that(fakeHttp$numberOfCOPYRequests, equals(1))
})

test_that("copy raises exception if server response code is not between 200 and 300", {

    uuid <- "aaaaa-j7d0g-ccccccccccccccc"
    response <- list()
    response$status_code <- 404
    fakeHttp <- FakeHttpRequest$new(serverResponse = response)

    REST <- RESTService$new("token", "https://host/",
                            fakeHttp, HttpParser$new(),
                            0, "https://webDavHost/")

    expect_that(REST$copy("file", "newDestination/file", uuid),
                throws_error("Server code: 404"))
})

test_that("getCollectionContent retreives correct content from WebDAV server", {

    uuid <- "aaaaa-j7d0g-ccccccccccccccc"
    expectedURL <- "https://webDavHost/c=aaaaa-j7d0g-ccccccccccccccc"
    returnContent <- list()
    returnContent$status_code <- 200
    returnContent$content <- c("animal", "animal/dog", "ball")

    fakeHttp <- FakeHttpRequest$new(expectedURL, returnContent)

    REST <- RESTService$new("token", "https://host/",
                            fakeHttp, FakeHttpParser$new(),
                            0, "https://webDavHost/")

    returnResult <- REST$getCollectionContent(uuid)
    returnedContentMatchExpected <- all.equal(returnResult,
                                              c("animal", "animal/dog", "ball"))

    expect_that(returnedContentMatchExpected, is_true())
    expect_that(fakeHttp$requestHeaderContainsAuthorizationField, is_true())
})

test_that("getCollectionContent raises exception if server returns empty response", {

    uuid <- "aaaaa-j7d0g-ccccccccccccccc"
    response <- ""
    fakeHttp <- FakeHttpRequest$new(serverResponse = response)

    REST <- RESTService$new("token", "https://host/",
                            fakeHttp, FakeHttpParser$new(),
                            0, "https://webDavHost/")

    expect_that(REST$getCollectionContent(uuid),
                throws_error("Response is empty, request may be misconfigured"))
})

test_that("getCollectionContent parses server response", {

    uuid <- "aaaaa-j7d0g-ccccccccccccccc"
    fakeHttpParser <- FakeHttpParser$new()
    REST <- RESTService$new("token", "https://host/",
                            FakeHttpRequest$new(), fakeHttpParser,
                            0, "https://webDavHost/")

    REST$getCollectionContent(uuid)

    expect_that(fakeHttpParser$parserCallCount, equals(1))
})

test_that("getCollectionContent raises exception if server returns empty response", {

    uuid <- "aaaaa-j7d0g-ccccccccccccccc"
    response <- ""
    fakeHttp <- FakeHttpRequest$new(serverResponse = response)

    REST <- RESTService$new("token", "https://host/",
                            fakeHttp, FakeHttpParser$new(),
                            0, "https://webDavHost/")

    expect_that(REST$getCollectionContent(uuid),
                throws_error("Response is empty, request may be misconfigured"))
})

test_that(paste("getCollectionContent raises exception if server",
                "response code is not between 200 and 300"), {

    uuid <- "aaaaa-j7d0g-ccccccccccccccc"
    response <- list()
    response$status_code <- 404
    fakeHttp <- FakeHttpRequest$new(serverResponse = response)

    REST <- RESTService$new("token", "https://host/",
                            fakeHttp, HttpParser$new(),
                            0, "https://webDavHost/")

    expect_that(REST$getCollectionContent(uuid),
                throws_error("Server code: 404"))
})


test_that("getResourceSize calls REST service properly", {

    uuid <- "aaaaa-j7d0g-ccccccccccccccc"
    expectedURL <- "https://webDavHost/c=aaaaa-j7d0g-ccccccccccccccc/file"
    response <- list()
    response$status_code <- 200
    response$content <- c(6, 2, 931, 12003)
    fakeHttp <- FakeHttpRequest$new(expectedURL, response)

    REST <- RESTService$new("token", "https://host/",
                            fakeHttp, FakeHttpParser$new(),
                            0, "https://webDavHost/")

    returnResult <- REST$getResourceSize("file", uuid)
    returnedContentMatchExpected <- all.equal(returnResult,
                                              c(6, 2, 931, 12003))

    expect_that(fakeHttp$URLIsProperlyConfigured, is_true())
    expect_that(fakeHttp$requestHeaderContainsAuthorizationField, is_true())
    expect_that(returnedContentMatchExpected, is_true())
})

test_that("getResourceSize raises exception if server returns empty response", {

    uuid <- "aaaaa-j7d0g-ccccccccccccccc"
    response <- ""
    fakeHttp <- FakeHttpRequest$new(serverResponse = response)

    REST <- RESTService$new("token", "https://host/",
                            fakeHttp, FakeHttpParser$new(),
                            0, "https://webDavHost/")

    expect_that(REST$getResourceSize("file", uuid),
                throws_error("Response is empty, request may be misconfigured"))
})

test_that(paste("getResourceSize raises exception if server",
                "response code is not between 200 and 300"), {

    uuid <- "aaaaa-j7d0g-ccccccccccccccc"
    response <- list()
    response$status_code <- 404
    fakeHttp <- FakeHttpRequest$new(serverResponse = response)

    REST <- RESTService$new("token", "https://host/",
                            fakeHttp, HttpParser$new(),
                            0, "https://webDavHost/")

    expect_that(REST$getResourceSize("file", uuid),
                throws_error("Server code: 404"))
})

test_that("getResourceSize parses server response", {

    uuid <- "aaaaa-j7d0g-ccccccccccccccc"
    fakeHttpParser <- FakeHttpParser$new()
    REST <- RESTService$new("token", "https://host/",
                            FakeHttpRequest$new(), fakeHttpParser,
                            0, "https://webDavHost/")

    REST$getResourceSize("file", uuid)

    expect_that(fakeHttpParser$parserCallCount, equals(1))
})

test_that("read calls REST service properly", {

    uuid <- "aaaaa-j7d0g-ccccccccccccccc"
    expectedURL <- "https://webDavHost/c=aaaaa-j7d0g-ccccccccccccccc/file"
    serverResponse <- list()
    serverResponse$status_code <- 200
    serverResponse$content <- "file content"

    fakeHttp <- FakeHttpRequest$new(expectedURL, serverResponse)

    REST <- RESTService$new("token", "https://host/",
                            fakeHttp, FakeHttpParser$new(),
                            0, "https://webDavHost/")

    returnResult <- REST$read("file", uuid, "text", 1024, 512)

    expect_that(fakeHttp$URLIsProperlyConfigured, is_true())
    expect_that(fakeHttp$requestHeaderContainsAuthorizationField, is_true())
    expect_that(fakeHttp$requestHeaderContainsRangeField, is_true())
    expect_that(returnResult, equals("file content"))
})

test_that("read raises exception if server response code is not between 200 and 300", {

    uuid <- "aaaaa-j7d0g-ccccccccccccccc"
    response <- list()
    response$status_code <- 404
    fakeHttp <- FakeHttpRequest$new(serverResponse = response)

    REST <- RESTService$new("token", "https://host/",
                            fakeHttp, HttpParser$new(),
                            0, "https://webDavHost/")

    expect_that(REST$read("file", uuid),
                throws_error("Server code: 404"))
})

test_that("read raises exception if contentType is not valid", {

    uuid <- "aaaaa-j7d0g-ccccccccccccccc"
    fakeHttp <- FakeHttpRequest$new()

    REST <- RESTService$new("token", "https://host/",
                            fakeHttp, HttpParser$new(),
                            0, "https://webDavHost/")

    expect_that(REST$read("file", uuid, "some invalid content type"),
                throws_error("Invalid contentType. Please use text or raw."))
})

test_that("read parses server response", {

    uuid <- "aaaaa-j7d0g-ccccccccccccccc"
    fakeHttpParser <- FakeHttpParser$new()
    REST <- RESTService$new("token", "https://host/",
                            FakeHttpRequest$new(), fakeHttpParser,
                            0, "https://webDavHost/")

    REST$read("file", uuid, "text", 1024, 512)

    expect_that(fakeHttpParser$parserCallCount, equals(1))
})

test_that("write calls REST service properly", {

    fileContent <- "new file content"
    uuid <- "aaaaa-j7d0g-ccccccccccccccc"
    expectedURL <- "https://webDavHost/c=aaaaa-j7d0g-ccccccccccccccc/file"
    fakeHttp <- FakeHttpRequest$new(expectedURL)

    REST <- RESTService$new("token", "https://host/",
                            fakeHttp, FakeHttpParser$new(),
                            0, "https://webDavHost/")

    REST$write("file", uuid, fileContent, "text/html")

    expect_that(fakeHttp$URLIsProperlyConfigured, is_true())
    expect_that(fakeHttp$requestBodyIsProvided, is_true())
    expect_that(fakeHttp$requestHeaderContainsAuthorizationField, is_true())
    expect_that(fakeHttp$requestHeaderContainsContentTypeField, is_true())
})

test_that("write raises exception if server response code is not between 200 and 300", {

    uuid <- "aaaaa-j7d0g-ccccccccccccccc"
    fileContent <- "new file content"
    response <- list()
    response$status_code <- 404
    fakeHttp <- FakeHttpRequest$new(serverResponse = response)

    REST <- RESTService$new("token", "https://host/",
                            fakeHttp, HttpParser$new(),
                            0, "https://webDavHost/")

    expect_that(REST$write("file", uuid, fileContent, "text/html"),
                throws_error("Server code: 404"))
})

test_that("getConnection calls REST service properly", {
    uuid <- "aaaaa-j7d0g-ccccccccccccccc"
    fakeHttp <- FakeHttpRequest$new()

    REST <- RESTService$new("token", "https://host/",
                            fakeHttp, FakeHttpParser$new(),
                            0, "https://webDavHost/")

    REST$getConnection("file", uuid, "r")

    expect_that(fakeHttp$numberOfgetConnectionCalls, equals(1))
})
