RESTService <- R6::R6Class(

    "RESTService",

    public = list(

        hostName   = NULL,
        token      = NULL,
        http       = NULL,
        httpParser = NULL,
        numRetries = NULL,

        initialize = function(token, hostName,
                              http, httpParser,
                              numRetries     = 0,
                              webDavHostName = NULL)
        {
            version <- "v1"

            self$token       <- token
            self$hostName    <- paste0("https://", hostName,
                                       "/arvados/", version, "/")
            self$http        <- http
            self$httpParser  <- httpParser
            self$numRetries  <- numRetries

            private$rawHostName    <- hostName
            private$webDavHostName <- webDavHostName
        },

        setNumConnRetries = function(newNumOfRetries)
        {
            self$numRetries <- newNumOfRetries
        },

        getWebDavHostName = function()
        {
            if(is.null(private$webDavHostName))
            {
                discoveryDocumentURL <- paste0("https://", private$rawHostName,
                                               "/discovery/v1/apis/arvados/v1/rest")

                headers <- list(Authorization = paste("OAuth2", self$token))

                serverResponse <- self$http$execute("GET", discoveryDocumentURL, headers,
                                                    retryTimes = self$numRetries)

                discoveryDocument <- self$httpParser$parseJSONResponse(serverResponse)
                private$webDavHostName <- discoveryDocument$keepWebServiceUrl

                if(is.null(private$webDavHostName))
                    stop("Unable to find WebDAV server.")
            }

            private$webDavHostName
        },

        getResource = function(resource, uuid)
        {
            resourceURL <- paste0(self$hostName, resource, "/", uuid)
            headers <- list(Authorization = paste("OAuth2", self$token))

            serverResponse <- self$http$execute("GET", resourceURL, headers,
                                                retryTimes = self$numRetries)

            resource <- self$httpParser$parseJSONResponse(serverResponse)

            if(!is.null(resource$errors))
                stop(resource$errors)

            resource
        },

        listResources = function(resource, filters = NULL, limit = 100, offset = 0)
        {
            resourceURL <- paste0(self$hostName, resource)
            headers <- list(Authorization = paste("OAuth2", self$token))
            body <- NULL

            serverResponse <- self$http$execute("GET", resourceURL, headers, body,
                                                filters, limit, offset,
                                                self$numRetries)

            resources <- self$httpParser$parseJSONResponse(serverResponse)

            if(!is.null(resources$errors))
                stop(resources$errors)

            resources
        },

        fetchAllItems = function(resourceURL, filters)
        {
            headers <- list(Authorization = paste("OAuth2", self$token))

            offset <- 0
            itemsAvailable <- .Machine$integer.max
            items <- c()
            while(length(items) < itemsAvailable)
            {
                serverResponse <- self$http$execute(verb       = "GET",
                                                    url        = resourceURL,
                                                    headers    = headers,
                                                    body       = NULL,
                                                    query      = filters,
                                                    limit      = NULL,
                                                    offset     = offset,
                                                    retryTimes = self$numRetries)

                parsedResponse <- self$httpParser$parseJSONResponse(serverResponse)

                if(!is.null(parsedResponse$errors))
                    stop(parsedResponse$errors)

                items          <- c(items, parsedResponse$items)
                offset         <- length(items)
                itemsAvailable <- parsedResponse$items_available
            }

            items
        },

        deleteResource = function(resource, uuid)
        {
            collectionURL <- paste0(self$hostName, resource, "/", uuid)
            headers <- list("Authorization" = paste("OAuth2", self$token),
                            "Content-Type"  = "application/json")

            serverResponse <- self$http$execute("DELETE", collectionURL, headers,
                                                retryTimes = self$numRetries)

            removedResource <- self$httpParser$parseJSONResponse(serverResponse)

            if(!is.null(removedResource$errors))
                stop(removedResource$errors)

            removedResource
        },

        updateResource = function(resource, uuid, newContent)
        {
            resourceURL <- paste0(self$hostName, resource, "/", uuid)
            headers <- list("Authorization" = paste("OAuth2", self$token),
                            "Content-Type"  = "application/json")

            newContent <- jsonlite::toJSON(newContent, auto_unbox = T)

            serverResponse <- self$http$execute("PUT", resourceURL, headers, newContent,
                                                retryTimes = self$numRetries)

            updatedResource <- self$httpParser$parseJSONResponse(serverResponse)

            if(!is.null(updatedResource$errors))
                stop(updatedResource$errors)

            updatedResource
        },

        createResource = function(resource, content)
        {
            resourceURL <- paste0(self$hostName, resource)
            headers <- list("Authorization" = paste("OAuth2", self$token),
                            "Content-Type"  = "application/json")

            content <- jsonlite::toJSON(content, auto_unbox = T)

            serverResponse <- self$http$execute("POST", resourceURL, headers, content,
                                                retryTimes = self$numRetries)

            newResource <- self$httpParser$parseJSONResponse(serverResponse)

            if(!is.null(newResource$errors))
                stop(newResource$errors)

            newResource
        },

        create = function(files, uuid)
        {
            sapply(files, function(filePath)
            {
                private$createNewFile(filePath, uuid, "text/html")
            })
        },

        delete = function(relativePath, uuid)
        {
            fileURL <- paste0(self$getWebDavHostName(), "c=",
                              uuid, "/", relativePath);
            headers <- list(Authorization = paste("OAuth2", self$token)) 

            serverResponse <- self$http$execute("DELETE", fileURL, headers,
                                                retryTimes = self$numRetries)

            if(serverResponse$status_code < 200 || serverResponse$status_code >= 300)
                stop(paste("Server code:", serverResponse$status_code))

            serverResponse
        },

        move = function(from, to, uuid)
        {
            collectionURL <- paste0(self$getWebDavHostName(), "c=", uuid, "/")
            fromURL <- paste0(collectionURL, from)
            toURL <- paste0(collectionURL, to)

            headers <- list("Authorization" = paste("OAuth2", self$token),
                           "Destination" = toURL)

            serverResponse <- self$http$execute("MOVE", fromURL, headers,
                                                retryTimes = self$numRetries)

            if(serverResponse$status_code < 200 || serverResponse$status_code >= 300)
                stop(paste("Server code:", serverResponse$status_code))

            serverResponse
        },

        getCollectionContent = function(uuid)
        {
            collectionURL <- URLencode(paste0(self$getWebDavHostName(),
                                              "c=", uuid))

            headers <- list("Authorization" = paste("OAuth2", self$token))

            response <- self$http$execute("PROPFIND", collectionURL, headers,
                                          retryTimes = self$numRetries)

            if(all(response == ""))
                stop("Response is empty, request may be misconfigured")

            if(response$status_code < 200 || response$status_code >= 300)
                stop(paste("Server code:", response$status_code))

            self$httpParser$getFileNamesFromResponse(response, collectionURL)
        },

        getResourceSize = function(relativePath, uuid)
        {
            collectionURL <- URLencode(paste0(self$getWebDavHostName(),
                                              "c=", uuid))

            subcollectionURL <- paste0(collectionURL, "/", relativePath);

            headers <- list("Authorization" = paste("OAuth2", self$token))

            response <- self$http$execute("PROPFIND", subcollectionURL, headers,
                                          retryTimes = self$numRetries)

            if(all(response == ""))
                stop("Response is empty, request may be misconfigured")

            if(response$status_code < 200 || response$status_code >= 300)
                stop(paste("Server code:", response$status_code))

            sizes <- self$httpParser$getFileSizesFromResponse(response,
                                                              collectionURL)
            as.numeric(sizes)
        },

        read = function(relativePath, uuid, contentType = "raw", offset = 0, length = 0)
        {
            fileURL <- paste0(self$getWebDavHostName(),
                             "c=", uuid, "/", relativePath);

            range <- paste0("bytes=", offset, "-")

            if(length > 0)
                range = paste0(range, offset + length - 1)

            if(offset == 0 && length == 0)
            {
                headers <- list(Authorization = paste("OAuth2", self$token))
            }
            else
            {
                headers <- list(Authorization = paste("OAuth2", self$token),
                                Range = range)
            }

            if(!(contentType %in% self$httpParser$validContentTypes))
                stop("Invalid contentType. Please use text or raw.")

            serverResponse <- self$http$execute("GET", fileURL, headers,
                                                retryTimes = self$numRetries)

            if(serverResponse$status_code < 200 || serverResponse$status_code >= 300)
                stop(paste("Server code:", serverResponse$status_code))

            self$httpParser$parseResponse(serverResponse, contentType)
        },

        write = function(relativePath, uuid, content, contentType)
        {
            fileURL <- paste0(self$getWebDavHostName(),
                             "c=", uuid, "/", relativePath);
            headers <- list(Authorization = paste("OAuth2", self$token),
                            "Content-Type" = contentType)
            body <- content

            serverResponse <- self$http$execute("PUT", fileURL, headers, body,
                                                retryTimes = self$numRetries)

            if(serverResponse$status_code < 200 || serverResponse$status_code >= 300)
                stop(paste("Server code:", serverResponse$status_code))

            self$httpParser$parseResponse(serverResponse, "text")
        },

        getConnection = function(uuid, relativePath, openMode)
        {
            fileURL <- paste0(self$getWebDavHostName(), 
                              "c=", uuid, "/", relativePath);
            headers <- list(Authorization = paste("OAuth2", self$token))

            h <- curl::new_handle()
            curl::handle_setheaders(h, .list = headers)

            conn <- curl::curl(url = fileURL, open = openMode, handle = h)

            conn
        }
    ),

    private = list(

        webDavHostName = NULL,
        rawHostName    = NULL,

        createNewFile = function(relativePath, uuid, contentType)
        {
            fileURL <- paste0(self$getWebDavHostName(), "c=",
                              uuid, "/", relativePath)
            headers <- list(Authorization = paste("OAuth2", self$token), 
                            "Content-Type" = contentType)
            body <- NULL

            serverResponse <- self$http$execute("PUT", fileURL, headers, body,
                                                retryTimes = self$numRetries)

            if(serverResponse$status_code < 200 || serverResponse$status_code >= 300)
                stop(paste("Server code:", serverResponse$status_code))

            paste("File created:", relativePath)
        }
    ),

    cloneable = FALSE
)
