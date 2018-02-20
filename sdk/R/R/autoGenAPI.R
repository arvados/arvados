#TODO: Some methods do the same thing like collecion.index and collection.list.
#      Make one implementation of the method and make other reference to it.

getAPIDocument <- function(){
    url <- "https://4xphq.arvadosapi.com/discovery/v1/apis/arvados/v1/rest"
    serverResponse <- httr::RETRY("GET", url = url)

    httr::content(serverResponse, as = "parsed", type = "application/json")
}

#' @export
generateAPI <- function()
{
    JSONDocument <- getAPIDocument()

    generateArvadosClasses(JSONDocument)
    generateArvadosAPIClass(JSONDocument)
}

generateArvadosAPIClass <- function(discoveryDocument)
{
    classMetaData     <- discoveryDocument$schemas
    functionResources <- discoveryDocument$resources
    resourceNames     <- names(functionResources)

    arvadosAPIHeader <- generateAPIClassHeader()
    arvadosAPIFooter <- generateAPIClassFooter()

    arvadosMethods <- Map(function(resource, resourceName)
    {
        methodNames <- names(resource$methods)

        functions <- Map(function(methodMetaData, methodName)
        {
            methodName <- paste0(resourceName, ".", methodName)
            createFunction(methodName, methodMetaData, classMetaData)

        }, resource$methods, methodNames)

        unlist(unname(functions))

    }, functionResources, resourceNames)

    arvadosClass <- c(arvadosAPIHeader, arvadosMethods, arvadosAPIFooter)

    #TODO: Save to a file or load in memory?
    fileConn <- file("ArvadosAPI.R", "w")
    writeLines(unlist(arvadosClass), fileConn)
    close(fileConn)
    NULL
}

getFunctionName <- function(functionMetaData)
{
    stringr::str_replace(functionMetaData$id, "arvados.", "")
}

#TODO: Make sure that arguments that are required always go first.
#      This is not the case if request$required is false.
getFunctionArguments <- function(functionMetaData)
{
    request <- functionMetaData$request
    requestArgument <- NULL

    if(!is.null(request))
        if(request$required)
            requestArgument <- names(request$properties)[1]
        else
            requestArgument <- paste(names(request$properties)[1], "=", "NULL")

    argNames <- names(functionMetaData$parameters)

    args <- sapply(argNames, function(argName)
    {
        arg <- functionMetaData$parameters[[argName]]

        if(!arg$required)
        {
            if(!is.null(arg$default))
                return(paste0(argName, " = ", "\"", arg$default, "\""))
            else
                return(paste(argName, "=", "NULL"))
        }

        argName
    })

    paste0(c(requestArgument, args), collapse = ", ")
}

getFunctionBody <- function(functionMetaData, classMetaData)
{
    url  <- getRequestURL(functionMetaData)
    headers <- getRequestHeaders()
    requestQueryList <- getRequestQueryList(functionMetaData)
    requestQueryList <- getRequestQueryList(functionMetaData)
    requestBody <- getRequestBody(functionMetaData)
    request <- getRequest(functionMetaData)
    response <- getResponse(functionMetaData)
    returnObject <- getReturnObject(functionMetaData, classMetaData)

    body <- c(url, headers, requestQueryList, requestBody, request, response, returnObject)
    paste0("\t\t\t", body)
}

getRequestBody <- function(functionMetaData)
{
    request <- functionMetaData$request

    if(is.null(request) || !request$required)
        return("body <- NULL")

    requestParameterName <- names(request$properties)[1]
    paste0("body <- ", requestParameterName, "$toJSON()")
}

getRequestHeaders <- function()
{
    paste0("headers <- list(Authorization = paste(\"OAuth2\", private$token),",
                            "\"Content-Type\" = \"application/json\")")
}

getReturnObject <- function(functionMetaData, classMetaData)
{
    returnClass <- functionMetaData$response[["$ref"]]
    classArguments <- getReturnClassArguments(returnClass, classMetaData)

    c(paste0(returnClass, "$new(", classArguments, ")"))
}

getReturnClassArguments <- function(className, classMetaData)
{
    classArguments <- unique(names(classMetaData[[className]]$properties))

    arguments <- sapply(classArguments, function(arg)
    {
        paste0(arg, " = resource$", arg)
    })

    paste0(arguments, collapse = ", ")
}

getRequest <- function(functionMetaData)
{
    method <- functionMetaData$httpMethod
    paste0("response <- private$http$exec(\"", method, "\", url, headers, body, queryArgs)")
}

getResponse <- function(functionMetaData)
{
    "resource <- private$httpParser$parseJSONResponse(response)"
}

getRequestURL <- function(functionMetaData)
{
    endPoint <- functionMetaData$path
    endPoint <- stringr::str_replace_all(endPoint, "\\{", "${")
    url <- c(paste0("endPoint <- stringr::str_interp(\"", endPoint, "\")"),
             paste0("url <- paste0(private$host, endPoint)"))
    url
}

getRequestQueryList <- function(functionMetaData)
{
    argNames <- names(functionMetaData$parameters)

    if(length(argNames) == 0)
        return("queryArgs <- NULL")

    queryListContent <- sapply(argNames, function(arg) paste0(arg, " = ", arg))

    paste0("queryArgs <- list(", paste0(queryListContent, collapse = ', ') , ")")
}

createFunction <- function(functionName, functionMetaData, classMetaData)
{
    args <- getFunctionArguments(functionMetaData)
    aditionalArgs <- 
    body <- getFunctionBody(functionMetaData, classMetaData)

    functionString <- c(paste0("\t\t", functionName, " = function(", args, ")"),
                       "\t\t{",
                           body,
                       "\t\t},\n")

    functionString
}

generateAPIClassHeader <- function()
{
    c("#' @export",
      "ArvadosAPI <- R6::R6Class(",
      "",
      "\t\"ArvadosAPI\",",
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
      "\t\t\tprivate$rawHost <- Sys.getenv(\"ARVADOS_API_HOST\")",
      "\t\t\tprivate$host <- paste0(\"https://\", private$rawHost, \"/arvados/v1/\")",
      "\t\t\tprivate$token <- Sys.getenv(\"ARVADOS_API_TOKEN\")",
      "\t\t\tprivate$numRetries  <- numRetries",
      "\t\t\tprivate$http  <- ArvadosR:::HttpRequest$new()",
      "\t\t\tprivate$httpParser  <- ArvadosR:::HttpParser$new()",
      "",
      "\t\t\tif(private$rawHost == \"\" | private$token == \"\")",
      "\t\t\t\tstop(paste(\"Please provide host name and authentification token\",",
      "\t\t\t\t\t\t   \"or set ARVADOS_API_HOST and ARVADOS_API_TOKEN\",",
      "\t\t\t\t\t\t   \"environment variables.\"))",
      "\t\t},\n")
}

generateAPIClassFooter <- function()
{
    c("\t\tgetHostName = function() private$host,",
      "\t\tgetToken = function() private$token",
      "\t),",
      "",
      "\tprivate = list(",
      "",
      "\t\ttoken = NULL,",
      "\t\trawHost = NULL,",
      "\t\thost = NULL,",
      "\t\thttp = NULL,",
      "\t\thttpParser = NULL,",
      "\t\tnumRetries = NULL",
      "\t),",
      "",
      "\tcloneable = FALSE",
      ")")
}

generateArvadosClasses <- function(resources)
{
    classes <- sapply(resources$schemas, function(classSchema)
    {
        getArvadosClass(classSchema)

    }, USE.NAMES = TRUE)

    unlist(unname(classes))

    fileConn <- file("ArvadosClasses.R", "w")
    writeLines(unlist(classes), fileConn)
    close(fileConn)
    NULL
}

getArvadosClass <- function(classSchema)
{
    name   <- classSchema$id
    fields <- unique(names(classSchema$properties))
    fieldsList <- paste0("c(", paste0("\"", fields, "\"", collapse = ", "), ")")
    constructorArgs <- paste0(fields, " = NULL", collapse = ", ")

    classString <- c(paste0(name, " <- R6::R6Class("),
                     "",
              paste0("\t\"", name, "\","),
                     "",
                     "\tpublic = list(",
              paste0("\t\t", fields, " = NULL,"),
                     "",
              paste0("\t\tinitialize = function(", constructorArgs, ") {"),
              paste0("\t\t\tself$", fields, " <- ", fields),
                     "\t\t\t",
              paste0("\t\t\tprivate$classFields <- ", fieldsList),
                     "\t\t},",
                     "",
                     "\t\ttoJSON = function() {",
                     "\t\t\tfields <- sapply(private$classFields, function(field)",
                     "\t\t\t{",
                     "\t\t\t\tself[[field]]",
                     "\t\t\t}, USE.NAMES = TRUE)",
                     "\t\t\t",
              paste0("\t\t\tjsonlite::toJSON(list(\"", tolower(name), "\" = Filter(Negate(is.null), fields)), auto_unbox = TRUE)"),
                     "\t\t}",
                     "\t),",
                     "",
                     "\tprivate = list(",
                     "\t\tclassFields = NULL",
                     "\t),",
                     "",
                     "\tcloneable = FALSE",
                     ")",
                     "")
}
