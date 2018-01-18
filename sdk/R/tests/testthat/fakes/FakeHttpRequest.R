FakeHttpRequest <- R6::R6Class(

    "FakeHttpRequest",

    public = list(

        serverMaxElementsPerRequest = NULL,

        content                                 = NULL,
        expectedURL                             = NULL,
        URLIsProperlyConfigured                 = NULL,
        expectedQueryFilters                    = NULL,
        queryFiltersAreCorrect                  = NULL,
        requestHeaderContainsAuthorizationField = NULL,
        requestHeaderContainsDestinationField   = NULL,
        JSONEncodedBodyIsProvided               = NULL,

        numberOfGETRequests    = NULL,
        numberOfDELETERequests = NULL,
        numberOfPUTRequests    = NULL,
        numberOfPOSTRequests   = NULL,
        numberOfMOVERequests   = NULL,

        initialize = function(expectedURL      = NULL,
                              serverResponse   = NULL,
                              expectedFilters  = NULL)
        {
            if(is.null(serverResponse))
            {
                self$content <- list()
                self$content$status_code <- 200
            }
            else
                self$content <- serverResponse

            self$expectedURL <- expectedURL
            self$URLIsProperlyConfigured <- FALSE
            self$expectedQueryFilters <- expectedFilters
            self$queryFiltersAreCorrect <- FALSE
            self$requestHeaderContainsAuthorizationField <- FALSE
            self$requestHeaderContainsDestinationField <- FALSE
            self$JSONEncodedBodyIsProvided <- FALSE

            self$numberOfGETRequests <- 0
            self$numberOfDELETERequests <- 0
            self$numberOfPUTRequests <- 0
            self$numberOfPOSTRequests <- 0
            self$numberOfMOVERequests <- 0

            self$serverMaxElementsPerRequest <- 5
        },

        GET = function(url, headers = NULL, queryFilters = NULL, limit = NULL, offset = NULL)
        {
            private$validateURL(url)
            private$validateHeaders(headers)
            private$validateFilters(queryFilters)
            self$numberOfGETRequests <- self$numberOfGETRequests + 1

            if(!is.null(self$content$items_available))
            {
                return(private$getElements(offset, limit))
            }
            else
                return(self$content)
        },

        PUT = function(url, headers = NULL, body = NULL,
                       queryFilters = NULL, limit = NULL, offset = NULL)
        {
            private$validateURL(url)
            private$validateHeaders(headers)
            private$validateBody(body)
            self$numberOfPUTRequests <- self$numberOfPUTRequests + 1

            self$content
        },

        POST = function(url, headers = NULL, body = NULL,
                        queryFilters = NULL, limit = NULL, offset = NULL)
        {
            private$validateURL(url)
            private$validateHeaders(headers)
            private$validateBody(body)
            self$numberOfPOSTRequests <- self$numberOfPOSTRequests + 1

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
            private$validateURL(url)
            private$validateHeaders(headers)
            self$content
        },

        MOVE = function(url, headers = NULL)
        {
            private$validateURL(url)
            private$validateHeaders(headers)
            self$numberOfMOVERequests <- self$numberOfMOVERequests + 1
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

            if(!is.null(headers$Destination))
                self$requestHeaderContainsDestinationField <- TRUE
        },

        validateBody = function(body)
        {
            if(!is.null(body) && class(body) == "json")           
                self$JSONEncodedBodyIsProvided <- TRUE
        },

        validateFilters = function(filters)
        {
            if(!is.null(self$expectedQueryFilters) &&
               !is.null(filters) &&
               all.equal(unname(filters), self$expectedQueryFilters))
            {
                self$queryFiltersAreCorrect <- TRUE
            }
        },

        getElements = function(offset, limit)
        {
            start <- 1
            elementCount <- self$serverMaxElementsPerRequest

            if(!is.null(offset))
            {
                if(offset > self$content$items_available)
                    stop("Invalid offset")
                
                start <- offset + 1
            }

            if(!is.null(limit))
                if(limit < self$serverMaxElementsPerRequest)
                    elementCount <- limit - 1


            serverResponse <- list()
            serverResponse$items_available <- self$content$items_available
            serverResponse$items <- self$content$items[start:(start + elementCount - 1)]

            if(start + elementCount > self$content$items_available)
            {
                elementCount = self$content$items_available - start
                serverResponse$items <- self$content$items[start:(start + elementCount)]
            }

            serverResponse
        }
    ),

    cloneable = FALSE
)
