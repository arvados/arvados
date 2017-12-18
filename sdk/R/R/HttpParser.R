#' HttpParser
#'
HttpParser <- R6::R6Class(

    "HttrParser",

    public = list(
        initialize = function() 
        {
        },

        parseJSONResponse = function(serverResponse) 
        {
            parsed_response <- httr::content(serverResponse, as = "parsed", type = "application/json")
        },

        #TODO rename this
        parseWebDAVResponse = function(response, uri)
        {
            text <- rawToChar(response$content)
            doc <- XML::xmlParse(text, asText=TRUE)
            base <- paste(paste("/", strsplit(uri, "/")[[1]][-1:-3], sep="", collapse=""), "/", sep="")
            result <- unlist(
                XML::xpathApply(doc, "//D:response/D:href", function(node) {
                    sub(base, "", URLdecode(XML::xmlValue(node)), fixed=TRUE)
                })
            )
            result <- result[result != ""]
            result
        },

        extractFileSizeFromWebDAVResponse = function(response, uri)    
        {
            text <- rawToChar(response$content)
            doc <- XML::xmlParse(text, asText=TRUE)

            base <- paste(paste("/", strsplit(uri, "/")[[1]][-1:-3], sep="", collapse=""), "/", sep="")
            result <- XML::xpathApply(doc, "//D:response/D:propstat/D:prop/D:getcontentlength", function(node) {
              XML::xmlValue(node)
            })

            unlist(result)
        }
    )
)
