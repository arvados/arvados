source("./R/Subcollection.R")

source("./R/ArvadosFile.R")

#' Arvados Collection Object
#'
#' Update description
#'
#' @examples arv = Collection$new(api, uuid)
#' @export CollectionTree
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
            if(startsWith(relativePath, "./"))
                relativePath <- substr(relativePath, 3, nchar(relativePath))

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
