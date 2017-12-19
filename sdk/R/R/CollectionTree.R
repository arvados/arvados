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
                branch = private$createBranch(splitPath)      
            })

            root <- Subcollection$new("")

            sapply(treeBranches, function(branch)
            {
                private$addBranch(root, branch)
            })

            root$.__enclos_env__$private$addToCollection(collection)
            private$tree <- root
        },

        getElement = function(relativePath)
        {
            if(endsWith(relativePath, "/"))
                relativePath <- substr(relativePath, 0, nchar(relativePath) - 1)

            splitPath <- unlist(strsplit(relativePath, "/", fixed = TRUE))
            returnElement = private$tree

            for(pathFragment in splitPath)
            {
                returnElement = returnElement$.__enclos_env__$private$getChild(pathFragment)

                if(is.null(returnElement))
                    return(NULL)
            }

            returnElement
        }
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
                    branch = ArvadosFile$new(splitPath[[elementIndex]])
                }
                else
                {
                    newFolder = Subcollection$new(splitPath[[elementIndex]])
                    newFolder$add(branch)
                    branch = newFolder
                }
            }
            
            branch
        },

        addBranch = function(container, node)
        {
            child = container$.__enclos_env__$private$getChild(node$getName())

            if(is.null(child))
            {
                container$add(node)
                #todo add it to collection
            }
            else
            {
                if("ArvadosFile" %in% class(child))
                {
                    child = private$replaceFileWithSubcollection(child)
                }

                private$addBranch(child, node$.__enclos_env__$private$getFirstChild())
            }
        },

        replaceFileWithSubcollection = function(arvadosFile)
        {
            subcollection <- Subcollection$new(arvadosFile$getName())
            fileParent <- arvadosFile$.__enclos_env__$private$parent
            fileParent$.__enclos_env__$private$removeChild(arvadosFile$getName())
            fileParent$add(subcollection)

            arvadosFile$.__enclos_env__$private$parent <- NULL

            subcollection
        }
    )
)
