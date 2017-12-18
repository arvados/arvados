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
            private$name       <- name
            private$http       <- HttpRequest$new()
            private$httpParser <- HttpParser$new()
        },

        getName = function() private$name,

        getFileList = function(fullpath = TRUE)
        {
            self$getName()
        },

        getSizeInBytes = function()
        {
            collectionURL <- URLencode(paste0(private$collection$api$getWebDavHostName(), "c=", private$collection$uuid))
            fileURL <- paste0(collectionURL, "/", self$getRelativePath());

            headers = list("Authorization" = paste("OAuth2", private$collection$api$getToken()))

            propfindResponse <- private$http$PROPFIND(fileURL, headers)

            sizes <- private$httpParser$extractFileSizeFromWebDAVResponse(propfindResponse, collectionURL)
            as.numeric(sizes)
        },

        removeFromCollection = function()
        {
            if(is.null(private$collection))
                stop("Subcollection doesn't belong to any collection.")
            
            private$collection$.__enclos_env__$private$deleteFromREST(self$getRelativePath())

            #todo rename this add to a collection
            private$addToCollection(NULL)
            private$detachFromParent()
        },

        getRelativePath = function()
        {
            relativePath <- c(private$name)
            parent <- private$parent

            #Recurse back to root
            while(!is.null(parent))
            {
                relativePath <- c(parent$getName(), relativePath)
                parent <- parent$getParent()
            }

            relativePath <- relativePath[relativePath != ""]
            paste0(relativePath, collapse = "/")
        },

        getParent = function() private$parent,

        read = function(offset = 0, length = 0)
        {
            #todo range is wrong fix it
            if(offset < 0 || length < 0)
            stop("Offset and length must be positive values.")

            range = paste0("bytes=", offset, "-")

            if(length > 0)
                range = paste0(range, offset + length - 1)
            
            fileURL = paste0(private$collection$api$getWebDavHostName(), "c=", private$collection$uuid, "/", self$getRelativePath());
            headers <- list(Authorization = paste("OAuth2", private$collection$api$getToken()), 
                            Range = range)

            serverResponse <- private$http$GET(fileURL, headers)

            if(serverResponse$status_code != 206)
                stop(paste("Server code:", serverResponse$status_code))

            parsedServerResponse <- httr::content(serverResponse, "raw")
            parsedServerResponse
        },
        
        write = function(content, contentType = "text/html")
        {
            fileURL = paste0(private$collection$api$getWebDavHostName(), "c=", private$collection$uuid, "/", self$getRelativePath());
            headers <- list(Authorization = paste("OAuth2", private$collection$api$getToken()), 
                            "Content-Type" = contentType)
            body <- content

            serverResponse <- private$http$PUT(fileURL, headers, body)

            if(serverResponse$status_code != 201)
                stop(paste("Server code:", serverResponse$status_code))

            parsedServerResponse <- httr::content(serverResponse, "text")
            parsedServerResponse
        }
    ),

    private = list(

        name         = NULL,
        size         = NULL,
        parent       = NULL,
        collection   = NULL,
        http         = NULL,
        httpParser   = NULL,

        getChild = function(name)
        {
            return(NULL)
        },

        getFirstChild = function()
        {
            return(NULL)
        },

        addToCollection = function(collection)
        {
            private$collection = collection
        },

        detachFromParent = function()
        {
            if(!is.null(private$parent))
            {
                private$parent$.__enclos_env__$private$removeChild(private$name)
                private$parent <- NULL
            }
        }
    ),
    
    cloneable = FALSE
)
