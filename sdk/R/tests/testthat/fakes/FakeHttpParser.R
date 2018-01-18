FakeHttpParser <- R6::R6Class(

    "HttrParser",

    public = list(

        parserCallCount = NULL,

        initialize = function() 
        {
            self$parserCallCount <- 0
        },

        parseJSONResponse = function(serverResponse) 
        {
            self$parserCallCount <- self$parserCallCount + 1
            serverResponse
        },

        parseWebDAVResponse = function(response, uri)
        {
            self$parserCallCount <- self$parserCallCount + 1
            response
        },

        extractFileSizeFromWebDAVResponse = function(response, uri)    
        {
            self$parserCallCount <- self$parserCallCount + 1
            response
        }
    )
)
