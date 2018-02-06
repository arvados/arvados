context("Http Request")


test_that("execyte raises exception if http verb is not valid", {

    http <- HttpRequest$new()
    expect_that(http$execute("FAKE VERB", "url"),
               throws_error("Http verb is not valid."))
}) 

test_that(paste("createQuery generates and encodes query portion of http",
                "request based on filters, limit and offset parameters"), {

    http <- HttpRequest$new()
    filters <- list(list("color", "=", "red"))
    limit <- 20
    offset <- 50
    expect_that(http$createQuery(filters, limit, offset),
                equals(paste0("/?filters=%5B%5B%22color%22%2C%22%3D%22%2C%22red",
                              "%22%5D%5D&limit=20&offset=50")))
}) 

test_that(paste("createQuery generates and empty string",
                "when filters, limit and offset parameters are set to NULL"), {

    http <- HttpRequest$new()
    expect_that(http$createQuery(NULL, NULL, NULL), equals(""))
}) 
