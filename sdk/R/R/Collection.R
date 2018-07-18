# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

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

		uuid = NULL,
        # api  = NULL,

		initialize = function(api, uuid)
        {
            # self$api <- api
            private$REST <- api$getRESTService()

            self$uuid <- uuid

            private$fileContent <- private$REST$getCollectionContent(uuid)
            private$tree <- CollectionTree$new(private$fileContent, self)
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

		toJSON = function()
        {
			fields <- sapply(private$classFields, function(field)
			{
				self[[field]]
			}, USE.NAMES = TRUE)

			jsonlite::toJSON(list("collection" =
                     Filter(Negate(is.null), fields)), auto_unbox = TRUE)
		},

		isEmpty = function() {
			fields <- sapply(private$classFields,
			                 function(field) self[[field]])

			if(any(sapply(fields, function(field) !is.null(field) && field != "")))
				FALSE
			else
				TRUE
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
