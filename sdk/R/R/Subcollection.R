#' Subcollection Class
#' 
#' @details 
#' Todo: Update description
#' Subcollection
#' 
#' @export Subcollection
#' @exportClass Subcollection
Subcollection <- setRefClass(
    "Subcollection",
    fields = list(
        name         = "character",
        relativePath = "character",
        content      = "list"
    ),
    methods = list(
        initialize = function(subcollectionName)
        {
            name <<- subcollectionName
            content <<- list()
        },
        add = function(subcollectionContent)
        {
            content <<- c(content, subcollectionContent)
        }
    )
)
