FakeRESTService <- R6::R6Class(

    "FakeRESTService",

    public = list(

        getResourceCallCount    = NULL,
        createResourceCallCount = NULL,
        listResourcesCallCount  = NULL,
        deleteResourceCallCount = NULL,
        updateResourceCallCount = NULL,
        fetchAllItemsCallCount  = NULL,

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
            self$getResourceCallCount    <- 0
            self$createResourceCallCount <- 0
            self$listResourcesCallCount  <- 0
            self$deleteResourceCallCount <- 0
            self$updateResourceCallCount <- 0
            self$fetchAllItemsCallCount  <- 0

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

        getWebDavHostName = function()
        {
        },

        getResource = function(resource, uuid)
        {
            self$getResourceCallCount <- self$getResourceCallCount + 1
            self$returnContent
        },

        listResources = function(resource, filters = NULL, limit = 100, offset = 0)
        {
            self$listResourcesCallCount <- self$listResourcesCallCount + 1
            self$returnContent
        },

        fetchAllItems = function(resourceURL, filters)
        {
            self$fetchAllItemsCallCount <- self$fetchAllItemsCallCount + 1
            self$returnContent
        },

        deleteResource = function(resource, uuid)
        {
            self$deleteResourceCallCount <- self$deleteResourceCallCount + 1
            self$returnContent
        },

        updateResource = function(resource, uuid, newContent)
        {
            self$updateResourceCallCount <- self$updateResourceCallCount + 1
            self$returnContent
        },

        createResource = function(resource, content)
        {
            self$createResourceCallCount <- self$createResourceCallCount + 1
            self$returnContent
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
