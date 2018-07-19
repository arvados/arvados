# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

HttpParser <- R6::R6Class(

    "HttrParser",

    public = list(

        validContentTypes = NULL,

        initialize = function()
        {
            self$validContentTypes <- c("text", "raw")
        },

        parseJSONResponse = function(serverResponse)
        {
            parsed_response <- httr::content(serverResponse,
                                             as = "parsed",
                                             type = "application/json")
        },

        parseResponse = function(serverResponse, outputType)
        {
            parsed_response <- httr::content(serverResponse, as = outputType)
        },

        getFileNamesFromResponse = function(response, uri)
        {
            text <- rawToChar(response$content)
            doc <- XML::xmlParse(text, asText=TRUE)
            base <- paste(paste("/", strsplit(uri, "/")[[1]][-1:-3], sep="", collapse=""), "/", sep="")
            result <- unlist(
                XML::xpathApply(doc, "//D:response/D:href", function(node) {
                    sub(base, "", URLdecode(XML::xmlValue(node)), fixed=TRUE)
                })
            )
            result <- result[result != ""]
            result[-1]
        },

        getFileSizesFromResponse = function(response, uri)
        {
            text <- rawToChar(response$content)
            doc <- XML::xmlParse(text, asText=TRUE)

            base <- paste(paste("/", strsplit(uri, "/")[[1]][-1:-3], sep="", collapse=""), "/", sep="")
            result <- XML::xpathApply(doc, "//D:response/D:propstat/D:prop/D:getcontentlength", function(node) {
              XML::xmlValue(node)
            })

            unlist(result)
        }
    )
)
