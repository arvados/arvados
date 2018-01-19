RESTService <- R6::R6Class(

    "RESTService",

    public = list(

        initialize = function(api)
        {
            private$api <- api
            private$http <- api$getHttpClient()
            private$httpParser <- api$getHttpParser()
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
            fileURL <- paste0(private$api$getWebDavHostName(), "c=",
                              uuid, "/", relativePath);
            headers <- list(Authorization = paste("OAuth2", private$api$getToken())) 

            serverResponse <- private$http$DELETE(fileURL, headers)

            if(serverResponse$status_code < 200 || serverResponse$status_code >= 300)
                stop(paste("Server code:", serverResponse$status_code))

            paste("File deleted:", relativePath)
        },

        move = function(from, to, uuid)
        {
            collectionURL <- paste0(private$api$getWebDavHostName(), "c=", uuid, "/")
            fromURL <- paste0(collectionURL, from)
            toURL <- paste0(collectionURL, to)

            headers <- list("Authorization" = paste("OAuth2", private$api$getToken()),
                           "Destination" = toURL)

            serverResponse <- private$http$MOVE(fromURL, headers)

            if(serverResponse$status_code < 200 || serverResponse$status_code >= 300)
                stop(paste("Server code:", serverResponse$status_code))

            serverResponse
        },

        getCollectionContent = function(uuid)
        {
            collectionURL <- URLencode(paste0(private$api$getWebDavHostName(),
                                              "c=", uuid))

            headers <- list("Authorization" = paste("OAuth2", private$api$getToken()))

            response <- private$http$PROPFIND(collectionURL, headers)

            if(all(response == ""))
                stop("Response is empty, request may be misconfigured")

            private$httpParser$getFileNamesFromResponse(response, collectionURL)
        },

        getResourceSize = function(relativePath, uuid)
        {
            collectionURL <- URLencode(paste0(private$api$getWebDavHostName(),
                                              "c=", uuid))

            subcollectionURL <- paste0(collectionURL, "/", relativePath);

            headers <- list("Authorization" = paste("OAuth2",
                                                   private$api$getToken()))

            response <- private$http$PROPFIND(subcollectionURL, headers)

            if(all(response == ""))
                stop("Response is empty, request may be misconfigured")

            sizes <- private$httpParser$getFileSizesFromResponse(response,
                                                             collectionURL)
            as.numeric(sizes)
        },

        read = function(relativePath, uuid, contentType = "raw", offset = 0, length = 0)
        {
            fileURL <- paste0(private$api$getWebDavHostName(),
                             "c=", uuid, "/", relativePath);

            range <- paste0("bytes=", offset, "-")

            if(length > 0)
                range = paste0(range, offset + length - 1)

            if(offset == 0 && length == 0)
            {
                headers <- list(Authorization = paste("OAuth2", private$api$getToken()))
            }
            else
            {
                headers <- list(Authorization = paste("OAuth2", private$api$getToken()),
                                Range = range)
            }

            if(!(contentType %in% private$httpParser$validContentTypes))
                stop("Invalid contentType. Please use text or raw.")

            serverResponse <- private$http$GET(fileURL, headers)

            if(serverResponse$status_code < 200 || serverResponse$status_code >= 300)
                stop(paste("Server code:", serverResponse$status_code))

            private$httpParser$parseResponse(serverResponse, contentType)
        },

        write = function(relativePath, uuid, content, contentType)
        {
            fileURL <- paste0(private$api$getWebDavHostName(),
                             "c=", uuid, "/", relativePath);
            headers <- list(Authorization = paste("OAuth2", private$api$getToken()),
                            "Content-Type" = contentType)
            body <- content

            serverResponse <- private$http$PUT(fileURL, headers, body)

            if(serverResponse$status_code < 200 || serverResponse$status_code >= 300)
                stop(paste("Server code:", serverResponse$status_code))

            private$httpParser$parseResponse(serverResponse, "text")
        }
    ),

    private = list(

        api        = NULL,
        http       = NULL,
        httpParser = NULL,

        createNewFile = function(relativePath, uuid, contentType)
        {
            fileURL <- paste0(private$api$getWebDavHostName(), "c=",
                              uuid, "/", relativePath);
            headers <- list(Authorization = paste("OAuth2", private$api$getToken()), 
                            "Content-Type" = contentType)
            body <- NULL

            serverResponse <- private$http$PUT(fileURL, headers, body)

            if(serverResponse$status_code < 200 || serverResponse$status_code >= 300)
                stop(paste("Server code:", serverResponse$status_code))

            paste("File created:", relativePath)
        }
    ),

    cloneable = FALSE
)
