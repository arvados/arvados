# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

source("fakes/FakeRESTService.R")

context("Subcollection")

test_that("getRelativePath returns path relative to the tree root", {

    animal <- Subcollection$new("animal")

    fish <- Subcollection$new("fish")
    animal$add(fish)

    expect_that(animal$getRelativePath(), equals("animal"))
    expect_that(fish$getRelativePath(), equals("animal/fish"))
})

test_that(paste("getFileListing by default returns sorted path of all files",
                "relative to the current subcollection"), {

    animal   <- Subcollection$new("animal")
    fish     <- Subcollection$new("fish")
    shark    <- ArvadosFile$new("shark")
    blueFish <- ArvadosFile$new("blueFish")

    animal$add(fish)
    fish$add(shark)
    fish$add(blueFish)

    result <- animal$getFileListing()

    #expect sorted array
    expectedResult <- c("animal/fish/blueFish", "animal/fish/shark")

    resultsMatch <- length(expectedResult) == length(result) &&
                    all(expectedResult == result)

    expect_that(resultsMatch, is_true())
})

test_that(paste("getFileListing returns sorted names of all direct children",
                "if fullPath is set to FALSE"), {

    animal <- Subcollection$new("animal")
    fish   <- Subcollection$new("fish")
    shark  <- ArvadosFile$new("shark")
    dog    <- ArvadosFile$new("dog")

    animal$add(fish)
    animal$add(dog)
    fish$add(shark)

    result <- animal$getFileListing(fullPath = FALSE)
    expectedResult <- c("dog", "fish")

    resultsMatch <- length(expectedResult) == length(result) &&
                    all(expectedResult == result)

    expect_that(resultsMatch, is_true())
})

test_that("add adds content to inside collection tree", {

    animal <- Subcollection$new("animal")
    fish   <- Subcollection$new("fish")
    dog    <- ArvadosFile$new("dog")

    animal$add(fish)
    animal$add(dog)

    animalContainsFish <- animal$get("fish")$getName() == fish$getName()
    animalContainsDog  <- animal$get("dog")$getName()  == dog$getName()

    expect_that(animalContainsFish, is_true())
    expect_that(animalContainsDog, is_true())
})

test_that("add raises exception if content name is empty string", {

    animal     <- Subcollection$new("animal")
    rootFolder <- Subcollection$new("")

    expect_that(animal$add(rootFolder),
                throws_error("Content has invalid name.", fixed = TRUE))
})

test_that(paste("add raises exception if ArvadosFile/Subcollection",
                "with same name already exists in the subcollection"), {

    animal     <- Subcollection$new("animal")
    fish       <- Subcollection$new("fish")
    secondFish <- Subcollection$new("fish")
    thirdFish  <- ArvadosFile$new("fish")

    animal$add(fish)

    expect_that(animal$add(secondFish),
                throws_error(paste("Subcollection already contains ArvadosFile or",
                                   "Subcollection with same name."), fixed = TRUE))
    expect_that(animal$add(thirdFish),
                throws_error(paste("Subcollection already contains ArvadosFile or",
                                   "Subcollection with same name."), fixed = TRUE))
})

test_that(paste("add raises exception if passed argument is",
                "not ArvadosFile or Subcollection"), {

    animal <- Subcollection$new("animal")
    number <- 10

    expect_that(animal$add(number),
                throws_error(paste("Expected AravodsFile or Subcollection object,",
                                   "got (numeric)."), fixed = TRUE))
})

test_that(paste("add post content to a REST service",
                "if subcollection belongs to a collection"), {

    collectionContent <- c("animal", "animal/fish")
    fakeREST <- FakeRESTService$new(collectionContent)

    api <- Arvados$new("myToken", "myHostName")
    api$setRESTService(fakeREST)

    collection <- Collection$new(api, "myUUID")
    animal <- collection$get("animal")
    dog <- ArvadosFile$new("dog")

    animal$add(dog)

    expect_that(fakeREST$createCallCount, equals(1))
})

test_that("remove removes content from subcollection", {

    animal <- Subcollection$new("animal")
    fish   <- Subcollection$new("fish")

    animal$add(fish)
    animal$remove("fish")

    returnValueAfterRemovalIsNull <- is.null(animal$get("fish"))

    expect_that(returnValueAfterRemovalIsNull, is_true())
})

test_that(paste("remove raises exception",
                "if content to remove doesn't exist in the subcollection"), {

    animal <- Subcollection$new("animal")

    expect_that(animal$remove("fish"),
                throws_error(paste("Subcollection doesn't contains ArvadosFile",
                                   "or Subcollection with specified name.")))
})

test_that("remove raises exception if passed argument is not character vector", {

    animal <- Subcollection$new("animal")
    number <- 10

    expect_that(animal$remove(number),
                throws_error(paste("Expected character,",
                                   "got (numeric)."), fixed = TRUE))
})

test_that(paste("remove removes content from REST service",
                "if subcollection belongs to a collection"), {

    collectionContent <- c("animal", "animal/fish", "animal/dog")
    fakeREST <- FakeRESTService$new(collectionContent)

    api <- Arvados$new("myToken", "myHostName")
    api$setRESTService(fakeREST)
    collection <- Collection$new(api, "myUUID")
    animal <- collection$get("animal")

    animal$remove("fish")

    expect_that(fakeREST$deleteCallCount, equals(1))
})

test_that(paste("get returns ArvadosFile or Subcollection",
                "if file or folder with given name exists"), {

    animal <- Subcollection$new("animal")
    fish   <- Subcollection$new("fish")
    dog    <- ArvadosFile$new("dog")

    animal$add(fish)
    animal$add(dog)

    returnedFish <- animal$get("fish")
    returnedDog  <- animal$get("dog")

    returnedFishIsSubcollection <- "Subcollection" %in% class(returnedFish)
    returnedDogIsArvadosFile    <- "ArvadosFile"   %in% class(returnedDog)

    expect_that(returnedFishIsSubcollection, is_true())
    expect_that(returnedFish$getName(), equals("fish"))

    expect_that(returnedDogIsArvadosFile, is_true())
    expect_that(returnedDog$getName(), equals("dog"))
})

test_that(paste("get returns NULL if file or folder",
                "with given name doesn't exists"), {

    animal <- Subcollection$new("animal")
    fish   <- Subcollection$new("fish")

    animal$add(fish)

    returnedDogIsNull <- is.null(animal$get("dog"))

    expect_that(returnedDogIsNull, is_true())
})

test_that("getFirst returns first child in the subcollection", {

    animal <- Subcollection$new("animal")
    fish   <- Subcollection$new("fish")

    animal$add(fish)

    expect_that(animal$getFirst()$getName(), equals("fish"))
})

test_that("getFirst returns NULL if subcollection contains no children", {

    animal <- Subcollection$new("animal")

    returnedElementIsNull <- is.null(animal$getFirst())

    expect_that(returnedElementIsNull, is_true())
})

test_that(paste("setCollection by default sets collection",
                "filed of subcollection and all its children"), {

    animal <- Subcollection$new("animal")
    fish   <- Subcollection$new("fish")
    animal$add(fish)

    animal$setCollection("myCollection")

    expect_that(animal$getCollection(), equals("myCollection"))
    expect_that(fish$getCollection(), equals("myCollection"))
})

test_that(paste("setCollection sets collection filed of subcollection only",
                "if parameter setRecursively is set to FALSE"), {

    animal <- Subcollection$new("animal")
    fish   <- Subcollection$new("fish")
    animal$add(fish)

    animal$setCollection("myCollection", setRecursively = FALSE)
    fishCollectionIsNull <- is.null(fish$getCollection())

    expect_that(animal$getCollection(), equals("myCollection"))
    expect_that(fishCollectionIsNull, is_true())
})

test_that(paste("move raises exception if subcollection",
                "doesn't belong to any collection"), {

    animal <- Subcollection$new("animal")

    expect_that(animal$move("new/location"),
                throws_error("Subcollection doesn't belong to any collection"))
})

test_that("move raises exception if new location contains content with the same name", {

    collectionContent <- c("animal",
                           "animal/fish",
                           "animal/dog",
                           "animal/fish/shark",
                           "fish")
    fakeREST <- FakeRESTService$new(collectionContent)

    api <- Arvados$new("myToken", "myHostName")
    api$setRESTService(fakeREST)
    collection <- Collection$new(api, "myUUID")
    fish <- collection$get("animal/fish")

    expect_that(fish$move("fish"),
                throws_error("Destination already contains content with same name."))

})

test_that(paste("move raises exception if newLocationInCollection",
                "parameter is invalid"), {

    collectionContent <- c("animal",
                           "animal/fish",
                           "animal/dog",
                           "animal/fish/shark",
                           "ball")
    fakeREST <- FakeRESTService$new(collectionContent)

    api <- Arvados$new("myToken", "myHostName")
    api$setRESTService(fakeREST)

    collection <- Collection$new(api, "myUUID")
    fish <- collection$get("animal/fish")

    expect_that(fish$move("objects/dog"),
                throws_error("Unable to get destination subcollection"))
})

test_that("move moves subcollection inside collection tree", {

    collectionContent <- c("animal",
                           "animal/fish",
                           "animal/dog",
                           "animal/fish/shark",
                           "ball")
    fakeREST <- FakeRESTService$new(collectionContent)

    api <- Arvados$new("myToken", "myHostName")
    api$setRESTService(fakeREST)
    collection <- Collection$new(api, "myUUID")
    fish <- collection$get("animal/fish")

    fish$move("fish")
    fishIsNullOnOldLocation <- is.null(collection$get("animal/fish"))
    fishExistsOnNewLocation <- !is.null(collection$get("fish"))

    expect_that(fishIsNullOnOldLocation, is_true())
    expect_that(fishExistsOnNewLocation, is_true())
})

test_that(paste("getSizeInBytes returns zero if subcollection",
                "is not part of a collection"), {

    animal <- Subcollection$new("animal")

    expect_that(animal$getSizeInBytes(), equals(0))
})

test_that(paste("getSizeInBytes delegates size calculation",
                "to REST service class"), {

    collectionContent <- c("animal", "animal/fish")
    returnSize <- 100
    fakeREST <- FakeRESTService$new(collectionContent, returnSize)

    api <- Arvados$new("myToken", "myHostName")
    api$setRESTService(fakeREST)
    collection <- Collection$new(api, "myUUID")
    animal <- collection$get("animal")

    resourceSize <- animal$getSizeInBytes()

    expect_that(resourceSize, equals(100))
})
