# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

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
        requestHeaderContainsRangeField         = NULL,
        requestHeaderContainsContentTypeField   = NULL,
        JSONEncodedBodyIsProvided               = NULL,
        requestBodyIsProvided                   = NULL,

        numberOfGETRequests        = NULL,
        numberOfDELETERequests     = NULL,
        numberOfPUTRequests        = NULL,
        numberOfPOSTRequests       = NULL,
        numberOfMOVERequests       = NULL,
        numberOfCOPYRequests       = NULL,
        numberOfgetConnectionCalls = NULL,

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

            self$expectedURL                             <- expectedURL
            self$URLIsProperlyConfigured                 <- FALSE
            self$expectedQueryFilters                    <- expectedFilters
            self$queryFiltersAreCorrect                  <- FALSE
            self$requestHeaderContainsAuthorizationField <- FALSE
            self$requestHeaderContainsDestinationField   <- FALSE
            self$requestHeaderContainsRangeField         <- FALSE
            self$requestHeaderContainsContentTypeField   <- FALSE
            self$JSONEncodedBodyIsProvided               <- FALSE
            self$requestBodyIsProvided                   <- FALSE

            self$numberOfGETRequests    <- 0
            self$numberOfDELETERequests <- 0
            self$numberOfPUTRequests    <- 0
            self$numberOfPOSTRequests   <- 0
            self$numberOfMOVERequests   <- 0
            self$numberOfCOPYRequests   <- 0

            self$numberOfgetConnectionCalls <- 0

            self$serverMaxElementsPerRequest <- 5
        },

        exec = function(verb, url, headers = NULL, body = NULL, query = NULL,
                        limit = NULL, offset = NULL, retryTimes = 0)
        {
            private$validateURL(url)
            private$validateHeaders(headers)
            private$validateFilters(queryFilters)
            private$validateBody(body)

            if(verb == "GET")
                self$numberOfGETRequests <- self$numberOfGETRequests + 1
            else if(verb == "POST")
                self$numberOfPOSTRequests <- self$numberOfPOSTRequests + 1
            else if(verb == "PUT")
                self$numberOfPUTRequests <- self$numberOfPUTRequests + 1
            else if(verb == "DELETE")
                self$numberOfDELETERequests <- self$numberOfDELETERequests + 1
            else if(verb == "MOVE")
                self$numberOfMOVERequests <- self$numberOfMOVERequests + 1
            else if(verb == "COPY")
                self$numberOfCOPYRequests <- self$numberOfCOPYRequests + 1
            else if(verb == "PROPFIND")
            {
                return(self$content)
            }

            if(!is.null(self$content$items_available))
                return(private$getElements(offset, limit))
            else
                return(self$content)
        },

        getConnection = function(url, headers, openMode)
        {
            self$numberOfgetConnectionCalls <- self$numberOfgetConnectionCalls + 1
            c(url, headers, openMode)
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

            if(!is.null(headers$Range))
                self$requestHeaderContainsRangeField <- TRUE

            if(!is.null(headers[["Content-Type"]]))
                self$requestHeaderContainsContentTypeField <- TRUE
        },

        validateBody = function(body)
        {
            if(!is.null(body))
            {
                self$requestBodyIsProvided <- TRUE

                if(class(body) == "json")
                    self$JSONEncodedBodyIsProvided <- TRUE
            }
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
