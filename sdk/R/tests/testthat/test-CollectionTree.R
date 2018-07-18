# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

context("CollectionTree")

test_that("constructor creates file tree from character array properly", {

    collection <- "myCollection"
    characterArray <- c("animal",
                        "animal/dog",
                        "boat")

    collectionTree <- CollectionTree$new(characterArray, collection)

    root   <- collectionTree$getTree()
    animal <- collectionTree$getElement("animal")
    dog    <- collectionTree$getElement("animal/dog")
    boat   <- collectionTree$getElement("boat")

    rootHasNoParent             <- is.null(root$getParent())
    rootIsOfTypeSubcollection   <- "Subcollection" %in% class(root)
    animalIsOfTypeSubcollection <- "Subcollection" %in% class(animal)
    dogIsOfTypeArvadosFile      <- "ArvadosFile" %in% class(dog)
    boatIsOfTypeArvadosFile     <- "ArvadosFile" %in% class(boat)
    animalsParentIsRoot         <- animal$getParent()$getName() == root$getName()
    animalContainsDog           <- animal$getFirst()$getName() == dog$getName()
    dogsParentIsAnimal          <- dog$getParent()$getName() == animal$getName()
    boatsParentIsRoot           <- boat$getParent()$getName() == root$getName()

    allElementsBelongToSameCollection <- root$getCollection()   == "myCollection" &&
                                         animal$getCollection() == "myCollection" &&
                                         dog$getCollection()    == "myCollection" &&
                                         boat$getCollection()   == "myCollection"

    expect_that(root$getName(), equals(""))
    expect_that(rootIsOfTypeSubcollection, is_true())
    expect_that(rootHasNoParent, is_true())
    expect_that(animalIsOfTypeSubcollection, is_true())
    expect_that(animalsParentIsRoot, is_true())
    expect_that(animalContainsDog, is_true())
    expect_that(dogIsOfTypeArvadosFile, is_true())
    expect_that(dogsParentIsAnimal, is_true())
    expect_that(boatIsOfTypeArvadosFile, is_true())
    expect_that(boatsParentIsRoot, is_true())
    expect_that(allElementsBelongToSameCollection, is_true())
})

test_that("getElement returns element from tree if element exists on specified path", {

    collection <- "myCollection"
    characterArray <- c("animal",
                        "animal/dog",
                        "boat")

    collectionTree <- CollectionTree$new(characterArray, collection)

    dog <- collectionTree$getElement("animal/dog")

    expect_that(dog$getName(), equals("dog"))
})

test_that("getElement returns NULL from tree if element doesn't exists on specified path", {

    collection <- "myCollection"
    characterArray <- c("animal",
                        "animal/dog",
                        "boat")

    collectionTree <- CollectionTree$new(characterArray, collection)

    fish <- collectionTree$getElement("animal/fish")
    fishIsNULL <- is.null(fish)

    expect_that(fishIsNULL, is_true())
})

test_that("getElement trims ./ from start of relativePath", {

    collection <- "myCollection"
    characterArray <- c("animal",
                        "animal/dog",
                        "boat")

    collectionTree <- CollectionTree$new(characterArray, collection)

    dog <- collectionTree$getElement("animal/dog")
    dogWithDotSlash <- collectionTree$getElement("./animal/dog")

    expect_that(dogWithDotSlash$getName(), equals(dog$getName()))
})

test_that("getElement trims / from end of relativePath", {

    collection <- "myCollection"
    characterArray <- c("animal",
                        "animal/dog",
                        "boat")

    collectionTree <- CollectionTree$new(characterArray, collection)

    animal <- collectionTree$getElement("animal")
    animalWithSlash <- collectionTree$getElement("animal/")

    expect_that(animalWithSlash$getName(), equals(animal$getName()))
})
