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

        getFileList = function(fullpath = TRUE)
        {
            self$getName()
        },

        getSizeInBytes = function()
        {
            collectionURL <- URLencode(paste0(private$collection$api$getWebDavHostName(),
                                              "c=", private$collection$uuid))
            fileURL <- paste0(collectionURL, "/", self$getRelativePath());

            headers = list("Authorization" = paste("OAuth2", private$collection$api$getToken()))

            propfindResponse <- private$http$PROPFIND(fileURL, headers)

            sizes <- private$httpParser$extractFileSizeFromWebDAVResponse(propfindResponse, collectionURL)
            as.numeric(sizes)
        },

        removeFromCollection = function()
        {
            if(is.null(private$collection))
                stop("ArvadosFile doesn't belong to any collection.")
            
            private$collection$.__enclos_env__$private$deleteFromREST(self$getRelativePath())

            private$addToCollection(NULL)
            private$detachFromParent()

            "Content removed successfully."
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

        read = function(contentType = "raw", offset = 0, length = 0)
        {
            if(is.null(private$collection))
                stop("ArvadosFile doesn't belong to any collection.")

            if(offset < 0 || length < 0)
                stop("Offset and length must be positive values.")

            if(!(contentType %in% private$http$validContentTypes))
                stop("Invalid contentType. Please use text or raw.")

            range = paste0("bytes=", offset, "-")

            if(length > 0)
                range = paste0(range, offset + length - 1)
            
            fileURL = paste0(private$collection$api$getWebDavHostName(),
                             "c=", private$collection$uuid, "/", self$getRelativePath());

            if(offset == 0 && length == 0)
            {
                headers <- list(Authorization = paste("OAuth2",
                                                      private$collection$api$getToken())) 
            }
            else
            {
                headers <- list(Authorization = paste("OAuth2", private$collection$api$getToken()), 
                                Range = range)
            }

            serverResponse <- private$http$GET(fileURL, headers)

            if(serverResponse$status_code < 200 || serverResponse$status_code >= 300)
                stop(paste("Server code:", serverResponse$status_code))

            parsedServerResponse <- httr::content(serverResponse, contentType)
            parsedServerResponse
        },
        
        write = function(content, contentType = "text/html")
        {
            if(is.null(private$collection))
                stop("ArvadosFile doesn't belong to any collection.")

            fileURL = paste0(private$collection$api$getWebDavHostName(), 
                             "c=", private$collection$uuid, "/", self$getRelativePath());
            headers <- list(Authorization = paste("OAuth2", private$collection$api$getToken()), 
                            "Content-Type" = contentType)
            body <- content

            serverResponse <- private$http$PUT(fileURL, headers, body)

            if(serverResponse$status_code < 200 || serverResponse$status_code >= 300)
                stop(paste("Server code:", serverResponse$status_code))

            parsedServerResponse <- httr::content(serverResponse, "text")
            parsedServerResponse
        },

        move = function(newLocation)
        {
            if(is.null(private$collection))
                stop("ArvadosFile doesn't belong to any collection.")

            if(endsWith(newLocation, paste0(private$name, "/")))
            {
                newLocation <- substr(newLocation, 0,
                                      nchar(newLocation) - nchar(paste0(private$name, "/")))
            }
            else if(endsWith(newLocation, private$name))
            {
                newLocation <- substr(newLocation, 0, nchar(newLocation) - nchar(private$name))
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
        }
    ),

    private = list(

        name       = NULL,
        size       = NULL,
        parent     = NULL,
        collection = NULL,
        http       = NULL,
        httpParser = NULL,

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
            private$collection <- collection
        },

        detachFromParent = function()
        {
            if(!is.null(private$parent))
            {
                private$parent$.__enclos_env__$private$removeChild(private$name)
                private$parent <- NULL
            }
        },

        attachToParent = function(parent)
        {
            parent$.__enclos_env__$private$children <- c(parent$.__enclos_env__$private$children, self)
            private$parent <- parent
        }
    ),
    
    cloneable = FALSE
)
