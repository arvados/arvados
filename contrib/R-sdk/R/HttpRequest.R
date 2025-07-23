# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

HttpRequest <- R6::R6Class(

    "HttrRequest",

    public = list(

        validContentTypes = NULL,
        validVerbs = NULL,

        initialize = function()
        {
            self$validContentTypes <- c("text", "raw")
            self$validVerbs <- c("GET", "POST", "PUT", "DELETE", "PROPFIND", "MOVE", "COPY")
        },

        exec = function(verb, url, headers = NULL, body = NULL, queryParams = NULL,
                        retryTimes = 0)
        {
            if(!(verb %in% self$validVerbs))
                stop("Http verb is not valid.")

            urlQuery <- self$createQuery(queryParams)
            url      <- paste0(url, urlQuery)

            config <- httr::add_headers(unlist(headers))
            if(toString(Sys.getenv("ARVADOS_API_HOST_INSECURE") == "TRUE"))
               config$options = list(ssl_verifypeer = 0L)

            response <- httr::RETRY(verb, url = url, body = body,
                                    config = config, times = retryTimes + 1)
        },

        createQuery = function(queryParams)
        {
            queryParams <- Filter(Negate(is.null), queryParams)

            query <- sapply(queryParams, function(param)
            {
                if(is.list(param) || length(param) > 1)
                    param <- RListToPythonList(param, ",")

                URLencode(as.character(param), reserved = T, repeated = T)

            }, USE.NAMES = TRUE)

            if(length(query) > 0)
            {
                query <- paste0(names(query), "=", query, collapse = "&")

                return(paste0("?", query))
            }

            return("")
        },

        getConnection = function(url, headers, openMode)
        {
            h <- curl::new_handle()
            curl::handle_setheaders(h, .list = headers)

            if(toString(Sys.getenv("ARVADOS_API_HOST_INSECURE") == "TRUE"))
               curl::handle_setopt(h, ssl_verifypeer = 0L)

            conn <- curl::curl(url = url, open = openMode, handle = h)
        }
    ),

    cloneable = FALSE
)
