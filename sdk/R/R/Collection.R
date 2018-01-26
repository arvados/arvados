source("./R/Subcollection.R")
source("./R/ArvadosFile.R")
source("./R/RESTService.R")
source("./R/util.R")

#' Arvados Collection Object
#'
#' Update description
#'
#' @examples arv = Collection$new(api, uuid)
#' @export Collection
Collection <- R6::R6Class(

    "Collection",

    public = list(

        api  = NULL,
        uuid = NULL,

        initialize = function(api, uuid)
        {
            self$api <- api
            private$REST <- api$getRESTService()

            self$uuid <- uuid

            private$fileContent <- private$REST$getCollectionContent(uuid)
            private$tree <- CollectionTree$new(private$fileContent, self)
        },

        add = function(content, relativePath = "")
        {
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

        create = function(fileNames, relativePath = "")
        {
            if(relativePath == ""  ||
               relativePath == "." ||
               relativePath == "./")
            {
                subcollection <- private$tree$getTree()
            }
            else
            {
                relativePath  <- trimFromEnd(relativePath, "/") 
                subcollection <- self$get(relativePath)
            }

            if(is.null(subcollection))
                stop(paste("Subcollection", relativePath, "doesn't exist."))

            if(is.character(fileNames))
            {
                arvadosFiles <- NULL
                sapply(fileNames, function(fileName)
                {
                    childWithSameName <- subcollection$get(fileName)
                    if(!is.null(childWithSameName))
                        stop("Destination already contains file with same name.")

                    newFile <- ArvadosFile$new(fileName)
                    subcollection$add(newFile)

                    arvadosFiles <<- c(arvadosFiles, newFile)
                })

                if(length(arvadosFiles) == 1)
                    return(arvadosFiles[[1]])
                else
                    return(arvadosFiles)
            }
            else 
            {
                stop(paste0("Expected character vector, got ",
                            paste0("(", paste0(class(fileNames), collapse = ", "), ")"),
                            "."))
            }
        },

        remove = function(paths)
        {
            if(is.character(paths))
            {
                sapply(paths, function(filePath)
                {
                    filePath <- trimFromEnd(filePath, "/")
                    file <- self$get(filePath)

                    if(is.null(file))
                        stop(paste("File", filePath, "doesn't exist."))

                    parent <- file$getParent()
                    parent$remove(file$getName())
                })
            }
            else 
            {
                stop(paste0("Expected character vector, got ",
                            paste0("(", paste0(class(paths), collapse = ", "), ")"),
                            "."))
            }
        },

        move = function(content, newLocation)
        {
            content <- trimFromEnd(content, "/")

            elementToMove <- self$get(content)

            if(is.null(elementToMove))
                stop("Element you want to move doesn't exist in the collection.")

            elementToMove$move(newLocation)
        },

        getFileListing = function()
        {
            content <- private$REST$getCollectionContent(self$uuid)
            content[order(tolower(content))]
        },

        get = function(relativePath)
        {
            private$tree$getElement(relativePath)
        },

        getRESTService = function() private$REST,
        setRESTService = function(newRESTService) private$REST <- newRESTService
    ),

    private = list(

        REST        = NULL,
        tree        = NULL,
        fileContent = NULL,

        generateTree = function(content)
        {
            treeBranches <- sapply(collectionContent, function(filePath)
            {
                splitPath <- unlist(strsplit(filePath$name, "/", fixed = TRUE))

                branch = private$createBranch(splitPath, filePath$fileSize)      
            })
        }
    ),

    cloneable = FALSE
)
