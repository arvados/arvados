# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

RESTService <- R6::R6Class(

    "RESTService",

    public = list(

        token      = NULL,
        http       = NULL,
        httpParser = NULL,
        numRetries = NULL,

        initialize = function(token, rawHost,
                              http, httpParser,
                              numRetries     = 0,
                              webDavHostName = NULL)
        {
            self$token      <- token
            self$http       <- http
            self$httpParser <- httpParser
            self$numRetries <- numRetries

            private$rawHostName    <- rawHost
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
                publicConfigURL <- paste0("https://", private$rawHostName,
                                               "/arvados/v1/config")

                serverResponse <- self$http$exec("GET", publicConfigURL, retryTimes = self$numRetries)

                configDocument <- self$httpParser$parseJSONResponse(serverResponse)
                private$webDavHostName <- configDocument$Services$WebDAVDownload$ExternalURL

                if(is.null(private$webDavHostName))
                    stop("Unable to find WebDAV server.")
            }

            private$webDavHostName
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

            serverResponse <- self$http$exec("DELETE", fileURL, headers,
                                             retryTimes = self$numRetries)

            if(serverResponse$status_code < 200 || serverResponse$status_code >= 300)
                stop(paste("Server code:", serverResponse$status_code))

            serverResponse
        },

        move = function(from, to, uuid)
        {
            collectionURL <- paste0(self$getWebDavHostName(), "c=", uuid, "/")
            fromURL <- paste0(collectionURL, from)
            toURL <- paste0(collectionURL, trimFromStart(to, "/"))

            headers <- list("Authorization" = paste("OAuth2", self$token),
                            "Destination" = toURL)

            serverResponse <- self$http$exec("MOVE", fromURL, headers,
                                             retryTimes = self$numRetries)

            if(serverResponse$status_code < 200 || serverResponse$status_code >= 300)
                stop(paste("Server code:", serverResponse$status_code))

            serverResponse
        },

        copy = function(from, to, uuid)
        {
            collectionURL <- paste0(self$getWebDavHostName(), "c=", uuid, "/")
            fromURL <- paste0(collectionURL, from)
            toURL <- paste0(collectionURL, trimFromStart(to, "/"))

            headers <- list("Authorization" = paste("OAuth2", self$token),
                            "Destination" = toURL)

            serverResponse <- self$http$exec("COPY", fromURL, headers,
                                             retryTimes = self$numRetries)

            if(serverResponse$status_code < 200 || serverResponse$status_code >= 300)
                stop(paste("Server code:", serverResponse$status_code))

            serverResponse
        },

       getCollectionContent = function(uuid, relativePath = NULL)

        {
            collectionURL <- URLencode(paste0(self$getWebDavHostName(),
                                             "c=", uuid, "/", relativePath))

            headers <- list("Authorization" = paste("Bearer", self$token))

            response <- self$http$exec("PROPFIND", collectionURL, headers,
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

            response <- self$http$exec("PROPFIND", subcollectionURL, headers,
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

            serverResponse <- self$http$exec("GET", fileURL, headers,
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

            serverResponse <- self$http$exec("PUT", fileURL, headers, body,
                                             retryTimes = self$numRetries)

            if(serverResponse$status_code < 200 || serverResponse$status_code >= 300)
                stop(paste("Server code:", serverResponse$status_code))

            self$httpParser$parseResponse(serverResponse, "text")
        },

        getConnection = function(relativePath, uuid, openMode)
        {
            fileURL <- paste0(self$getWebDavHostName(),
                              "c=", uuid, "/", relativePath);
            headers <- list(Authorization = paste("OAuth2", self$token))

            conn <- self$http$getConnection(fileURL, headers, openMode)
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

            serverResponse <- self$http$exec("PUT", fileURL, headers, body,
                                             retryTimes = self$numRetries)

            if (serverResponse$status_code < 200){ # to wyrzuca błędy
                stop(paste("Server code:", serverResponse$status_code))}
            else if (serverResponse$status_code >= 300 & serverResponse$status_code < 422) {
                stop(paste("Server code:", serverResponse$status_code))}
            else if (serverResponse$status_code == 422 ) {
                stop(paste("Project of that name already exists. If you want to change it use project_update() instead"))}

            paste("File created:", relativePath)
        }
    ),

    cloneable = FALSE
)
