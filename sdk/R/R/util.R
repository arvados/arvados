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
