# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

#' R6 Class Representing a ArvadosFile
#'
#' @description
#' ArvadosFile class represents a file inside Arvados collection.

#' @export
ArvadosFile <- R6::R6Class(

    "ArvadosFile",

    public = list(

        #' @description
        #' Initialize new enviroment.
        #' @param name Name of the new enviroment.
        #' @return A new `ArvadosFile` object.
        #' @examples
        #' myFile   <- ArvadosFile$new("myFile")
        initialize = function(name)
        {
            if(name == "")
                stop("Invalid name.")

            private$name <- name
        },

        #' @description
        #' Returns name of the file.
        #' @examples
        #' arvadosFile$getName()
        getName = function() private$name,

        #' @description
        #' Returns collections file content as character vector.
        #' @param fullPath Checking if TRUE.
        #' @examples
        #' arvadosFile$getFileListing()
        getFileListing = function(fullpath = TRUE)
        {
            self$getName()
        },

        #' @description
        #' Returns collections content size in bytes.
        #' @examples
        #' arvadosFile$getSizeInBytes()
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

        #' @description
        #' Returns collection UUID.
        getCollection = function() private$collection,

        #' @description
        #' Sets new collection.
        setCollection = function(collection, setRecursively = TRUE)
        {
            private$collection <- collection
        },

        #' @description
        #' Returns file path relative to the root.
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

        #' @description
        #' Returns project UUID.
        getParent = function() private$parent,

        #' @description
        #' Sets project collection.
        setParent = function(newParent) private$parent <- newParent,

        #' @description
        #' Read file content.
        #' @param contentType Type of content. Possible is "text", "raw".
        #' @param offset Describes the location of a piece of data compared to another location
        #' @param length Length of content
        #' @examples
        #' collection <- Collection$new(arv, collectionUUID)
        #' arvadosFile <- collection$get(fileName)
        #' fileContent <- arvadosFile$read("text")
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

        #' @description
        #' Get connection opened in "read" or "write" mode.
        #' @param rw Type of connection.
        #' @examples
        #' collection <- Collection$new(arv, collectionUUID)
        #' arvadosFile <- collection$get(fileName)
        #' arvConnection <- arvadosFile$connection("w")
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

        #' @description
        #' Write connections content to a file or override current content of the file.
        #' @examples
        #' collection <- Collection$new(arv, collectionUUID)
        #' arvadosFile <- collection$get(fileName)
        #' myFile$write("This is new file content")
        #' arvadosFile$flush()
        flush = function()
        {
            v <- textConnectionValue(private$buffer)
            close(private$buffer)
            self$write(paste(v, collapse='\n'))
        },

        #' @description
        #' Write to file or override current content of the file.
        #' @param content File to write.
        #' @param contentType Type of content. Possible is "text", "raw".
        #' @examples
        #' collection <- Collection$new(arv, collectionUUID)
        #' arvadosFile <- collection$get(fileName)
        #' myFile$write("This is new file content")
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

        #' @description
        #' Moves file to a new location inside collection.
        #' @param destination Path to new folder.
        #' @examples
        #' arvadosFile$move(newPath)
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

        #' @description
        #' Copies file to a new location inside collection.
        #' @param destination Path to new folder.
        #' @examples
        #' arvadosFile$copy("NewName.format")
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

        #' @description
        #' Duplicate file and gives it a new name.
        #' @param newName New name for duplicated file.
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
            #parent$.__enclos_env__$private$children <- c(parent$.__enclos_env__$private$children, self)
            #private$parent <- parent
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

            #private$parent$.__enclos_env__$private$removeChild(private$name)
            #private$parent <- NULL
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
