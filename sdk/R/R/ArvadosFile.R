#' ArvadosFile Object
#'
#' Update description
#'
#' @export ArvadosFile
ArvadosFile <- R6::R6Class(

    "ArvadosFile",

    public = list(

        initialize = function(name, relativePath, api, collection)
        {
            private$name         <- name
            private$relativePath <- relativePath
            private$api          <- api
            private$collection   <- collection
            private$http         <- HttpRequest$new()
            private$httpParser   <- HttpParser$new()
        },

        getName = function() private$name,

        getRelativePath = function() private$relativePath,

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

            #TODO(Fudo): Move this to HttpRequest.R
            # serverResponse <- httr::GET(url = fileURL,
                                        # config = httr::add_headers(unlist(headers)))
            serverResponse <- private$http$GET(fileURL, headers)
            parsed_response <- httr::content(serverResponse, "raw")

        }
    ),

    private = list(

        name         = NULL,
        relativePath = NULL,
        parent       = NULL,
        api          = NULL,
        collection   = NULL,
        http         = NULL,
        httpParser   = NULL
    ),
    
    cloneable = FALSE
)
