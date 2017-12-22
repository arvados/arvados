#' Arvados SubCollection Object
#'
#' Update description
#'
#' @export Subcollection
Subcollection <- R6::R6Class(

    "Subcollection",

    public = list(

        initialize = function(name)
        {
            private$name       <- name
            private$http       <- HttpRequest$new()
            private$httpParser <- HttpParser$new()
        },
        
        add = function(content)
        {
            if("ArvadosFile"   %in% class(content) ||
               "Subcollection" %in% class(content))
            {
                if(!is.null(content$.__enclos_env__$private$collection))
                    stop("ArvadosFile/Subcollection already belongs to a collection.")

                childWithSameName <- private$getChild(content$getName())
                if(!is.null(childWithSameName))
                    stop("Subcollection already contains ArvadosFile
                          or Subcollection with same name.")

                if(!is.null(private$collection))
                {       
                    contentPath <- paste0(self$getRelativePath(),
                                          "/", content$getFileList())

                    private$collection$.__enclos_env__$private$createFilesOnREST(contentPath)
                    content$.__enclos_env__$private$addToCollection(private$collection)
                }

                private$children <- c(private$children, content)
                content$.__enclos_env__$private$parent = self

                "Content added successfully."
            }
            else
            {
                stop(paste("Expected AravodsFile or Subcollection object, got",
                           class(content), "."))
            }
        },

        removeFromCollection = function()
        {
            if(is.null(private$collection))
                stop("Subcollection doesn't belong to any collection.")

            if(private$name == "")
                stop("Unable to delete root folder.")

            collectionList <- paste0(self$getRelativePath(),
                                     "/", self$getFileList(fullpath = FALSE))
            sapply(collectionList, function(file)
            {
                private$collection$.__enclos_env__$private$deleteFromREST(file)
            })

            private$addToCollection(NULL)
            private$dettachFromParent()

            "Content removed successfully."
        },

        getFileList = function(fullpath = TRUE)
        {
            content <- NULL

            if(fullpath)
            {
                for(child in private$children)
                    content <- c(content, child$getFileList())

                if(private$name != "")
                    content <- unlist(paste0(private$name, "/", content))
            }
            else
            {
                for(child in private$children)
                    content <- c(content, child$getName())
            }

            content
        },

        getSizeInBytes = function()
        {
            collectionURL <- URLencode(paste0(private$collection$api$getWebDavHostName(),
                                              "c=", private$collection$uuid))
            subcollectionURL <- paste0(collectionURL, "/", self$getRelativePath(), "/");

            headers = list("Authorization" = paste("OAuth2", private$collection$api$getToken()))

            propfindResponse <- private$http$PROPFIND(subcollectionURL, headers)

            sizes <- private$httpParser$extractFileSizeFromWebDAVResponse(propfindResponse, collectionURL)
            sizes <- as.numeric(sizes[-1])

            sum(sizes)
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

        move = function(newLocation)
        {
            if(is.null(private$collection))
                stop("Subcollection doesn't belong to any collection.")

            if(endsWith(newLocation, paste0(private$name, "/")))
            {
                newLocation <- substr(newLocation, 0,
                                      nchar(newLocation) - nchar(paste0(private$name, "/")))
            }
            else if(endsWith(newLocation, private$name))
            {
                newLocation <- substr(newLocation, 0,
                                      nchar(newLocation) - nchar(private$name))
            }
            else
            {
                stop("Destination path is not valid.")
            }

            newParent <- private$collection$get(newLocation)

            if(is.null(newParent))
            {
                stop("Unable to get destination subcollection.")
            }

            status <- private$collection$.__enclos_env__$private$moveOnREST(self$getRelativePath(),
                                                                            paste0(newParent$getRelativePath(), "/", self$getName()))

            private$attachToParent(newParent)

            "Content moved successfully."
        },

        getParent = function() private$parent
    ),

    private = list(

        name       = NULL,
        children   = NULL,
        parent     = NULL,
        collection = NULL,
        http       = NULL,
        httpParser = NULL,

        getChild = function(name)
        {
            for(child in private$children)
            {
                if(child$getName() == name)
                    return(child)
            }

            return(NULL)
        },

        getFirstChild = function()
        {
            if(length(private$children) == 0)
               return(NULL)

            private$children[[1]]
        },

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

        addToCollection = function(collection)
        {
            for(child in private$children)
                child$.__enclos_env__$private$addToCollection(collection)

            private$collection = collection
        },

        dettachFromParent = function()
        {
            if(!is.null(private$parent))
            {
                private$parent$.__enclos_env__$private$removeChild(private$name)
                private$parent <- NULL
            }
            else
                stop("Parent doesn't exists.")
        },

        attachToParent = function(parent)
        {
            if(private$name != "")
            {
                parent$.__enclos_env__$private$children <- c(parent$.__enclos_env__$private$children, self)
                private$parent <- parent
            }
        }
    ),
    
    cloneable = FALSE
)
