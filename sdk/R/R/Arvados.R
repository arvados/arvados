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

            host_name  <- Sys.getenv("ARVADOS_API_HOST");
            token <- Sys.getenv("ARVADOS_API_TOKEN");

            if(host_name == "" | token == "")
                stop(paste0("Please provide host name and authentification token",
                            " or set ARVADOS_API_HOST and ARVADOS_API_TOKEN",
                            " environmental variables."))

            version <- "v1"
            host  <- paste0("https://", host_name, "/arvados/", version, "/")

            private$http       <- HttpRequest$new()
            private$httpParser <- HttpParser$new()
            private$token      <- token
            private$host       <- host
            private$rawHost    <- host_name
        },

        getToken    = function() private$token,
        getHostName = function() private$host,

        getHttpClient = function() private$http,
        setHttpClient = function(newClient) private$http <- newClient,

        getHttpParser = function() private$httpParser,
        setHttpParser = function(newParser) private$httpParser <- newParser,

        getWebDavHostName = function()
        {
            if(is.null(private$webDavHostName))
            {
                discoveryDocumentURL <- paste0("https://", private$rawHost,
                                               "/discovery/v1/apis/arvados/v1/rest")

                headers <- list(Authorization = paste("OAuth2", private$token))

                serverResponse <- private$http$GET(discoveryDocumentURL, headers)

                discoveryDocument <- private$httpParser$parseJSONResponse(serverResponse)
                private$webDavHostName <- discoveryDocument$keepWebServiceUrl

                if(is.null(private$webDavHostName))
                    stop("Unable to find WebDAV server.")
            }

            private$webDavHostName
        },

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

            if(!is.null(filters))
                names(filters) <- c("collection")

            serverResponse <- private$http$GET(collectionURL, headers, filters,
                                               limit, offset)

            collections <- private$httpParser$parseJSONResponse(serverResponse)

            if(!is.null(collections$errors))
                stop(collections$errors)

            collections
        },

        listAllCollections = function(filters = NULL)
        {
            if(!is.null(filters))
                names(filters) <- c("collection")

            collectionURL <- paste0(private$host, "collections")
            private$fetchAllItems(collectionURL, filters)
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

        updateCollection = function(uuid, newContent)
        {
            collectionURL <- paste0(private$host, "collections/", uuid)
            headers <- list("Authorization" = paste("OAuth2", private$token),
                            "Content-Type"  = "application/json")

            body <- list(list())
            #test if this is needed
            names(body) <- c("collection")
            body$collection <- newContent

            body <- jsonlite::toJSON(body, auto_unbox = T)

            serverResponse <- private$http$PUT(collectionURL, headers, body)

            collection <- private$httpParser$parseJSONResponse(serverResponse)

            if(!is.null(collection$errors))
                stop(collection$errors)

            collection
        },

        createCollection = function(content)
        {
            collectionURL <- paste0(private$host, "collections")
            headers <- list("Authorization" = paste("OAuth2", private$token),
                            "Content-Type"  = "application/json")

            body <- list(list())
            names(body) <- c("collection")
            body$collection <- content

            body <- jsonlite::toJSON(body, auto_unbox = T)

            serverResponse <- private$http$POST(collectionURL, headers, body)

            collection <- private$httpParser$parseJSONResponse(serverResponse)

            if(!is.null(collection$errors))
                stop(collection$errors)

            collection
        },

        getProject = function(uuid)
        {
            projectURL <- paste0(private$host, "groups/", uuid)
            headers <- list(Authorization = paste("OAuth2", private$token))

            serverResponse <- private$http$GET(projectURL, headers)

            project <- private$httpParser$parseJSONResponse(serverResponse)

            if(!is.null(project$errors))
                stop(project$errors)

            project
        },

        createProject = function(content)
        {
            projectURL <- paste0(private$host, "groups")
            headers <- list("Authorization" = paste("OAuth2", private$token),
                            "Content-Type"  = "application/json")

            body <- list(list())
            names(body) <- c("group")
            body$group <- c("group_class" = "project", content)
            body <- jsonlite::toJSON(body, auto_unbox = T)

            serverResponse <- private$http$POST(projectURL, headers, body)

            project <- private$httpParser$parseJSONResponse(serverResponse)

            if(!is.null(project$errors))
                stop(project$errors)

            project
        },

        updateProject = function(uuid, newContent)
        {
            projectURL <- paste0(private$host, "groups/", uuid)
            headers <- list("Authorization" = paste("OAuth2", private$token),
                            "Content-Type"  = "application/json")

            body <- list(list())
            names(body) <- c("group")
            body$group <- newContent
            body <- jsonlite::toJSON(body, auto_unbox = T)

            serverResponse <- private$http$PUT(projectURL, headers, body)

            project <- private$httpParser$parseJSONResponse(serverResponse)

            if(!is.null(project$errors))
                stop(project$errors)

            project
        },

        listProjects = function(filters = NULL, limit = 100, offset = 0)
        {
            projectURL <- paste0(private$host, "groups")
            headers <- list(Authorization = paste("OAuth2", private$token))

            if(!is.null(filters))
                names(filters) <- c("groups")

            filters[[length(filters) + 1]] <- list("group_class", "=", "project")

            serverResponse <- private$http$GET(projectURL, headers, filters,
                                               limit, offset)

            projects <- private$httpParser$parseJSONResponse(serverResponse)

            if(!is.null(projects$errors))
                stop(projects$errors)

            projects
        },

        listAllProjects = function(filters = NULL)
        {
            if(!is.null(filters))
                names(filters) <- c("groups")

            filters[[length(filters) + 1]] <- list("group_class", "=", "project")

            projectURL <- paste0(private$host, "groups")

            private$fetchAllItems(projectURL, filters)
        },

        deleteProject = function(uuid)
        {
            projectURL <- paste0(private$host, "groups/", uuid)
            headers <- list("Authorization" = paste("OAuth2", private$token),
                            "Content-Type"  = "application/json")

            serverResponse <- private$http$DELETE(projectURL, headers)

            project <- private$httpParser$parseJSONResponse(serverResponse)

            if(!is.null(project$errors))
                stop(project$errors)

            project
        }
    ),

    private = list(

        token          = NULL,
        host           = NULL,
        rawHost        = NULL,
        webDavHostName = NULL,
        http           = NULL,
        httpParser     = NULL,

        fetchAllItems = function(resourceURL, filters)
        {
            headers <- list(Authorization = paste("OAuth2", private$token))

            offset <- 0
            itemsAvailable <- .Machine$integer.max
            items <- c()
            while(length(items) < itemsAvailable)
            {
                serverResponse <- private$http$GET(url          = resourceURL,
                                                   headers      = headers,
                                                   queryFilters = filters,
                                                   limit        = NULL,
                                                   offset       = offset)

                parsedResponse <- private$httpParser$parseJSONResponse(serverResponse)

                if(!is.null(parsedResponse$errors))
                    stop(parsedResponse$errors)

                items          <- c(items, parsedResponse$items)
                offset         <- length(items)
                itemsAvailable <- parsedResponse$items_available
            }

            items
        }
    ),

    cloneable = FALSE
)
