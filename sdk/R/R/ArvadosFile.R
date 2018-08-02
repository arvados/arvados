# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

source("./R/util.R")

#' ArvadosFile
#'
#' ArvadosFile class represents a file inside Arvados collection.
#'
#' @section Usage:
#' \preformatted{file = ArvadosFile$new(name)}
#'
#' @section Arguments:
#' \describe{
#'   \item{name}{Name of the file.}
#' }
#'
#' @section Methods:
#' \describe{
#'   \item{getName()}{Returns name of the file.}
#'   \item{getRelativePath()}{Returns file path relative to the root.}
#'   \item{read(contentType = "raw", offset = 0, length = 0)}{Read file content.}
#'   \item{write(content, contentType = "text/html")}{Write to file (override current content of the file).}
#'   \item{connection(rw)}{Get connection opened in "read" or "write" mode.}
#'   \item{flush()}{Write connections content to a file (override current content of the file).}
#'   \item{remove(name)}{Removes ArvadosFile or Subcollection specified by name from the subcollection.}
#'   \item{getSizeInBytes()}{Returns file size in bytes.}
#'   \item{move(destination)}{Moves file to a new location inside collection.}
#'   \item{copy(destination)}{Copies file to a new location inside collection.}
#' }
#'
#' @name ArvadosFile
#' @examples
#' \dontrun{
#' myFile <- ArvadosFile$new("myFile")
#'
#' myFile$write("This is new file content")
#' fileContent <- myFile$read()
#' fileContent <- myFile$read("text")
#' fileContent <- myFile$read("raw", offset = 8, length = 4)
#'
#' #Write a table:
#' arvConnection <- myFile$connection("w")
#' write.table(mytable, arvConnection)
#' arvadosFile$flush()
#'
#' #Read a table:
#' arvConnection <- myFile$connection("r")
#' mytable <- read.table(arvConnection)
#'
#' myFile$move("newFolder/myFile")
#' myFile$copy("newFolder/myFile")
#' }
NULL

#' @export
ArvadosFile <- R6::R6Class(

    "ArvadosFile",

    public = list(

        initialize = function(name)
        {
            if(name == "")
                stop("Invalid name.")

            private$name <- name
        },

        getName = function() private$name,

        getFileListing = function(fullpath = TRUE)
        {
            self$getName()
        },

        getSizeInBytes = function()
        {
            if(is.null(private$collection))
                return(0)

            REST <- private$collection$getRESTService()

            fileSize <- REST$getResourceSize(self$getRelativePath(),
                                             private$collection$uuid)
            fileSize
        },

        get = function(fileLikeObjectName)
        {
            return(NULL)
        },

        getFirst = function()
        {
            return(NULL)
        },

        getCollection = function() private$collection,

        setCollection = function(collection, setRecursively = TRUE)
        {
            private$collection <- collection
        },

        getRelativePath = function()
        {
            relativePath <- c(private$name)
            parent <- private$parent

            while(!is.null(parent))
            {
                relativePath <- c(parent$getName(), relativePath)
                parent <- parent$getParent()
            }

            relativePath <- relativePath[relativePath != ""]
            paste0(relativePath, collapse = "/")
        },

        getParent = function() private$parent,

        setParent = function(newParent) private$parent <- newParent,

        read = function(contentType = "raw", offset = 0, length = 0)
        {
            if(is.null(private$collection))
                stop("ArvadosFile doesn't belong to any collection.")

            if(offset < 0 || length < 0)
                stop("Offset and length must be positive values.")

            REST <- private$collection$getRESTService()

            fileContent <- REST$read(self$getRelativePath(),
                                     private$collection$uuid,
                                     contentType, offset, length)
            fileContent
        },

        connection = function(rw)
        {
            if (rw == "r" || rw == "rb")
            {
                REST <- private$collection$getRESTService()
                return(REST$getConnection(self$getRelativePath(),
                                          private$collection$uuid,
                                          rw))
            }
            else if (rw == "w")
            {
                private$buffer <- textConnection(NULL, "w")

                return(private$buffer)
            }
        },

        flush = function()
        {
            v <- textConnectionValue(private$buffer)
            close(private$buffer)
            self$write(paste(v, collapse='\n'))
        },

        write = function(content, contentType = "text/html")
        {
            if(is.null(private$collection))
                stop("ArvadosFile doesn't belong to any collection.")

            REST <- private$collection$getRESTService()

            writeResult <- REST$write(self$getRelativePath(),
                                      private$collection$uuid,
                                      content, contentType)
            writeResult
        },

        move = function(destination)
        {
            if(is.null(private$collection))
                stop("ArvadosFile doesn't belong to any collection.")

            destination <- trimFromEnd(destination, "/")
            nameAndPath <- splitToPathAndName(destination)

            newParent <- private$collection$get(nameAndPath$path)

            if(is.null(newParent))
                stop("Unable to get destination subcollection.")

            childWithSameName <- newParent$get(nameAndPath$name)

            if(!is.null(childWithSameName))
                stop("Destination already contains content with same name.")

            REST <- private$collection$getRESTService()
            REST$move(self$getRelativePath(),
                      paste0(newParent$getRelativePath(), "/", nameAndPath$name),
                      private$collection$uuid)

            private$dettachFromCurrentParent()
            private$attachToNewParent(self, newParent)

            private$parent <- newParent
            private$name <- nameAndPath$name

            self
        },

        copy = function(destination)
        {
            if(is.null(private$collection))
                stop("ArvadosFile doesn't belong to any collection.")

            destination <- trimFromEnd(destination, "/")
            nameAndPath <- splitToPathAndName(destination)

            newParent <- private$collection$get(nameAndPath$path)

            if(is.null(newParent))
                stop("Unable to get destination subcollection.")

            childWithSameName <- newParent$get(nameAndPath$name)

            if(!is.null(childWithSameName))
                stop("Destination already contains content with same name.")

            REST <- private$collection$getRESTService()
            REST$copy(self$getRelativePath(),
                      paste0(newParent$getRelativePath(), "/", nameAndPath$name),
                      private$collection$uuid)

            newFile <- self$duplicate(nameAndPath$name)
            newFile$setCollection(self$getCollection())
            private$attachToNewParent(newFile, newParent)
            newFile$setParent(newParent)

            newFile
        },

        duplicate = function(newName = NULL)
        {
            name <- if(!is.null(newName)) newName else private$name
            newFile <- ArvadosFile$new(name)
            newFile
        }
    ),

    private = list(

        name       = NULL,
        size       = NULL,
        parent     = NULL,
        collection = NULL,
        buffer     = NULL,

        attachToNewParent = function(content, newParent)
        {
            # We temporary set parents collection to NULL. This will ensure that
            # add method doesn't post this file on REST.
            # We also need to set content's collection to NULL because
            # add method throws exception if we try to add content that already
            # belongs to a collection.
            parentsCollection <- newParent$getCollection()
            content$setCollection(NULL, setRecursively = FALSE)
            newParent$setCollection(NULL, setRecursively = FALSE)
            newParent$add(content)
            content$setCollection(parentsCollection, setRecursively = FALSE)
            newParent$setCollection(parentsCollection, setRecursively = FALSE)
        },

        dettachFromCurrentParent = function()
        {
            # We temporary set parents collection to NULL. This will ensure that
            # remove method doesn't remove this file from REST.
            parent <- private$parent
            parentsCollection <- parent$getCollection()
            parent$setCollection(NULL, setRecursively = FALSE)
            parent$remove(private$name)
            parent$setCollection(parentsCollection, setRecursively = FALSE)
        }
    ),

    cloneable = FALSE
)

#' print.ArvadosFile
#'
#' Custom print function for ArvadosFile class
#'
#' @param x Instance of ArvadosFile class
#' @param ... Optional arguments.
#' @export
print.ArvadosFile = function(x, ...)
{
    collection   <- NULL
    relativePath <- x$getRelativePath()

    if(!is.null(x$getCollection()))
    {
        collection <- x$getCollection()$uuid
        relativePath <- paste0("/", relativePath)
    }

    cat(paste0("Type:          ", "\"", "ArvadosFile", "\""), sep = "\n")
    cat(paste0("Name:          ", "\"", x$getName(),   "\""), sep = "\n")
    cat(paste0("Relative path: ", "\"", relativePath,  "\""), sep = "\n")
    cat(paste0("Collection:    ", "\"", collection,    "\""), sep = "\n")
}
