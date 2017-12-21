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
            query <- private$createQuery(queryFilters, limit, offset)
            url <- paste0(url, query)

            serverResponse <- httr::GET(url = url, config = headers)
        },

        PUT = function(url, headers = NULL, body = NULL,
                       queryFilters = NULL, limit = 100, offset = 0)
        {
            headers <- httr::add_headers(unlist(headers))
            query <- private$createQuery(queryFilters, limit, offset)
            url <- paste0(url, query)

            serverResponse <- httr::PUT(url = url, config = headers, body = body)
        },

        POST = function(url, headers = NULL, body = NULL,
                        queryFilters = NULL, limit = 100, offset = 0)
        {
            headers <- httr::add_headers(unlist(headers))
            query <- private$createQuery(queryFilters, limit, offset)
            url <- paste0(url, query)

            serverResponse <- httr::POST(url = url, config = headers, body = body)
        },

        DELETE = function(url, headers = NULL, body = NULL,
                          queryFilters = NULL, limit = NULL, offset = NULL)
        {
            headers <- httr::add_headers(unlist(headers))
            query <- private$createQuery(queryFilters, limit, offset)
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
        }
    ),

    private = list(

        #Todo(Fudo): Refactor this and find a better way to build
        # Python array from R list (recursion?)
        createQuery = function(filters, limit, offset)
        {
            finalQuery <- NULL

            if(!is.null(filters))
            {
                filters <- sapply(filters, function(filter)
                {
                    if(length(filter) != 3)
                        stop("Filter list must have exacthey 3 elements.")

                    attributeAndOperator = filter[c(1, 2)]
                    filterList = filter[[3]]
                    filterListIsPrimitive = TRUE
                    if(length(filterList) > 1)
                        filterListIsPrimitive = FALSE

                    attributeAndOperator <- sapply(attributeAndOperator, function(component) {
                        component <- paste0("\"", component, "\"")
                    })

                    filterList <- sapply(unlist(filterList), function(filter) {
                        filter <- paste0("\"", filter, "\"")
                    })

                    filterList <- paste(filterList, collapse = ",+")

                    if(!filterListIsPrimitive)
                        filterList <- paste0("[", filterList, "]")

                    filter <- c(attributeAndOperator, filterList)

                    queryParameter <- paste(filter, collapse = ",+")
                    queryParameter <- paste0("[", queryParameter, "]")
        
                })

                filters <- paste(filters, collapse = ",+")
                filters <- paste0("[", filters, "]")

                encodedQuery <- URLencode(filters, reserved = T, repeated = T)

                #Todo(Fudo): This is a hack for now. Find a proper solution.
                encodedQuery <- stringr::str_replace_all(encodedQuery, "%2B", "+")

                finalQuery <- c(finalQuery, paste0("filters=", encodedQuery))

                finalQuery
            }

            if(!is.null(limit))
            {
                if(!is.numeric(limit))
                    stop("Limit must be a numeric type.")
                
                finalQuery <- c(finalQuery, paste0("limit=", limit))
            }

            if(!is.null(offset))
            {
                if(!is.numeric(offset))
                    stop("Offset must be a numeric type.")
                
                finalQuery <- c(finalQuery, paste0("offset=", offset))
            }

            if(length(finalQuery) > 1)
            {
                finalQuery <- paste0(finalQuery, collapse = "&")
                finalQuery <- paste0("?", finalQuery)
            }

            finalQuery
        }
    ),

    cloneable = FALSE
)
