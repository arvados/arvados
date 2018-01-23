source("./R/RESTService.R")
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

        initialize = function(authToken = NULL, hostName = NULL)
        {
            if(!is.null(hostName))
               Sys.setenv(ARVADOS_API_HOST = hostName)

            if(!is.null(authToken))
                Sys.setenv(ARVADOS_API_TOKEN = authToken)

            hostName  <- Sys.getenv("ARVADOS_API_HOST");
            token <- Sys.getenv("ARVADOS_API_TOKEN");

            if(hostName == "" | token == "")
                stop(paste0("Please provide host name and authentification token",
                            " or set ARVADOS_API_HOST and ARVADOS_API_TOKEN",
                            " environment variables."))

            private$REST  <- RESTService$new(token, hostName, NULL,
                                             HttpRequest$new(), HttpParser$new())
            private$token <- private$REST$token
            private$host  <- private$REST$hostName
        },

        getToken          = function() private$REST$token,
        getHostName       = function() private$REST$hostName,
        getWebDavHostName = function() private$REST$getWebDavHostName(),
        getRESTService    = function() private$REST,
        setRESTService    = function(newRESTService) private$REST <- newRESTService,

        getCollection = function(uuid)
        {
            collection <- private$REST$getResource("collections", uuid)
            collection
        },

        listCollections = function(filters = NULL, limit = 100, offset = 0)
        {
            if(!is.null(filters))
                names(filters) <- c("collection")

            collections <- private$REST$listResources("collections", filters,
                                                      limit, offset)
            collections
        },

        listAllCollections = function(filters = NULL)
        {
            if(!is.null(filters))
                names(filters) <- c("collection")

            collectionURL <- paste0(private$host, "collections")
            allCollection <- private$REST$fetchAllItems(collectionURL, filters)
            allCollection
        },

        deleteCollection = function(uuid)
        {
            removedCollection <- private$REST$deleteResource("collections", uuid)
            removedCollection
        },

        updateCollection = function(uuid, newContent)
        {
            body <- list(list())
            names(body) <- c("collection")
            body$collection <- newContent

            updatedCollection <- private$REST$updateResource("collections",
                                                             uuid, body)
            updatedCollection
        },

        createCollection = function(content)
        {
            body <- list(list())
            names(body) <- c("collection")
            body$collection <- content

            newCollection <- private$REST$createResource("collections", body)
            newCollection
        },

        getProject = function(uuid)
        {
            project <- private$REST$getResource("groups", uuid)
            project
        },

        createProject = function(content)
        {
            body <- list(list())
            names(body) <- c("group")
            body$group <- c("group_class" = "project", content)

            newProject <- private$REST$createResource("groups", body)
            newProject
        },

        updateProject = function(uuid, newContent)
        {
            body <- list(list())
            names(body) <- c("group")
            body$group <- newContent

            updatedProject <- private$REST$updateResource("groups",
                                                          uuid, body)
            updatedProject
        },

        listProjects = function(filters = NULL, limit = 100, offset = 0)
        {
            if(!is.null(filters))
                names(filters) <- c("groups")

            filters[[length(filters) + 1]] <- list("group_class", "=", "project")

            projects <- private$REST$listResources("groups", filters,
                                                   limit, offset)
            projects
        },

        listAllProjects = function(filters = NULL)
        {
            if(!is.null(filters))
                names(filters) <- c("groups")

            filters[[length(filters) + 1]] <- list("group_class", "=", "project")

            projectURL <- paste0(private$host, "groups")

            result <- private$REST$fetchAllItems(projectURL, filters)
            result
        },

        deleteProject = function(uuid)
        {
            removedProject <- private$REST$deleteResource("groups", uuid)
            removedProject
        }
    ),

    private = list(

        token          = NULL,
        host           = NULL,
        REST           = NULL
    ),

    cloneable = FALSE
)
