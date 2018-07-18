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

            treeBranches <- sapply(fileContent, function(filePath)
            {
                splitPath <- unlist(strsplit(filePath, "/", fixed = TRUE))
                branch <- private$createBranch(splitPath)
            })

            root <- Subcollection$new("")

            sapply(treeBranches, function(branch)
            {
                private$addBranch(root, branch)
            })

            root$setCollection(collection)
            private$tree <- root
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

        createBranch = function(splitPath)
        {
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
                container$add(node)
            }
            else
            {
                # Note: REST always returns folder name alone before other folder
                # content, so in first iteration we don't know if it's a file
                # or folder since its just a name, so we assume it's a file.
                # If we encounter that same name again we know
                # it's a folder so we need to replace ArvadosFile with Subcollection.
                if("ArvadosFile" %in% class(child))
                {
                    child = private$replaceFileWithSubcollection(child)
                }

                private$addBranch(child, node$getFirst())
            }
        },

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
