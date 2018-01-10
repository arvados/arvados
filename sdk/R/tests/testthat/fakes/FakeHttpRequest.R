FakeHttpRequest <- R6::R6Class(

    "FakeHttpRequest",

    public = list(

        content                                 = NULL,
        expectedURL                             = NULL,
        URLIsProperlyConfigured                 = NULL,
        requestHeaderContainsAuthorizationField = NULL,

        numberOfGETRequests = NULL,
        numberOfDELETERequests = NULL,

        initialize = function(expectedURL = NULL, serverResponse = NULL)
        {
            self$content <- serverResponse
            self$expectedURL <- expectedURL
            self$requestHeaderContainsAuthorizationField <- FALSE
            self$URLIsProperlyConfigured <- FALSE

            self$numberOfGETRequests <- 0
            self$numberOfDELETERequests <- 0
        },

        GET = function(url, headers = NULL, queryFilters = NULL, limit = NULL, offset = NULL)
        {
            private$validateURL(url)
            private$validateHeaders(headers)
            self$numberOfGETRequests <- self$numberOfGETRequests + 1

            self$content
        },

        PUT = function(url, headers = NULL, body = NULL,
                       queryFilters = NULL, limit = NULL, offset = NULL)
        {
            self$content
        },

        POST = function(url, headers = NULL, body = NULL,
                        queryFilters = NULL, limit = NULL, offset = NULL)
        {
            self$content
        },

        DELETE = function(url, headers = NULL, body = NULL,
                          queryFilters = NULL, limit = NULL, offset = NULL)
        {
            private$validateURL(url)
            private$validateHeaders(headers)
            self$numberOfDELETERequests <- self$numberOfDELETERequests + 1
            self$content
        },

        PROPFIND = function(url, headers = NULL)
        {
            self$content
        },

        MOVE = function(url, headers = NULL)
        {
            self$content
        }
    ),

    private = list(

        validateURL = function(url) 
        {
            if(!is.null(self$expectedURL) && url == self$expectedURL)
                self$URLIsProperlyConfigured <- TRUE
        },

        validateHeaders = function(headers) 
        {
            if(!is.null(headers$Authorization))
                self$requestHeaderContainsAuthorizationField <- TRUE
        }
    ),

    cloneable = FALSE
)
