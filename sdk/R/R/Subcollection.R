source("./R/util.R")

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

        move = function(newLocationInCollection)
        {
            if(is.null(private$collection))
                stop("Subcollection doesn't belong to any collection")

            newLocationInCollection <- trimFromEnd(newLocationInCollection, "/")
            newParentLocation <- trimFromEnd(newLocationInCollection, private$name)

            newParent <- private$collection$get(newParentLocation)

            if(is.null(newParent))
            {
                stop("Unable to get destination subcollection")
            }

            childWithSameName <- newParent$get(private$name)

            if(!is.null(childWithSameName))
                stop("Destination already contains content with same name.")

            REST <- private$collection$getRESTService()
            REST$move(self$getRelativePath(),
                      paste0(newParent$getRelativePath(), "/", self$getName()),
                      private$collection$uuid)

            private$dettachFromCurrentParent()
            private$attachToNewParent(newParent)

            "Content moved successfully"
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
        }
    ),
    
    cloneable = FALSE
)
