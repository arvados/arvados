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

            collection
            parsed_response <- httr::content(serverResponse, "raw")
        },
        
        write = function(content, contentType)
        {
            fileURL = paste0(private$api$getWebDavHostName(), "c=", private$collection$uuid, "/", private$relativePath);
            headers <- list(Authorization = paste("OAuth2", private$api$getToken()), 
                            "Content-Type" = contentType)
            body <- content

            serverResponse <- private$http$PUT(fileURL, headers, body)

            if(serverResponse$status_code != 201)
                stop(paste("Server code:", serverResponse$status_code))

            #Note(Fudo): Everything went well we need to update file size 
            # in collection tree.

            #Todo(Fudo): Move this into HttpRequest
            uri <- URLencode(paste0(private$api$getWebDavHostName(), "c=", private$collection$uuid))
            h <- curl::new_handle()
            curl::handle_setopt(h, customrequest = "PROPFIND")

            curl::handle_setheaders(h, "Authorization" = paste("OAuth2", private$api$getToken()))
            propfindResponse <- curl::curl_fetch_memory(fileURL, h)

            fileInfo <- private$httpParser$parseWebDAVResponse(propfindResponse, uri)

            private$size <- fileInfo[[1]]$fileSize
            private$collection$update(self, "File size changed")

            parsed_response <- httr::content(serverResponse, "text")
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
        httpParser   = NULL
    ),
    
    cloneable = FALSE
)
