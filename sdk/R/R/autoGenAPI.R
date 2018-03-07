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
    JSONDocument <- getAPIDocument()

    generateArvadosClasses(JSONDocument)
    generateArvadosAPIClass(JSONDocument)
}

#NOTE: Arvados class generation:

generateArvadosAPIClass <- function(discoveryDocument)
{
    classMetaData   <- discoveryDocument$schemas
    methodResources <- discoveryDocument$resources
    resourceNames   <- names(methodResources)

    doc <- generateMethodsDocumentation(methodResources, resourceNames)
    arvadosAPIHeader <- generateAPIClassHeader()
    arvadosClassMethods <- generateClassContent(methodResources, 
                                                resourceNames, classMetaData)
    arvadosAPIFooter <- generateAPIClassFooter()

    arvadosClass <- c(doc,
                      arvadosAPIHeader,
                      arvadosClassMethods,
                      arvadosAPIFooter)

    #TODO: Save to a file or load in memory?
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

generateClassContent <- function(methodResources, resourceNames, classMetaData)
{
    arvadosMethods <- Map(function(resource, resourceName)
    {
        methodNames <- names(resource$methods)

        functions <- Map(function(methodMetaData, methodName)
        {
            methodName <- paste0(resourceName, ".", methodName)
            createMethod(methodName, methodMetaData, classMetaData)

        }, resource$methods, methodNames)

        unlist(unname(functions))

    }, methodResources, resourceNames)

    arvadosMethods
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

createMethod <- function(name, methodMetaData, classMetaData)
{
    args      <- getMethodArguments(methodMetaData)
    signature <- getMethodSignature(name, args)
    body      <- getMethodBody(methodMetaData, classMetaData)

    c(signature,
      "\t\t{",
          body,
      "\t\t},\n")
}

#TODO: Make sure that arguments that are required always go first.
#      This is not the case if request$required is false.
getMethodArguments <- function(methodMetaData)
{
    request <- methodMetaData$request
    requestArgs <- NULL

    if(!is.null(request))
    {
        if(request$required)
            requestArgs <- names(request$properties)
        else
            requestArgs <- paste(names(request$properties), "=", "NULL")
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

getMethodBody <- function(methodMetaData, classMetaData)
{
    url              <- getRequestURL(methodMetaData)
    headers          <- getRequestHeaders()
    requestQueryList <- getRequestQueryList(methodMetaData)
    requestBody      <- getRequestBody(methodMetaData)
    request          <- getRequest(methodMetaData)
    response         <- getResponse(methodMetaData)
    errorCheck       <- getErrorCheckingCode()
    returnObject     <- getReturnObject(methodMetaData, classMetaData)
    returnStatement  <- getReturnObjectValidationCode()

    body <- c(url,
              headers,
              requestQueryList,
              requestBody, "",
              request, response, "",
              errorCheck, "",
              returnObject, "",
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
    args <- names(methodMetaData$parameters)

    if(length(args) == 0)
        return("queryArgs <- NULL")

    args <- sapply(args, function(arg) paste0(arg, " = ", arg))
    collapsedArgs <- paste0(args, collapse = ", ")

    if(nchar(collapsedArgs) > 40)
        return(formatArgs("queryArgs <- list(", "\t", args, ")", 40))
    else
        return(paste0("queryArgs <- list(", collapsedArgs, ")"))
}

getRequestBody <- function(methodMetaData)
{
    request <- methodMetaData$request

    if(is.null(request) || !request$required)
        return("body <- NULL")

    requestParameterName <- names(request$properties)[1]
    paste0("body <- ", requestParameterName, "$toJSON()")
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

getReturnObject <- function(methodMetaData, classMetaData)
{
    returnClass <- methodMetaData$response[["$ref"]]
    classArguments <- getReturnClassArguments(returnClass, classMetaData)

    if(returnClass == "Collection")
        return(c(formatArgs("result <- Collection$new(", "\t",
                            classArguments, ")", 40),
                 "",
                 "result$setRESTService(private$REST)"))

    formatArgs(paste0("result <- ", returnClass, "$new("),
               "\t", classArguments, ")", 40)
}

getReturnObjectValidationCode <- function()
{
    c("if(result$isEmpty())",
      "\tresource",
      "else",
      "\tresult")
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


#NOTE: Arvados class documentation:

generateMethodsDocumentation <- function(methodResources, resourceNames)
{
    methodsDoc <- unlist(unname(Map(function(resource, resourceName)
    {
        methodNames <- names(resource$methods)

        methodDoc <- Map(function(methodMetaData, methodName)
        {
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

#NOTE: API Classes generation:

generateArvadosClasses <- function(resources)
{
    classes <- sapply(resources$schemas, function(classSchema)
    {
        #NOTE: Collection is implemented manually.
        if(classSchema$id != "Collection")
            getArvadosClass(classSchema)

    }, USE.NAMES = TRUE)

    fileConn <- file("./R/ArvadosClasses.R", "w")
    writeLines(unlist(classes), fileConn)
    close(fileConn)
    NULL
}

getArvadosClass <- function(classSchema)
{
    name            <- classSchema$id
    fields          <- unique(names(classSchema$properties))
    constructorArgs <- paste(fields, "= NULL")
    documentation   <- getClassDocumentation(classSchema, constructorArgs)

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
                                         paste0("\"", fields, "\""), ")", 40)),
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
                     "\t\t},",
                     "",
                     "\t\tisEmpty = function() {",
                     "\t\t\tfields <- sapply(private$classFields,",
                     "\t\t\t                 function(field) self[[field]])",
                     "",
              paste0("\t\t\tif(any(sapply(fields, function(field) !is.null(field)",
                     " && field != \"\")))"),
                     "\t\t\t\tFALSE",
                     "\t\t\telse",
                     "\t\t\t\tTRUE",
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

#NOTE: API Classes documentation:

getClassDocumentation <- function(classSchema, constructorArgs)
{
    name                     <- classSchema$id
    description              <- classSchema$description
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

#NOTE: Utility functions:

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
