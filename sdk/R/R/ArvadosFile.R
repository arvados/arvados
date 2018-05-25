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
#'   \item{move(newLocation)}{Moves file to a new location inside collection.}
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

        setCollection = function(collection)
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
                return(REST$getConnection(private$collection$uuid,
                                          self$getRelativePath(),
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

        move = function(newLocation)
        {
            if(is.null(private$collection))
                stop("ArvadosFile doesn't belong to any collection")

            newLocation <- trimFromEnd(newLocation, "/")
            nameAndPath <- splitToPathAndName(newLocation)

            newParent <- private$collection$get(nameAndPath$path)

            if(is.null(newParent))
            {
                stop("Unable to get destination subcollection")
            }

            childWithSameName <- newParent$get(nameAndPath$name)

            if(!is.null(childWithSameName))
                stop("Destination already contains content with same name.")

            REST <- private$collection$getRESTService()
            REST$move(self$getRelativePath(),
                      paste0(newParent$getRelativePath(), "/", nameAndPath$name),
                      private$collection$uuid)

            private$dettachFromCurrentParent()
            private$attachToNewParent(newParent)

            private$name <- nameAndPath$name

            "Content moved successfully."
        }
    ),

    private = list(

        name       = NULL,
        size       = NULL,
        parent     = NULL,
        collection = NULL,
        buffer     = NULL,

        attachToNewParent = function(newParent)
        {
            #Note: We temporary set parents collection to NULL. This will ensure that
            #      add method doesn't post file on REST.
            parentsCollection <- newParent$getCollection()
            newParent$setCollection(NULL, setRecursively = FALSE)

            newParent$add(self)

            newParent$setCollection(parentsCollection, setRecursively = FALSE)

            private$parent <- newParent
        },

        dettachFromCurrentParent = function()
        {
            #Note: We temporary set parents collection to NULL. This will ensure that
            #      remove method doesn't remove this subcollection from REST.
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

    cat(paste0("Type:          ", "\"", "ArvadosFile",         "\""), sep = "\n")
    cat(paste0("Name:          ", "\"", x$getName(),           "\""), sep = "\n")
    cat(paste0("Relative path: ", "\"", relativePath,          "\""), sep = "\n")
    cat(paste0("Collection:    ", "\"", collection,            "\""), sep = "\n")
}
