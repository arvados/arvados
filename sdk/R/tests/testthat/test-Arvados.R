context("Arvados API")

test_that("Arvados constructor will use environment variables if no parameters are passed to it", {

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

test_that("Arvados constructor preferes constructor fields over environment variables", {

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
