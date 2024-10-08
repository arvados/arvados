# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

library(jsonlite)

getAPIDocument <- function(loc)
{
    if (length(grep("^[a-z]+://", loc)) > 0) {
        library(httr)
        serverResponse <- httr::RETRY("GET", url = loc)
        httr::content(serverResponse, as = "parsed", type = "application/json")
    } else {
        jsonlite::read_json(loc)
    }
}

#' generateAPI
#'
#' Autogenerate classes to interact with Arvados from the Arvados discovery document.
#'
#' @export
generateAPI <- function(discoveryDocument)
{
    methodResources <- discoveryDocument$resources
    resourceNames   <- names(methodResources)

    classDoc <- genAPIClassDoc(methodResources, resourceNames)
    arvadosAPIHeader <- genAPIClassHeader()
    arvadosClassMethods <- genClassContent(methodResources, resourceNames)
    arvadosProjectMethods <- genProjectMethods(methodResources)
    arvadosAPIFooter <- genAPIClassFooter()

    arvadosClass <- c(classDoc,
                      arvadosAPIHeader,
                      arvadosClassMethods,
                      arvadosProjectMethods,
                      arvadosAPIFooter)

    fileConn <- file("./R/Arvados.R", "w")
    writeLines(c(
    "# Copyright (C) The Arvados Authors. All rights reserved.",
    "#",
    "# SPDX-License-Identifier: Apache-2.0",
    "",
    "#' Arvados",
    "#'",
    "#' This class implements a full REST client to the Arvados API.",
    "#'"), fileConn)
    writeLines(unlist(arvadosClass), fileConn)
    close(fileConn)
    NULL
}

genAPIClassHeader <- function()
{
    c("#' @export",
      "Arvados <- R6::R6Class(",
      "",
      "\t\"Arvados\",",
      "",
      "\tpublic = list(",
      "",
      "\t\t#' @description Create a new Arvados API client.",
      "\t\t#' @param authToken Authentification token. If not specified ARVADOS_API_TOKEN environment variable will be used.",
      "\t\t#' @param hostName Host name. If not specified ARVADOS_API_HOST environment variable will be used.",
      "\t\t#' @param numRetries Number which specifies how many times to retry failed service requests.",
      "\t\t#' @return A new `Arvados` object.",
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

genProjectMethods <- function(methodResources)
{
    toCallArg <- function(arg) {
        callArg <- strsplit(arg, " *=")[[1]][1]
        paste(callArg, callArg, sep=" = ")
    }
    toCallArgs <- function(argList) {
        paste0(Map(toCallArg, argList), collapse=", ")
    }
    groupsMethods <- methodResources[["groups"]][["methods"]]
    getArgs <- getMethodArguments(groupsMethods[["get"]])
    createArgs <- getMethodArguments(groupsMethods[["create"]])
    updateArgs <- getMethodArguments(groupsMethods[["update"]])
    listArgs <- getMethodArguments(groupsMethods[["list"]])
    deleteArgs <- getMethodArguments(groupsMethods[["delete"]])

    c("\t\t#' @description An alias for `groups_get`.",
      getMethodParams(groupsMethods[["get"]]),
      "\t\t#' @return A Group object.",
      getMethodSignature("project_get", getArgs),
      "\t\t{",
      paste("\t\t\tself$groups_get(", toCallArgs(getArgs), ")", sep=""),
      "\t\t},",
      "",
      "\t\t#' @description A wrapper for `groups_create` that sets `group_class=\"project\"`.",
      getMethodParams(groupsMethods[["create"]]),
      "\t\t#' @return A Group object.",
      getMethodSignature("project_create", createArgs),
      "\t\t{",
      "\t\t\tgroup <- c(\"group_class\" = \"project\", group)",
      paste("\t\t\tself$groups_create(", toCallArgs(createArgs), ")", sep=""),
      "\t\t},",
      "",
      "\t\t#' @description A wrapper for `groups_update` that sets `group_class=\"project\"`.",
      getMethodParams(groupsMethods[["update"]]),
      "\t\t#' @return A Group object.",
      getMethodSignature("project_update", updateArgs),
      "\t\t{",
      "\t\t\tgroup <- c(\"group_class\" = \"project\", group)",
      paste("\t\t\tself$groups_update(", toCallArgs(updateArgs), ")", sep=""),
      "\t\t},",
      "",
      "\t\t#' @description A wrapper for `groups_list` that adds a filter for `group_class=\"project\"`.",
      getMethodParams(groupsMethods[["list"]]),
      "\t\t#' @return A GroupList object.",
      getMethodSignature("project_list", listArgs),
      "\t\t{",
      "\t\t\tfilters[[length(filters) + 1]] <- list(\"group_class\", \"=\", \"project\")",
      paste("\t\t\tself$groups_list(", toCallArgs(listArgs), ")", sep=""),
      "\t\t},",
      "",
      "\t\t#' @description An alias for `groups_delete`.",
      getMethodParams(groupsMethods[["delete"]]),
      "\t\t#' @return A Group object.",
      getMethodSignature("project_delete", deleteArgs),
      "\t\t{",
      paste("\t\t\tself$groups_delete(", toCallArgs(deleteArgs), ")", sep=""),
      "\t\t},",
      "",
      "\t\t#' @description Test whether or not a project exists.",
      getMethodParams(groupsMethods[["get"]]),
      getMethodSignature("project_exist", getArgs),
      "\t\t{",
      paste("\t\t\tresult <- try(self$groups_get(", toCallArgs(getArgs), "))", sep=""),
      "\t\t\tif(inherits(result, \"try-error\"))",
      "\t\t\t\texists <- FALSE",
      "\t\t\telse",
      "\t\t\t\texists <- result['group_class'] == \"project\"",
      "\t\t\tcat(format(exists))",
      "\t\t},",
      "",
      "\t\t#' @description A convenience wrapper for `project_update` to set project metadata properties.",
      "\t\t#' @param listProperties List of new properties.",
      "\t\t#' @param uuid UUID of the project to update.",
      "\t\t#' @return A Group object.",
      "\t\tproject_properties_set = function(listProperties, uuid)",
      "\t\t{",
      "\t\t\tself$project_update(list(\"properties\" = listProperties), uuid)",
      "\t\t},",
      "",
      "\t\t#' @description Get a project and update it with additional properties.",
      "\t\t#' @param properties List of new properties.",
      "\t\t#' @param uuid UUID of the project to update.",
      "\t\t#' @return A Group object.",
      "\t\tproject_properties_append = function(properties, uuid)",
      "\t\t{",
      "\t\t\tproj <- private$get_project_by_list(uuid, list('uuid', 'properties'))",
      "\t\t\tnewListOfProperties <- c(proj$properties, properties)",
      "\t\t\tuniqueProperties <- unique(unlist(newListOfProperties))",
      "\t\t\tnewProperties <- suppressWarnings(newListOfProperties[which(newListOfProperties == uniqueProperties)])",
      "\t\t\tself$project_properties_set(newProperties, proj$uuid)",
      "\t\t},",
      "",
      "\t\t#' @description Get properties of a project.",
      "\t\t#' @param uuid The UUID of the project to query.",
      "\t\tproject_properties_get = function(uuid)",
      "\t\t{",
      "\t\t\tprivate$get_project_by_list(uuid, list('uuid', 'properties'))$properties",
      "\t\t},",
      "",
      "\t\t#' @description Delete one property from a project by name.",
      "\t\t#' @param oneProp Name of the property to delete.",
      "\t\t#' @param uuid The UUID of the project to update.",
      "\t\t#' @return A Group object.",
      "\t\tproject_properties_delete = function(oneProp, uuid)",
      "\t\t{",
      "\t\t\tprojProp <- self$project_properties_get(uuid)",
      "\t\t\tprojProp[[oneProp]] <- NULL",
      "\t\t\tself$project_properties_set(projProp, uuid)",
      "\t\t},",
      "",
      "\t\t#' @description Convenience wrapper of `links_list` to create a permission link.",
      "\t\t#' @param type The type of permission: one of `'can_read'`, `'can_write'`, or `'can_manage'`.",
      "\t\t#' @param uuid The UUID of the object to grant permission to.",
      "\t\t#' @param user The UUID of the user or group who receives this permission.",
      "\t\t#' @return A Link object if one was updated, else NULL.",
      "\t\tproject_permission_give = function(type, uuid, user)",
      "\t\t{",
      "\t\t\tlink <- list(",
      "\t\t\t\t'link_class' = 'permission',",
      "\t\t\t\t'name' = type,",
      "\t\t\t\t'head_uuid' = uuid,",
      "\t\t\t\t'tail_uuid' = user)",
      "\t\t\tself$links_create(link)",
      "\t\t},",
      "",
      "\t\t#' @description Find an existing permission link and update its level.",
      "\t\t#' @param typeOld The type of permission to find: one of `'can_read'`, `'can_write'`, or `'can_manage'`.",
      "\t\t#' @param typeNew The type of permission to set: one of `'can_read'`, `'can_write'`, or `'can_manage'`.",
      "\t\t#' @param uuid The UUID of the object to grant permission to.",
      "\t\t#' @param user The UUID of the user or group who receives this permission.",
      "\t\t#' @return A Link object if one was updated, else NULL.",
      "\t\tproject_permission_update = function(typeOld, typeNew, uuid, user)",
      "\t\t{",
      "\t\t\tlinks <- self$links_list(filters = list(",
      "\t\t\t\t\tlist('link_class', '=', 'permission'),",
      "\t\t\t\t\tlist('name', '=', typeOld),",
      "\t\t\t\t\tlist('head_uuid', '=', uuid),",
      "\t\t\t\t\tlist('tail_uuid', '=', user)",
      "\t\t\t\t), select=list('uuid'), count = 'none')$items",
      "\t\t\tif (length(links) == 0) {",
      "\t\t\t\tcat(format('No permission granted'))",
      "\t\t\t} else {",
      "\t\t\t\tself$links_update(list('name' = typeNew), links[[1]]$uuid)",
      "\t\t\t}",
      "\t\t},",
      "",
      "\t\t#' @description Delete an existing permission link.",
      "\t\t#' @param type The type of permission to delete: one of `'can_read'`, `'can_write'`, or `'can_manage'`.",
      "\t\t#' @param uuid The UUID of the object to grant permission to.",
      "\t\t#' @param user The UUID of the user or group who receives this permission.",
      "\t\t#' @return A Link object if one was deleted, else NULL.",
      "\t\tproject_permission_delete = function(type, uuid, user)",
      "\t\t{",
      "\t\t\tlinks <- self$links_list(filters = list(",
      "\t\t\t\t\tlist('link_class', '=', 'permission'),",
      "\t\t\t\t\tlist('name', '=', type),",
      "\t\t\t\t\tlist('head_uuid', '=', uuid),",
      "\t\t\t\t\tlist('tail_uuid', '=', user)",
      "\t\t\t\t), select=list('uuid'), count = 'none')$items",
      "\t\t\tif (length(links) == 0) {",
      "\t\t\t\tcat(format('No permission granted'))",
      "\t\t\t} else {",
      "\t\t\t\tself$links_delete(links[[1]]$uuid)",
      "\t\t\t}",
      "\t\t},",
      "",
      "\t\t#' @description Check for an existing permission link.",
      "\t\t#' @param type The type of permission to check: one of `'can_read'`, `'can_write'`, `'can_manage'`, or `NULL` (the default).",
      "\t\t#' @param uuid The UUID of the object to check permission on.",
      "\t\t#' @param user The UUID of the user or group to check permission for.",
      "\t\t#' @return If `type` is `NULL`, the list of matching permission links.",
      "\t\t#' Otherwise, prints and invisibly returns the level of the found permission link.",
      "\t\tproject_permission_check = function(uuid, user, type = NULL)",
      "\t\t{",
      "\t\t\tfilters <- list(",
      "\t\t\t\tlist('link_class', '=', 'permission'),",
      "\t\t\t\tlist('head_uuid', '=', uuid),",
      "\t\t\t\tlist('tail_uuid', '=', user))",
      "\t\t\tif (!is.null(type)) {",
      "\t\t\t\tfilters <- c(filters, list(list('name', '=', type)))",
      "\t\t\t}",
      "\t\t\tlinks <- self$links_list(filters = filters, count='none')$items",
      "\t\t\tif (is.null(type)) {",
      "\t\t\t\tlinks",
      "\t\t\t} else {",
      "\t\t\t\tprint(links[[1]]$name)",
      "\t\t\t}",
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

            methodName <- paste0(resourceName, "_", methodName)
            unlist(c(
                   getMethodDoc(methodName, methodMetaData),
                   createMethod(methodName, methodMetaData)
            ))

        }, resource$methods, methodNames)

        unlist(unname(functions))

    }, methodResources, resourceNames)

    arvadosMethods
}

genAPIClassFooter <- function()
{
    c("\t\t#' @description Return the host name of this client's Arvados API server.",
      "\t\t#' @return Hostname string.",
      "\t\tgetHostName = function() private$host,",
      "",
      "\t\t#' @description Return the Arvados API token used by this client.",
      "\t\t#' @return API token string.",
      "\t\tgetToken = function() private$token,",
      "",
      "\t\t#' @description Set the RESTService object used by this client.",
      "\t\tsetRESTService = function(newREST) private$REST <- newREST,",
      "",
      "\t\t#' @description Return the RESTService object used by this client.",
      "\t\t#' @return RESTService object.",
      "\t\tgetRESTService = function() private$REST",
      "\t),",
      "",
      "\tprivate = list(",
      "\t\ttoken = NULL,",
      "\t\thost = NULL,",
      "\t\tREST = NULL,",
      "\t\tnumRetries = NULL,",
      "\t\tget_project_by_list = function(uuid, select = NULL)",
      "\t\t{",
      "\t\t\tself$groups_list(",
      "\t\t\t\tfilters = list(list('uuid', '=', uuid), list('group_class', '=', 'project')),",
      "\t\t\t\tselect = select,",
      "\t\t\t\tcount = 'none'",
      "\t\t\t)$items[[1]]",
      "\t\t}",
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

normalizeParamName <- function(name)
{
    # Downcase the first letter
    name <- sub("^(\\w)", "\\L\\1", name, perl=TRUE)
    # Convert snake_case to camelCase
    gsub("_(uuid\\b|id\\b|\\w)", "\\U\\1", name, perl=TRUE)
}

getMethodArguments <- function(methodMetaData)
{
    request <- methodMetaData$request
    requestArgs <- NULL

    if(!is.null(request))
    {
        resourceName <- normalizeParamName(request$properties[[1]][[1]])

        if(request$required)
            requestArgs <- resourceName
        else
            requestArgs <- paste(resourceName, "=", "NULL")
    }

    argNames <- names(methodMetaData$parameters)

    args <- sapply(argNames, function(argName)
    {
        arg <- methodMetaData$parameters[[argName]]
        argName <- normalizeParamName(argName)

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
    errorCheck       <- getErrorCheckingCode(methodMetaData)
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

    queryArgs <- sapply(queryArgs, function(arg) {
        arg <- normalizeParamName(arg)
        paste(arg, "=", arg)
    })
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

    resourceName <- normalizeParamName(request$properties[[1]][[1]])

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

getErrorCheckingCode <- function(methodMetaData)
{
    if ("ensure_unique_name" %in% names(methodMetaData$parameters)) {
        body <- c("\tif (identical(sub('Entity:.*', '', resource$errors), '//railsapi.internal/arvados/v1/collections: 422 Unprocessable ')) {",
                  "\t\tresource <- cat(format('An object with the given name already exists with this owner. If you want to update it use the update method instead'))",
                  "\t} else {",
                  "\t\tstop(resource$errors)",
                  "\t}")
    } else {
        body <- "\tstop(resource$errors)"
    }
    c("if(!is.null(resource$errors)) {", body, "}")
}

getReturnObject <- function()
{
    "resource"
}

genAPIClassDoc <- function(methodResources, resourceNames)
{
    c("#' @examples",
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
      "")
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
    description <- paste("\t\t#' @description", gsub("\n", "\n\t\t#' ", methodMetaData$description))
    params      <- getMethodParams(methodMetaData)
    returnValue <- paste("\t\t#' @return", methodMetaData$response[["$ref"]], "object.")

    c(description, params, returnValue)
}

getMethodParams <- function(methodMetaData)
{
    request <- methodMetaData$request
    requestDoc <- NULL

    if(!is.null(request))
    {
        requestDoc <- unname(unlist(sapply(request$properties, function(prop)
                             {
                                 className <- sapply(prop, function(ref) ref)
                                 objectName <- normalizeParamName(className)
                                 paste("\t\t#' @param", objectName, className, "object.")
                             })))
    }

    argNames <- names(methodMetaData$parameters)

    argsDoc <- unname(unlist(sapply(argNames, function(argName)
    {
        arg <- methodMetaData$parameters[[argName]]
        paste("\t\t#' @param",
              normalizeParamName(argName),
              gsub("\n", "\n\t\t#' ", arg$description)
        )
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

args <- commandArgs(TRUE)
if (length(args) == 0) {
   loc <- "arvados-v1-discovery.json"
} else {
   loc <- args[[1]]
}
discoveryDocument <- getAPIDocument(loc)
generateAPI(discoveryDocument)
