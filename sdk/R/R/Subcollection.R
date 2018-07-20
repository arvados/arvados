# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

source("./R/util.R")

#' Subcollection
#'
#' Subcollection class represents a folder inside Arvados collection.
#' It is essentially a composite of arvadosFiles and other subcollections.
#'
#' @section Usage:
#' \preformatted{subcollection = Subcollection$new(name)}
#'
#' @section Arguments:
#' \describe{
#'   \item{name}{Name of the subcollection.}
#' }
#'
#' @section Methods:
#' \describe{
#'   \item{getName()}{Returns name of the subcollection.}
#'   \item{getRelativePath()}{Returns subcollection path relative to the root.}
#'   \item{add(content)}{Adds ArvadosFile or Subcollection specified by content to the subcollection.}
#'   \item{remove(name)}{Removes ArvadosFile or Subcollection specified by name from the subcollection.}
#'   \item{get(relativePath)}{If relativePath is valid, returns ArvadosFile or Subcollection specified by relativePath, else returns NULL.}
#'   \item{getFileListing()}{Returns subcollections file content as character vector.}
#'   \item{getSizeInBytes()}{Returns subcollections content size in bytes.}
#'   \item{move(newLocation)}{Moves subcollection to a new location inside collection.}
#' }
#'
#' @name Subcollection
#' @examples
#' \dontrun{
#' myFolder <- Subcollection$new("myFolder")
#' myFile   <- ArvadosFile$new("myFile")
#'
#' myFolder$add(myFile)
#' myFolder$get("myFile")
#' myFolder$remove("myFile")
#'
#' myFolder$move("newLocation/myFolder")
#' }
NULL

#' @export
Subcollection <- R6::R6Class(

    "Subcollection",

    public = list(

        initialize = function(name)
        {
            private$name <- name
        },

        getName = function() private$name,

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

        add = function(content)
        {
            if("ArvadosFile"   %in% class(content) ||
               "Subcollection" %in% class(content))
            {
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

        getFileListing = function(fullPath = TRUE)
        {
            content <- private$getContentAsCharVector(fullPath)
            content[order(tolower(content))]
        },

        getSizeInBytes = function()
        {
            if(is.null(private$collection))
                return(0)

            REST <- private$collection$getRESTService()

            fileSizes <- REST$getResourceSize(paste0(self$getRelativePath(), "/"),
                                              private$collection$uuid)
            return(sum(fileSizes))
        },

        move = function(newLocation)
        {
            if(is.null(private$collection))
                stop("Subcollection doesn't belong to any collection")

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
        },

        get = function(name)
        {
            for(child in private$children)
            {
                if(child$getName() == name)
                    return(child)
            }

            return(NULL)
        },

        getFirst = function()
        {
            if(length(private$children) == 0)
               return(NULL)

            private$children[[1]]
        },

        setCollection = function(collection, setRecursively = TRUE)
        {
            private$collection = collection

            if(setRecursively)
            {
                for(child in private$children)
                    child$setCollection(collection)
            }
        },

        getCollection = function() private$collection,

        getParent = function() private$parent,

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
