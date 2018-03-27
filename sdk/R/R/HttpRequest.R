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

        exec = function(verb, url, headers = NULL, body = NULL, queryParams = NULL,
                        retryTimes = 0)
        {
            if(!(verb %in% self$validVerbs))
                stop("Http verb is not valid.")

            headers  <- httr::add_headers(unlist(headers))
            urlQuery <- self$createQuery(queryParams)
            url      <- paste0(url, urlQuery)

            # times = 1 regular call + numberOfRetries
            response <- httr::RETRY(verb, url = url, body = body,
                                    config = headers, times = retryTimes + 1)
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

                return(paste0("/?", query))
            }

            return("")
        }
    ),

    cloneable = FALSE
)
