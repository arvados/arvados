source("./R/util.R")

HttpRequest <- R6::R6Class(

    "HttrRequest",

    public = list(

        validContentTypes = NULL,
        validVerbs = NULL,

        initialize = function() 
        {
            self$validContentTypes <- c("text", "raw")
            self$validVerbs <- c("GET", "POST", "PUT", "DELETE", "PROPFIND", "MOVE")
        },

        execute = function(verb, url, headers = NULL, body = NULL, query = NULL,
                           limit = NULL, offset = NULL, retryTimes = 0)
        {
            if(!(verb %in% self$validVerbs))
                stop("Http verb is not valid.")

            headers  <- httr::add_headers(unlist(headers))
            urlQuery <- self$createQuery(query, limit, offset)
            url      <- paste0(url, urlQuery)

            # times = 1 regular call + numberOfRetries
            response <- httr::RETRY(verb, url = url, body = body,
                                    config = headers, times = retryTimes + 1)
        },

        exec = function(verb, url, headers = NULL, body = NULL, queryParams = NULL,
                        retryTimes = 0)
        {
            if(!(verb %in% self$validVerbs))
                stop("Http verb is not valid.")

            headers  <- httr::add_headers(unlist(headers))
            urlQuery <- self$genQuery(queryParams)
            url      <- paste0(url, urlQuery)

            # times = 1 regular call + numberOfRetries
            response <- httr::RETRY(verb, url = url, body = body,
                                    config = headers, times = retryTimes + 1)
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
        },

        genQuery = function(queryParams)
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

                return(paste0("/?", query))
            }

            return("")
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
