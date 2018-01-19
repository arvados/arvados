FakeRESTService <- R6::R6Class(

    "FakeRESTService",

    public = list(

        createCallCount               = NULL,
        deleteCallCount               = NULL,
        moveCallCount                 = NULL,
        getCollectionContentCallCount = NULL,
        getResourceSizeCallCount      = NULL,
        readCallCount                 = NULL,
        writeCallCount                = NULL,
        writeBuffer                   = NULL,

        collectionContent = NULL,
        returnContent = NULL,

        initialize = function(collectionContent = NULL, returnContent = NULL)
        {
            self$createCallCount               <- 0
            self$deleteCallCount               <- 0
            self$moveCallCount                 <- 0
            self$getCollectionContentCallCount <- 0
            self$getResourceSizeCallCount      <- 0
            self$readCallCount                 <- 0
            self$writeCallCount                <- 0

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
            self$getCollectionContentCallCount <- self$getCollectionContentCallCount + 1
            self$collectionContent
        },

        getResourceSize = function(uuid, relativePathToResource)
        {
            self$getResourceSizeCallCount <- self$getResourceSizeCallCount + 1
            self$returnContent
        },
        
        read = function(relativePath, uuid, contentType = "text", offset = 0, length = 0)
        {
            self$readCallCount <- self$readCallCount + 1
            self$returnContent
        },

        write = function(uuid, relativePath, content, contentType)
        {
            self$writeBuffer <- content
            self$writeCallCount <- self$writeCallCount + 1
            self$returnContent
        }
    ),

    cloneable = FALSE
)
