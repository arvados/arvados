# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

context("Utility function")

test_that("listAll always returns all resource items from server", {

    serverResponseLimit <- 3
    itemsAvailable <- 8
    items <- list("collection1", "collection2", "collection3", "collection4",
                  "collection5", "collection6", "collection7", "collection8")

    testFunction <- function(offset, ...)
    {
        response <- list()
        response$items_available <- itemsAvailable

        maxIndex <- offset + serverResponseLimit
        lastElementIndex <- if(maxIndex < itemsAvailable) maxIndex else itemsAvailable

        response$items <- items[(offset + 1):lastElementIndex]
        response
    }

    result <- listAll(testFunction)

    expect_that(length(result), equals(8))
})

test_that("trimFromStart trims string correctly if string starts with trimCharacters", {

    sample <- "./something/random"
    trimCharacters <- "./something/"

    result <- trimFromStart(sample, trimCharacters)

    expect_that(result, equals("random"))
})

test_that("trimFromStart returns original string if string doesn't starts with trimCharacters", {

    sample <- "./something/random"
    trimCharacters <- "./nothing/"

    result <- trimFromStart(sample, trimCharacters)

    expect_that(result, equals("./something/random"))
})

test_that("trimFromEnd trims string correctly if string ends with trimCharacters", {

    sample <- "./something/random"
    trimCharacters <- "/random"

    result <- trimFromEnd(sample, trimCharacters)

    expect_that(result, equals("./something"))
})

test_that("trimFromEnd returns original string if string doesn't end with trimCharacters", {

    sample <- "./something/random"
    trimCharacters <- "specific"

    result <- trimFromStart(sample, trimCharacters)

    expect_that(result, equals("./something/random"))
})

test_that("RListToPythonList converts nested R list to char representation of Python list", {

    sample <- list("insert", list("random", list("text")), list("here"))

    result              <- RListToPythonList(sample)
    resultWithSeparator <- RListToPythonList(sample, separator = ",+")

    expect_that(result, equals("[\"insert\", [\"random\", \"text\"], \"here\"]"))
    expect_that(resultWithSeparator,
                equals("[\"insert\",+[\"random\",+\"text\"],+\"here\"]"))
})

test_that("appendToStartIfNotExist appends characters to beginning of a string", {

    sample <- "New Year"
    charactersToAppend <- "Happy "

    result <- appendToStartIfNotExist(sample, charactersToAppend)

    expect_that(result, equals("Happy New Year"))
})

test_that(paste("appendToStartIfNotExist returns original string if string",
                "doesn't start with specified characters"), {

    sample <- "Happy New Year"
    charactersToAppend <- "Happy"

    result <- appendToStartIfNotExist(sample, charactersToAppend)

    expect_that(result, equals("Happy New Year"))
})

test_that(paste("splitToPathAndName splits relative path to file/folder",
                "name and rest of the path"), {

    relativePath <- "path/to/my/file.exe"

    result <- splitToPathAndName( relativePath)

    expect_that(result$name, equals("file.exe"))
    expect_that(result$path, equals("path/to/my"))
})
