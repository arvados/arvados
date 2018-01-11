#' ArvadosFile Object
#'
#' Update description
#'
#' @export ArvadosFile
ArvadosFile <- R6::R6Class(

    "ArvadosFile",

    public = list(

        initialize = function(name)
        {
            private$name             <- name
            private$http             <- HttpRequest$new()
            private$httpParser       <- HttpParser$new()
        },

        getName = function() private$name,

        getFileListing = function(fullpath = TRUE)
        {
            self$getName()
        },

        getSizeInBytes = function()
        {
            collectionURL <- URLencode(paste0(private$collection$api$getWebDavHostName(),
                                              "c=", private$collection$uuid))
            fileURL <- paste0(collectionURL, "/", self$getRelativePath());

            headers = list("Authorization" = paste("OAuth2", private$collection$api$getToken()))

            propfindResponse <- private$http$PROPFIND(fileURL, headers)

            sizes <- private$httpParser$extractFileSizeFromWebDAVResponse(propfindResponse, collectionURL)
            as.numeric(sizes)
        },

        get = function(fileLikeObjectName)
        {
            return(NULL)
        },

        getFirst = function()
        {
            return(NULL)
        },

        getCollection = function() private$collection,

        setCollection = function(collection)
        {
            private$collection <- collection
        },

        getRelativePath = function()
        {
            relativePath <- c(private$name)
            parent <- private$parent

            while(!is.null(parent))
            {
                relativePath <- c(parent$getName(), relativePath)
                parent <- parent$getParent()
            }

            relativePath <- relativePath[relativePath != ""]
            paste0(relativePath, collapse = "/")
        },

        getParent = function() private$parent,

        setParent = function(newParent) private$parent <- newParent,

        read = function(contentType = "raw", offset = 0, length = 0)
        {
            if(is.null(private$collection))
                stop("ArvadosFile doesn't belong to any collection.")

            if(offset < 0 || length < 0)
                stop("Offset and length must be positive values.")

            if(!(contentType %in% private$http$validContentTypes))
                stop("Invalid contentType. Please use text or raw.")

            range = paste0("bytes=", offset, "-")

            if(length > 0)
                range = paste0(range, offset + length - 1)

            fileURL = paste0(private$collection$api$getWebDavHostName(),
                             "c=", private$collection$uuid, "/", self$getRelativePath());

            if(offset == 0 && length == 0)
            {
                headers <- list(Authorization = paste("OAuth2",
                                                      private$collection$api$getToken()))
            }
            else
            {
                headers <- list(Authorization = paste("OAuth2", private$collection$api$getToken()),
                                Range = range)
            }

            serverResponse <- private$http$GET(fileURL, headers)

            if(serverResponse$status_code < 200 || serverResponse$status_code >= 300)
                stop(paste("Server code:", serverResponse$status_code))

            parsedServerResponse <- httr::content(serverResponse, contentType)
            parsedServerResponse
        },

	connection = function(rw)
	{
	  if (rw == "r") {
	    return(textConnection(self$read("text")))
	  } else if (rw == "w") {
	    private$buffer <- textConnection(NULL, "w")
	    return(private$buffer)
	  }
	},

	flush = function() {
	  v <- textConnectionValue(private$buffer)
	  close(private$buffer)
	  self$write(paste(v, collapse='\n'))
	},

        write = function(content, contentType = "text/html")
        {
            if(is.null(private$collection))
                stop("ArvadosFile doesn't belong to any collection.")

            fileURL = paste0(private$collection$api$getWebDavHostName(),
                             "c=", private$collection$uuid, "/", self$getRelativePath());
            headers <- list(Authorization = paste("OAuth2", private$collection$api$getToken()),
                            "Content-Type" = contentType)
            body <- content

            serverResponse <- private$http$PUT(fileURL, headers, body)

            if(serverResponse$status_code < 200 || serverResponse$status_code >= 300)
                stop(paste("Server code:", serverResponse$status_code))

            parsedServerResponse <- httr::content(serverResponse, "text")
            parsedServerResponse
        },

        move = function(newLocation)
        {
            #todo test if file can be moved

            if(is.null(private$collection))
                stop("ArvadosFile doesn't belong to any collection.")

            if(endsWith(newLocation, paste0(private$name, "/")))
            {
                newLocation <- substr(newLocation, 0,
                                      nchar(newLocation)
                                      - nchar(paste0(private$name, "/")))
            }
            else if(endsWith(newLocation, private$name))
            {
                newLocation <- substr(newLocation, 0,
                                      nchar(newLocation) - nchar(private$name))
            }
            else
            {
                stop("Destination path is not valid.")
            }

            newParent <- private$collection$get(newLocation)

            if(is.null(newParent))
            {
                stop("Unable to get destination subcollection.")
            }

            childWithSameName <- newParent$get(private$name)

            if(!is.null(childWithSameName))
                stop("Destination already contains file with same name.")

            status <- private$collection$moveOnREST(self$getRelativePath(),
                                                    paste0(newParent$getRelativePath(),
                                                           "/", self$getName()))

            #Note: We temporary set parents collection to NULL. This will ensure that
            #      add method doesn't post file on REST server.
            parentsCollection <- newParent$getCollection()
            newParent$setCollection(NULL, setRecursively = FALSE)

            newParent$add(self)

            newParent$setCollection(parentsCollection, setRecursively = FALSE)

            private$parent <- newParent

            "Content moved successfully."
        }
    ),

    private = list(

        name       = NULL,
        size       = NULL,
        parent     = NULL,
        collection = NULL,
        http       = NULL,
        httpParser = NULL,
        buffer     = NULL
    ),

    cloneable = FALSE
)
