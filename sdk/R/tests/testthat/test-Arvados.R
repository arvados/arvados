context("Arvados API")

source("fakes/FakeRESTService.R")

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
                                    " environment variables.")))
}) 

test_that("getCollection delegates operation to RESTService class", {

    arv <- Arvados$new("token", "hostName")
    fakeREST <- FakeRESTService$new()
    arv$setRESTService(fakeREST)
    collectionUUID <- "aaaaa-j7d0g-ccccccccccccccc"

    arv$getCollection(collectionUUID)

    expect_that(fakeREST$getResourceCallCount, equals(1))
}) 

test_that("listCollection delegates operation to RESTService class", {

    arv <- Arvados$new("token", "hostName")
    fakeREST <- FakeRESTService$new()
    arv$setRESTService(fakeREST)

    arv$listCollections()

    expect_that(fakeREST$listResourcesCallCount, equals(1))
}) 

test_that("listAllCollection delegates operation to RESTService class", {

    arv <- Arvados$new("token", "hostName")
    fakeREST <- FakeRESTService$new()
    arv$setRESTService(fakeREST)

    arv$listAllCollections()

    expect_that(fakeREST$fetchAllItemsCallCount, equals(1))
}) 

test_that("deleteCollection delegates operation to RESTService class", {

    arv <- Arvados$new("token", "hostName")
    fakeREST <- FakeRESTService$new()
    arv$setRESTService(fakeREST)
    collectionUUID <- "aaaaa-j7d0g-ccccccccccccccc"

    arv$deleteCollection(collectionUUID)

    expect_that(fakeREST$deleteResourceCallCount, equals(1))
}) 

test_that("updateCollection delegates operation to RESTService class", {

    arv <- Arvados$new("token", "hostName")
    fakeREST <- FakeRESTService$new()
    arv$setRESTService(fakeREST)
    newCollectionContent <- list(newName = "Brand new shiny name")
    collectionUUID <- "aaaaa-j7d0g-ccccccccccccccc"

    arv$updateCollection(collectionUUID, newCollectionContent)

    expect_that(fakeREST$updateResourceCallCount, equals(1))
}) 

test_that("createCollection delegates operation to RESTService class", {

    arv <- Arvados$new("token", "hostName")
    fakeREST <- FakeRESTService$new()
    arv$setRESTService(fakeREST)
    collectionContent <- list(newName = "Brand new shiny name")

    arv$createCollection(collectionContent)

    expect_that(fakeREST$createResourceCallCount, equals(1))
}) 

test_that("getProject delegates operation to RESTService class", {

    arv <- Arvados$new("token", "hostName")
    fakeREST <- FakeRESTService$new()
    arv$setRESTService(fakeREST)
    projectUUID <- "aaaaa-j7d0g-ccccccccccccccc"

    arv$getCollection(projectUUID)

    expect_that(fakeREST$getResourceCallCount, equals(1))
}) 

test_that("listProjects delegates operation to RESTService class", {

    arv <- Arvados$new("token", "hostName")
    fakeREST <- FakeRESTService$new()
    arv$setRESTService(fakeREST)

    arv$listCollections()

    expect_that(fakeREST$listResourcesCallCount, equals(1))
}) 

test_that("listAllProjects delegates operation to RESTService class", {

    arv <- Arvados$new("token", "hostName")
    fakeREST <- FakeRESTService$new()
    arv$setRESTService(fakeREST)

    arv$listAllProjects()

    expect_that(fakeREST$fetchAllItemsCallCount, equals(1))
}) 

test_that("deleteProject delegates operation to RESTService class", {

    arv <- Arvados$new("token", "hostName")
    fakeREST <- FakeRESTService$new()
    arv$setRESTService(fakeREST)
    projectUUID <- "aaaaa-j7d0g-ccccccccccccccc"

    arv$deleteCollection(projectUUID)

    expect_that(fakeREST$deleteResourceCallCount, equals(1))
}) 

test_that("updateProject delegates operation to RESTService class", {

    arv <- Arvados$new("token", "hostName")
    fakeREST <- FakeRESTService$new()
    arv$setRESTService(fakeREST)
    newProjectContent <- list(newName = "Brand new shiny name")
    projectUUID <- "aaaaa-j7d0g-ccccccccccccccc"

    arv$updateCollection(projectUUID, newProjectContent)

    expect_that(fakeREST$updateResourceCallCount, equals(1))
}) 

test_that("createProject delegates operation to RESTService class", {

    arv <- Arvados$new("token", "hostName")
    fakeREST <- FakeRESTService$new()
    arv$setRESTService(fakeREST)
    projectContent <- list(newName = "Brand new shiny name")

    arv$createCollection(projectContent)

    expect_that(fakeREST$createResourceCallCount, equals(1))
}) 
