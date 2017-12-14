#' Arvados SubCollection Object
#'
#' Update description
#'
#' @export Subcollection
Subcollection <- R6::R6Class(

    "Subcollection",

    public = list(

        initialize = function(name, relativePath, children)
        {
            private$name <- name
            private$relativePath <- relativePath
            private$children <- children
        },

        getName = function() private$name,

        getRelativePath = function() private$relativePath,

        getSizeInBytes = function()
        {
            overallSize = 0
            for(child in private$children)
                overallSize = overallSize + child$getSizeInBytes()

            overallSize
        },

        setParent = function(parent) private$parent <- parent
    ),

    private = list(

        name         = NULL,
        relativePath = NULL,
        children     = NULL,
        parent       = NULL
    ),
    
    cloneable = FALSE
)
