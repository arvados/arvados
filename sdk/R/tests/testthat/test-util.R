context("Utility function")

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
