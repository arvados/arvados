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

RListToPythonList <- function(sample, separator = ", ")
{
    pythonArrayContent <- sapply(sample, function(sampleUnit)
    {
        if((is.vector(sampleUnit) || is.list(sampleUnit)) &&
            length(sampleUnit) > 1)
        {
            return(RListToPythonList(sampleUnit, separator))
        }
        else
        {
            return(paste0("\"", sampleUnit, "\""))
        }
    })

    return(paste0("[", paste0(pythonArrayContent, collapse = separator), "]"))
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
