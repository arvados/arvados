source("./R/custom_classes.R")

HttpRequest <- setRefClass(

    "HttrRequest",

    fields = list(
        send_method         = "character",
        server_base_url     = "character",
        server_relative_url = "character",
        auth_token          = "character",
        allowed_methods     = "list",
        query_filters       = "ANY",
        response_limit      = "ANY",
        query_offset        = "ANY"
    ),

    methods = list(
        initialize = function(method,
                              token,
                              base_url,
                              relative_url,
                              filters = NULL,
                              limit = 100,
                              offset = 0) 
        {
            send_method         <<- method
            auth_token          <<- token
            server_base_url     <<- base_url
            server_relative_url <<- relative_url
            query_filters       <<- filters
            response_limit      <<- limit
            query_offset        <<- offset
        },

        execute = function() 
        {
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
            #Todo(Fudo): Implement this later on.
            print("POST method")
        },

        putRequest = function() 
        {
            #Todo(Fudo): Implement this later on.
            print("PUT method")
        },

        deleteRequest = function() 
        {
            #Todo(Fudo): Implement this later on.
            print("DELETE method")
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
            finalQuery <- ""

            if(!is.null(query_filters))
            {
                filters <- sapply(query_filters, function(filter)
                {
                    filter <- sapply(filter, function(component) 
                    {
                        component <- paste0("\"", component, "\"")
                    })
                    
                    queryParameter <- paste(filter, collapse = ",+")
                    queryParameter <- paste0("[[", queryParameter, "]]")
                })

                encodedQuery <- URLencode(filters, reserved = T, repeated = T)

                #Todo(Fudo): Hardcoded for now. Look for a better solution.
                finalQuery <- paste0("?alt=json&filters=", encodedQuery)

                #Todo(Fudo): This is a hack for now. Find a proper solution.
                finalQuery <- str_replace_all(finalQuery, "%2B", "+")
            }

            finalQuery
        }
    )
)
