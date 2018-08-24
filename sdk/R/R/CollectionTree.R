# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

source("./R/Subcollection.R")
source("./R/ArvadosFile.R")
source("./R/util.R")

CollectionTree <- R6::R6Class(
    "CollectionTree",
    public = list(

        pathsList = NULL,

        initialize = function(fileContent, collection)
        {
            self$pathsList <- fileContent
            treeBranches <- sapply(fileContent, function(filePath) self$createBranch(filePath))
            root <- Subcollection$new("")
            sapply(treeBranches, function(branch) self$addBranch(root, branch))
            root$setCollection(collection)
            private$tree <- root
        },

        createBranch = function(filePath)
        {
            splitPath <- unlist(strsplit(filePath, "/", fixed = TRUE))
            branch <- NULL
            lastElementIndex <- length(splitPath)

            for(elementIndex in lastElementIndex:1)
            {
                if(elementIndex == lastElementIndex)
                {
                    branch <- ArvadosFile$new(splitPath[[elementIndex]])
                }
                else
                {
                    newFolder <- Subcollection$new(splitPath[[elementIndex]])
                    newFolder$add(branch)
                    branch <- newFolder
                }
            }

            branch
        },

        addBranch = function(container, node)
        {
            child <- container$get(node$getName())

            if(is.null(child))
            {
                # Make sure we are don't make any REST call while adding child
                collection <- container$getCollection()
                container$setCollection(NULL, setRecursively = FALSE)
                container$add(node)
                container$setCollection(collection, setRecursively = FALSE)
            }
            else
            {
                # Note: REST always returns folder name alone before other folder
                # content, so in first iteration we don't know if it's a file
                # or folder since its just a name, so we assume it's a file.
                # If we encounter that same name again we know
                # it's a folder so we need to replace ArvadosFile with Subcollection.
                if("ArvadosFile" %in% class(child))
                    child = private$replaceFileWithSubcollection(child)

                self$addBranch(child, node$getFirst())
            }
        },

        getElement = function(relativePath)
        {
            relativePath <- trimFromStart(relativePath, "./")
            relativePath <- trimFromEnd(relativePath, "/")

            if(endsWith(relativePath, "/"))
                relativePath <- substr(relativePath, 0, nchar(relativePath) - 1)

            splitPath <- unlist(strsplit(relativePath, "/", fixed = TRUE))
            returnElement <- private$tree

            for(pathFragment in splitPath)
            {
                returnElement <- returnElement$get(pathFragment)

                if(is.null(returnElement))
                    return(NULL)
            }

            returnElement
        },

        getTree = function() private$tree
    ),

    private = list(

        tree = NULL,

        replaceFileWithSubcollection = function(arvadosFile)
        {
            subcollection <- Subcollection$new(arvadosFile$getName())
            fileParent <- arvadosFile$getParent()
            fileParent$remove(arvadosFile$getName())
            fileParent$add(subcollection)

            arvadosFile$setParent(NULL)

            subcollection
        }
    )
)
