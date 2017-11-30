source("./R/HttpRequest.R")
source("./R/HttpParser.R")

#' Arvados SDK Object
#'
#' All Arvados logic is inside this class
#'
#' @field token represents user authentification token.
#' @field host represents server name we wish to connect to.
#' @export Arvados
Arvados <- setRefClass(

    "Arvados",

    fields = list(
        token = "character",
        host  = "character"
    ),

    methods = list(

        initialize = function(auth_token, host_name) 
        {
            #Todo(Fudo): Validate token
            token <<- auth_token
            host  <<- host_name
        }
    )
)

#' collection_get
#'
#' Get Arvados collection
#'
#' @name collection_get
#' @field uuid UUID of the given collection
Arvados$methods(

    collection_get = function(uuid) 
    {
        collection_relative_url <- paste0("collections/", uuid)
        http_request <- HttpRequest("GET", token, host, collection_relative_url) 
        server_response <- http_request$execute()

        httpParser <- HttpParser()
        collection <- httpParser$parseCollectionGet(server_response)

        if(!is.null(collection$errors))
            stop(collection$errors)       

        class(collection) <- "ArvadosCollection"

        return(collection)
    }
)

#' collection_list
#'
#' List Arvados collections based on filter matching
#'
#' @name collection_list
#' @field uuid UUID of the given collection
Arvados$methods(

    collection_list = function(filters = NULL, limit = NULL, offset = NULL) 
    {
        #Todo(Fudo): Implement limit and offset
        collection_relative_url <- "collections"
        http_request <- HttpRequest("GET", token, host, collection_relative_url, filters, limit, offset) 
        server_response <- http_request$execute()

        httpParser <- HttpParser()
        collection <- httpParser$parseCollectionGet(server_response)

        if(!is.null(collection$errors))
            stop(collection$errors)       

        class(collection) <- "ArvadosCollectionList"

        return(collection)
    }
)
