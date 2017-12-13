source("./R/Subcollection.R")
source("./R/ArvadosFile.R")
source("./R/FileTree.R")

#' Arvados Collection Object
#'
#' Update description
#'
#' @examples arv = Collection$new(api, uuid)
#' @export Collection
Collection <- R6::R6Class(

    "Collection",

    public = list(

        #Todo(Fudo): Encapsulate this?
        uuid                     = NULL,
        etag                     = NULL,
        owner_uuid               = NULL,
        created_at               = NULL,
        modified_by_client_uuid  = NULL,
        modified_by_user_uuid    = NULL,
        modified_at              = NULL,
        portable_data_hash       = NULL,
        replication_desired      = NULL,
        replication_confirmed_at = NULL,
        replication_confirmed    = NULL,
        updated_at               = NULL,
        manifest_text            = NULL,
        name                     = NULL,
        description              = NULL,
        properties               = NULL,
        delete_at                = NULL,
        file_names               = NULL,
        trash_at                 = NULL,
        is_trashed               = NULL,

        initialize = function(api, uuid)
        {
            private$api <- api
            result <- private$api$getCollection(uuid)

            self$uuid                     <- result$uuid                               
            self$etag                     <- result$etag                               
            self$owner_uuid               <- result$owner_uuid                         
            self$created_at               <- result$created_at                         
            self$modified_by_client_uuid  <- result$modified_by_client_uuid            
            self$modified_by_user_uuid    <- result$modified_by_user_uuid              
            self$modified_at              <- result$modified_at                        
            self$portable_data_hash       <- result$portable_data_hash                 
            self$replication_desired      <- result$replication_desired                
            self$replication_confirmed_at <- result$replication_confirmed_at           
            self$replication_confirmed    <- result$replication_confirmed              
            self$updated_at               <- result$updated_at                         
            self$manifest_text            <- result$manifest_text                      
            self$name                     <- result$name                               
            self$description              <- result$description                        
            self$properties               <- result$properties                         
            self$delete_at                <- result$delete_at                          
            self$file_names               <- result$file_names                         
            self$trash_at                 <- result$trash_at                           
            self$is_trashed               <- result$is_trashed                         

            private$fileItems <- private$getCollectionContent()

            private$fileTree <- FileTree$new(private$fileItems)
        },

        printFileContent = function()
        {
            private$fileTree$printContent(private$fileTree$getRoot(), 0)
        },

        getFileContent = function()
        {
            sapply(private$fileItems, function(file)
            {
                file$name
            })
        },

        get = function(relativePath)
        {
            treeNode <- private$fileTree$traverseInOrder(private$fileTree$getRoot(), function(node)
            {
                if(node$relativePath == relativePath)
                    return(node)
                else
                    return(NULL)
            })

            if(!is.null(treeNode))
            {
                return(private$createSubcollectionTree(treeNode))
            }
            else
            {
                return(NULL)
            }
        },

        update = function(subcollection, event)
        {
            #Todo(Fudo): Add some king of check here later on.
            if(event == "File size changed")
            {
                private$handleFileSizeChange(subcollection$getRelativePath(),
                                             subcollection$getSizeInBytes())
            }
        }
    ),

    active = list(
        items = function(value)
        {
            if(missing(value))
                return(private$fileItems)
            else
                print("Value is read-only.")

            return(NULL)
        }
    ),
    
    private = list(

        fileItems = NULL,
        api       = NULL,
        fileTree  = NULL,

        handleFileSizeChange = function(filePath, newSize)
        {
            node <- private$fileTree$getNode(filePath)
            node$size <- newSize
        },

        createSubcollectionTree = function(treeNode)
        {
            if(treeNode$hasChildren())
            {
                children = NULL

                for(child in treeNode$children)
                {
                    child <- private$createSubcollectionTree(child)
                    children <- c(children, child)                   
                }

                return(Subcollection$new(treeNode$name, treeNode$relativePath, children))
            }
            else
            {
                if(treeNode$type == "file")
                    return(ArvadosFile$new(treeNode$name, treeNode$relativePath, treeNode$size, private$api, self))
                else 
                    return(Subcollection$new(treeNode$name, treeNode$relativePath, NULL))
            }
        },

        getCollectionContent = function()
        {
            uri <- URLencode(paste0(private$api$getWebDavHostName(), "c=", self$uuid))

            # fetch directory listing via curl and parse XML response
            h <- curl::new_handle()
            curl::handle_setopt(h, customrequest = "PROPFIND")

            curl::handle_setheaders(h, "Authorization" = paste("OAuth2", private$api$getToken()))
            response <- curl::curl_fetch_memory(uri, h)

            parsedResponse <- HttpParser$new()$parseWebDAVResponse(response, uri)
            parsedResponse[-1]
        }

    ),

    cloneable = FALSE
)
