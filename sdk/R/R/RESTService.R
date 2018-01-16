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

            print(paste("File deleted:", relativePath))
        },

        move = function(from, to, uuid)
        {
            #Todo Do we need this URLencode?
            collectionURL <- URLencode(paste0(private$api$getWebDavHostName(), "c=",
                                              uuid, "/"))
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
            collectionURL <- URLencode(paste0(private$api$getWebDavHostName(), "c=", uuid))

            headers = list("Authorization" = paste("OAuth2", private$api$getToken()))

            response <- private$http$PROPFIND(collectionURL, headers)

            parsedResponse <- private$httpParser$parseWebDAVResponse(response, collectionURL)
            parsedResponse[-1]
        },

        getResourceSize = function(uuid, relativePathToResource)
        {
            collectionURL <- URLencode(paste0(private$api$getWebDavHostName(),
                                              "c=", uuid))
            subcollectionURL <- paste0(collectionURL, "/",
                                       relativePathToResource, "/");

            headers = list("Authorization" = paste("OAuth2",
                                                   private$api$getToken()))

            propfindResponse <- private$http$PROPFIND(subcollectionURL, headers)

            sizes <- private$httpParser$extractFileSizeFromWebDAVResponse(propfindResponse,
                                                                          collectionURL)
            sizes <- as.numeric(sizes[-1])

            return(sum(sizes))
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

            print(paste("File created:", relativePath))
        }
    ),

    cloneable = FALSE
)
