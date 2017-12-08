source("./R/HttpRequest.R")
#' ArvadosFile Class
#' 
#' @details 
#' Todo: Update description
#' Subcollection
#' 
#' @export ArvadosFile
#' @exportClass ArvadosFile
ArvadosFile <- setRefClass(
    "ArvadosFile",
    fields = list(
        name         = "character",
        relativePath = "character",

        read = "function"


    ),
    methods = list(
        initialize = function(subcollectionName, api)
        {
            name <<- subcollectionName

            read <<- function(offset = 0, length = 0)
            {
                if(offset < 0 || length < 0)
                stop("Offset and length must be positive values.")

                range = paste0("bytes=", offset, "-")

                if(length > 0)
                    range = paste0(range, offset + length - 1)
                
                fileURL = paste0(api$getWebDavHostName(), relativePath);
                headers <- list(Authorization = paste("OAuth2", api$getWebDavToken()), 
                                Range = range)

                #TODO(Fudo): Move this to HttpRequest.R
                # serverResponse <- httr::GET(url = fileURL,
                                            # config = httr::add_headers(unlist(headers)))
                http <- HttpRequest()
                serverResponse <- http$GET(fileURL, headers)
                parsed_response <- httr::content(serverResponse, "raw")
            }
        }
    )
)
