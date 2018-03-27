getAPIDocument <- function(){
    url <- "https://4xphq.arvadosapi.com/discovery/v1/apis/arvados/v1/rest"
    serverResponse <- httr::RETRY("GET", url = url)

    httr::content(serverResponse, as = "parsed", type = "application/json")
}

#' @export
generateAPI <- function()
{
    #TODO: Consider passing discovery document URL as parameter.
    #TODO: Consider passing location where to create new files.
    discoveryDocument <- getAPIDocument()

    methodResources <- discoveryDocument$resources
    resourceNames   <- names(methodResources)

    doc <- generateMethodsDocumentation(methodResources, resourceNames)
    arvadosAPIHeader <- generateAPIClassHeader()
    arvadosProjectMethods <- generateProjectMethods()
    arvadosClassMethods <- generateClassContent(methodResources, resourceNames)
    arvadosAPIFooter <- generateAPIClassFooter()

    arvadosClass <- c(doc,
                      arvadosAPIHeader,
                      arvadosProjectMethods,
                      arvadosClassMethods,
                      arvadosAPIFooter)

    fileConn <- file("./R/Arvados.R", "w")
    writeLines(unlist(arvadosClass), fileConn)
    close(fileConn)
    NULL
}

generateAPIClassHeader <- function()
{
    c("#' @export",
      "Arvados <- R6::R6Class(",
      "",
      "\t\"Arvados\",",
      "",
      "\tpublic = list(",
      "",
      "\t\tinitialize = function(authToken = NULL, hostName = NULL, numRetries = 0)",
      "\t\t{",
      "\t\t\tif(!is.null(hostName))",
      "\t\t\t\tSys.setenv(ARVADOS_API_HOST = hostName)",
      "",
      "\t\t\tif(!is.null(authToken))",
      "\t\t\t\tSys.setenv(ARVADOS_API_TOKEN = authToken)",
      "",
      "\t\t\thostName <- Sys.getenv(\"ARVADOS_API_HOST\")",
      "\t\t\ttoken    <- Sys.getenv(\"ARVADOS_API_TOKEN\")",
      "",
      "\t\t\tif(hostName == \"\" | token == \"\")",
      "\t\t\t\tstop(paste(\"Please provide host name and authentification token\",",
      "\t\t\t\t\t\t   \"or set ARVADOS_API_HOST and ARVADOS_API_TOKEN\",",
      "\t\t\t\t\t\t   \"environment variables.\"))",
      "",
      "\t\t\tprivate$token <- token",
      "\t\t\tprivate$host  <- paste0(\"https://\", hostName, \"/arvados/v1/\")",
      "\t\t\tprivate$numRetries <- numRetries",
      "\t\t\tprivate$REST <- RESTService$new(token, hostName,",
      "\t\t\t                                HttpRequest$new(), HttpParser$new(),",
      "\t\t\t                                numRetries)",
      "",
      "\t\t},\n")
}

generateProjectMethods <- function()
{
    c("\t\tprojects.get = function(uuid)",
      "\t\t{",
      "\t\t\tself$groups.get(uuid)",
      "\t\t},",
      "",
      "\t\tprojects.create = function(group, ensure_unique_name = \"false\")",
      "\t\t{",
      "\t\t\tgroup <- c(\"group_class\" = \"project\", group)",
      "\t\t\tself$groups.create(group, ensure_unique_name)",
      "\t\t},",
      "",
      "\t\tprojects.update = function(group, uuid)",
      "\t\t{",
      "\t\t\tgroup <- c(\"group_class\" = \"project\", group)",
      "\t\t\tself$groups.update(group, uuid)",
      "\t\t},",
      "",
      "\t\tprojects.list = function(filters = NULL, where = NULL,",
      "\t\t\torder = NULL, select = NULL, distinct = NULL,",
      "\t\t\tlimit = \"100\", offset = \"0\", count = \"exact\",",
      "\t\t\tinclude_trash = NULL)",
      "\t\t{",
      "\t\t\tfilters[[length(filters) + 1]] <- list(\"group_class\", \"=\", \"project\")",
      "\t\t\tself$groups.list(filters, where, order, select, distinct,",
      "\t\t\t                 limit, offset, count, include_trash)",
      "\t\t},",
      "",
      "\t\tprojects.delete = function(uuid)",
      "\t\t{",
      "\t\t\tself$groups.delete(uuid)",
      "\t\t},",
      "")
}

generateClassContent <- function(methodResources, resourceNames)
{
    arvadosMethods <- Map(function(resource, resourceName)
    {
        methodNames <- names(resource$methods)

        functions <- Map(function(methodMetaData, methodName)
        {
            #NOTE: Index, show and destroy are aliases for the preferred names
            # "list", "get" and "delete". Until they are removed from discovery
            # document we will filter them here.
            if(methodName %in% c("index", "show", "destroy"))
               return(NULL)

            methodName <- paste0(resourceName, ".", methodName)
            createMethod(methodName, methodMetaData)

        }, resource$methods, methodNames)

        unlist(unname(functions))

    }, methodResources, resourceNames)

    arvadosMethods
}

generateAPIClassFooter <- function()
{
    c("\t\tgetHostName = function() private$host,",
      "\t\tgetToken = function() private$token,",
      "\t\tsetRESTService = function(newREST) private$REST <- newREST,",
      "\t\tgetRESTService = function() private$REST",
      "\t),",
      "",
      "\tprivate = list(",
      "",
      "\t\ttoken = NULL,",
      "\t\thost = NULL,",
      "\t\tREST = NULL,",
      "\t\tnumRetries = NULL",
      "\t),",
      "",
      "\tcloneable = FALSE",
      ")")
}

createMethod <- function(name, methodMetaData)
{
    args      <- getMethodArguments(methodMetaData)
    signature <- getMethodSignature(name, args)
    body      <- getMethodBody(methodMetaData)

    c(signature,
      "\t\t{",
          body,
      "\t\t},\n")
}

getMethodArguments <- function(methodMetaData)
{
    request <- methodMetaData$request
    requestArgs <- NULL

    if(!is.null(request))
    {
        resourceName <- tolower(request$properties[[1]][[1]])

        if(request$required)
            requestArgs <- resourceName
        else
            requestArgs <- paste(resourceName, "=", "NULL")
    }

    argNames <- names(methodMetaData$parameters)

    args <- sapply(argNames, function(argName)
    {
        arg <- methodMetaData$parameters[[argName]]

        if(!arg$required)
        {
            if(!is.null(arg$default))
                return(paste0(argName, " = ", "\"", arg$default, "\""))
            else
                return(paste(argName, "=", "NULL"))
        }

        argName
    })

    c(requestArgs, args)
}

getMethodSignature <- function(methodName, args)
{
    collapsedArgs <- paste0(args, collapse = ", ")
    lineLengthLimit <- 40

    if(nchar(collapsedArgs) > lineLengthLimit)
    {
        return(paste0("\t\t",
                      formatArgs(paste(methodName, "= function("),
                                 "\t", args, ")", lineLengthLimit)))
    }
    else
    {
        return(paste0("\t\t", methodName, " = function(", collapsedArgs, ")"))
    }
}

getMethodBody <- function(methodMetaData)
{
    url              <- getRequestURL(methodMetaData)
    headers          <- getRequestHeaders()
    requestQueryList <- getRequestQueryList(methodMetaData)
    requestBody      <- getRequestBody(methodMetaData)
    request          <- getRequest(methodMetaData)
    response         <- getResponse(methodMetaData)
    errorCheck       <- getErrorCheckingCode()
    returnStatement  <- getReturnObject()

    body <- c(url,
              headers,
              requestQueryList, "",
              requestBody, "",
              request, response, "",
              errorCheck, "",
              returnStatement)

    paste0("\t\t\t", body)
}

getRequestURL <- function(methodMetaData)
{
    endPoint <- methodMetaData$path
    endPoint <- stringr::str_replace_all(endPoint, "\\{", "${")
    url <- c(paste0("endPoint <- stringr::str_interp(\"", endPoint, "\")"),
             paste0("url <- paste0(private$host, endPoint)"))
    url
}

getRequestHeaders <- function()
{
    c("headers <- list(Authorization = paste(\"OAuth2\", private$token), ",
      "                \"Content-Type\" = \"application/json\")")
}

getRequestQueryList <- function(methodMetaData)
{
    queryArgs <- names(Filter(function(arg) arg$location == "query",
                        methodMetaData$parameters))

    if(length(queryArgs) == 0)
        return("queryArgs <- NULL")

    queryArgs <- sapply(queryArgs, function(arg) paste0(arg, " = ", arg))
    collapsedArgs <- paste0(queryArgs, collapse = ", ")

    lineLengthLimit <- 40

    if(nchar(collapsedArgs) > lineLengthLimit)
        return(formatArgs("queryArgs <- list(", "\t\t\t\t  ", queryArgs, ")",
                          lineLengthLimit))
    else
        return(paste0("queryArgs <- list(", collapsedArgs, ")"))
}

getRequestBody <- function(methodMetaData)
{
    request <- methodMetaData$request

    if(is.null(request) || !request$required)
        return("body <- NULL")

    resourceName <- tolower(request$properties[[1]][[1]])

    requestParameterName <- names(request$properties)[1]

    c(paste0("if(length(", resourceName, ") > 0)"),
      paste0("\tbody <- jsonlite::toJSON(list(", resourceName, " = ", resourceName, "), "),
             "\t                         auto_unbox = TRUE)",
      "else",
      "\tbody <- NULL")
}

getRequest <- function(methodMetaData)
{
    method <- methodMetaData$httpMethod
    c(paste0("response <- private$REST$http$exec(\"", method, "\", url, headers, body,"),
      "                                   queryArgs, private$numRetries)")
}

getResponse <- function(methodMetaData)
{
    "resource <- private$REST$httpParser$parseJSONResponse(response)"
}

getErrorCheckingCode <- function()
{
    c("if(!is.null(resource$errors))",
      "\tstop(resource$errors)")
}

getReturnObject <- function()
{
    "resource"
}

#NOTE: Arvados class documentation:

generateMethodsDocumentation <- function(methodResources, resourceNames)
{
    methodsDoc <- unlist(unname(Map(function(resource, resourceName)
    {
        methodNames <- names(resource$methods)

        methodDoc <- Map(function(methodMetaData, methodName)
        {
            #NOTE: Index, show and destroy are aliases for the preferred names
            # "list", "get" and "delete". Until they are removed from discovery
            # document we will filter them here.
            if(methodName %in% c("index", "show", "destroy"))
               return(NULL)

            methodName <- paste0(resourceName, ".", methodName)
            getMethodDocumentation(methodName, methodMetaData)

        }, resource$methods, methodNames)

        unlist(unname(methodDoc))

    }, methodResources, resourceNames)))
    
    methodsDoc
}

getMethodDocumentation <- function(methodName, methodMetaData)
{
    name        <- paste("#' @name", methodName)
    usage       <- getMethodUsage(methodName, methodMetaData)
    description <- paste("#'", methodName, "is a method defined in Arvados class.")
    params      <- getMethodDescription(methodMetaData)
    returnValue <- paste("#' @return", methodMetaData$response[["$ref"]], "object.")

    c(description,
      "#' ",
      usage,
      params,
      returnValue,
      name,
      "NULL",
      "")
}

getMethodUsage <- function(methodName, methodMetaData)
{
    lineLengthLimit <- 40
    args <- getMethodArguments(methodMetaData)
    c(formatArgs(paste0("#' @usage arv$", methodName,
                        "("), "#' \t", args, ")", lineLengthLimit))
}

getMethodDescription <- function(methodMetaData)
{
    request <- methodMetaData$request
    requestDoc <- NULL

    if(!is.null(request))
    {
        requestDoc <- unname(unlist(sapply(request$properties, function(prop)
                             {
                                 className <- sapply(prop, function(ref) ref)
                                 objectName <- paste0(tolower(substr(className, 1, 1)),
                                                      substr(className, 2, nchar(className)))
                                 paste("#' @param", objectName, className, "object.") 
                             })))
    }

    argNames <- names(methodMetaData$parameters)

    argsDoc <- unname(unlist(sapply(argNames, function(argName)
    {
        arg <- methodMetaData$parameters[[argName]]
        argDescription <- arg$description
        paste("#' @param", argName, argDescription) 
    })))

    c(requestDoc, argsDoc)
}

#NOTE: Utility functions:

# This function is used to split very long lines of code into smaller chunks.
# This is usually the case when we pass a lot of named argumets to a function.
formatArgs <- function(prependAtStart, prependToEachSplit,
                       args, appendAtEnd, lineLength)
{
    if(length(args) > 1)
    {
        args[1:(length(args) - 1)] <- paste0(args[1:(length(args) - 1)], ",") 
    }

    args[1] <- paste0(prependAtStart, args[1])
    args[length(args)] <- paste0(args[length(args)], appendAtEnd)

    argsLength <- length(args)
    argLines <- list()
    index <- 1

    while(index <= argsLength)
    {
        line <- args[index]
        index <- index + 1

        while(nchar(line) < lineLength && index <= argsLength)
        {
            line <- paste(line, args[index])
            index <- index + 1
        }

        argLines <- c(argLines, line)
    }
    
    argLines <- unlist(argLines)
    argLinesLen <- length(argLines)

    if(argLinesLen > 1)
        argLines[2:argLinesLen] <- paste0(prependToEachSplit, argLines[2:argLinesLen]) 

    argLines
}
