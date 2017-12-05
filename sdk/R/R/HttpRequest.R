source("./R/custom_classes.R")

HttpRequest <- setRefClass(

    "HttrRequest",

    fields = list(

        GET    = "function",
        PUT    = "function",
        POST   = "function",
        DELETE = "function"
    ),

    methods = list(
        initialize = function() 
        {
            # Public methods
            GET <<- function(url, headers = NULL, body = NULL,
                             queryFilters = NULL, limit = 100, offset = 0)
            {
                print(limit)
                headers <- httr::add_headers(unlist(headers))
                query <- .createQuery(queryFilters, limit, offset)
                url <- paste0(url, query)
                print(url)

                serverResponse <- httr::GET(url = url, config = headers)
            }

            PUT <<- function(url, headers = NULL, body = NULL,
                             queryFilters = NULL, limit = 100, offset = 0)
            {
                headers <- httr::add_headers(unlist(headers))
                query <- .createQuery(queryFilters, limit, offset)
                url <- paste0(url, query)

                serverResponse <- httr::PUT(url = url, config = headers, body = body)
            }

            POST <<- function(url, headers = NULL, body = NULL,
                              queryFilters = NULL, limit = 100, offset = 0)
            {
                headers <- httr::add_headers(unlist(headers))
                query <- .createQuery(queryFilters, limit, offset)
                url <- paste0(url, query)

                serverResponse <- httr::POST(url = url, config = headers, body = body)
            }

            DELETE <<- function(url, headers = NULL, body = NULL,
                             queryFilters = NULL, limit = 100, offset = 0)
            {
                headers <- httr::add_headers(unlist(headers))
                query <- .createQuery(queryFilters, limit, offset)
                url <- paste0(url, query)

                serverResponse <- httr::DELETE(url = url, config = headers)
            }

            # Private methods
            .createQuery <- function(filters, limit, offset)
            {
                finalQuery <- "?alt=json"

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

                    finalQuery <- paste0(finalQuery, "&filters=", encodedQuery)

                    #Todo(Fudo): This is a hack for now. Find a proper solution.
                    finalQuery <- stringr::str_replace_all(finalQuery, "%2B", "+")
                }

                if(!is.null(limit))
                {
                    if(!is.numeric(limit))
                        stop("Limit must be a numeric type.")
                    
                    finalQuery <- paste0(finalQuery, "&limit=", limit)
                }

                if(!is.null(offset))
                {
                    if(!is.numeric(offset))
                        stop("Offset must be a numeric type.")
                    
                    finalQuery <- paste0(finalQuery, "&offset=", offset)
                }

                finalQuery
            }
        }
    )
)
