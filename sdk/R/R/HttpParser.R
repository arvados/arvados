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

        #Todo(Fudo): Test this.
        parseWebDAVResponse = function(response, uri)
        {
            text <- rawToChar(response$content)
            doc <- XML::xmlParse(text, asText=TRUE)

            # calculate relative paths
            base <- paste(paste("/", strsplit(uri, "/")[[1]][-1:-3], sep="", collapse=""), "/", sep="")
            result <- XML::xpathApply(doc, "//D:response", function(node) {
                result = list()
                children = XML::xmlChildren(node)

                result$name = sub(base, "", URLdecode(XML::xmlValue(children$href)), fixed=TRUE)
                sizeXMLNode = XML::xmlChildren(XML::xmlChildren(children$propstat)$prop)$getcontentlength
                result$fileSize = as.numeric(XML::xmlValue(sizeXMLNode))

                result
            })

            result
        }
    )
)
