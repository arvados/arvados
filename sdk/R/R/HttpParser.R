#' HttpParser
#'
HttpParser <- setRefClass(

    "HttrParser",

    fields = list(
    ),

    methods = list(
        initialize = function() 
        {
        },

        parseCollectionGet = function(server_response) 
        {
            parsed_response <- httr::content(server_response, as = "parsed", type = "application/json")

            #Todo(Fudo): Create new Collection object and populate it
        },

        parseWebDAVResponse = function(response, uri)
        {
            #Todo(Fudo): Move this to HttpParser.
            text <- rawToChar(response$content)
            doc <- XML::xmlParse(text, asText=TRUE)

            # calculate relative paths
            base <- paste(paste("/", strsplit(uri, "/")[[1]][-1:-3], sep="", collapse=""), "/", sep="")
            result <- unlist(
                XML::xpathApply(doc, "//D:response/D:href", function(node) {
                    sub(base, "", URLdecode(xmlValue(node)), fixed=TRUE)
                })
            )
            result[result != ""]
        }

    )
)
