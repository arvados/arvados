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
        relativePath = "character"
    ),
    methods = list(
        initialize = function(subcollectionName)
        {
            name <<- subcollectionName
        }
    )
)
