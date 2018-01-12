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
                childWithSameName <- self$get(content$getName())
                if(!is.null(childWithSameName))
                    stop("Subcollection already contains ArvadosFile
                          or Subcollection with same name.")

                if(!is.null(private$collection))
                {       
                    if(self$getRelativePath() != "")
                        contentPath <- paste0(self$getRelativePath(),
                                              "/", content$getFileListing())
                    else
                        contentPath <- content$getFileListing()

                    private$collection$createFilesOnREST(contentPath)
                    content$setCollection(private$collection)
                }

                private$children <- c(private$children, content)
                content$setParent(self)

                "Content added successfully."
            }
            else
            {
                stop(paste("Expected AravodsFile or Subcollection object, got",
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
                    stop("Subcollection doesn't contains ArvadosFile
                          or Subcollection with same name.")

                if(!is.null(private$collection))
                {
                    private$collection$deleteFromREST(child$getRelativePath())
                    child$setCollection(NULL)
                }

                private$removeChild(name)
                child$setParent(NULL)

                "Content removed"
            }
            else
            {
                stop(paste("Expected character, got",
                           paste0("(", paste0(class(name), collapse = ", "), ")"),
                           "."))
            }
        },

        getFileListing = function(fullpath = TRUE)
        {
            content <- NULL

            if(fullpath)
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

            status <- private$collection$moveOnREST(self$getRelativePath(),
                                                    paste0(newParent$getRelativePath(),
                                                           "/", self$getName()))

            #Note: We temporary set parents collection to NULL. This will ensure that
            #      add method doesn't post file on REST server.
            parentsCollection <- newParent$getCollection()
            newParent$setCollection(NULL, setRecursively = FALSE)

            newParent$add(self)

            newParent$setCollection(parentsCollection, setRecursively = FALSE)

            private$parent <- newParent

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
        http       = NULL,
        httpParser = NULL,

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
        }
    ),
    
    cloneable = FALSE
)
