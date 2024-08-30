# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

getAPIDocument <- function(){
    url <- "https://jutro.arvadosapi.com/discovery/v1/apis/arvados/v1/rest"
    serverResponse <- httr::RETRY("GET", url = url)

    httr::content(serverResponse, as = "parsed", type = "application/json")
}

#' generateAPI
#'
#' Autogenerate classes to interact with Arvados from the Arvados discovery document.
#'
#' @export
generateAPI <- function()
{
    #TODO: Consider passing discovery document URL as parameter.
    #TODO: Consider passing location where to create new files.
    discoveryDocument <- getAPIDocument()

    methodResources <- discoveryDocument$resources
    resourceNames   <- names(methodResources)

    methodDoc <- genMethodsDoc(methodResources, resourceNames)
    classDoc <- genAPIClassDoc(methodResources, resourceNames)
    arvadosAPIHeader <- genAPIClassHeader()
    arvadosProjectMethods <- genProjectMethods()
    arvadosClassMethods <- genClassContent(methodResources, resourceNames)
    arvadosAPIFooter <- genAPIClassFooter()

    arvadosClass <- c(methodDoc,
                      classDoc,
                      arvadosAPIHeader,
                      arvadosProjectMethods,
                      arvadosClassMethods,
                      arvadosAPIFooter)

    fileConn <- file("./R/Arvados.R", "w")
    writeLines(c(
    "# Copyright (C) The Arvados Authors. All rights reserved.",
    "#",
    "# SPDX-License-Identifier: Apache-2.0", ""), fileConn)
    writeLines(unlist(arvadosClass), fileConn)
    close(fileConn)
    NULL
}

genAPIClassHeader <- function()
{
    c("Arvados <- R6::R6Class(",
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

genProjectMethods <- function()
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

genClassContent <- function(methodResources, resourceNames)
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

genAPIClassFooter <- function()
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
    c("headers <- list(Authorization = paste(\"Bearer\", private$token), ",
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

genMethodsDoc <- function(methodResources, resourceNames)
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
            getMethodDoc(methodName, methodMetaData)

        }, resource$methods, methodNames)

        unlist(unname(methodDoc))

    }, methodResources, resourceNames)))

    projectDoc <- genProjectMethodsDoc()

    c(methodsDoc, projectDoc)
}

genAPIClassDoc <- function(methodResources, resourceNames)
{
    c("#' Arvados",
      "#'",
      "#' Arvados class gives users ability to access Arvados REST API.",
      "#'" ,
      "#' @section Usage:",
      "#' \\preformatted{arv = Arvados$new(authToken = NULL, hostName = NULL, numRetries = 0)}",
      "#'",
      "#' @section Arguments:",
      "#' \\describe{",
      "#' \t\\item{authToken}{Authentification token. If not specified ARVADOS_API_TOKEN environment variable will be used.}",
      "#' \t\\item{hostName}{Host name. If not specified ARVADOS_API_HOST environment variable will be used.}",
      "#' \t\\item{numRetries}{Number which specifies how many times to retry failed service requests.}",
      "#' }",
      "#'",
      "#' @section Methods:",
      "#' \\describe{",
      getAPIClassMethodList(methodResources, resourceNames),
      "#' }",
      "#'",
      "#' @name Arvados",
      "#' @examples",
      "#' \\dontrun{",
      "#' arv <- Arvados$new(\"your Arvados token\", \"example.arvadosapi.com\")",
      "#'",
      "#' collection <- arv$collections.get(\"uuid\")",
      "#'",
      "#' collectionList <- arv$collections.list(list(list(\"name\", \"like\", \"Test%\")))",
      "#' collectionList <- listAll(arv$collections.list, list(list(\"name\", \"like\", \"Test%\")))",
      "#'",
      "#' deletedCollection <- arv$collections.delete(\"uuid\")",
      "#'",
      "#' updatedCollection <- arv$collections.update(list(name = \"New name\", description = \"New description\"),",
      "#'                                             \"uuid\")",
      "#'",
      "#' createdCollection <- arv$collections.create(list(name = \"Example\",",
      "#'                                                  description = \"This is a test collection\"))",
      "#' }",
      "NULL",
      "",
      "#' @export")
}

getAPIClassMethodList <- function(methodResources, resourceNames)
{
    methodList <- unlist(unname(Map(function(resource, resourceName)
    {
        methodNames <- names(resource$methods)
        paste0(resourceName,
               ".",
               methodNames[!(methodNames %in% c("index", "show", "destroy"))])

    }, methodResources, resourceNames)))

    hardcodedMethods <- c("projects.create", "projects.get",
                          "projects.list", "projects.update", "projects.delete")
    paste0("#' \t\\item{}{\\code{\\link{", sort(c(methodList, hardcodedMethods)), "}}}")
}

getMethodDoc <- function(methodName, methodMetaData)
{
    name        <- paste("#' @name", methodName)
    usage       <- getMethodUsage(methodName, methodMetaData)
    description <- paste("#'", methodName, "is a method defined in Arvados class.")
    params      <- getMethodDescription(methodMetaData)
    returnValue <- paste("#' @return", methodMetaData$response[["$ref"]], "object.")

    c(paste("#'", methodName),
      "#' ",
      description,
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
        paste("#' @param", argName, gsub("\n", "\n#' ", arg$description))
    })))

    c(requestDoc, argsDoc)
}

genProjectMethodsDoc <- function()
{
    #TODO: Manually update this documentation to reflect changes in discovery document.
    c("#' project.get",
    "#' ",
    "#' projects.get is equivalent to groups.get method.",
    "#' ",
    "#' @usage arv$projects.get(uuid)",
    "#' @param uuid The UUID of the Group in question.",
    "#' @return Group object.",
    "#' @name projects.get",
    "NULL",
    "",
    "#' project.create",
    "#' ",
    "#' projects.create wrapps groups.create method by setting group_class attribute to \"project\".",
    "#' ",
    "#' @usage arv$projects.create(group, ensure_unique_name = \"false\")",
    "#' @param group Group object.",
    "#' @param ensure_unique_name Adjust name to ensure uniqueness instead of returning an error on (owner_uuid, name) collision.",
    "#' @return Group object.",
    "#' @name projects.create",
    "NULL",
    "",
    "#' project.update",
    "#' ",
    "#' projects.update wrapps groups.update method by setting group_class attribute to \"project\".",
    "#' ",
    "#' @usage arv$projects.update(group, uuid)",
    "#' @param group Group object.",
    "#' @param uuid The UUID of the Group in question.",
    "#' @return Group object.",
    "#' @name projects.update",
    "NULL",
    "",
    "#' project.delete",
    "#' ",
    "#' projects.delete is equivalent to groups.delete method.",
    "#' ",
    "#' @usage arv$project.delete(uuid)",
    "#' @param uuid The UUID of the Group in question.",
    "#' @return Group object.",
    "#' @name projects.delete",
    "NULL",
    "",
    "#' project.list",
    "#' ",
    "#' projects.list wrapps groups.list method by setting group_class attribute to \"project\".",
    "#' ",
    "#' @usage arv$projects.list(filters = NULL,",
    "#' 	where = NULL, order = NULL, distinct = NULL,",
    "#' 	limit = \"100\", offset = \"0\", count = \"exact\",",
    "#' 	include_trash = NULL, uuid = NULL, recursive = NULL)",
    "#' @param filters ",
    "#' @param where ",
    "#' @param order ",
    "#' @param distinct ",
    "#' @param limit ",
    "#' @param offset ",
    "#' @param count ",
    "#' @param include_trash Include items whose is_trashed attribute is true.",
    "#' @param uuid ",
    "#' @param recursive Include contents from child groups recursively.",
    "#' @return Group object.",
    "#' @name projects.list",
    "NULL",
    "")
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
