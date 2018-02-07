source("./R/RESTService.R")
source("./R/HttpRequest.R")
source("./R/HttpParser.R")

#' Arvados
#' 
#' Arvados class gives users ability to manipulate collections and projects.
#' 
#' @section Usage:
#' \preformatted{arv = Arvados$new(authToken, hostName, numRetries = 0)}
#'
#' @section Arguments:
#' \describe{
#'   \item{authToken}{Authentification token. If not specified ARVADOS_API_TOKEN environment variable will be used.}
#'   \item{hostName}{Host name. If not specified ARVADOS_API_HOST environment variable will be used.}
#'   \item{numRetries}{Number which specifies how many times to retry failed service requests.}
#' }
#' 
#' @section Methods:
#' \describe{
#'   \item{getToken()}{Returns authentification token currently in use.}
#'   \item{getHostName()}{Returns host name currently in use.}
#'   \item{getNumRetries()}{Returns number which specifies how many times to retry failed service requests.}
#'   \item{setNumRetries(newNumOfRetries)}{Sets number which specifies how many times to retry failed service requests.}
#'   \item{getCollection(uuid)}{Get collection with specified UUID.}
#'   \item{listCollections(filters = NULL, limit = 100, offset = 0)}{Returns list of collections based on filters parameter.}
#'   \item{listAllCollections(filters = NULL)}{Lists all collections, based on filters parameter, even if the number of items is greater than maximum API limit.}
#'   \item{deleteCollection(uuid)}{Deletes collection with specified UUID.}
#'   \item{updateCollection(uuid, newContent)}{Updates collection with specified UUID.}
#'   \item{createCollection(content)}{Creates new collection.}
#'   \item{getProject(uuid)}{Get project with specified UUID.}
#'   \item{listProjects(filters = NULL, limit = 100, offset = 0)}{Returns list of projects based on filters parameter.}
#'   \item{listAllProjects(filters = NULL)}{Lists all projects, based on filters parameter, even if the number of items is greater than maximum API limit.}
#'   \item{deleteProject(uuid)}{Deletes project with specified UUID.}
#'   \item{updateProject(uuid, newContent)}{Updates project with specified UUID.}
#'   \item{createProject(content)}{Creates new project.}
#' }
#'
#' @name Arvados
#' @examples
#' \dontrun{
#' arv <- Arvados$new("your Arvados token", "example.arvadosapi.com")
#'
#' collection <- arv$getCollection("uuid")
#'
#' collectionList <- arv$listCollections(list(list("name", "like", "Test%")))
#' collectionList <- arv$listAllCollections(list(list("name", "like", "Test%")))
#'
#' deletedCollection <- arv$deleteCollection("uuid")
#'
#' updatedCollection <- arv$updateCollection("uuid", list(name = "New name", description = "New description"))
#'
#' createdCollection <- arv$createCollection(list(name = "Example", description = "This is a test collection"))
#' }
NULL

#' @export
Arvados <- R6::R6Class(

    "Arvados",

    public = list(

        initialize = function(authToken = NULL, hostName = NULL, numRetries = 0)
        {
            if(!is.null(hostName))
               Sys.setenv(ARVADOS_API_HOST = hostName)

            if(!is.null(authToken))
                Sys.setenv(ARVADOS_API_TOKEN = authToken)

            hostName  <- Sys.getenv("ARVADOS_API_HOST");
            token     <- Sys.getenv("ARVADOS_API_TOKEN");

            if(hostName == "" | token == "")
                stop(paste("Please provide host name and authentification token",
                           "or set ARVADOS_API_HOST and ARVADOS_API_TOKEN",
                           "environment variables."))

            private$numRetries  <- numRetries
            private$REST  <- RESTService$new(token, hostName,
                                             HttpRequest$new(), HttpParser$new(),
                                             numRetries)

            private$token <- private$REST$token
            private$host  <- private$REST$hostName
        },

        getToken          = function() private$REST$token,
        getHostName       = function() private$REST$hostName,
        getWebDavHostName = function() private$REST$getWebDavHostName(),
        getRESTService    = function() private$REST,
        setRESTService    = function(newRESTService) private$REST <- newRESTService,

        getNumRetries = function() private$REST$numRetries,
        setNumRetries = function(newNumOfRetries)
        {
            private$REST$setNumRetries(newNumOfRetries)
        },

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

            updatedProject <- private$REST$updateResource("groups", uuid, body)
            updatedProject
        },

        listProjects = function(filters = NULL, limit = 100, offset = 0)
        {
            if(!is.null(filters))
                names(filters) <- c("groups")

            filters[[length(filters) + 1]] <- list("group_class", "=", "project")

            projects <- private$REST$listResources("groups", filters, limit, offset)
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

        token      = NULL,
        host       = NULL,
        REST       = NULL,
        numRetries = NULL
    ),

    cloneable = FALSE
)

#' @export print.Arvados
print.Arvados = function(arvados)
{
    cat(paste0("Type:  ", "\"", "Arvados",             "\""), sep = "\n")
    cat(paste0("Host:  ", "\"", arvados$getHostName(), "\""), sep = "\n")
    cat(paste0("Token: ", "\"", arvados$getToken(),    "\""), sep = "\n")
}
