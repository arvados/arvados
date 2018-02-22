source("./R/Subcollection.R")
source("./R/ArvadosFile.R")
source("./R/RESTService.R")
source("./R/util.R")

#' Collection
#' 
#' Collection class provides interface for working with Arvados collections.
#' 
#' @section Usage:
#' \preformatted{collection = Collection$new(arv, uuid)}
#'
#' @section Arguments:
#' \describe{
#'   \item{arv}{Arvados object.}
#'   \item{uuid}{UUID of a collection.}
#' }
#' 
#' @section Methods:
#' \describe{
#'   \item{add(content)}{Adds ArvadosFile or Subcollection specified by content to the collection.}
#'   \item{create(fileNames, relativePath = "")}{Creates one or more ArvadosFiles and adds them to the collection at specified path.}
#'   \item{remove(fileNames)}{Remove one or more files from the collection.}
#'   \item{move(content, newLocation)}{Moves ArvadosFile or Subcollection to another location in the collection.}
#'   \item{getFileListing()}{Returns collections file content as character vector.}
#'   \item{get(relativePath)}{If relativePath is valid, returns ArvadosFile or Subcollection specified by relativePath, else returns NULL.}
#' }
#'
#' @name Collection
#' @examples
#' \dontrun{
#' arv <- Arvados$new("your Arvados token", "example.arvadosapi.com")
#' collection <- Collection$new(arv, "uuid")
#'
#' newFile <- ArvadosFile$new("myFile")
#' collection$add(newFile, "myFolder")
#'
#' createdFiles <- collection$create(c("main.cpp", lib.dll), "cpp/src/")
#'
#' collection$remove("location/to/my/file.cpp")
#'
#' collection$move("folder/file.cpp", "file.cpp")
#'
#' arvadosFile <- collection$get("location/to/my/file.cpp")
#' arvadosSubcollection <- collection$get("location/to/my/directory/")
#' }
NULL

#' @export
Collection <- R6::R6Class(

    "Collection",

    public = list(

		uuid                     = NULL,
		etag                     = NULL,
		owner_uuid               = NULL,
		created_at               = NULL,
		modified_by_client_uuid  = NULL,
		modified_by_user_uuid    = NULL,
		modified_at              = NULL,
		portable_data_hash       = NULL,
		replication_desired      = NULL,
		replication_confirmed_at = NULL,
		replication_confirmed    = NULL,
		updated_at               = NULL,
		manifest_text            = NULL,
		name                     = NULL,
		description              = NULL,
		properties               = NULL,
		delete_at                = NULL,
		file_names               = NULL,
		trash_at                 = NULL,
		is_trashed               = NULL,

		initialize = function(uuid = NULL, etag = NULL, owner_uuid = NULL,
                              created_at = NULL, modified_by_client_uuid = NULL,
                              modified_by_user_uuid = NULL, modified_at = NULL,
                              portable_data_hash = NULL, replication_desired = NULL,
                              replication_confirmed_at = NULL,
                              replication_confirmed = NULL, updated_at = NULL,
                              manifest_text = NULL, name = NULL, description = NULL,
                              properties = NULL, delete_at = NULL, file_names = NULL,
                              trash_at = NULL, is_trashed = NULL) 
        {
			self$uuid                     <- uuid
			self$etag                     <- etag
			self$owner_uuid               <- owner_uuid
			self$created_at               <- created_at
			self$modified_by_client_uuid  <- modified_by_client_uuid
			self$modified_by_user_uuid    <- modified_by_user_uuid
			self$modified_at              <- modified_at
			self$portable_data_hash       <- portable_data_hash
			self$replication_desired      <- replication_desired
			self$replication_confirmed_at <- replication_confirmed_at
			self$replication_confirmed    <- replication_confirmed
			self$updated_at               <- updated_at
			self$manifest_text            <- manifest_text
			self$name                     <- name
			self$description              <- description
			self$properties               <- properties
			self$delete_at                <- delete_at
			self$file_names               <- file_names
			self$trash_at                 <- trash_at
			self$is_trashed               <- is_trashed
			
			private$classFields <- c("uuid", "etag", "owner_uuid", 
                                     "created_at", "modified_by_client_uuid",
                                     "modified_by_user_uuid", "modified_at",
                                     "portable_data_hash", "replication_desired",
                                     "replication_confirmed_at",
                                     "replication_confirmed", "updated_at",
                                     "manifest_text", "name", "description", 
                                     "properties", "delete_at", "file_names",
                                     "trash_at", "is_trashed")
        },

        add = function(content, relativePath = "")
        {
            if(is.null(private$tree))
                private$genereateCollectionTreeStructure()

            if(relativePath == ""  ||
               relativePath == "." ||
               relativePath == "./")
            {
                subcollection <- private$tree$getTree()
            }
            else
            {
                relativePath <- trimFromEnd(relativePath, "/")
                subcollection <- self$get(relativePath)
            }

            if(is.null(subcollection))
                stop(paste("Subcollection", relativePath, "doesn't exist."))

            if("ArvadosFile"   %in% class(content) ||
               "Subcollection" %in% class(content))
            {
                if(content$getName() == "")
                    stop("Content has invalid name.")

                subcollection$add(content)
                content
            }
            else
            {
                stop(paste0("Expected AravodsFile or Subcollection object, got ",
                            paste0("(", paste0(class(content), collapse = ", "), ")"),
                            "."))
            }
        },

        create = function(fileNames, relativePath = "")
        {
            if(is.null(private$tree))
                private$genereateCollectionTreeStructure()

            if(relativePath == ""  ||
               relativePath == "." ||
               relativePath == "./")
            {
                subcollection <- private$tree$getTree()
            }
            else
            {
                relativePath  <- trimFromEnd(relativePath, "/") 
                subcollection <- self$get(relativePath)
            }

            if(is.null(subcollection))
                stop(paste("Subcollection", relativePath, "doesn't exist."))

            if(is.character(fileNames))
            {
                arvadosFiles <- NULL
                sapply(fileNames, function(fileName)
                {
                    childWithSameName <- subcollection$get(fileName)
                    if(!is.null(childWithSameName))
                        stop("Destination already contains file with same name.")

                    newFile <- ArvadosFile$new(fileName)
                    subcollection$add(newFile)

                    arvadosFiles <<- c(arvadosFiles, newFile)
                })

                if(length(arvadosFiles) == 1)
                    return(arvadosFiles[[1]])
                else
                    return(arvadosFiles)
            }
            else 
            {
                stop(paste0("Expected character vector, got ",
                            paste0("(", paste0(class(fileNames), collapse = ", "), ")"),
                            "."))
            }
        },

        remove = function(paths)
        {
            if(is.null(private$tree))
                private$genereateCollectionTreeStructure()

            if(is.character(paths))
            {
                sapply(paths, function(filePath)
                {
                    filePath <- trimFromEnd(filePath, "/")
                    file <- self$get(filePath)

                    if(is.null(file))
                        stop(paste("File", filePath, "doesn't exist."))

                    parent <- file$getParent()

                    if(is.null(parent))
                        stop("You can't delete root folder.")

                    parent$remove(file$getName())
                })

                "Content removed"
            }
            else 
            {
                stop(paste0("Expected character vector, got ",
                            paste0("(", paste0(class(paths), collapse = ", "), ")"),
                            "."))
            }
        },

        move = function(content, newLocation)
        {
            if(is.null(private$tree))
                private$genereateCollectionTreeStructure()

            content <- trimFromEnd(content, "/")

            elementToMove <- self$get(content)

            if(is.null(elementToMove))
                stop("Content you want to move doesn't exist in the collection.")

            elementToMove$move(newLocation)
        },

        getFileListing = function()
        {
            if(is.null(private$tree))
                private$genereateCollectionTreeStructure()

            content <- private$REST$getCollectionContent(self$uuid)
            content[order(tolower(content))]
        },

        get = function(relativePath)
        {
            if(is.null(private$tree))
                private$genereateCollectionTreeStructure()

            private$tree$getElement(relativePath)
        },

        getRESTService = function() private$REST,
        setRESTService = function(newRESTService) private$REST <- newRESTService
    ),

    private = list(

        REST        = NULL,
        tree        = NULL,
        fileContent = NULL,
        classFields = NULL,

        genereateCollectionTreeStructure = function()
        {
            if(is.null(self$uuid))
                stop("Collection uuid is not defined.")

            if(is.null(private$REST))
                stop("REST service is not defined.")

            private$fileContent <- private$REST$getCollectionContent(self$uuid)
            private$tree <- CollectionTree$new(private$fileContent, self)
        }
    ),

    cloneable = FALSE
)

#' print.Collection
#'
#' Custom print function for Collection class
#'
#' @param x Instance of Collection class
#' @param ... Optional arguments.
#' @export 
print.Collection = function(x, ...)
{
    cat(paste0("Type: ", "\"", "Arvados Collection", "\""), sep = "\n")
    cat(paste0("uuid: ", "\"", x$uuid,               "\""), sep = "\n")
}
