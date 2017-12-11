source("./R/HttpRequest.R")
source("./R/HttpParser.R")

#' Arvados SDK Object
#'
#' All Arvados logic is inside this class
#'
#' @field token Token represents user authentification token.
#' @field host Host represents server name we wish to connect to.
#' @examples arv = Arvados$new("token", "host_name")
#' @export Arvados
Arvados <- R6::R6Class(

    "Arvados",

    public = list(

        initialize = function(auth_token = NULL, host_name = NULL) 
        {
            if(!is.null(host_name))
               Sys.setenv(ARVADOS_API_HOST  = host_name)

            if(!is.null(auth_token))
                Sys.setenv(ARVADOS_API_TOKEN = auth_token)

            host  <- Sys.getenv("ARVADOS_API_HOST");
            token <- Sys.getenv("ARVADOS_API_TOKEN");

            if(host == "" | token == "")
                stop("Please provide host name and authentification token or set ARVADOS_API_HOST and ARVADOS_API_TOKEN environmental variables.")

            discoveryDocumentURL <- paste0("https://", host, "/discovery/v1/apis/arvados/v1/rest")

            version <- "v1"
            host  <- paste0("https://", host, "/arvados/", version, "/")

            private$http <- HttpRequest$new()
            private$httpParser <- HttpParser$new()
            private$token <- token
            private$host <- host
            
            headers <- list(Authorization = paste("OAuth2", private$token))

            serverResponse <- private$http$GET(discoveryDocumentURL, headers)

            discoveryDocument <- private$httpParser$parseJSONResponse(serverResponse)
            private$webDavHostName <- discoveryDocument$keepWebServiceUrl
        },

        getToken    = function() private$token,
        getHostName = function() private$host,

        #Todo(Fudo): Hardcoded credentials to WebDAV server. Remove them later
        getWebDavHostName = function() private$webDavHostName,

        getCollection = function(uuid) 
        {
            collectionURL <- paste0(private$host, "collections/", uuid)
            headers <- list(Authorization = paste("OAuth2", private$token))

            serverResponse <- private$http$GET(collectionURL, headers)

            collection <- private$httpParser$parseJSONResponse(serverResponse)

            if(!is.null(collection$errors))
                stop(collection$errors)       

            collection
        },

        listCollections = function(filters = NULL, limit = 100, offset = 0) 
        {
            collectionURL <- paste0(private$host, "collections")
            headers <- list(Authorization = paste("OAuth2", private$token))

            serverResponse <- private$http$GET(collectionURL, headers, NULL, filters, limit, offset)

            collection <- private$httpParser$parseJSONResponse(serverResponse)

            if(!is.null(collection$errors))
                stop(collection$errors)       

            collection
        },

        deleteCollection = function(uuid) 
        {
            collectionURL <- paste0(private$host, "collections/", uuid)
            headers <- list("Authorization" = paste("OAuth2", private$token),
                            "Content-Type"  = "application/json")

            serverResponse <- private$http$DELETE(collectionURL, headers)

            collection <- private$httpParser$parseJSONResponse(serverResponse)

            if(!is.null(collection$errors))
                stop(collection$errors)       

            collection
        },

        updateCollection = function(uuid, body) 
        {
            collectionURL <- paste0(private$host, "collections/", uuid)
            headers <- list("Authorization" = paste("OAuth2", private$token),
                            "Content-Type"  = "application/json")

            body <- jsonlite::toJSON(body, auto_unbox = T)

            serverResponse <- private$http$PUT(collectionURL, headers, body)

            collection <- private$httpParser$parseJSONResponse(serverResponse)

            if(!is.null(collection$errors))
                stop(collection$errors)       

            collection
        },

        createCollection = function(body) 
        {
            collectionURL <- paste0(private$host, "collections")
            headers <- list("Authorization" = paste("OAuth2", private$token),
                            "Content-Type"  = "application/json")
            body <- jsonlite::toJSON(body, auto_unbox = T)

            serverResponse <- private$http$POST(collectionURL, headers, body)

            collection <- private$httpParser$parseJSONResponse(serverResponse)

            if(!is.null(collection$errors))
                stop(collection$errors)       

            collection
        }

    ),
    
    private = list(

        token          = NULL,
        host           = NULL,
        webDavHostName = NULL,
        http           = NULL,
        httpParser     = NULL
    ),
    
    cloneable = FALSE
)
