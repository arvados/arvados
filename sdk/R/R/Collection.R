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
#'   \item{create(files)}{Creates one or more ArvadosFiles and adds them to the collection at specified path.}
#'   \item{remove(fileNames)}{Remove one or more files from the collection.}
#'   \item{move(content, destination)}{Moves ArvadosFile or Subcollection to another location in the collection.}
#'   \item{copy(content, destination)}{Copies ArvadosFile or Subcollection to another location in the collection.}
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

		initialize = function(api, uuid)
        {
            private$REST <- api$getRESTService()
            self$uuid <- uuid
        },

        add = function(content, relativePath = "")
        {
            if(is.null(private$tree))
                private$generateCollectionTreeStructure()

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
                if(!is.null(content$getCollection()))
                    stop("Content already belongs to a collection.")

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

        create = function(files)
        {
            if(is.null(private$tree))
                private$generateCollectionTreeStructure()

            if(is.character(files))
            {
                sapply(files, function(file)
                {
                    childWithSameName <- self$get(file)
                    if(!is.null(childWithSameName))
                        stop("Destination already contains file with same name.")

                    newTreeBranch <- private$tree$createBranch(file)
                    private$tree$addBranch(private$tree$getTree(), newTreeBranch)

                    private$REST$create(file, self$uuid)
                    newTreeBranch$setCollection(self)
                })

                "Created"
            }
            else
            {
                stop(paste0("Expected character vector, got ",
                            paste0("(", paste0(class(files), collapse = ", "), ")"),
                            "."))
            }
        },

        remove = function(paths)
        {
            if(is.null(private$tree))
                private$generateCollectionTreeStructure()

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

        move = function(content, destination)
        {
            if(is.null(private$tree))
                private$generateCollectionTreeStructure()

            content <- trimFromEnd(content, "/")

            elementToMove <- self$get(content)

            if(is.null(elementToMove))
                stop("Content you want to move doesn't exist in the collection.")

            elementToMove$move(destination)
        },

        copy = function(content, destination)
        {
            if(is.null(private$tree))
                private$generateCollectionTreeStructure()

            content <- trimFromEnd(content, "/")

            elementToCopy <- self$get(content)

            if(is.null(elementToCopy))
                stop("Content you want to copy doesn't exist in the collection.")

            elementToCopy$copy(destination)
        },

        refresh = function()
        {
            if(!is.null(private$tree))
            {
                private$tree$getTree()$setCollection(NULL, setRecursively = TRUE)
                private$tree <- NULL
            }
        },

        getFileListing = function()
        {
            if(is.null(private$tree))
                private$generateCollectionTreeStructure()

            content <- private$REST$getCollectionContent(self$uuid)
            content[order(tolower(content))]
        },

        get = function(relativePath)
        {
            if(is.null(private$tree))
                private$generateCollectionTreeStructure()

            private$tree$getElement(relativePath)
        },

        getRESTService = function() private$REST,
        setRESTService = function(newRESTService) private$REST <- newRESTService
    ),

    private = list(

        REST        = NULL,
        tree        = NULL,
        fileContent = NULL,

        generateCollectionTreeStructure = function()
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
