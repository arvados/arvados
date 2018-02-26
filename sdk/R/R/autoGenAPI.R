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
    fileConn <- file("./R/Arvados.R", "w")
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
            requestArgument <- names(request$properties)
        else
            requestArgument <- paste(names(request$properties), "=", "NULL")

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

    paste0(c(requestArgument, args))
}

getFunctionBody <- function(functionMetaData, classMetaData)
{
    url  <- getRequestURL(functionMetaData)
    headers <- getRequestHeaders()
    requestQueryList <- getRequestQueryList(functionMetaData)
    requestBody <- getRequestBody(functionMetaData)
    request <- getRequest(functionMetaData)
    response <- getResponse(functionMetaData)
    errorCheck <- getErrorCheckingCode()
    returnObject <- getReturnObject(functionMetaData, classMetaData)

    body <- c(url,
              headers,
              requestQueryList,
              requestBody, "",
              request, response, "",
              errorCheck, "",
              returnObject)

    paste0("\t\t\t", body)
}

getErrorCheckingCode <- function()
{
    c("if(!is.null(resource$errors))", "\tstop(resource$errors)")
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
    c("headers <- list(Authorization = paste(\"OAuth2\", private$token), ",
      "                \"Content-Type\" = \"application/json\")")
}

getReturnObject <- function(functionMetaData, classMetaData)
{
    returnClass <- functionMetaData$response[["$ref"]]
    classArguments <- getReturnClassArguments(returnClass, classMetaData)


    if(returnClass == "Collection")
        return(c(formatArgs("collection <- Collection$new(", "\t",
                            classArguments, ")", 40),
                 "",
                 "collection$setRESTService(private$REST)",
                 "collection"))

    formatArgs(paste0(returnClass, "$new("), "\t", classArguments, ")", 40)
}

getReturnClassArguments <- function(className, classMetaData)
{
    classArguments <- unique(names(classMetaData[[className]]$properties))

    arguments <- sapply(classArguments, function(arg)
    {
        paste0(arg, " = resource$", arg)
    })

    arguments
}

getRequest <- function(functionMetaData)
{
    method <- functionMetaData$httpMethod
    c(paste0("response <- private$REST$http$exec(\"", method, "\", url, headers, body,"),
      "                                   queryArgs, private$numRetries)")
}

getResponse <- function(functionMetaData)
{
    "resource <- private$REST$httpParser$parseJSONResponse(response)"
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
    args <- names(functionMetaData$parameters)

    if(length(args) == 0)
        return("queryArgs <- NULL")

    args <- sapply(args, function(arg) paste0(arg, " = ", arg))
    collapsedArgs <- paste0(args, collapse = ", ")

    if(nchar(collapsedArgs) > 40)
        return(formatArgs("queryArgs <- list(", "\t", args, ")", 40))
    else
        return(paste0("queryArgs <- list(", collapsedArgs, ")"))
}

createFunction <- function(functionName, functionMetaData, classMetaData)
{
    args <- getFunctionArguments(functionMetaData)
    body <- getFunctionBody(functionMetaData, classMetaData)
    funSignature <- getFunSignature(functionName, args)

    functionString <- c(funSignature,
                        "\t\t{",
                            body,
                        "\t\t},\n")

    functionString
}

getFunSignature <- function(funName, args)
{
    collapsedArgs <- paste0(args, collapse = ", ")

    if(nchar(collapsedArgs) > 40)
    {
        return(paste0("\t\t",
                      formatArgs(paste(funName, "= function("),
                                 "\t", args, ")", 40)))
    }
    else
    {
        return(paste0("\t\t", funName, " = function(", collapsedArgs, ")"))
    }
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

generateAPIClassFooter <- function()
{
    c("\t\tgetHostName = function() private$host,",
      "\t\tgetToken = function() private$token,",
      "\t\tsetRESTService = function(newREST) private$REST <- newREST",
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

generateArvadosClasses <- function(resources)
{
    classes <- sapply(resources$schemas, function(classSchema)
    {
        #NOTE: Collection is implemented manually.
        if(classSchema$id != "Collection")
            getArvadosClass(classSchema)

    }, USE.NAMES = TRUE)

    unlist(unname(classes))

    fileConn <- file("./R/ArvadosClasses.R", "w")
    writeLines(unlist(classes), fileConn)
    close(fileConn)
    NULL
}

getArvadosClass <- function(classSchema)
{
    name   <- classSchema$id
    fields <- unique(names(classSchema$properties))
    constructorArgs <- paste(fields, "= NULL")
    documentation <- getClassDocumentation(classSchema, constructorArgs)

    classString <- c(documentation,
              paste0(name, " <- R6::R6Class("),
                     "",
              paste0("\t\"", name, "\","),
                     "",
                     "\tpublic = list(",
              paste0("\t\t", fields, " = NULL,"),
                     "",
              paste0("\t\t", formatArgs("initialize = function(", "\t\t",
                                        constructorArgs, ")", 40)),
                     "\t\t{",
              paste0("\t\t\tself$", fields, " <- ", fields),
                     "\t\t\t",
              paste0("\t\t\t", formatArgs("private$classFields <- c(", "\t",
                                         fields, ")", 40)),
                     "\t\t},",
                     "",
                     "\t\ttoJSON = function() {",
                     "\t\t\tfields <- sapply(private$classFields, function(field)",
                     "\t\t\t{",
                     "\t\t\t\tself[[field]]",
                     "\t\t\t}, USE.NAMES = TRUE)",
                     "\t\t\t",
              paste0("\t\t\tjsonlite::toJSON(list(\"", tolower(name), "\" = 
                     Filter(Negate(is.null), fields)), auto_unbox = TRUE)"),
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

getClassDocumentation <- function(classSchema, constructorArgs)
{
    name <- classSchema$id
    description <- classSchema$description
    nameLowercaseFirstLetter <- paste0(tolower(substr(name, 1, 1)),
                                       substr(name, 2, nchar(name)))
    c(paste0("#' ", name),
             "#' ",
      paste0("#' ", description),
             "#' ",
             "#' @section Usage:",
             formatArgs(paste0("#' \\preformatted{",
                               nameLowercaseFirstLetter, " -> ", name, "$new("),
                        "#' \t", constructorArgs, ")", 50),

             "#' }",
             "#' ",
      paste0("#' @section Arguments:"),
             "#'   \\describe{",
      paste0("#'     ", getClassArgumentDescription(classSchema)),
             "#'   }",
             "#' ",
      paste0("#' @name ", name),
             "NULL",
             "",
             "#' @export")
}

getClassArgumentDescription <- function(classSchema)
{
    argDoc <- sapply(classSchema$properties, function(arg)
    {    
        paste0("{", arg$description, "}")
    }, USE.NAMES = TRUE)

    paste0("\\item{", names(classSchema$properties), "}", argDoc)
}

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
