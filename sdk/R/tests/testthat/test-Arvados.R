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

test_that("listCollections delegates operation to RESTService class", {

    arv <- Arvados$new("token", "hostName")
    fakeREST <- FakeRESTService$new()
    arv$setRESTService(fakeREST)

    arv$listCollections()

    expect_that(fakeREST$listResourcesCallCount, equals(1))
}) 

test_that("listCollections filter paramerter must be named 'collection'", {

    filters <- list(list("name", "like", "MyCollection"))
    names(filters) <- c("collection")
    fakeREST <- FakeRESTService$new(expectedFilterContent = filters)
    arv <- Arvados$new("token", "hostName")
    arv$setRESTService(fakeREST)

    arv$listCollections(list(list("name", "like", "MyCollection")))

    expect_that(fakeREST$filtersAreConfiguredCorrectly, is_true())
}) 

test_that("listAllCollections delegates operation to RESTService class", {

    arv <- Arvados$new("token", "hostName")
    fakeREST <- FakeRESTService$new()
    arv$setRESTService(fakeREST)

    arv$listAllCollections()

    expect_that(fakeREST$fetchAllItemsCallCount, equals(1))
}) 

test_that("listAllCollections filter paramerter must be named 'collection'", {

    filters <- list(list("name", "like", "MyCollection"))
    names(filters) <- c("collection")
    fakeREST <- FakeRESTService$new(expectedFilterContent = filters)
    arv <- Arvados$new("token", "hostName")
    arv$setRESTService(fakeREST)

    arv$listAllCollections(list(list("name", "like", "MyCollection")))

    expect_that(fakeREST$filtersAreConfiguredCorrectly, is_true())
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

test_that("updateCollection adds content to request parameter named 'collection'", {

    collectionUUID <- "aaaaa-j7d0g-ccccccccccccccc"
    body <- list(list())
    names(body) <- c("collection")
    body$collection <- list(name = "MyCollection", desc = "No description")
    fakeREST <- FakeRESTService$new(returnContent = body)
    arv <- Arvados$new("token", "hostName")
    arv$setRESTService(fakeREST)

    arv$updateCollection(collectionUUID, 
                         list(name = "MyCollection", desc = "No description"))

    expect_that(fakeREST$bodyIsConfiguredCorrectly, is_true())
}) 

test_that("createCollection delegates operation to RESTService class", {

    arv <- Arvados$new("token", "hostName")
    fakeREST <- FakeRESTService$new()
    arv$setRESTService(fakeREST)
    collectionContent <- list(newName = "Brand new shiny name")

    arv$createCollection(collectionContent)

    expect_that(fakeREST$createResourceCallCount, equals(1))
}) 

test_that("createCollection adds content to request parameter named 'collection'", {

    body <- list(list())
    names(body) <- c("collection")
    body$collection <- list(name = "MyCollection", desc = "No description")
    fakeREST <- FakeRESTService$new(returnContent = body)
    arv <- Arvados$new("token", "hostName")
    arv$setRESTService(fakeREST)

    arv$createCollection(list(name = "MyCollection", desc = "No description"))

    expect_that(fakeREST$bodyIsConfiguredCorrectly, is_true())
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

test_that("listProjects filter contains additional 'group_class' field by default", {

    filters <- list(list("name", "like", "MyProject"))
    names(filters) <- c("groups")
    filters[[length(filters) + 1]] <- list("group_class", "=", "project")

    fakeREST <- FakeRESTService$new(expectedFilterContent = filters)
    arv <- Arvados$new("token", "hostName")
    arv$setRESTService(fakeREST)

    arv$listProjects(list(list("name", "like", "MyProject")))

    expect_that(fakeREST$filtersAreConfiguredCorrectly, is_true())
}) 

test_that("listAllProjects delegates operation to RESTService class", {

    arv <- Arvados$new("token", "hostName")
    fakeREST <- FakeRESTService$new()
    arv$setRESTService(fakeREST)

    arv$listAllProjects()

    expect_that(fakeREST$fetchAllItemsCallCount, equals(1))
}) 

test_that("listAllProjects filter contains additional 'group_class' field by default", {

    filters <- list(list("name", "like", "MyProject"))
    names(filters) <- c("groups")
    filters[[length(filters) + 1]] <- list("group_class", "=", "project")

    fakeREST <- FakeRESTService$new(expectedFilterContent = filters)
    arv <- Arvados$new("token", "hostName")
    arv$setRESTService(fakeREST)

    arv$listAllProjects(list(list("name", "like", "MyProject")))

    expect_that(fakeREST$filtersAreConfiguredCorrectly, is_true())
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

test_that("updateProject adds content to request parameter named 'group'", {

    projectUUID <- "aaaaa-j7d0g-ccccccccccccccc"
    body <- list(list())
    names(body) <- c("group")
    body$group <- list(name = "MyProject", desc = "No description")

    fakeREST <- FakeRESTService$new(returnContent = body)
    arv <- Arvados$new("token", "hostName")
    arv$setRESTService(fakeREST)

    arv$updateProject(projectUUID,
                      list(name = "MyProject", desc = "No description"))

    expect_that(fakeREST$bodyIsConfiguredCorrectly, is_true())
}) 

test_that("createProject delegates operation to RESTService class", {

    arv <- Arvados$new("token", "hostName")
    fakeREST <- FakeRESTService$new()
    arv$setRESTService(fakeREST)
    projectContent <- list(newName = "Brand new shiny name")

    arv$createCollection(projectContent)

    expect_that(fakeREST$createResourceCallCount, equals(1))
}) 

test_that("createProject request body contains 'goup_class' filed", {

    body <- list(list())
    names(body) <- c("group")
    body$group <- c("group_class" = "project",
                    list(name = "MyProject", desc = "No description"))

    fakeREST <- FakeRESTService$new(returnContent = body)
    arv <- Arvados$new("token", "hostName")
    arv$setRESTService(fakeREST)

    arv$createProject(list(name = "MyProject", desc = "No description"))

    expect_that(fakeREST$bodyIsConfiguredCorrectly, is_true())
}) 
