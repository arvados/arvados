source("./R/HttpRequest.R")
source("./R/HttpParser.R")

#' Arvados SDK Object
#'
#' All Arvados logic is inside this class
#'
#' @field token Token represents user authentification token.
#' @field host Host represents server name we wish to connect to.
#' @examples arv = Arvados("token", "host_name")
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
            version <- "v1"
            #Todo(Fudo): Validate token
            token <<- auth_token
            host  <<- paste0("https://", host_name, "/arvados/", version, "/")
        }
    )
)

#' collection_get
#'
#' Get Arvados collection
#'
#' @name collection_get
#' @field uuid UUID of the given collection
#' @examples arv = Arvados("token", "hostName")
#' @examples arv$collection_get("uuid")
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
#' Retreive list of collections based on provided filter.
#'
#' @name collection_list
#' @field filters List of filters we want to use to retreive list of collections.
#' @field limit Limits the number of result returned by server.
#' @field offset Offset from beginning of the result set.
#' @examples arv = Arvados("token", "hostName")
#' @examples arv$collection_list(list("uuid", "=" "aaaaa-bbbbb-ccccccccccccccc"))
Arvados$methods(

    collection_list = function(filters = NULL, limit = NULL, offset = NULL) 
    {
        #Todo(Fudo): Implement limit and offset
        collection_relative_url <- "collections"
        http_request <- HttpRequest("GET", token, host, collection_relative_url,
                                    body = NULL,  filters, limit, offset) 

        server_response <- http_request$execute()

        httpParser <- HttpParser()
        collection <- httpParser$parseCollectionGet(server_response)

        if(!is.null(collection$errors))
            stop(collection$errors)       

        class(collection) <- "ArvadosCollectionList"

        return(collection)
    }
)

#' collection_create
#'
#' Create Arvados collection
#'
#' @name collection_create
#' @field body Structure of the collection we want to create.
#' @examples arv = Arvados("token", "hostName")
#' @examples arv$collection_create(list(collection = list(name = "myCollection")))
Arvados$methods(

    collection_create = function(body) 
    {
        collection_relative_url <- paste0("collections/", "/?alt=json")
        body = jsonlite::toJSON(body, auto_unbox = T)

        http_request    <- HttpRequest("POST", token, host, collection_relative_url, body) 
        server_response <- http_request$execute()

        httpParser <- HttpParser()
        collection <- httpParser$parseCollectionGet(server_response)

        if(!is.null(collection$errors))
            stop(collection$errors)       

        class(collection) <- "ArvadosCollection"

        return(collection)
    }
)

#' collection_delete
#'
#' Delete Arvados collection
#'
#' @name collection_delete
#' @field uuid UUID of the collection we want to delete.
#' @examples arv = Arvados("token", "hostName")
#' @examples arv$collection_delete(uuid = "aaaaa-bbbbb-ccccccccccccccc")
Arvados$methods(

    collection_delete = function(uuid) 
    {
        collection_relative_url <- paste0("collections/", uuid, "/?alt=json")

        http_request    <- HttpRequest("DELETE", token, host, collection_relative_url) 
        server_response <- http_request$execute()

        httpParser <- HttpParser()
        collection <- httpParser$parseCollectionGet(server_response)

        if(!is.null(collection$errors))
            stop(collection$errors)       

        class(collection) <- "ArvadosCollection"

        return(collection)
    }
)

#' collection_update
#'
#' Update Arvados collection
#'
#' @name collection_update
#' @field uuid UUID of the collection we want to update.
#' @field body New structure of the collection.
#' @examples arv = Arvados("token", "hostName")
#' @examples arv$collection_update(uuid = "aaaaa-bbbbb-ccccccccccccccc", list(collection = list(name = "newName")))
Arvados$methods(

    collection_update = function(uuid, body) 
    {
        collection_relative_url <- paste0("collections/", uuid, "/?alt=json")
        body = jsonlite::toJSON(body, auto_unbox = T)

        http_request    <- HttpRequest("PUT", token, host, collection_relative_url, body) 
        server_response <- http_request$execute()

        httpParser <- HttpParser()
        collection <- httpParser$parseCollectionGet(server_response)

        if(!is.null(collection$errors))
            stop(collection$errors)       

        class(collection) <- "ArvadosCollection"

        return(collection)
    }
)
