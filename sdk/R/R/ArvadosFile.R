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

            fileContent <- REST$read(self$getRelativePath(),
                                     private$collection$uuid,
                                     contentType, offset, length)
            fileContent
        },

        connection = function(rw)
        {
            if (rw == "r" || rw == "rb") 
            {
                REST <- private$collection$getRESTService()
                return(REST$getConnection(private$collection$uuid,
                                          self$getRelativePath(),
                                          rw))
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

            writeResult <- REST$write(self$getRelativePath(),
                                      private$collection$uuid,
                                      content, contentType)
            writeResult
        },

        move = function(newLocation)
        {
            if(is.null(private$collection))
                stop("ArvadosFile doesn't belong to any collection")


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

#' @export print.ArvadosFile
print.ArvadosFile = function(arvadosFile)
{
    collection   <- NULL
    relativePath <- arvadosFile$getRelativePath()

    if(!is.null(arvadosFile$getCollection()))
    {
        collection <- arvadosFile$getCollection()$uuid
        relativePath <- paste0("/", relativePath)
    }

    cat(paste0("Type:          ", "\"", "ArvadosFile", "\""), sep = "\n")
    cat(paste0("Name:          ", "\"", arvadosFile$getName(), "\""), sep = "\n")
    cat(paste0("Relative path: ", "\"", relativePath, "\"") , sep = "\n")
    cat(paste0("Collection:    ", "\"", collection, "\""), sep = "\n")
}
