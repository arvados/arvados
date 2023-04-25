# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

source("fakes/FakeRESTService.R")

context("Collection")

test_that(paste("constructor creates file tree from text content",
                "retreived form REST service"), {

    collectionContent <- c("animal", "animal/fish", "ball")
    fakeREST <- FakeRESTService$new(collectionContent)

    api <- Arvados$new("myToken", "myHostName")
    api$setRESTService(fakeREST)
    collection <- Collection$new(api, "myUUID")

    root <- collection$get("")

    expect_that(fakeREST$getCollectionContentCallCount, equals(1))
    expect_that(root$getName(), equals(""))
})

test_that(paste("add raises exception if passed argumet is not",
                "ArvadosFile or Subcollection"), {

    collectionContent <- c("animal", "animal/fish", "ball")
    fakeREST <- FakeRESTService$new(collectionContent)

    api <- Arvados$new("myToken", "myHostName")
    api$setRESTService(fakeREST)
    collection <- Collection$new(api, "myUUID")

    newNumber <- 10

    expect_that(collection$add(newNumber),
    throws_error(paste("Expected AravodsFile or Subcollection",
                       "object, got (numeric)."), fixed = TRUE))
})

test_that("add raises exception if relative path is not valid", {

    collectionContent <- c("animal", "animal/fish", "ball")
    fakeREST <- FakeRESTService$new(collectionContent)

    api <- Arvados$new("myToken", "myHostName")
    api$setRESTService(fakeREST)
    collection <- Collection$new(api, "myUUID")

    newPen <- ArvadosFile$new("pen")

    expect_that(collection$add(newPen, "objects"),
                throws_error("Subcollection objects doesn't exist.",
                              fixed = TRUE))
})

test_that("add raises exception if content name is empty string", {

    collectionContent <- c("animal", "animal/fish")
    fakeREST <- FakeRESTService$new(collectionContent)

    api <- Arvados$new("myToken", "myHostName")
    api$setRESTService(fakeREST)
    collection <- Collection$new(api, "myUUID")

    rootFolder <- Subcollection$new("")

    expect_that(collection$add(rootFolder),
                throws_error("Content has invalid name.", fixed = TRUE))
})

test_that(paste("add adds ArvadosFile or Subcollection",
                "to local tree structure and remote REST service"), {

    collectionContent <- c("animal", "animal/fish", "ball")
    fakeREST <- FakeRESTService$new(collectionContent)

    api <- Arvados$new("myToken", "myHostName")
    api$setRESTService(fakeREST)
    collection <- Collection$new(api, "myUUID")

    newDog <- ArvadosFile$new("dog")
    collection$add(newDog, "animal")

    dog <- collection$get("animal/dog")
    dogExistsInCollection <- !is.null(dog) && dog$getName() == "dog"

    expect_true(dogExistsInCollection)
    expect_that(fakeREST$createCallCount, equals(1))
})

test_that("create raises exception if passed argumet is not character vector", {

    collectionContent <- c("animal", "animal/fish", "ball")
    fakeREST <- FakeRESTService$new(collectionContent)

    api <- Arvados$new("myToken", "myHostName")
    api$setRESTService(fakeREST)
    collection <- Collection$new(api, "myUUID")

    expect_that(collection$create(10),
                throws_error("Expected character vector, got (numeric).",
                             fixed = TRUE))
})

test_that(paste("create adds files specified by fileNames",
                "to local tree structure and remote REST service"), {

    fakeREST <- FakeRESTService$new()
    api <- Arvados$new("myToken", "myHostName")
    api$setRESTService(fakeREST)
    collection <- Collection$new(api, "myUUID")

    collection$create(c("animal/dog", "animal/cat"))

    dog <- collection$get("animal/dog")
    cat <- collection$get("animal/cat")
    dogExistsInCollection <- !is.null(dog) && dog$getName() == "dog"
    catExistsInCollection <- !is.null(cat) && cat$getName() == "cat"

    expect_true(dogExistsInCollection)
    expect_true(catExistsInCollection)
    expect_that(fakeREST$createCallCount, equals(2))
})

test_that("remove raises exception if passed argumet is not character vector", {

    collectionContent <- c("animal", "animal/fish", "ball")
    fakeREST <- FakeRESTService$new(collectionContent)

    api <- Arvados$new("myToken", "myHostName")
    api$setRESTService(fakeREST)
    collection <- Collection$new(api, "myUUID")

    expect_that(collection$remove(10),
                throws_error("Expected character vector, got (numeric).",
                             fixed = TRUE))
})

test_that("remove raises exception if user tries to remove root folder", {

    collectionContent <- c("animal", "animal/fish")
    fakeREST <- FakeRESTService$new(collectionContent)

    api <- Arvados$new("myToken", "myHostName")
    api$setRESTService(fakeREST)
    collection <- Collection$new(api, "myUUID")

    expect_that(collection$remove(""),
                throws_error("You can't delete root folder.", fixed = TRUE))
})

test_that(paste("remove removes files specified by paths",
                "from local tree structure and from remote REST service"), {

    collectionContent <- c("animal", "animal/fish", "animal/dog", "animal/cat", "ball")
    fakeREST <- FakeRESTService$new(collectionContent)

    api <- Arvados$new("myToken", "myHostName")
    api$setRESTService(fakeREST)
    collection <- Collection$new(api, "myUUID")

    collection$remove(c("animal/dog", "animal/cat"))

    dog <- collection$get("animal/dog")
    cat <- collection$get("animal/dog")
    dogExistsInCollection <- !is.null(dog) && dog$getName() == "dog"
    catExistsInCollection <- !is.null(cat) && cat$getName() == "cat"

    expect_false(dogExistsInCollection)
    expect_false(catExistsInCollection)
    expect_that(fakeREST$deleteCallCount, equals(2))
})

test_that(paste("move moves content to a new location inside file tree",
                "and on REST service"), {

    collectionContent <- c("animal", "animal/dog", "ball")
    fakeREST <- FakeRESTService$new(collectionContent)

    api <- Arvados$new("myToken", "myHostName")
    api$setRESTService(fakeREST)
    collection <- Collection$new(api, "myUUID")

    collection$move("animal/dog", "dog")

    dogIsNullOnOldLocation <- is.null(collection$get("animal/dog"))
    dogExistsOnNewLocation <- !is.null(collection$get("dog"))

    expect_true(dogIsNullOnOldLocation)
    expect_true(dogExistsOnNewLocation)
    expect_that(fakeREST$moveCallCount, equals(1))
})

test_that("move raises exception if new location is not valid", {

    collectionContent <- c("animal", "animal/fish", "ball")
    fakeREST <- FakeRESTService$new(collectionContent)

    api <- Arvados$new("myToken", "myHostName")
    api$setRESTService(fakeREST)
    collection <- Collection$new(api, "myUUID")

    expect_that(collection$move("fish", "object"),
                throws_error("Content you want to move doesn't exist in the collection.",
                             fixed = TRUE))
})

test_that("getFileListing returns sorted collection content received from REST service", {

    collectionContent <- c("animal", "animal/fish", "ball")
    fakeREST <- FakeRESTService$new(collectionContent)

    api <- Arvados$new("myToken", "myHostName")
    api$setRESTService(fakeREST)
    collection <- Collection$new(api, "myUUID")

    contentMatchExpected <- all(collection$getFileListing() ==
                                c("animal", "animal/fish", "ball"))

    expect_true(contentMatchExpected)
    #2 calls because Collection$new calls getFileListing once
    expect_that(fakeREST$getCollectionContentCallCount, equals(2))

})

test_that("get returns arvados file or subcollection from internal tree structure", {

    collectionContent <- c("animal", "animal/fish", "ball")
    fakeREST <- FakeRESTService$new(collectionContent)

    api <- Arvados$new("myToken", "myHostName")
    api$setRESTService(fakeREST)
    collection <- Collection$new(api, "myUUID")

    fish <- collection$get("animal/fish")
    fishIsNotNull <- !is.null(fish)

    expect_true(fishIsNotNull)
    expect_that(fish$getName(), equals("fish"))

    ball <- collection$get("ball")
    ballIsNotNull <- !is.null(ball)

    expect_true(ballIsNotNull)
    expect_that(ball$getName(), equals("ball"))
})

test_that(paste("copy copies content to a new location inside file tree",
                "and on REST service"), {

    collectionContent <- c("animal", "animal/dog", "ball")
    fakeREST <- FakeRESTService$new(collectionContent)

    api <- Arvados$new("myToken", "myHostName")
    api$setRESTService(fakeREST)
    collection <- Collection$new(api, "myUUID")

    collection$copy("animal/dog", "dog")

    dogExistsOnOldLocation <- !is.null(collection$get("animal/dog"))
    dogExistsOnNewLocation <- !is.null(collection$get("dog"))

    expect_true(dogExistsOnOldLocation)
    expect_true(dogExistsOnNewLocation)
    expect_that(fakeREST$copyCallCount, equals(1))
})

test_that("copy raises exception if new location is not valid", {

    collectionContent <- c("animal", "animal/fish", "ball")
    fakeREST <- FakeRESTService$new(collectionContent)

    api <- Arvados$new("myToken", "myHostName")
    api$setRESTService(fakeREST)
    collection <- Collection$new(api, "myUUID")

    expect_that(collection$copy("fish", "object"),
                throws_error("Content you want to copy doesn't exist in the collection.",
                             fixed = TRUE))
})

test_that("refresh invalidates current tree structure", {

    collectionContent <- c("animal", "animal/fish", "ball")
    fakeREST <- FakeRESTService$new(collectionContent)

    api <- Arvados$new("myToken", "myHostName")
    api$setRESTService(fakeREST)
    collection <- Collection$new(api, "aaaaa-j7d0g-ccccccccccccccc")

    # Before refresh
    fish <- collection$get("animal/fish")
    expect_that(fish$getName(), equals("fish"))
    expect_that(fish$getCollection()$uuid, equals("aaaaa-j7d0g-ccccccccccccccc"))

    collection$refresh()

    # After refresh
    expect_that(fish$getName(), equals("fish"))
    expect_true(is.null(fish$getCollection()))
})
