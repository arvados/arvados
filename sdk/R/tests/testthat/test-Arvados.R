context("Arvados API")

source("fakes/FakeHttpRequest.R")
source("fakes/FakeHttpParser.R")

test_that("Constructor will use environment variables if no parameters are passed to it", {

    Sys.setenv(ARVADOS_API_HOST  = "environment_api_host")
    Sys.setenv(ARVADOS_API_TOKEN = "environment_api_token")

    arv <- Arvados$new()

    Sys.unsetenv("ARVADOS_API_HOST")
    Sys.unsetenv("ARVADOS_API_TOKEN")

    expect_that("https://environment_api_host/arvados/v1/",
                equals(arv$getHostName())) 

    expect_that("environment_api_token",
                equals(arv$getToken())) 
}) 

test_that("Constructor preferes constructor fields over environment variables", {

    Sys.setenv(ARVADOS_API_HOST  = "environment_api_host")
    Sys.setenv(ARVADOS_API_TOKEN = "environment_api_token")

    arv <- Arvados$new("constructor_api_token", "constructor_api_host")

    Sys.unsetenv("ARVADOS_API_HOST")
    Sys.unsetenv("ARVADOS_API_TOKEN")

    expect_that("https://constructor_api_host/arvados/v1/",
                equals(arv$getHostName())) 

    expect_that("constructor_api_token",
                equals(arv$getToken())) 
}) 

test_that("Constructor raises exception if fields and environment variables are not provided", {

    expect_that(Arvados$new(),
                throws_error(paste0("Please provide host name and authentification token",
                                    " or set ARVADOS_API_HOST and ARVADOS_API_TOKEN",
                                    " environmental variables.")))
}) 

test_that("getWebDavHostName calls REST service properly", {

    hostName <- "hostName"
    token    <- "token"
    arv      <- Arvados$new(token, hostName)

    serverResponse <- list(keepWebServiceUrl = "https://myWebDavServer.com")
    expectedURL    <- paste0("https://", hostName,
                             "/discovery/v1/apis/arvados/v1/rest")

    httpRequest <- FakeHttpRequest$new(expectedURL, serverResponse)
    arv$setHttpClient(httpRequest)
    arv$setHttpParser(FakeHttpParser$new())

    webDAVHostName <- arv$getWebDavHostName()

    expect_that(httpRequest$URLIsProperlyConfigured, is_true())
    expect_that(httpRequest$requestHeaderContainsAuthorizationField, is_true())
    expect_that(httpRequest$numberOfGETRequests, equals(1))
}) 

test_that("getWebDavHostName returns webDAV host name properly", {

    arv <- Arvados$new("token", "hostName")

    serverResponse <- list(keepWebServiceUrl = "https://myWebDavServer.com")

    httpRequest <- FakeHttpRequest$new(expectedURL = NULL, serverResponse)
    arv$setHttpClient(httpRequest)
    arv$setHttpParser(FakeHttpParser$new())

    expect_that("https://myWebDavServer.com", equals(arv$getWebDavHostName())) 
}) 

test_that("getCollection calls REST service properly", {

    arv <- Arvados$new("token", "hostName")

    serverResponse <- NULL
    collectionUUID <- "aaaaa-j7d0g-ccccccccccccccc"
    expectedURL    <- paste0(arv$getHostName(), "collections/", collectionUUID)

    httpRequest <- FakeHttpRequest$new(expectedURL, serverResponse)
    arv$setHttpClient(httpRequest)
    arv$setHttpParser(FakeHttpParser$new())

    arv$getCollection(collectionUUID)

    expect_that(httpRequest$URLIsProperlyConfigured, is_true())
    expect_that(httpRequest$requestHeaderContainsAuthorizationField, is_true())
    expect_that(httpRequest$numberOfGETRequests, equals(1))
}) 

test_that("getCollection parses server response", {

    arv <- Arvados$new("token", "hostName")

    httpParser <- FakeHttpParser$new()
    arv$setHttpParser(httpParser)
    arv$setHttpClient(FakeHttpRequest$new())

    collectionUUID <- "aaaaa-j7d0g-ccccccccccccccc"
    arv$getCollection(collectionUUID)

    expect_that(httpParser$parserCallCount, equals(1))
}) 

test_that("getCollection raises exception if response contains errors field", {

    arv <- Arvados$new("token", "hostName")
    
    serverResponse <- list(errors = 404)
    arv$setHttpClient(FakeHttpRequest$new(NULL, serverResponse))
    arv$setHttpParser(FakeHttpParser$new())

    collectionUUID <- "aaaaa-j7d0g-ccccccccccccccc"
    
    expect_that(arv$getCollection(collectionUUID), 
                throws_error(404))
}) 

test_that("listCollections calls REST service properly", {

    arv <- Arvados$new("token", "hostName")

    serverResponse <- NULL
    expectedURL    <- paste0(arv$getHostName(), "collections")

    httpRequest <- FakeHttpRequest$new(expectedURL, serverResponse)
    arv$setHttpClient(httpRequest)
    arv$setHttpParser(FakeHttpParser$new())

    arv$listCollections()

    expect_that(httpRequest$URLIsProperlyConfigured, is_true())
    expect_that(httpRequest$requestHeaderContainsAuthorizationField, is_true())
    expect_that(httpRequest$numberOfGETRequests, equals(1))
}) 

test_that("listCollections parses server response", {

    arv <- Arvados$new("token", "hostName")

    httpParser <- FakeHttpParser$new()
    arv$setHttpParser(httpParser)
    arv$setHttpClient(FakeHttpRequest$new())

    arv$listCollections()

    expect_that(httpParser$parserCallCount, equals(1))
}) 

test_that("listCollections raises exception if response contains errors field", {

    arv <- Arvados$new("token", "hostName")
    
    serverResponse <- list(errors = 404)
    expectedURL <- NULL
    arv$setHttpClient(FakeHttpRequest$new(expectedURL, serverResponse))
    arv$setHttpParser(FakeHttpParser$new())

    expect_that(arv$listCollections(), 
                throws_error(404))
}) 

test_that("deleteCollection calls REST service properly", {

    arv <- Arvados$new("token", "hostName")

    serverResponse <- NULL
    collectionUUID <- "aaaaa-j7d0g-ccccccccccccccc"
    expectedURL    <- paste0(arv$getHostName(), "collections/", collectionUUID)

    httpRequest <- FakeHttpRequest$new(expectedURL, serverResponse)
    arv$setHttpClient(httpRequest)
    arv$setHttpParser(FakeHttpParser$new())

    arv$deleteCollection(collectionUUID)

    expect_that(httpRequest$URLIsProperlyConfigured, is_true())
    expect_that(httpRequest$requestHeaderContainsAuthorizationField, is_true())
    expect_that(httpRequest$numberOfDELETERequests, equals(1))
}) 

test_that("deleteCollection parses server response", {

    arv <- Arvados$new("token", "hostName")

    httpParser <- FakeHttpParser$new()
    arv$setHttpParser(httpParser)
    arv$setHttpClient(FakeHttpRequest$new())

    collectionUUID <- "aaaaa-j7d0g-ccccccccccccccc"
    arv$deleteCollection(collectionUUID)

    expect_that(httpParser$parserCallCount, equals(1))
}) 

test_that("getCollection raises exception if response contains errors field", {

    arv <- Arvados$new("token", "hostName")
    
    serverResponse <- list(errors = 404)
    arv$setHttpClient(FakeHttpRequest$new(NULL, serverResponse))
    arv$setHttpParser(FakeHttpParser$new())

    collectionUUID <- "aaaaa-j7d0g-ccccccccccccccc"
    
    expect_that(arv$deleteCollection(collectionUUID), 
                throws_error(404))
}) 
