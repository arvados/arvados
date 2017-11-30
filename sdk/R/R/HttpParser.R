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
        }
    )
)
