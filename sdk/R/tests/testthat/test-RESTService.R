source("fakes/FakeArvados.R")

context("REST service")

test_that("create calls REST service properly", {

    expectedURL <- "https://webdavHost/c=aaaaa-j7d0g-ccccccccccccccc/file"
    fakeHttp <- FakeHttpRequest$new(expectedURL)
    fakeHttpParser <- FakeHttpParser$new()

    arv <- FakeArvados$new("token",
                           "https://host/",
                           "https://webdavHost/",
                           fakeHttp,
                           fakeHttpParser)

    REST <- RESTService$new(arv)

    uuid <- "aaaaa-j7d0g-ccccccccccccccc"
    REST$create("file", uuid)

    expect_that(fakeHttp$URLIsProperlyConfigured, is_true())
    expect_that(fakeHttp$requestHeaderContainsAuthorizationField, is_true())
    expect_that(fakeHttp$numberOfPUTRequests, equals(1))
}) 

test_that("create raises exception if server response code is not between 200 and 300", {

    response <- list()
    response$status_code <- 404
    fakeHttp <- FakeHttpRequest$new(serverResponse = response)

    arv <- FakeArvados$new("token",
                           "https://host/",
                           "https://webdavHost/",
                           fakeHttp,
                           FakeHttpParser$new())

    uuid <- "aaaaa-j7d0g-ccccccccccccccc"
    REST <- RESTService$new(arv)

    expect_that(REST$create("file", uuid),
                throws_error("Server code: 404"))
}) 

test_that("delete calls REST service properly", {

    expectedURL <- "https://webdavHost/c=aaaaa-j7d0g-ccccccccccccccc/file"
    fakeHttp <- FakeHttpRequest$new(expectedURL)
    fakeHttpParser <- FakeHttpParser$new()

    arv <- FakeArvados$new("token",
                           "https://host/",
                           "https://webdavHost/",
                           fakeHttp,
                           fakeHttpParser)

    REST <- RESTService$new(arv)

    uuid <- "aaaaa-j7d0g-ccccccccccccccc"
    REST$delete("file", uuid)

    expect_that(fakeHttp$URLIsProperlyConfigured, is_true())
    expect_that(fakeHttp$requestHeaderContainsAuthorizationField, is_true())
    expect_that(fakeHttp$numberOfDELETERequests, equals(1))
}) 

test_that("delete raises exception if server response code is not between 200 and 300", {

    response <- list()
    response$status_code <- 404
    fakeHttp <- FakeHttpRequest$new(serverResponse = response)

    arv <- FakeArvados$new("token",
                           "https://host/",
                           "https://webdavHost/",
                           fakeHttp,
                           FakeHttpParser$new())

    uuid <- "aaaaa-j7d0g-ccccccccccccccc"
    REST <- RESTService$new(arv)

    expect_that(REST$delete("file", uuid),
                throws_error("Server code: 404"))
}) 

test_that("move calls REST service properly", {

    expectedURL <- "https://webdavHost/c=aaaaa-j7d0g-ccccccccccccccc/file"
    fakeHttp <- FakeHttpRequest$new(expectedURL)
    fakeHttpParser <- FakeHttpParser$new()

    arv <- FakeArvados$new("token",
                           "https://host/",
                           "https://webdavHost/",
                           fakeHttp,
                           fakeHttpParser)

    REST <- RESTService$new(arv)

    uuid <- "aaaaa-j7d0g-ccccccccccccccc"
    REST$move("file", "newDestination/file", uuid)

    expect_that(fakeHttp$URLIsProperlyConfigured, is_true())
    expect_that(fakeHttp$requestHeaderContainsAuthorizationField, is_true())
    expect_that(fakeHttp$requestHeaderContainsDestinationField, is_true())
    expect_that(fakeHttp$numberOfMOVERequests, equals(1))
}) 

test_that("move raises exception if server response code is not between 200 and 300", {

    response <- list()
    response$status_code <- 404
    fakeHttp <- FakeHttpRequest$new(serverResponse = response)

    arv <- FakeArvados$new("token",
                           "https://host/",
                           "https://webdavHost/",
                           fakeHttp,
                           FakeHttpParser$new())

    uuid <- "aaaaa-j7d0g-ccccccccccccccc"
    REST <- RESTService$new(arv)

    expect_that(REST$move("file", "newDestination/file", uuid),
                throws_error("Server code: 404"))
}) 

test_that("getCollectionContent retreives correct content from WebDAV server", {

    expectedURL <- "https://webdavHost/c=aaaaa-j7d0g-ccccccccccccccc"
    returnContent <- c("animal", "animal/dog", "ball")

    fakeHttp <- FakeHttpRequest$new(expectedURL, returnContent)
    fakeHttpParser <- FakeHttpParser$new()

    arv <- FakeArvados$new("token",
                           "https://host/",
                           "https://webdavHost/",
                           fakeHttp,
                           fakeHttpParser)

    REST <- RESTService$new(arv)

    uuid <- "aaaaa-j7d0g-ccccccccccccccc"
    returnResult <- REST$getCollectionContent(uuid)
    returnedContentMatchExpected <- all.equal(returnResult,
                                              c("animal", "animal/dog", "ball"))

    expect_that(returnedContentMatchExpected, is_true())
    expect_that(fakeHttp$requestHeaderContainsAuthorizationField, is_true())
}) 

test_that("getCollectionContent raises exception if server returns empty response", {

    response <- ""
    fakeHttp <- FakeHttpRequest$new(serverResponse = response)

    arv <- FakeArvados$new("token",
                           "https://host/",
                           "https://webdavHost/",
                           fakeHttp,
                           FakeHttpParser$new())

    uuid <- "aaaaa-j7d0g-ccccccccccccccc"
    REST <- RESTService$new(arv)

    expect_that(REST$getCollectionContent(uuid),
                throws_error("Response is empty, request may be misconfigured"))
}) 

test_that("getCollectionContent parses server response", {

    fakeHttp <- FakeHttpRequest$new()
    fakeHttpParser <- FakeHttpParser$new()

    arv <- FakeArvados$new("token",
                           "https://host/",
                           "https://webdavHost/",
                           fakeHttp,
                           fakeHttpParser)

    REST <- RESTService$new(arv)

    uuid <- "aaaaa-j7d0g-ccccccccccccccc"
    REST$getCollectionContent(uuid)

    expect_that(fakeHttpParser$parserCallCount, equals(1))
}) 

test_that("getResourceSize calls REST service properly", {

    expectedURL <- "https://webdavHost/c=aaaaa-j7d0g-ccccccccccccccc/file"
    expectedContent <- c("6", "2", "931", "12003")
    fakeHttp <- FakeHttpRequest$new(expectedURL, expectedContent)
    fakeHttpParser <- FakeHttpParser$new()

    arv <- FakeArvados$new("token",
                           "https://host/",
                           "https://webdavHost/",
                           fakeHttp,
                           fakeHttpParser)

    REST <- RESTService$new(arv)

    uuid <- "aaaaa-j7d0g-ccccccccccccccc"
    returnResult <- REST$getResourceSize("file", uuid)
    returnedContentMatchExpected <- all.equal(returnResult,
                                              as.numeric(expectedContent))

    expect_that(fakeHttp$URLIsProperlyConfigured, is_true())
    expect_that(fakeHttp$requestHeaderContainsAuthorizationField, is_true())
    expect_that(returnedContentMatchExpected, is_true())
}) 

test_that("getResourceSize raises exception if server returns empty response", {

    response <- ""
    fakeHttp <- FakeHttpRequest$new(serverResponse = response)

    arv <- FakeArvados$new("token",
                           "https://host/",
                           "https://webdavHost/",
                           fakeHttp,
                           FakeHttpParser$new())

    uuid <- "aaaaa-j7d0g-ccccccccccccccc"
    REST <- RESTService$new(arv)

    expect_that(REST$getResourceSize("file", uuid),
                throws_error("Response is empty, request may be misconfigured"))
}) 

test_that("getResourceSize parses server response", {

    fakeHttp <- FakeHttpRequest$new()
    fakeHttpParser <- FakeHttpParser$new()

    arv <- FakeArvados$new("token",
                           "https://host/",
                           "https://webdavHost/",
                           fakeHttp,
                           fakeHttpParser)

    REST <- RESTService$new(arv)

    uuid <- "aaaaa-j7d0g-ccccccccccccccc"
    REST$getResourceSize("file", uuid)

    expect_that(fakeHttpParser$parserCallCount, equals(1))
}) 

test_that("read calls REST service properly", {

    expectedURL <- "https://webdavHost/c=aaaaa-j7d0g-ccccccccccccccc/file"
    serverResponse <- list()
    serverResponse$status_code <- 200
    serverResponse$content <- "file content"

    fakeHttp <- FakeHttpRequest$new(expectedURL, serverResponse)
    fakeHttpParser <- FakeHttpParser$new()

    arv <- FakeArvados$new("token",
                           "https://host/",
                           "https://webdavHost/",
                           fakeHttp,
                           fakeHttpParser)

    REST <- RESTService$new(arv)

    uuid <- "aaaaa-j7d0g-ccccccccccccccc"
    returnResult <- REST$read("file", uuid, "text", 1024, 512)

    expect_that(fakeHttp$URLIsProperlyConfigured, is_true())
    expect_that(fakeHttp$requestHeaderContainsAuthorizationField, is_true())
    expect_that(fakeHttp$requestHeaderContainsRangeField, is_true())
    expect_that(returnResult, equals("file content"))
}) 

test_that("read raises exception if server response code is not between 200 and 300", {

    response <- list()
    response$status_code <- 404
    fakeHttp <- FakeHttpRequest$new(serverResponse = response)

    arv <- FakeArvados$new("token",
                           "https://host/",
                           "https://webdavHost/",
                           fakeHttp,
                           FakeHttpParser$new())

    uuid <- "aaaaa-j7d0g-ccccccccccccccc"
    REST <- RESTService$new(arv)

    expect_that(REST$read("file", uuid),
                throws_error("Server code: 404"))
}) 

test_that("read raises exception if contentType is not valid", {

    fakeHttp <- FakeHttpRequest$new()

    arv <- FakeArvados$new("token",
                           "https://host/",
                           "https://webdavHost/",
                           fakeHttp,
                           FakeHttpParser$new())

    uuid <- "aaaaa-j7d0g-ccccccccccccccc"
    REST <- RESTService$new(arv)

    expect_that(REST$read("file", uuid, "some invalid content type"),
                throws_error("Invalid contentType. Please use text or raw."))
}) 

test_that("read parses server response", {

    fakeHttp <- FakeHttpRequest$new()
    fakeHttpParser <- FakeHttpParser$new()

    arv <- FakeArvados$new("token",
                           "https://host/",
                           "https://webdavHost/",
                           fakeHttp,
                           fakeHttpParser)

    REST <- RESTService$new(arv)

    uuid <- "aaaaa-j7d0g-ccccccccccccccc"
    REST$getCollectionContent(uuid)

    expect_that(fakeHttpParser$parserCallCount, equals(1))
}) 

test_that("write calls REST service properly", {

    expectedURL <- "https://webdavHost/c=aaaaa-j7d0g-ccccccccccccccc/file"
    fakeHttp <- FakeHttpRequest$new(expectedURL)
    fakeHttpParser <- FakeHttpParser$new()

    arv <- FakeArvados$new("token",
                           "https://host/",
                           "https://webdavHost/",
                           fakeHttp,
                           fakeHttpParser)

    REST <- RESTService$new(arv)

    fileContent <- "new file content" 
    uuid <- "aaaaa-j7d0g-ccccccccccccccc"
    REST$write("file", uuid, fileContent, "text/html")

    expect_that(fakeHttp$URLIsProperlyConfigured, is_true())
    expect_that(fakeHttp$requestBodyIsProvided, is_true())
    expect_that(fakeHttp$requestHeaderContainsAuthorizationField, is_true())
    expect_that(fakeHttp$requestHeaderContainsContentTypeField, is_true())
}) 

test_that("write raises exception if server response code is not between 200 and 300", {

    response <- list()
    response$status_code <- 404
    fakeHttp <- FakeHttpRequest$new(serverResponse = response)

    arv <- FakeArvados$new("token",
                           "https://host/",
                           "https://webdavHost/",
                           fakeHttp,
                           FakeHttpParser$new())

    uuid <- "aaaaa-j7d0g-ccccccccccccccc"
    fileContent <- "new file content" 
    REST <- RESTService$new(arv)

    expect_that(REST$write("file", uuid, fileContent, "text/html"),
                throws_error("Server code: 404"))
}) 
