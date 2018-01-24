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
