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
                subcollection <- private$tree$getTree()
            }
            else
            {
                if(endsWith(relativePath, "/") && nchar(relativePath) > 0)
                    relativePath <- substr(relativePath, 1, nchar(relativePath) - 1)

                subcollection <- self$get(relativePath)
            }

            if(is.null(subcollection))
                stop(paste("Subcollection", relativePath, "doesn't exist."))

            if("ArvadosFile"   %in% class(content) ||
               "Subcollection" %in% class(content))
            {
                subcollection$add(content)

                content
            }
            else
            {
                contentClass <- paste(class(content), collapse = ", ")
                stop(paste("Expected AravodsFile or Subcollection object, got",
                           paste0("(", contentClass, ")"), "."))
            }
        },

        create = function(fileNames, relativePath = "")
        {
            if(relativePath == "" ||
               relativePath == "." ||
               relativePath == "./")
            {
                subcollection <- private$tree$getTree()
            }
            else
            {
                if(endsWith(relativePath, "/") && nchar(relativePath) > 0)
                    relativePath <- substr(relativePath, 1, nchar(relativePath) - 1)

                subcollection <- self$get(relativePath)
            }

            if(is.null(subcollection))
                stop(paste("Subcollection", relativePath, "doesn't exist."))

            if(is.character(fileNames))
            {
                arvadosFiles <- NULL
                sapply(fileNames, function(fileName)
                {
                    childWithSameName <- subcollection$get(fileName)
                    if(!is.null(childWithSameName))
                        stop("Destination already contains file with same name.")

                    newFile <- ArvadosFile$new(fileName)
                    subcollection$add(newFile)

                    arvadosFiles <<- c(arvadosFiles, newFile)
                })

                if(length(arvadosFiles) == 1)
                    return(arvadosFiles[[1]])
                else
                    return(arvadosFiles)
            }
            else 
            {
                contentClass <- paste(class(fileNames), collapse = ", ")
                stop(paste("Expected character vector, got",
                           paste0("(", contentClass, ")"), "."))
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

                    parent <- file$getParent()
                    parent$remove(filePath)
                })
            }
            else if("ArvadosFile"   %in% class(content) ||
                    "Subcollection" %in% class(content))
            {
                if(is.null(content$getCollection()) || 
                   content$getCollection()$uuid != self$uuid)
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
        },
        
        #Todo: Move these methods to another class.
        createFilesOnREST = function(files)
        {
            sapply(files, function(filePath)
            {
                self$createNewFile(filePath, NULL, "text/html")
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

        generateTree = function(content)
        {
            treeBranches <- sapply(collectionContent, function(filePath)
            {
                splitPath <- unlist(strsplit(filePath$name, "/", fixed = TRUE))

                branch = private$createBranch(splitPath, filePath$fileSize)      
            })
        }
    ),

    cloneable = FALSE
)
