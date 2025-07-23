# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

FakeHttpParser <- R6::R6Class(

    "FakeHttrParser",

    public = list(

        validContentTypes = NULL,
        parserCallCount = NULL,

        initialize = function()
        {
            self$parserCallCount <- 0
            self$validContentTypes <- c("text", "raw")
        },

        parseJSONResponse = function(serverResponse)
        {
            self$parserCallCount <- self$parserCallCount + 1

            if(!is.null(serverResponse$content))
                return(serverResponse$content)

            serverResponse
        },

        parseResponse = function(serverResponse, outputType)
        {
            self$parserCallCount <- self$parserCallCount + 1

            if(!is.null(serverResponse$content))
                return(serverResponse$content)

            serverResponse
        },

        getFileNamesFromResponse = function(serverResponse, uri)
        {
            self$parserCallCount <- self$parserCallCount + 1

            if(!is.null(serverResponse$content))
                return(serverResponse$content)

            serverResponse
        },

        getFileSizesFromResponse = function(serverResponse, uri)
        {
            self$parserCallCount <- self$parserCallCount + 1

            if(!is.null(serverResponse$content))
                return(serverResponse$content)

            serverResponse
        }
    )
)
