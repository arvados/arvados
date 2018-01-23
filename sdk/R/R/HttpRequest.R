source("./R/util.R")

HttpRequest <- R6::R6Class(

    "HttrRequest",

    public = list(

        validContentTypes = NULL,

        initialize = function() 
        {
            self$validContentTypes <- c("text", "raw")
        },

        GET = function(url, headers = NULL, queryFilters = NULL, limit = NULL, offset = NULL)
        {
            headers <- httr::add_headers(unlist(headers))
            query <- self$createQuery(queryFilters, limit, offset)
            url <- paste0(url, query)

            serverResponse <- httr::GET(url = url, config = headers)
        },

        PUT = function(url, headers = NULL, body = NULL,
                       queryFilters = NULL, limit = NULL, offset = NULL)
        {
            headers <- httr::add_headers(unlist(headers))
            query <- self$createQuery(queryFilters, limit, offset)
            url <- paste0(url, query)

            serverResponse <- httr::PUT(url = url, config = headers, body = body)
        },

        POST = function(url, headers = NULL, body = NULL,
                        queryFilters = NULL, limit = NULL, offset = NULL)
        {
            headers <- httr::add_headers(unlist(headers))
            query <- self$createQuery(queryFilters, limit, offset)
            url <- paste0(url, query)

            serverResponse <- httr::POST(url = url, config = headers, body = body)
        },

        DELETE = function(url, headers = NULL, body = NULL,
                          queryFilters = NULL, limit = NULL, offset = NULL)
        {
            headers <- httr::add_headers(unlist(headers))
            query <- self$createQuery(queryFilters, limit, offset)
            url <- paste0(url, query)

            serverResponse <- httr::DELETE(url = url, config = headers)
        },

        PROPFIND = function(url, headers = NULL)
        {
            h <- curl::new_handle()
            curl::handle_setopt(h, customrequest = "PROPFIND")
            curl::handle_setheaders(h, .list = headers)

        propfindResponse <- curl::curl_fetch_memory(url, h)
        },

        MOVE = function(url, headers = NULL)
        {
            h <- curl::new_handle()
            curl::handle_setopt(h, customrequest = "MOVE")
            curl::handle_setheaders(h, .list = headers)

            propfindResponse <- curl::curl_fetch_memory(url, h)
        },

        createQuery = function(filters, limit, offset)
        {
            finalQuery <- NULL

            finalQuery <- c(finalQuery, private$createFiltersQuery(filters))
            finalQuery <- c(finalQuery, private$createLimitQuery(limit))
            finalQuery <- c(finalQuery, private$createOffsetQuery(offset))

            finalQuery <- finalQuery[!is.null(finalQuery)]
            finalQuery <- paste0(finalQuery, collapse = "&")

            if(finalQuery != "")
                finalQuery <- paste0("/?", finalQuery)

            finalQuery
        }
    ),

    private = list(

        createFiltersQuery = function(filters)
        {
            if(!is.null(filters))
            {
                filters <- RListToPythonList(filters, ",")
                encodedQuery <- URLencode(filters, reserved = T, repeated = T)

                return(paste0("filters=", encodedQuery))
            }

            return(NULL)
        },

        createLimitQuery = function(limit)
        {
            if(!is.null(limit))
            {
                limit <- suppressWarnings(as.numeric(limit))

                if(is.na(limit))
                    stop("Limit must be a numeric type.")
                
                return(paste0("limit=", limit))
            }

            return(NULL)
        },

        createOffsetQuery = function(offset)
        {
            if(!is.null(offset))
            {
                offset <- suppressWarnings(as.numeric(offset))

                if(is.na(offset))
                    stop("Offset must be a numeric type.")
                
                return(paste0("offset=", offset))
            }

            return(NULL)
        }
    ),

    cloneable = FALSE
)
