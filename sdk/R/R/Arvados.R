source("./R/HttpRequest.R")
source("./R/HttpParser.R")
source("./R/custom_classes.R")

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

        getToken          = "function",
        getHostName       = "function",

        #Todo(Fudo): These are hardcoded and for debug only. Remove them later on.
        getWebDavToken    = "function",
        getWebDavHostName = "function",

        collection_get    = "function",
        collection_list   = "function",
        collection_create = "function",
        collection_update = "function",
        collection_delete = "function"
    ),

    methods = list(

        initialize = function(auth_token = NULL, host_name = NULL, webDavToken = NULL, webDavHostName = NULL) 
        {
            # Private state
            if(!is.null(host_name))
               Sys.setenv(ARVADOS_API_HOST  = host_name)

            if(!is.null(auth_token))
                Sys.setenv(ARVADOS_API_TOKEN = auth_token)

            host  <- Sys.getenv("ARVADOS_API_HOST");
            token <- Sys.getenv("ARVADOS_API_TOKEN");

            if(host == "" | token == "")
                stop("Please provide host name and authentification token or set ARVADOS_API_HOST and ARVADOS_API_TOKEN environmental variables.")

            version <- "v1"
            host  <- paste0("https://", host, "/arvados/", version, "/")

            # Public methods
            getToken <<- function() { token }
            getHostName <<- function() { host }

            #Todo(Fudo): Hardcoded credentials to WebDAV server. Remove them later
            getWebDavToken    <<- function() { webDavToken }
            getWebDavHostName <<- function() { webDavHostName }

            collection_get <<- function(uuid) 
            {
                collection_url <- paste0(host, "collections/", uuid)
                headers <- list(Authorization = paste("OAuth2", token))

                http <- HttpRequest() 
                serverResponse <- http$GET(collection_url, headers)

                httpParser <- HttpParser()
                collection <- httpParser$parseJSONResponse(serverResponse)

                if(!is.null(collection$errors))
                    stop(collection$errors)       

                collection
            }

            collection_list <<- function(filters = NULL, limit = 100, offset = 0) 
            {
                collection_url <- paste0(host, "collections")
                headers <- list(Authorization = paste("OAuth2", token))

                http <- HttpRequest() 
                serverResponse <- http$GET(collection_url, headers, NULL, filters, limit, offset)

                httpParser <- HttpParser()
                collection <- httpParser$parseJSONResponse(serverResponse)

                if(!is.null(collection$errors))
                    stop(collection$errors)       

                collection
            }

            collection_delete <<- function(uuid) 
            {
                collection_url <- paste0(host, "collections/", uuid)
                headers <- list("Authorization" = paste("OAuth2", token),
                                "Content-Type"  = "application/json")

                http <- HttpRequest() 
                serverResponse <- http$DELETE(collection_url, headers)

                httpParser <- HttpParser()
                collection <- httpParser$parseJSONResponse(serverResponse)

                if(!is.null(collection$errors))
                    stop(collection$errors)       

                collection
            }

            collection_update <<- function(uuid, body) 
            {
                collection_url <- paste0(host, "collections/", uuid)
                headers <- list("Authorization" = paste("OAuth2", token),
                                "Content-Type"  = "application/json")
                body <- jsonlite::toJSON(body, auto_unbox = T)

                http <- HttpRequest() 
                serverResponse <- http$PUT(collection_url, headers, body)

                httpParser <- HttpParser()
                collection <- httpParser$parseJSONResponse(serverResponse)

                if(!is.null(collection$errors))
                    stop(collection$errors)       

                collection
            }

            collection_create <<- function(body) 
            {
                collection_url <- paste0(host, "collections")
                headers <- list("Authorization" = paste("OAuth2", token),
                                "Content-Type"  = "application/json")
                body <- jsonlite::toJSON(body, auto_unbox = T)

                http <- HttpRequest() 
                serverResponse <- http$POST(collection_url, headers, body)

                httpParser <- HttpParser()
                collection <- httpParser$parseJSONResponse(serverResponse)

                if(!is.null(collection$errors))
                    stop(collection$errors)       

                collection
            }
        }
    )
)
