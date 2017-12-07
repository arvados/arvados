source("./R/Arvados.R")
source("./R/HttpParser.R")
source("./R/Subcollection.R")
source("./R/ArvadosFile.R")

#' Collection Class
#' 
#' @details 
#' Todo: Update description
#' Collection
#' 
#' @param uuid Object ID
#' @param etag Object version
#' @param owner_uuid No description
#' @param created_at No description
#' @param modified_by_client_uuid No description
#' @param modified_by_user_uuid No description
#' @param modified_at No description
#' @param portable_data_hash No description
#' @param replication_desired No description
#' @param replication_confirmed_at No description
#' @param replication_confirmed No description
#' @param updated_at No description
#' @param manifest_text No description
#' @param name No description
#' @param description No description
#' @param properties No description
#' @param delete_at No description
#' @param file_names No description
#' @param trash_at No description
#' @param is_trashed No description
#' 
#' @export Collection

#' @exportClass Collection
Collection <- setRefClass(

    "Collection",

    fields = list(uuid                     = "ANY",
                  items                    = "ANY",
                  fileContent              = "ANY",
                  etag                     = "ANY",
                  owner_uuid               = "ANY",
                  created_at               = "ANY",
                  modified_by_client_uuid  = "ANY",
                  modified_by_user_uuid    = "ANY",
                  modified_at              = "ANY",
                  portable_data_hash       = "ANY",
                  replication_desired      = "ANY",
                  replication_confirmed_at = "ANY",
                  replication_confirmed    = "ANY",
                  updated_at               = "ANY",
                  manifest_text            = "ANY",
                  name                     = "ANY",
                  description              = "ANY",
                  properties               = "ANY",
                  delete_at                = "ANY",
                  file_names               = "ANY",
                  trash_at                 = "ANY",
                  is_trashed               = "ANY",

                  getCollectionContent = "function",
                  get                  = "function"
    ),

    methods = list(

        initialize = function(api, uuid) 
        {
            result <- api$collection_get(uuid)
            
            # Private members
            uuid                     <<- result$uuid                               
            etag                     <<- result$etag                               
            owner_uuid               <<- result$owner_uuid                         
            created_at               <<- result$created_at                         
            modified_by_client_uuid  <<- result$modified_by_client_uuid            
            modified_by_user_uuid    <<- result$modified_by_user_uuid              
            modified_at              <<- result$modified_at                        
            portable_data_hash       <<- result$portable_data_hash                 
            replication_desired      <<- result$replication_desired                
            replication_confirmed_at <<- result$replication_confirmed_at           
            replication_confirmed    <<- result$replication_confirmed              
            updated_at               <<- result$updated_at                         
            manifest_text            <<- result$manifest_text                      
            name                     <<- result$name                               
            description              <<- result$description                        
            properties               <<- result$properties                         
            delete_at                <<- result$delete_at                          
            file_names               <<- result$file_names                         
            trash_at                 <<- result$trash_at                           
            is_trashed               <<- result$is_trashed                         

            # Public methods

            getCollectionContent <<- function()
            {
                #TODO(Fudo): Use proper URL here.
                uri <- URLencode(api$getWebDavHostName())

                # fetch directory listing via curl and parse XML response
                h <- curl::new_handle()
                curl::handle_setopt(h, customrequest = "PROPFIND")

                #TODO(Fudo): Use proper token here.
                curl::handle_setheaders(h, "Authorization" = paste("OAuth2", api$getWebDavToken()))
                response <- curl::curl_fetch_memory(uri, h)

                HttpParser()$parseWebDAVResponse(response, uri)
            }

            get <<- function(pathToTheFile)
            {
                fileWithPath <- unlist(stringr::str_split(pathToTheFile, "/"))
                fileWithPath <- fileWithPath[fileWithPath != ""]

                findFileIfExists <- function(name, node)
                {
                    matchPosition <- match(name, sapply(node$content, function(nodeInSubcollection) {nodeInSubcollection$name}), -1)
                    if(matchPosition != -1)
                    {
                        return(node$content[[matchPosition]])
                    }
                    else
                    {
                        return(NULL)
                    }
                }
                
                nodeToCheck = .self$items
                for(fileNameIndex in 1:length(fileWithPath))
                {
                    nodeToCheck <- findFileIfExists(fileWithPath[fileNameIndex], nodeToCheck)
                    if(is.null(nodeToCheck))
                        stop("File or folder you asked for is not part of the collection.")
                }

                nodeToCheck
            }


            # Private methods
            .createCollectionContentTree <- function(fileStructure)
            {
                #TODO(Fudo): Refactot this.
                treeBranches <- sapply(fileStructure, function(filePath)
                {
                    fileWithPath <- unlist(str_split(filePath, "/"))
                    file <- fileWithPath[length(fileWithPath), drop = T]

                    if(file != "")
                    {
                        file <- ArvadosFile(file)
                        file$relativePath <- filePath
                    }
                    else
                    {
                        file <- NULL
                    }

                    folders <- fileWithPath[-length(fileWithPath)]

                    subcollections <- sapply(folders, function(folder)
                    {
                        folder <- Subcollection(folder)
                        unname(folder)
                    })

                    if(!is.null(file))
                        subcollections <- c(subcollections, file)

                    if(length(subcollections) > 1)
                    {
                        for(subcollectionIndex in 1:(length(subcollections) - 1))
                        {
                            subcollections[[subcollectionIndex]]$relativePath <- paste(folders[1:(subcollectionIndex)], collapse = "/")
                            subcollections[[subcollectionIndex]]$add(subcollections[[subcollectionIndex + 1]])
                        }
                    }
                    subcollections[[1]]
                })

                root <- Subcollection(".")

                addIfExists <- function(firstNode, secondNode)
                {
                    firstNodeContent <- sapply(firstNode$content, function(node) {node$name})
                    if(length(firstNodeContent) == 0)
                    {
                        firstNode$add(secondNode)
                        return()
                    }

                    matchPosition <- match(secondNode$name, firstNodeContent, -1)
                    if(matchPosition != -1)
                    {
                        addIfExists(firstNode$content[[matchPosition]], secondNode$content[[1]])
                    }
                    else
                    {
                        firstNode$add(secondNode)
                    }
                }

                sapply(treeBranches, function(branch)
                {
                    addIfExists(root, branch)
                })

                root
            }

            #Todo(Fudo): This is dummy data. Real content will come from WebDAV server.
            testFileStructure <- c("math.h", "main.cpp", "emptyFolder/",
                                   "java/render.java", "java/test/observer.java",
                                   "java/test/observable.java",
                                   "csharp/this.cs", "csharp/is.cs",
                                   "csharp/dummy.cs", "csharp/file.cs")
            items  <<- getCollectionContent()
            fileContent  <<- .createCollectionContentTree(testFileStructure)
        }
    )
)
