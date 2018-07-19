# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

source("fakes/FakeRESTService.R")

context("ArvadosFile")

test_that("constructor raises error if  file name is empty string", {

    expect_that(ArvadosFile$new(""), throws_error("Invalid name."))
})

test_that("getFileListing always returns file name", {

    dog <- ArvadosFile$new("dog")

    expect_that(dog$getFileListing(), equals("dog"))
})

test_that("get always returns NULL", {

    dog <- ArvadosFile$new("dog")

    responseIsNull <- is.null(dog$get("something"))
    expect_that(responseIsNull, is_true())
})

test_that("getFirst always returns NULL", {

    dog <- ArvadosFile$new("dog")

    responseIsNull <- is.null(dog$getFirst())
    expect_that(responseIsNull, is_true())
})

test_that(paste("getSizeInBytes returns zero if arvadosFile",
                "is not part of a collection"), {

    dog <- ArvadosFile$new("dog")

    expect_that(dog$getSizeInBytes(), equals(0))
})

test_that(paste("getSizeInBytes delegates size calculation",
                "to REST service class"), {

    collectionContent <- c("animal", "animal/fish")
    returnSize <- 100
    fakeREST <- FakeRESTService$new(collectionContent, returnSize)

    api <- Arvados$new("myToken", "myHostName")
    api$setRESTService(fakeREST)
    collection <- Collection$new(api, "myUUID")
    fish <- collection$get("animal/fish")

    resourceSize <- fish$getSizeInBytes()

    expect_that(resourceSize, equals(100))
})

test_that("getRelativePath returns path relative to the tree root", {

    animal <- Subcollection$new("animal")
    fish <- Subcollection$new("fish")
    shark <- ArvadosFile$new("shark")

    animal$add(fish)
    fish$add(shark)

    expect_that(shark$getRelativePath(), equals("animal/fish/shark"))
})

test_that("read raises exception if file doesn't belong to a collection", {

    dog <- ArvadosFile$new("dog")

    expect_that(dog$read(),
                throws_error("ArvadosFile doesn't belong to any collection."))
})

test_that("read raises exception offset or length is negative number", {


    collectionContent <- c("animal", "animal/fish")
    fakeREST <- FakeRESTService$new(collectionContent)

    api <- Arvados$new("myToken", "myHostName")
    api$setRESTService(fakeREST)
    collection <- Collection$new(api, "myUUID")
    fish <- collection$get("animal/fish")

    expect_that(fish$read(contentType = "text", offset = -1),
                throws_error("Offset and length must be positive values."))
    expect_that(fish$read(contentType = "text", length = -1),
                throws_error("Offset and length must be positive values."))
    expect_that(fish$read(contentType = "text", offset = -1, length = -1),
                throws_error("Offset and length must be positive values."))
})

test_that("read delegates reading operation to REST service class", {

    collectionContent <- c("animal", "animal/fish")
    readContent <- "my file"
    fakeREST <- FakeRESTService$new(collectionContent, readContent)

    api <- Arvados$new("myToken", "myHostName")
    api$setRESTService(fakeREST)
    collection <- Collection$new(api, "myUUID")
    fish <- collection$get("animal/fish")

    fileContent <- fish$read("text")

    expect_that(fileContent, equals("my file"))
    expect_that(fakeREST$readCallCount, equals(1))
})

test_that(paste("connection delegates connection creation ro RESTService class",
                "which returns curl connection opened in read mode when",
                "'r' of 'rb' is passed as argument"), {

    collectionContent <- c("animal", "animal/fish")
    fakeREST <- FakeRESTService$new(collectionContent)

    api <- Arvados$new("myToken", "myHostName")
    api$setRESTService(fakeREST)
    collection <- Collection$new(api, "myUUID")
    fish <- collection$get("animal/fish")

    connection <- fish$connection("r")

    expect_that(fakeREST$getConnectionCallCount, equals(1))
})

test_that(paste("connection returns textConnection opened",
                "in write mode when 'w' is passed as argument"), {

    collectionContent <- c("animal", "animal/fish")
    fakeREST <- FakeRESTService$new(collectionContent)

    api <- Arvados$new("myToken", "myHostName")
    api$setRESTService(fakeREST)
    collection <- Collection$new(api, "myUUID")
    fish <- collection$get("animal/fish")

    connection <- fish$connection("w")

    writeLines("file", connection)
    writeLines("content", connection)

    writeResult <- textConnectionValue(connection)

    expect_that(writeResult[1], equals("file"))
    expect_that(writeResult[2], equals("content"))
})

test_that("flush sends data stored in a connection to a REST server", {


    collectionContent <- c("animal", "animal/fish")
    fakeREST <- FakeRESTService$new(collectionContent)

    api <- Arvados$new("myToken", "myHostName")
    api$setRESTService(fakeREST)
    collection <- Collection$new(api, "myUUID")
    fish <- collection$get("animal/fish")

    connection <- fish$connection("w")

    writeLines("file content", connection)

    fish$flush()

    expect_that(fakeREST$writeBuffer, equals("file content"))
})

test_that("write raises exception if file doesn't belong to a collection", {

    dog <- ArvadosFile$new("dog")

    expect_that(dog$write(),
                throws_error("ArvadosFile doesn't belong to any collection."))
})

test_that("write delegates writing operation to REST service class", {


    collectionContent <- c("animal", "animal/fish")
    fakeREST <- FakeRESTService$new(collectionContent)

    api <- Arvados$new("myToken", "myHostName")
    api$setRESTService(fakeREST)
    collection <- Collection$new(api, "myUUID")
    fish <- collection$get("animal/fish")

    fileContent <- fish$write("new file content")

    expect_that(fakeREST$writeBuffer, equals("new file content"))
})

test_that(paste("move raises exception if arvados file",
                "doesn't belong to any collection"), {

    animal <- ArvadosFile$new("animal")

    expect_that(animal$move("new/location"),
                throws_error("ArvadosFile doesn't belong to any collection"))
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
    dog <- collection$get("animal/dog")

    expect_that(dog$move("objects/dog"),
                throws_error("Unable to get destination subcollection"))
})

test_that("move raises exception if new location contains content with the same name", {


    collectionContent <- c("animal",
                           "animal/fish",
                           "animal/dog",
                           "animal/fish/shark",
                           "dog")

    fakeREST <- FakeRESTService$new(collectionContent)

    api <- Arvados$new("myToken", "myHostName")
    api$setRESTService(fakeREST)
    collection <- Collection$new(api, "myUUID")
    dog <- collection$get("animal/dog")

    expect_that(dog$move("dog"),
                throws_error("Destination already contains content with same name."))

})

test_that("move moves arvados file inside collection tree", {


    collectionContent <- c("animal",
                           "animal/fish",
                           "animal/dog",
                           "animal/fish/shark",
                           "ball")

    fakeREST <- FakeRESTService$new(collectionContent)

    api <- Arvados$new("myToken", "myHostName")
    api$setRESTService(fakeREST)
    collection <- Collection$new(api, "myUUID")
    dog <- collection$get("animal/dog")

    dog$move("dog")
    dogIsNullOnOldLocation <- is.null(collection$get("animal/dog"))
    dogExistsOnNewLocation <- !is.null(collection$get("dog"))

    expect_that(dogIsNullOnOldLocation, is_true())
    expect_that(dogExistsOnNewLocation, is_true())
})
