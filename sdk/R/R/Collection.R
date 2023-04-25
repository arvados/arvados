# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

#' R6 Class Representing Arvados Collection
#'
#' @description
#' Collection class provides interface for working with Arvados collections,
#' for exaplme actions like creating, updating, moving or removing are possible.
#'
#' @seealso
#' \code{\link{https://github.com/arvados/arvados/tree/main/sdk/R}}
#'
#' @export

Collection <- R6::R6Class(

    "Collection",

    public = list(

        #' @field uuid Autentic for Collection UUID.
        uuid = NULL,

        #' @description
        #' Initialize new enviroment.
        #' @param api Arvados enviroment.
        #' @param uuid The UUID Autentic for Collection UUID.
        #' @return A new `Collection` object.
        #' @examples
        #' collection <- Collection$new(arv, CollectionUUID)
        initialize = function(api, uuid)
        {
            private$REST <- api$getRESTService()
            self$uuid <- uuid
        },

        #' @description
        #' Adds ArvadosFile or Subcollection specified by content to the collection. Used only with ArvadosFile or Subcollection.
        #' @param content Content to be added.
        #' @param relativePath Path to add content.
        add = function(content, relativePath = "")
        {
            if(is.null(private$tree))
                private$generateCollectionTreeStructure()

            if(relativePath == ""  ||
               relativePath == "." ||
               relativePath == "./")
            {
                subcollection <- private$tree$getTree()
            }
            else
            {
                relativePath <- trimFromEnd(relativePath, "/")
                subcollection <- self$get(relativePath)
            }

            if(is.null(subcollection))
                stop(paste("Subcollection", relativePath, "doesn't exist."))

            if("ArvadosFile"   %in% class(content) ||
               "Subcollection" %in% class(content))
            {
                if(!is.null(content$getCollection()))
                    stop("Content already belongs to a collection.")

                if(content$getName() == "")
                    stop("Content has invalid name.")

                subcollection$add(content)
                content
            }
            else
            {
                stop(paste0("Expected AravodsFile or Subcollection object, got ",
                            paste0("(", paste0(class(content), collapse = ", "), ")"),
                            "."))
            }
        },

        #' @description
        #' Read file content.
        #' @param file Name of the file.
        #' @param col Collection from which the file is read.
        #' @param sep  Separator used in reading tsv, csv file format.
        #' @param istable Used in reading txt file to check if the file is table or not.
        #' @param fileclass Used in reading fasta file to set file class.
        #' @param Ncol Used in reading binary file to set numbers of columns in data.frame.
        #' @param Nrow Used in reading binary file to set numbers of rows in data.frame size.
        #' @examples
        #' collection <- Collection$new(arv, collectionUUID)
        #' readFile <- collection$readArvFile(arvadosFile, istable = 'yes')                    # table
        #' readFile <- collection$readArvFile(arvadosFile, istable = 'no')                     # text
        #' readFile <- collection$readArvFile(arvadosFile)                                     # xlsx, csv, tsv, rds, rdata
        #' readFile <- collection$readArvFile(arvadosFile, fileclass = 'fasta')                # fasta
        #' readFile <- collection$readArvFile(arvadosFile, Ncol= 4, Nrow = 32)                 # binary, only numbers
        #' readFile <- collection$readArvFile(arvadosFile, Ncol = 5, Nrow = 150, istable = "factor") # binary with factor or text
        readArvFile = function(file, con, sep = ',', istable = NULL, fileclass = "SeqFastadna", Ncol = NULL, Nrow = NULL, wantedFunction = NULL)
        {
            arvFile <- self$get(file)
            FileName <- arvFile$getName()
            FileName <- tolower(FileName)
            FileFormat <- gsub(".*\\.", "", FileName)

            # set enviroment
            ARVADOS_API_TOKEN <- Sys.getenv("ARVADOS_API_TOKEN")
            ARVADOS_API_HOST <- Sys.getenv("ARVADOS_API_HOST")
            my_collection <- self$uuid
            key <- gsub("/", "_", ARVADOS_API_TOKEN)

            Sys.setenv(
                "AWS_ACCESS_KEY_ID" = key,
                "AWS_SECRET_ACCESS_KEY" = key,
                "AWS_DEFAULT_REGION" = "collections",
                "AWS_S3_ENDPOINT" = gsub("api[.]", "", ARVADOS_API_HOST))

            if (FileFormat == "txt") {
                if (is.null(istable)){
                    stop(paste('You need to paste whether it is a text or table file'))
                } else if (istable == 'no') {
                    fileContent <- arvFile$read("text") # used to read
                    fileContent <- gsub("[\r\n]", " ", fileContent)
                } else if (istable == 'yes') {
                    arvConnection <- arvFile$connection("r") # used to make possible use different function later
                    fileContent <- read.table(arvConnection)
                }
            }
            else if (FileFormat  == "xlsx") {
                fileContent <- aws.s3::s3read_using(FUN = openxlsx::read.xlsx, object = file, bucket = my_collection)
            }
            else if (FileFormat == "csv" || FileFormat == "tsv") {
                arvConnection <- arvFile$connection("r")
                if (FileFormat == "tsv"){
                    mytable <- read.table(arvConnection, sep = '\t')
                } else if (FileFormat == "csv" & sep == '\t') {
                    mytable <- read.table(arvConnection, sep = '\t')
                } else if (FileFormat == "csv") {
                    mytable <- read.table(arvConnection, sep = ',')
                } else {
                    stop(paste('File format not supported, use arvadosFile$connection() and customise it'))
                }
            }
            else if (FileFormat == "fasta") {
                fastafile <- aws.s3::s3read_using(FUN = seqinr::read.fasta, as.string = TRUE, object = file, bucket = my_collection)
            }
            else if (FileFormat == "dat" || FileFormat == "bin") {
                fileContent <- gzcon(arvFile$connection("rb"))

                # function to precess data to binary format
                read_bin.file <- function(fileContent) {
                    # read binfile
                    column.names <- readBin(fileContent, character(), n = Ncol)
                    bindata <- readBin(fileContent, numeric(), Nrow*Ncol+Ncol)
                    # check
                    res <- which(bindata < 0.0000001)
                    if (is.list(res)) {
                        bindata <- bindata[-res]
                    } else {
                        bindata <- bindata
                    }
                    # make a dataframe
                    data <- data.frame(matrix(data = NA, nrow = Nrow, ncol = Ncol))
                    for (i in 1:Ncol) {
                        data[,i] <- bindata[(1+Nrow*(i-1)):(Nrow*i)]
                    }
                    colnames(data) = column.names

                    len <- which(is.na(data[,Ncol])) # error if sth went wrong
                    if (length(len) == 0) {
                        data
                    } else {
                        stop(paste("there is a factor or text in the table, customize the function by typing more arguments"))
                    }
                }
                if (is.null(Nrow) | is.null(Ncol)){
                    stop(paste('You need to specify numbers of columns and rows'))
                }
                if (is.null(istable)) {
                    fileContent <- read_bin.file(fileContent) # call a function
                } else if (istable == "factor") { # if there is a table with col name
                    fileContent <- read_bin.file(fileContent)
                }
            }
            else if (FileFormat == "rds" || FileFormat == "rdata") {
                arvConnection <- arvFile$connection("rb")
                mytable <- readRDS(gzcon(arvConnection))
            }
            else {
                stop(parse(('File format not supported, use arvadosFile$connection() and customise it')))
            }
        },

        #' @description
        #' Write file content
        #' @param name Name of the file.
        #' @param file File to be saved.
        #' @param istable Used in writing txt file to check if the file is table or not.
        #' @examples
        #' collection <- Collection$new(arv, collectionUUID)
        #' writeFile <- collection$writeFile(name = "myoutput.csv", file = file, fileFormat = "csv", istable = NULL, collectionUUID = collectionUUID)             # csv
        #' writeFile <- collection$writeFile(name = "myoutput.tsv", file = file, fileFormat = "tsv", istable = NULL, collectionUUID = collectionUUID)             # tsv
        #' writeFile <- collection$writeFile(name = "myoutput.fasta", file = file, fileFormat = "fasta", istable = NULL, collectionUUID = collectionUUID)         # fasta
        #' writeFile <- collection$writeFile(name = "myoutputtable.txt", file = file, fileFormat = "txt", istable = "yes", collectionUUID = collectionUUID)       # txt table
        #' writeFile <- collection$writeFile(name = "myoutputtext.txt", file = file, fileFormat = "txt", istable = "no", collectionUUID = collectionUUID)         # txt text
        #' writeFile <- collection$writeFile(name = "myoutputbinary.dat", file = file, fileFormat = "dat", collectionUUID = collectionUUID)                       # binary
        #' writeFile <- collection$writeFile(name = "myoutputxlsx.xlsx", file = file, fileFormat = "xlsx", collectionUUID = collectionUUID)                       # xlsx
        writeFile = function(name, file, collectionUUID, fileFormat, istable = NULL, seqName = NULL) {

            # set enviroment
            ARVADOS_API_TOKEN <- Sys.getenv("ARVADOS_API_TOKEN")
            ARVADOS_API_HOST <- Sys.getenv("ARVADOS_API_HOST")
            my_collection <- self$uuid
            key <- gsub("/", "_", ARVADOS_API_TOKEN)

            Sys.setenv(
                "AWS_ACCESS_KEY_ID" = key,
                "AWS_SECRET_ACCESS_KEY" = key,
                "AWS_DEFAULT_REGION" = "collections",
                "AWS_S3_ENDPOINT" = gsub("api[.]", "", ARVADOS_API_HOST))

            # save file
            if (fileFormat == "txt") {
                if (istable == "yes") {
                    aws.s3::s3write_using(file, FUN = write.table, object = name, bucket = collectionUUID)
                } else if (istable == "no") {
                    aws.s3::s3write_using(file, FUN = writeChar, object = name, bucket = collectionUUID)
                } else {
                    stop(paste("Specify parametr istable"))
                }
            } else if (fileFormat == "csv") {
                aws.s3::s3write_using(file, FUN = write.csv, object = name, bucket = collectionUUID)
            } else if (fileFormat == "tsv") {
                aws.s3::s3write_using(file, FUN = write.table, row.names = FALSE, sep = "\t", object = name, bucket = collectionUUID)
            } else if (fileFormat == "fasta") {
                aws.s3::s3write_using(file, FUN = seqinr::write.fasta, name = seqName, object = name, bucket = collectionUUID)
            } else if (fileFormat == "xlsx") {
                aws.s3::s3write_using(file, FUN = openxlsx::write.xlsx, object = name, bucket = collectionUUID)
            } else if (fileFormat == "dat" || fileFormat == "bin") {
                aws.s3::s3write_using(file, FUN = writeBin, object = name, bucket = collectionUUID)
            } else {
                stop(parse(('File format not supported, use arvadosFile$connection() and customise it')))
            }
        },

        #' @description
        #' Creates one or more ArvadosFiles and adds them to the collection at specified path.
        #' @param files Content to be created.
        #' @examples
        #' collection <- arv$collections_create(name = collectionTitle, description = collectionDescription, owner_uuid = collectionOwner, properties = list("ROX37196928443768648" = "ROX37742976443830153"))
        create = function(files)
        {
            if(is.null(private$tree))
                private$generateCollectionTreeStructure()

            if(is.character(files))
            {
                sapply(files, function(file)
                {
                    childWithSameName <- self$get(file)
                    if(!is.null(childWithSameName))
                        stop("Destination already contains file with same name.")

                    newTreeBranch <- private$tree$createBranch(file)
                    private$tree$addBranch(private$tree$getTree(), newTreeBranch)

                    private$REST$create(file, self$uuid)
                    newTreeBranch$setCollection(self)
                    newTreeBranch
                })
            }
            else
            {
                stop(paste0("Expected character vector, got ",
                            paste0("(", paste0(class(files), collapse = ", "), ")"),
                            "."))
            }
        },

        #' @description
        #' Remove one or more files from the collection.
        #' @param paths Content to be removed.
        #' @examples
        #' collection$remove(fileName.format)
        remove = function(paths)
        {
            if(is.null(private$tree))
                private$generateCollectionTreeStructure()

            if(is.character(paths))
            {
                sapply(paths, function(filePath)
                {
                    filePath <- trimFromEnd(filePath, "/")
                    file <- self$get(filePath)

                    if(is.null(file))
                        stop(paste("File", filePath, "doesn't exist."))

                    parent <- file$getParent()

                    if(is.null(parent))
                        stop("You can't delete root folder.")

                    parent$remove(file$getName())
                })

                "Content removed"
            }
            else
            {
                stop(paste0("Expected character vector, got ",
                            paste0("(", paste0(class(paths), collapse = ", "), ")"),
                            "."))
            }
        },

        #' @description
        #' Moves ArvadosFile or Subcollection to another location in the collection.
        #' @param content Content to be moved.
        #' @param destination Path to move content.
        #' @examples
        #' collection$move("fileName.format", path)
        move = function(content, destination)
        {
            if(is.null(private$tree))
                private$generateCollectionTreeStructure()

            content <- trimFromEnd(content, "/")

            elementToMove <- self$get(content)

            if(is.null(elementToMove))
                stop("Content you want to move doesn't exist in the collection.")

            elementToMove$move(destination)
        },

        #' @description
        #' Copies ArvadosFile or Subcollection to another location in the collection.
        #' @param content Content to be moved.
        #' @param destination Path to move content.
        #' @examples
        #' copied <- collection$copy("oldName.format", "newName.format")
        copy = function(content, destination)
        {
            if(is.null(private$tree))
                private$generateCollectionTreeStructure()

            content <- trimFromEnd(content, "/")

            elementToCopy <- self$get(content)

            if(is.null(elementToCopy))
                stop("Content you want to copy doesn't exist in the collection.")

            elementToCopy$copy(destination)
        },

        #' @description
        #' Refreshes the environment.
        #' @examples
        #' collection$refresh()
        refresh = function()
        {
            if(!is.null(private$tree))
            {
                private$tree$getTree()$setCollection(NULL, setRecursively = TRUE)
                private$tree <- NULL
            }
        },

        #' @description
        #' Returns collections file content as character vector.
        #' @examples
        #' list <- collection$getFileListing()
        getFileListing = function()
        {
            if(is.null(private$tree))
                private$generateCollectionTreeStructure()

            content <- private$REST$getCollectionContent(self$uuid)
            content[order(tolower(content))]
        },

        #' @description
        #' If relativePath is valid, returns ArvadosFile or Subcollection specified by relativePath, else returns NULL.
        #' @param relativePath Path from content is taken.
        #' @examples
        #' arvadosFile <- collection$get(fileName)
        get = function(relativePath)
        {
            if(is.null(private$tree))
                private$generateCollectionTreeStructure()

            private$tree$getElement(relativePath)
        },

        getRESTService = function() private$REST,
        setRESTService = function(newRESTService) private$REST <- newRESTService
    ),
    private = list(

        REST        = NULL,
        #' @tree beautiful tree of sth
        tree        = NULL,
        fileContent = NULL,

        generateCollectionTreeStructure = function(relativePath = NULL)
        {
            if(is.null(self$uuid))
                stop("Collection uuid is not defined.")

            if(is.null(private$REST))
                stop("REST service is not defined.")

            private$fileContent <- private$REST$getCollectionContent(self$uuid, relativePath)
            private$tree <- CollectionTree$new(private$fileContent, self)
        }
    ),

    cloneable = FALSE
)

#' print.Collection
#'
#' Custom print function for Collection class
#'
#' @param x Instance of Collection class
#' @param ... Optional arguments.
#' @export
print.Collection = function(x, ...)
{
    cat(paste0("Type: ", "\"", "Arvados Collection", "\""), sep = "\n")
    cat(paste0("uuid: ", "\"", x$uuid,               "\""), sep = "\n")
}







