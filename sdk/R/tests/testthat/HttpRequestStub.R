HttpRequest <- R6::R6Class(

    "HttrRequestStub",

    public = list(

        validContentTypes = NULL,

        content = NULL,

        initialize = function(returnContent) 
        {
            self$validContentTypes <- c("text", "raw")
            self$content <- returnContent
        },

        GET = function(url, headers = NULL, queryFilters = NULL, limit = NULL, offset = NULL)
        {
            return self$content
        },

        PUT = function(url, headers = NULL, body = NULL,
                       queryFilters = NULL, limit = NULL, offset = NULL)
        {
            return self$content
        },

        POST = function(url, headers = NULL, body = NULL,
                        queryFilters = NULL, limit = NULL, offset = NULL)
        {
            return self$content
        },

        DELETE = function(url, headers = NULL, body = NULL,
                          queryFilters = NULL, limit = NULL, offset = NULL)
        {
            return self$content
        },

        PROPFIND = function(url, headers = NULL)
        {
            return self$content
        },

        MOVE = function(url, headers = NULL)
        {
            return self$content
        }
    ),

    cloneable = FALSE
)
