FakeRESTService <- R6::R6Class(

    "FakeRESTService",

    public = list(

        createCallCount          = NULL,
        deleteCallCount          = NULL,
        moveCallCount            = NULL,
        getResourceSizeCallCount = NULL,

        collectionContent = NULL,
        returnContent = NULL,

        initialize = function(collectionContent = NULL, returnContent = NULL)
        {
            self$createCallCount <- 0
            self$deleteCallCount <- 0
            self$moveCallCount   <- 0
            self$getResourceSizeCallCount   <- 0

            self$collectionContent <- collectionContent
            self$returnContent <- returnContent
        },

        create = function(files, uuid)
        {
            self$createCallCount <- self$createCallCount + 1

            self$returnContent
        },

        delete = function(relativePath, uuid)
        {
            self$deleteCallCount <- self$deleteCallCount + 1
            self$returnContent
        },

        move = function(from, to, uuid)
        {
            self$moveCallCount <- self$moveCallCount + 1
            self$returnContent
        },

        getCollectionContent = function(uuid)
        {
            self$collectionContent
        },

        getResourceSize = function(uuid, relativePathToResource)
        {
            self$getResourceSizeCallCount <- self$getResourceSizeCallCount + 1
            self$returnContent
        }
    ),

    cloneable = FALSE
)
