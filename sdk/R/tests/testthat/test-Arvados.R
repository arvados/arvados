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

test_that("listAllCollections always returns all collections from server", {

    arv <- Arvados$new("token", "hostName")
    
    expectedURL <- NULL
    serverResponse <- list(items_available = 8,
                           items = list("collection1",
                                        "collection2",
                                        "collection3",
                                        "collection4",
                                        "collection5",
                                        "collection6",
                                        "collection7",
                                        "collection8"))

    httpRequest <- FakeHttpRequest$new(expectedURL, serverResponse)
    httpParser  <- FakeHttpParser$new()

    httpRequest$serverMaxElementsPerRequest <- 3

    arv$setHttpClient(httpRequest)
    arv$setHttpParser(httpParser)

    result <- arv$listAllCollections()

    expect_that(length(result), equals(8))
    expect_that(httpRequest$numberOfGETRequests, equals(3))
    expect_that(httpParser$parserCallCount, equals(3))
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

test_that("deleteCollection raises exception if response contains errors field", {

    arv <- Arvados$new("token", "hostName")
    
    serverResponse <- list(errors = 404)
    arv$setHttpClient(FakeHttpRequest$new(NULL, serverResponse))
    arv$setHttpParser(FakeHttpParser$new())

    collectionUUID <- "aaaaa-j7d0g-ccccccccccccccc"
    
    expect_that(arv$deleteCollection(collectionUUID), 
                throws_error(404))
}) 

test_that("updateCollection calls REST service properly", {

    arv <- Arvados$new("token", "hostName")

    newCollectionContent <- list(newName = "Brand new shiny name")
    collectionUUID <- "aaaaa-j7d0g-ccccccccccccccc"
    expectedURL    <- paste0(arv$getHostName(), "collections/", collectionUUID)
    serverResponse <- NULL

    httpRequest <- FakeHttpRequest$new(expectedURL, serverResponse)
    arv$setHttpClient(httpRequest)
    arv$setHttpParser(FakeHttpParser$new())

    arv$updateCollection(collectionUUID, newCollectionContent)

    expect_that(httpRequest$URLIsProperlyConfigured, is_true())
    expect_that(httpRequest$requestHeaderContainsAuthorizationField, is_true())
    expect_that(httpRequest$JSONEncodedBodyIsProvided, is_true())
    expect_that(httpRequest$numberOfPUTRequests, equals(1))
}) 

test_that("updateCollection parses server response", {

    arv <- Arvados$new("token", "hostName")

    httpParser <- FakeHttpParser$new()
    arv$setHttpParser(httpParser)
    arv$setHttpClient(FakeHttpRequest$new())

    collectionUUID <- "aaaaa-j7d0g-ccccccccccccccc"
    newCollectionContent <- list(newName = "Brand new shiny name")

    arv$updateCollection(collectionUUID, newCollectionContent)

    expect_that(httpParser$parserCallCount, equals(1))
}) 

test_that("updateCollection raises exception if response contains errors field", {

    arv <- Arvados$new("token", "hostName")
    
    expectedURL <- NULL
    serverResponse <- list(errors = 404)
    arv$setHttpClient(FakeHttpRequest$new(expectedURL, serverResponse))
    arv$setHttpParser(FakeHttpParser$new())

    collectionUUID <- "aaaaa-j7d0g-ccccccccccccccc"
    newCollectionContent <- list(newName = "Brand new shiny name")
    
    expect_that(arv$updateCollection(collectionUUID, newCollectionContent), 
                throws_error(404))
}) 

test_that("createCollection calls REST service properly", {

    arv <- Arvados$new("token", "hostName")

    collectionContent <- list(name = "My favorite collection")
    expectedURL    <- paste0(arv$getHostName(), "collections")
    serverResponse <- NULL

    httpRequest <- FakeHttpRequest$new(expectedURL, serverResponse)
    arv$setHttpClient(httpRequest)
    arv$setHttpParser(FakeHttpParser$new())

    arv$createCollection(collectionContent)

    expect_that(httpRequest$URLIsProperlyConfigured, is_true())
    expect_that(httpRequest$requestHeaderContainsAuthorizationField, is_true())
    expect_that(httpRequest$JSONEncodedBodyIsProvided, is_true())
    expect_that(httpRequest$numberOfPOSTRequests, equals(1))
}) 

test_that("createCollection parses server response", {

    arv <- Arvados$new("token", "hostName")

    httpParser <- FakeHttpParser$new()
    arv$setHttpParser(httpParser)
    arv$setHttpClient(FakeHttpRequest$new())

    collectionContent <- list(name = "My favorite collection")

    arv$createCollection(collectionContent)

    expect_that(httpParser$parserCallCount, equals(1))
}) 

test_that("createCollection raises exception if response contains errors field", {

    arv <- Arvados$new("token", "hostName")
    
    expectedURL <- NULL
    serverResponse <- list(errors = 404)
    arv$setHttpClient(FakeHttpRequest$new(expectedURL, serverResponse))
    arv$setHttpParser(FakeHttpParser$new())

    collectionContent <- list(name = "My favorite collection")
    
    expect_that(arv$createCollection(collectionContent), 
                throws_error(404))
}) 

test_that("getProject calls REST service properly", {

    arv <- Arvados$new("token", "hostName")

    serverResponse <- NULL
    projectUUID <- "aaaaa-j7d0g-ccccccccccccccc"
    expectedURL    <- paste0(arv$getHostName(), "groups/", projectUUID)

    httpRequest <- FakeHttpRequest$new(expectedURL, serverResponse)
    arv$setHttpClient(httpRequest)
    arv$setHttpParser(FakeHttpParser$new())

    arv$getProject(projectUUID)

    expect_that(httpRequest$URLIsProperlyConfigured, is_true())
    expect_that(httpRequest$requestHeaderContainsAuthorizationField, is_true())
    expect_that(httpRequest$numberOfGETRequests, equals(1))
}) 

test_that("getProject parses server response", {

    arv <- Arvados$new("token", "hostName")

    httpParser <- FakeHttpParser$new()
    arv$setHttpParser(httpParser)
    arv$setHttpClient(FakeHttpRequest$new())

    projectUUID <- "aaaaa-j7d0g-ccccccccccccccc"
    arv$getProject(projectUUID)

    expect_that(httpParser$parserCallCount, equals(1))
}) 

test_that("getProject raises exception if response contains errors field", {

    arv <- Arvados$new("token", "hostName")
    
    serverResponse <- list(errors = 404)
    arv$setHttpClient(FakeHttpRequest$new(NULL, serverResponse))
    arv$setHttpParser(FakeHttpParser$new())

    projectUUID <- "aaaaa-j7d0g-ccccccccccccccc"
    
    expect_that(arv$getProject(projectUUID), 
                throws_error(404))
}) 

test_that("createProject calls REST service properly", {

    arv <- Arvados$new("token", "hostName")

    projectContent <- list(name = "My favorite project")
    expectedURL    <- paste0(arv$getHostName(), "groups")
    serverResponse <- NULL

    httpRequest <- FakeHttpRequest$new(expectedURL, serverResponse)
    arv$setHttpClient(httpRequest)
    arv$setHttpParser(FakeHttpParser$new())

    arv$createProject(projectContent)

    expect_that(httpRequest$URLIsProperlyConfigured, is_true())
    expect_that(httpRequest$requestHeaderContainsAuthorizationField, is_true())
    expect_that(httpRequest$JSONEncodedBodyIsProvided, is_true())
    expect_that(httpRequest$numberOfPOSTRequests, equals(1))
}) 

test_that("createProject parses server response", {

    arv <- Arvados$new("token", "hostName")

    httpParser <- FakeHttpParser$new()
    arv$setHttpParser(httpParser)
    arv$setHttpClient(FakeHttpRequest$new())

    projectContent <- list(name = "My favorite project")

    arv$createProject(projectContent)

    expect_that(httpParser$parserCallCount, equals(1))
}) 

test_that("createProject raises exception if response contains errors field", {

    arv <- Arvados$new("token", "hostName")
    
    expectedURL <- NULL
    serverResponse <- list(errors = 404)
    arv$setHttpClient(FakeHttpRequest$new(expectedURL, serverResponse))
    arv$setHttpParser(FakeHttpParser$new())

    projectContent <- list(name = "My favorite project")
    
    expect_that(arv$createProject(projectContent), 
                throws_error(404))
}) 

test_that("updateProject calls REST service properly", {

    arv <- Arvados$new("token", "hostName")

    newProjectContent <- list(newName = "Brand new shiny name")
    projectUUID <- "aaaaa-j7d0g-ccccccccccccccc"
    expectedURL    <- paste0(arv$getHostName(), "groups/", projectUUID)
    serverResponse <- NULL

    httpRequest <- FakeHttpRequest$new(expectedURL, serverResponse)
    arv$setHttpClient(httpRequest)
    arv$setHttpParser(FakeHttpParser$new())

    arv$updateProject(projectUUID, newProjectContent)

    expect_that(httpRequest$URLIsProperlyConfigured, is_true())
    expect_that(httpRequest$requestHeaderContainsAuthorizationField, is_true())
    expect_that(httpRequest$JSONEncodedBodyIsProvided, is_true())
    expect_that(httpRequest$numberOfPUTRequests, equals(1))
}) 

test_that("updateProject parses server response", {

    arv <- Arvados$new("token", "hostName")

    httpParser <- FakeHttpParser$new()
    arv$setHttpParser(httpParser)
    arv$setHttpClient(FakeHttpRequest$new())

    newProjectContent <- list(newName = "Brand new shiny name")
    projectUUID <- "aaaaa-j7d0g-ccccccccccccccc"

    arv$updateProject(projectUUID, newProjectContent)

    expect_that(httpParser$parserCallCount, equals(1))
}) 

test_that("updateProject raises exception if response contains errors field", {

    arv <- Arvados$new("token", "hostName")
    
    expectedURL <- NULL
    serverResponse <- list(errors = 404)
    arv$setHttpClient(FakeHttpRequest$new(expectedURL, serverResponse))
    arv$setHttpParser(FakeHttpParser$new())

    newProjectContent <- list(newName = "Brand new shiny name")
    projectUUID <- "aaaaa-j7d0g-ccccccccccccccc"
    
    expect_that(arv$updateProject(projectUUID, newProjectContent), 
                throws_error(404))
}) 

test_that("listProjects calls REST service properly", {

    arv <- Arvados$new("token", "hostName")

    serverResponse <- NULL
    expectedURL    <- paste0(arv$getHostName(), "groups")
    expectedFilters <- list(list("name" = "My project"), 
                            list("group_class", "=", "project"))

    httpRequest <- FakeHttpRequest$new(expectedURL, serverResponse, expectedFilters)
    arv$setHttpClient(httpRequest)
    arv$setHttpParser(FakeHttpParser$new())

    arv$listProjects(list(list("name" = "My project")))

    expect_that(httpRequest$URLIsProperlyConfigured, is_true())
    expect_that(httpRequest$queryFiltersAreCorrect, is_true())
    expect_that(httpRequest$requestHeaderContainsAuthorizationField, is_true())
    expect_that(httpRequest$numberOfGETRequests, equals(1))
}) 

test_that("listProjects parses server response", {

    arv <- Arvados$new("token", "hostName")

    httpParser <- FakeHttpParser$new()
    arv$setHttpParser(httpParser)
    arv$setHttpClient(FakeHttpRequest$new())

    arv$listProjects()

    expect_that(httpParser$parserCallCount, equals(1))
}) 

test_that("listProjects raises exception if response contains errors field", {

    arv <- Arvados$new("token", "hostName")
    
    serverResponse <- list(errors = 404)
    expectedURL <- NULL
    arv$setHttpClient(FakeHttpRequest$new(expectedURL, serverResponse))
    arv$setHttpParser(FakeHttpParser$new())

    expect_that(arv$listProjects(), 
                throws_error(404))
}) 

test_that("listAllProjects always returns all projects from server", {

    arv <- Arvados$new("token", "hostName")
    
    expectedURL <- NULL
    serverResponse <- list(items_available = 8,
                           items = list("project1",
                                        "project2",
                                        "project3",
                                        "project4",
                                        "project5",
                                        "project6",
                                        "project7",
                                        "project8"))

    expectedFilters <- list(list("name" = "My project"), 
                            list("group_class", "=", "project"))

    httpRequest <- FakeHttpRequest$new(expectedURL, serverResponse, expectedFilters)
    httpParser  <- FakeHttpParser$new()

    httpRequest$serverMaxElementsPerRequest <- 3

    arv$setHttpClient(httpRequest)
    arv$setHttpParser(httpParser)

    result <- arv$listAllProjects(list(list("name" = "My project")))

    expect_that(length(result), equals(8))
    expect_that(httpRequest$queryFiltersAreCorrect, is_true())
    expect_that(httpRequest$numberOfGETRequests, equals(3))
    expect_that(httpParser$parserCallCount, equals(3))
}) 

test_that("deleteProject calls REST service properly", {

    arv <- Arvados$new("token", "hostName")

    serverResponse <- NULL
    projectUUID <- "aaaaa-j7d0g-ccccccccccccccc"
    expectedURL    <- paste0(arv$getHostName(), "groups/", projectUUID)

    httpRequest <- FakeHttpRequest$new(expectedURL, serverResponse)
    arv$setHttpClient(httpRequest)
    arv$setHttpParser(FakeHttpParser$new())

    arv$deleteProject(projectUUID)

    expect_that(httpRequest$URLIsProperlyConfigured, is_true())
    expect_that(httpRequest$requestHeaderContainsAuthorizationField, is_true())
    expect_that(httpRequest$numberOfDELETERequests, equals(1))
}) 

test_that("deleteProject parses server response", {

    arv <- Arvados$new("token", "hostName")

    httpParser <- FakeHttpParser$new()
    arv$setHttpParser(httpParser)
    arv$setHttpClient(FakeHttpRequest$new())

    projectUUID <- "aaaaa-j7d0g-ccccccccccccccc"
    arv$deleteProject(projectUUID)

    expect_that(httpParser$parserCallCount, equals(1))
}) 

test_that("deleteCollection raises exception if response contains errors field", {

    arv <- Arvados$new("token", "hostName")
    
    serverResponse <- list(errors = 404)
    expectedURL <- NULL
    arv$setHttpClient(FakeHttpRequest$new(expectedURL, serverResponse))
    arv$setHttpParser(FakeHttpParser$new())

    projectUUID <- "aaaaa-j7d0g-ccccccccccccccc"
    
    expect_that(arv$deleteProject(projectUUID), 
                throws_error(404))
}) 
