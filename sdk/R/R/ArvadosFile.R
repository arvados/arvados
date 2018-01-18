source("./R/util.R")

#' ArvadosFile Object
#'
#' Update description
#'
#' @export ArvadosFile
ArvadosFile <- R6::R6Class(

    "ArvadosFile",

    public = list(

        initialize = function(name)
        {
            private$name             <- name
            private$http             <- HttpRequest$new()
            private$httpParser       <- HttpParser$new()
        },

        getName = function() private$name,

        getFileListing = function(fullpath = TRUE)
        {
            self$getName()
        },

        getSizeInBytes = function()
        {
            if(is.null(private$collection))
                return(0)

            REST <- private$collection$getRESTService()

            fileSize <- REST$getResourceSize(self$getRelativePath(),
                                             private$collection$uuid)

            fileSize
        },

        get = function(fileLikeObjectName)
        {
            return(NULL)
        },

        getFirst = function()
        {
            return(NULL)
        },

        getCollection = function() private$collection,

        setCollection = function(collection)
        {
            private$collection <- collection
        },

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

        getParent = function() private$parent,

        setParent = function(newParent) private$parent <- newParent,

        read = function(contentType = "raw", offset = 0, length = 0)
        {
            if(is.null(private$collection))
                stop("ArvadosFile doesn't belong to any collection.")

            if(offset < 0 || length < 0)
                stop("Offset and length must be positive values.")

            REST <- private$collection$getRESTService()

            REST$read(private$collection$uuid,
                      self$getRelativePath(),
                      contentType, offset, length)
        },

        connection = function(rw)
        {
            if (rw == "r") 
            {
                return(textConnection(self$read("text")))
            }
            else if (rw == "w") 
            {
                private$buffer <- textConnection(NULL, "w")

                return(private$buffer)
            }
        },

        flush = function() 
        {
            v <- textConnectionValue(private$buffer)
            close(private$buffer)
            self$write(paste(v, collapse='\n'))
        },

        write = function(content, contentType = "text/html")
        {
            if(is.null(private$collection))
                stop("ArvadosFile doesn't belong to any collection.")

            REST <- private$collection$getRESTService()

            result <- REST$write(private$collection$uuid,
                                 self$getRelativePath(),
                                 content, contentType)
        },

        move = function(newLocationInCollection)
        {
            if(is.null(private$collection))
                stop("ArvadosFile doesn't belong to any collection")

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

            "Content moved successfully."
        }
    ),

    private = list(

        name       = NULL,
        size       = NULL,
        parent     = NULL,
        collection = NULL,
        http       = NULL,
        httpParser = NULL,
        buffer     = NULL,

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
