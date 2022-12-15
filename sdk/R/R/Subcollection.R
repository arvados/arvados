# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

#' R6 Class Representing a Subcollection
#'
#' @description
#' Subcollection class represents a folder inside Arvados collection.
#' It is essentially a composite of arvadosFiles and other subcollections.

#' @export
Subcollection <- R6::R6Class(

    "Subcollection",

    public = list(

        #' @description
        #' Initialize new enviroment.
        #' @param name Name of the new enviroment.
        #' @return A new `Subcollection` object.
        initialize = function(name)
        {
            private$name <- name
        },

        #' @description
        #' Returns name of the file.
        getName = function() private$name,

        #' @description
        #' Returns Subcollection's path relative to the root.
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
        #' Adds ArvadosFile or Subcollection specified by content to the Subcollection.
        #' @param content Content to be added.
        add = function(content)
        {
            if("ArvadosFile"   %in% class(content) ||
               "Subcollection" %in% class(content))
            {
                if(!is.null(content$getCollection()))
                    stop("Content already belongs to a collection.")

                if(content$getName() == "")
                    stop("Content has invalid name.")

                childWithSameName <- self$get(content$getName())

                if(!is.null(childWithSameName))
                    stop(paste("Subcollection already contains ArvadosFile",
                               "or Subcollection with same name."))

                if(!is.null(private$collection))
                {
                    if(self$getRelativePath() != "")
                        contentPath <- paste0(self$getRelativePath(),
                                              "/", content$getFileListing())
                    else
                        contentPath <- content$getFileListing()

                    REST <- private$collection$getRESTService()
                    REST$create(contentPath, private$collection$uuid)
                    content$setCollection(private$collection)
                }

                private$children <- c(private$children, content)
                content$setParent(self)

                "Content added successfully."
            }
            else
            {
                stop(paste0("Expected AravodsFile or Subcollection object, got ",
                            paste0("(", paste0(class(content), collapse = ", "), ")"),
                            "."))
            }
        },

        #' @description
        #' Removes ArvadosFile or Subcollection specified by name from the Subcollection.
        #' @param name Name of the file to be removed.
        remove = function(name)
        {
            if(is.character(name))
            {
                child <- self$get(name)

                if(is.null(child))
                    stop(paste("Subcollection doesn't contains ArvadosFile",
                               "or Subcollection with specified name."))

                if(!is.null(private$collection))
                {
                    REST <- private$collection$getRESTService()
                    REST$delete(child$getRelativePath(), private$collection$uuid)

                    child$setCollection(NULL)
                }

                private$removeChild(name)
                child$setParent(NULL)

                "Content removed"
            }
            else
            {
                stop(paste0("Expected character, got ",
                            paste0("(", paste0(class(name), collapse = ", "), ")"),
                            "."))
            }
        },

        #' @description
        #' Returns Subcollections file content as character vector.
        #' @param fullPath Checking if the path to file exists.
        getFileListing = function(fullPath = TRUE)
        {
            content <- private$getContentAsCharVector(fullPath)
            content[order(tolower(content))]
        },

        #' @description
        #' Returns subcollections content size in bytes.
        getSizeInBytes = function()
        {
            if(is.null(private$collection))
                return(0)

            REST <- private$collection$getRESTService()

            fileSizes <- REST$getResourceSize(paste0(self$getRelativePath(), "/"),
                                              private$collection$uuid)
            return(sum(fileSizes))
        },

        #' @description
        #' Moves Subcollection to a new location inside collection.
        #' @param destination Path to move the file.
        move = function(destination)
        {
            if(is.null(private$collection))
                stop("Subcollection doesn't belong to any collection.")

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
        #' Copies Subcollection to a new location inside collection.
        #' @param destination Path to copy the file.
        copy = function(destination)
        {
            if(is.null(private$collection))
                stop("Subcollection doesn't belong to any collection.")

            destination <- trimFromEnd(destination, "/")
            nameAndPath <- splitToPathAndName(destination)

            newParent <- private$collection$get(nameAndPath$path)

            if(is.null(newParent) || !("Subcollection" %in% class(newParent)))
                stop("Unable to get destination subcollection.")

            childWithSameName <- newParent$get(nameAndPath$name)

            if(!is.null(childWithSameName))
                stop("Destination already contains content with same name.")

            REST <- private$collection$getRESTService()
            REST$copy(self$getRelativePath(),
                      paste0(newParent$getRelativePath(), "/", nameAndPath$name),
                      private$collection$uuid)

            newContent <- self$duplicate(nameAndPath$name)
            newContent$setCollection(self$getCollection(), setRecursively = TRUE)
            newContent$setParent(newParent)
            private$attachToNewParent(newContent, newParent)

            newContent
        },

        #' @description
        #' Duplicate Subcollection and gives it a new name.
        #' @param newName New name for duplicated file.
        duplicate = function(newName = NULL)
        {
            name <- if(!is.null(newName)) newName else private$name
            root <- Subcollection$new(name)
            for(child in private$children)
                root$add(child$duplicate())

            root
        },

        #' @description
        #' If name is valid, returns ArvadosFile or Subcollection specified by relativePath, else returns NULL.
        #' @param name Name of the file.
        get = function(name)
        {
            for(child in private$children)
            {
                if(child$getName() == name)
                    return(child)
            }

            return(NULL)
        },

        #' @description
        #' Returns files in Subcollection.
        getFirst = function()
        {
            if(length(private$children) == 0)
                return(NULL)

            private$children[[1]]
        },

        #' @description
        #' Sets Collection by its UUID.
        setCollection = function(collection, setRecursively = TRUE)
        {
            private$collection = collection

            if(setRecursively)
            {
                for(child in private$children)
                    child$setCollection(collection)
            }
        },

        #' @description
        #' Returns Collection of Subcollection.
        getCollection = function() private$collection,

        #' @description
        #' Returns Collection UUID.
        getParent = function() private$parent,

        #' @description
        #' Sets new Collection.
        setParent = function(newParent) private$parent <- newParent
    ),

    private = list(

        name       = NULL,
        children   = NULL,
        parent     = NULL,
        collection = NULL,

        removeChild = function(name)
        {
            numberOfChildren = length(private$children)
            if(numberOfChildren > 0)
            {
                for(childIndex in 1:numberOfChildren)
                {
                    if(private$children[[childIndex]]$getName() == name)
                    {
                        private$children = private$children[-childIndex]
                        return()
                    }
                }
            }
        },

        attachToNewParent = function(content, newParent)
        {
            # We temporary set parents collection to NULL. This will ensure that
            # add method doesn't post this subcollection to REST.
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
            # remove method doesn't remove this subcollection from REST.
            parent <- private$parent
            parentsCollection <- parent$getCollection()
            parent$setCollection(NULL, setRecursively = FALSE)
            parent$remove(private$name)
            parent$setCollection(parentsCollection, setRecursively = FALSE)
        },

        getContentAsCharVector = function(fullPath = TRUE)
        {
            content <- NULL

            if(fullPath)
            {
                for(child in private$children)
                    content <- c(content, child$getFileListing())

                if(private$name != "")
                    content <- unlist(paste0(private$name, "/", content))
            }
            else
            {
                for(child in private$children)
                    content <- c(content, child$getName())
            }

            content
        }
    ),

    cloneable = FALSE
)

#' print.Subcollection
#'
#' Custom print function for Subcollection class
#'
#' @param x Instance of Subcollection class
#' @param ... Optional arguments.
#' @export
print.Subcollection = function(x, ...)
{
    collection   <- NULL
    relativePath <- x$getRelativePath()

    if(!is.null(x$getCollection()))
    {
        collection <- x$getCollection()$uuid

        if(!x$getName() == "")
            relativePath <- paste0("/", relativePath)
    }

    cat(paste0("Type:          ", "\"", "Arvados Subcollection", "\""), sep = "\n")
    cat(paste0("Name:          ", "\"", x$getName(),             "\""), sep = "\n")
    cat(paste0("Relative path: ", "\"", relativePath,            "\""), sep = "\n")
    cat(paste0("Collection:    ", "\"", collection,              "\""), sep = "\n")
}
