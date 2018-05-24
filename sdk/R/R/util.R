# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

#' listAll
#'
#' List all resources even if the number of items is greater than maximum API limit.
#'
#' @param fn Arvados method used to retrieve items from REST service.
#' @param ... Optional arguments which will be pased to fn .
#' @examples
#' \dontrun{
#' arv <- Arvados$new("your Arvados token", "example.arvadosapi.com")
#' cl <- listAll(arv$collections.list, filters = list(list("name", "like", "test%"))
#' }
#' @export 
listAll <- function(fn, ...)
{
    offset <- 0
    itemsAvailable <- .Machine$integer.max
    items <- c()

    while(length(items) < itemsAvailable)
    {
        serverResponse <- fn(offset = offset, ...)

        if(!is.null(serverResponse$errors))
            stop(serverResponse$errors)

        items          <- c(items, serverResponse$items)
        offset         <- length(items)
        itemsAvailable <- serverResponse$items_available
    }

    items
}


#NOTE: Package private functions

trimFromStart <- function(sample, trimCharacters)
{
    if(startsWith(sample, trimCharacters))
        sample <- substr(sample, nchar(trimCharacters) + 1, nchar(sample))

    sample
}

trimFromEnd <- function(sample, trimCharacters)
{
    if(endsWith(sample, trimCharacters))
        sample <- substr(sample, 0, nchar(sample) - nchar(trimCharacters))

    sample
}

RListToPythonList <- function(RList, separator = ", ")
{
    pythonArrayContent <- sapply(RList, function(elementInList)
    {
        if((is.vector(elementInList) || is.list(elementInList)) &&
            length(elementInList) > 1)
        {
            return(RListToPythonList(elementInList, separator))
        }
        else
        {
            return(paste0("\"", elementInList, "\""))
        }
    })

    pythonArray <- paste0("[", paste0(pythonArrayContent, collapse = separator), "]")
    pythonArray
}

appendToStartIfNotExist <- function(sample, characters)
{
    if(!startsWith(sample, characters))
        sample <- paste0(characters, sample)

    sample
}

splitToPathAndName = function(path)
{
    path <- appendToStartIfNotExist(path, "/")
    components <- unlist(stringr::str_split(path, "/"))
    nameAndPath <- list()
    nameAndPath$name <- components[length(components)]
    nameAndPath$path <- trimFromStart(paste0(components[-length(components)], collapse = "/"),
                                      "/")
    nameAndPath
}
