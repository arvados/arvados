#' ArvadosFile Object
#'
#' Update description
#'
#' @export ArvadosFile
ArvadosFile <- R6::R6Class(

    "ArvadosFile",

    public = list(

        initialize = function(name, relativePath, size, api, collection)
        {
            private$name         <- name
            private$size         <- size
            private$relativePath <- relativePath
            private$api          <- api
            private$collection   <- collection
            private$http         <- HttpRequest$new()
            private$httpParser   <- HttpParser$new()
        },

        getName = function() private$name,

        getRelativePath = function() private$relativePath,

        getSizeInBytes = function() private$size,

        read = function(offset = 0, length = 0)
        {
            if(offset < 0 || length < 0)
            stop("Offset and length must be positive values.")

            range = paste0("bytes=", offset, "-")

            if(length > 0)
                range = paste0(range, offset + length - 1)
            
            fileURL = paste0(private$api$getWebDavHostName(), "c=", private$collection$uuid, "/", private$relativePath);
            headers <- list(Authorization = paste("OAuth2", private$api$getToken()), 
                            Range = range)

            serverResponse <- private$http$GET(fileURL, headers)

            if(serverResponse$status_code != 206)
                stop(paste("Server code:", serverResponse$status_code))

            parsedServerResponse <- httr::content(serverResponse, "raw")
            parsedServerResponse
        },
        
        write = function(content, contentType = "text/html")
        {
            fileURL = paste0(private$api$getWebDavHostName(), "c=", private$collection$uuid, "/", private$relativePath);
            headers <- list(Authorization = paste("OAuth2", private$api$getToken()), 
                            "Content-Type" = contentType)
            body <- content

            serverResponse <- private$http$PUT(fileURL, headers, body)

            if(serverResponse$status_code != 201)
                stop(paste("Server code:", serverResponse$status_code))

            private$notifyCollectionThatFileSizeChanges()

            parsedServerResponse <- httr::content(serverResponse, "text")
            parsedServerResponse
        }
    ),

    private = list(

        name         = NULL,
        relativePath = NULL,
        size         = NULL,
        parent       = NULL,
        api          = NULL,
        collection   = NULL,
        http         = NULL,
        httpParser   = NULL,

        notifyCollectionThatFileSizeChanges = function()
        {
            collectionURL <- URLencode(paste0(private$api$getWebDavHostName(), "c=", private$collection$uuid))
            fileURL = paste0(collectionURL, "/", private$relativePath);
            headers = list("Authorization" = paste("OAuth2", private$api$getToken()))

            propfindResponse <- private$http$PROPFIND(fileURL, headers)

            fileInfo <- private$httpParser$parseWebDAVResponse(propfindResponse, collectionURL)

            private$size <- fileInfo[[1]]$fileSize
            private$collection$update(self, "File size changed")
        }
    ),
    
    cloneable = FALSE
)
