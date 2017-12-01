source("./R/custom_classes.R")

HttpRequest <- setRefClass(

    "HttrRequest",

    fields = list(
        send_method         = "character",
        server_base_url     = "character",
        server_relative_url = "character",
        auth_token          = "character",
        allowed_methods     = "list",
        request_body        = "ANY",
        query_filters       = "ANY",
        response_limit      = "ANY",
        query_offset        = "ANY"
    ),

    methods = list(
        initialize = function(method,
                              token,
                              base_url,
                              relative_url,
                              body = NULL,
                              filters = NULL,
                              limit = 100,
                              offset = 0) 
        {
            send_method         <<- method
            auth_token          <<- token
            server_base_url     <<- base_url
            server_relative_url <<- relative_url
            request_body        <<- body
            query_filters       <<- filters
            response_limit      <<- limit
            query_offset        <<- offset
        },

        execute = function() 
        {
            #Todo(Fudo): Get rid of the switch and make this module more general.
            http_method <- switch(send_method,
                                  "GET"    = .self$getRequest,
                                  "POST"   = .self$postRequest,
                                  "PUT"    = .self$putRequest,
                                  "DELETE" = .self$deleteRequest,
                                  "PATCH"  = .self$pathcRequest)
            http_method()
        },

        getRequest = function() 
        {
            requestHeaders <- httr::add_headers(Authorization = .self$getAuthHeader())
            requestQuery   <- .self$generateQuery()
            url            <- paste0(server_base_url, server_relative_url, requestQuery)

            server_data <- httr::GET(url    = url,
                                     config = requestHeaders)
        },

        #Todo(Fudo): Try to make this more generic
        postRequest = function() 
        {
            url <- paste0(server_base_url, server_relative_url)
            requestHeaders <- httr::add_headers("Authorization" = .self$getAuthHeader(),
                                                "Content-Type"  = "application/json")
            response <- POST(url, body = request_body, config = requestHeaders)
        },

        putRequest = function() 
        {
            url <- paste0(server_base_url, server_relative_url)
            requestHeaders <- httr::add_headers("Authorization" = .self$getAuthHeader(),
                                                "Content-Type"  = "application/json")

            response <- PUT(url, body = request_body, config = requestHeaders)
        },

        deleteRequest = function() 
        {
            url <- paste0(server_base_url, server_relative_url)
            requestHeaders <- httr::add_headers("Authorization" = .self$getAuthHeader(),
                                                "Content-Type"  = "application/json")
            response <- DELETE(url, config = requestHeaders)
        },

        pathcRequest = function() 
        {
            #Todo(Fudo): Implement this later on.
            print("PATCH method")
        },

        getAuthHeader = function() 
        {
            auth_method <- "OAuth2"
            auth_header <- paste(auth_method, auth_token)
        },

        generateQuery = function() 
        {
            #Todo(Fudo): This function is a mess, refactor it
            finalQuery <- "?alt=json"

            if(!is.null(query_filters))
            {
                filters <- sapply(query_filters, function(filter)
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

            if(!is.null(response_limit))
            {
                if(!is.numeric(response_limit))
                    stop("Limit must be a numeric type.")
                
                finalQuery <- paste0(finalQuery, "&limit=", response_limit)
            }

            if(!is.null(query_offset))
            {
                if(!is.numeric(query_offset))
                    stop("Offset must be a numeric type.")
                
                finalQuery <- paste0(finalQuery, "&offset=", query_offset)
            }

            finalQuery
        }
    )
)
