source("./R/Subcollection.R")
source("./R/ArvadosFile.R")
source("./R/HttpRequest.R")
source("./R/HttpParser.R")

#' Arvados Collection Object
#'
#' Update description
#'
#' @examples arv = Collection$new(api, uuid)
#' @export Collection
Collection <- R6::R6Class(

    "Collection",

    public = list(

        api  = NULL,
        uuid = NULL,

        initialize = function(api, uuid)
        {
            self$api <- api
            private$http <- HttpRequest$new()
            private$httpParser <- HttpParser$new()

            self$uuid <- uuid
            collection <- self$api$getCollection(uuid)

            private$fileContent <- private$getCollectionContent()
            private$tree <- CollectionTree$new(private$fileContent, self)
        },

        add = function(content, relativePath = "")
        {
            if(relativePath == "" ||
               relativePath == "." ||
               relativePath == "./")
            {
                subcollection <- private$tree$.__enclos_env__$private$tree
            }
            else
            {
                if(endsWith(relativePath, "/") && nchar(relativePath) > 0)
                    relativePath <- substr(relativePath, 1, nchar(relativePath) - 1)

                subcollection <- self$get(relativePath)
            }

            if(is.null(subcollection))
                stop(paste("Subcollection", relativePath, "doesn't exist."))

            if(is.character(content))
            {
                sapply(content, function(fileName)
                {
                    subcollection$add(ArvadosFile$new(fileName))
                })
            }
            else if("ArvadosFile"   %in% class(content) ||
                    "Subcollection" %in% class(content))
            {
                subcollection$add(content)
            }
        },

        remove = function(content)
        {
            if(is.character(content))
            {
                sapply(content, function(filePath)
                {
                    if(endsWith(filePath, "/") && nchar(filePath) > 0)
                        filePath <- substr(filePath, 1, nchar(filePath) - 1)

                    file <- self$get(filePath)

                    if(is.null(file))
                        stop(paste("File", filePath, "doesn't exist."))

                    file$removeFromCollection()
                })
            }
            else if("ArvadosFile"   %in% class(content) ||
                    "Subcollection" %in% class(content))
            {
                if(is.null(content$.__enclos_env__$private$collection) || 
                   content$.__enclos_env__$private$collection$uuid != self$uuid)
                    stop("Subcollection doesn't belong to this collection.")

                content$removeFromCollection()
            }
        },

        move = function(content, newLocation)
        {
            if(endsWith(content, "/"))
                content <- substr(content, 0, nchar(content) - 1)

            elementToMove <- self$get(content)

            if(is.null(elementToMove))
                stop("Element you want to move doesn't exist in the collection.")

            elementToMove$move(newLocation)
        },

        getFileListing = function() private$getCollectionContent(),

        get = function(relativePath)
        {
            private$tree$getElement(relativePath)
        }
    ),

    private = list(

        http       = NULL,
        httpParser = NULL,
        tree       = NULL,

        fileContent = NULL,

        getCollectionContent = function()
        {
            collectionURL <- URLencode(paste0(self$api$getWebDavHostName(), "c=", self$uuid))

            headers = list("Authorization" = paste("OAuth2", self$api$getToken()))

            response <- private$http$PROPFIND(collectionURL, headers)

            parsedResponse <- private$httpParser$parseWebDAVResponse(response, collectionURL)
            parsedResponse[-1]
        },

        createFilesOnREST = function(files)
        {
            sapply(files, function(filePath)
            {
                private$createNewFile(filePath, NULL, "text/html")
            })
        },
        
        generateTree = function(content)
        {
            treeBranches <- sapply(collectionContent, function(filePath)
            {
                splitPath <- unlist(strsplit(filePath$name, "/", fixed = TRUE))

                branch = private$createBranch(splitPath, filePath$fileSize)      
            })
        },

        createNewFile = function(relativePath, content, contentType)
        {
            fileURL <- paste0(self$api$getWebDavHostName(), "c=", self$uuid, "/", relativePath);
            headers <- list(Authorization = paste("OAuth2", self$api$getToken()), 
                            "Content-Type" = contentType)
            body <- content

            serverResponse <- private$http$PUT(fileURL, headers, body)

            if(serverResponse$status_code < 200 || serverResponse$status_code >= 300)
                stop(paste("Server code:", serverResponse$status_code))

            print(paste("File created:", relativePath))
        },
        
        deleteFromREST = function(relativePath)
        {
            fileURL <- paste0(self$api$getWebDavHostName(), "c=", self$uuid, "/", relativePath);
            headers <- list(Authorization = paste("OAuth2", self$api$getToken())) 

            serverResponse <- private$http$DELETE(fileURL, headers)

            if(serverResponse$status_code < 200 || serverResponse$status_code >= 300)
                stop(paste("Server code:", serverResponse$status_code))

            print(paste("File deleted:", relativePath))
        },

        moveOnREST = function(from, to)
        {
            collectionURL <- URLencode(paste0(self$api$getWebDavHostName(), "c=", self$uuid, "/"))
            fromURL <- paste0(collectionURL, from)
            toURL <- paste0(collectionURL, to)

            headers = list("Authorization" = paste("OAuth2", self$api$getToken()),
                           "Destination" = toURL)

            serverResponse <- private$http$MOVE(fromURL, headers)

            if(serverResponse$status_code < 200 || serverResponse$status_code >= 300)
                stop(paste("Server code:", serverResponse$status_code))

            serverResponse
        }
    ),

    cloneable = FALSE
)
