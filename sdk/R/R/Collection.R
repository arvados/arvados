source("./R/Subcollection.R")
source("./R/ArvadosFile.R")

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

            #Todo(Fudo): Replace this when you get access to webDAV server.
            private$fileItems <- private$getCollectionContent()

            private$fileTree <- private$generateTree(private$fileItems)
        },

        printFileContent = function(pretty = TRUE)
        {
            if(pretty)
                private$fileTree$printContent(0)
            else
                print(private$fileItems)

        },

        get = function(relativePath)
        {
            treeNode <- private$traverseInOrder(private$fileTree, function(node)
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

        api       = NULL,
        fileItems = NULL,
        fileTree  = NULL,

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
                    return(ArvadosFile$new(treeNode$name, treeNode$relativePath, private$api, self))
                else if(treeNode$type == "folder" || treeNode$type == "root")
                    return(Subcollection$new(treeNode$name, treeNode$relativePath, NULL))
            }
        },

        createSubcollectionFromNode = function(treeNode, children)
        {
            subcollection = NULL
            if(treeNode$type == "file")
                subcollection = ArvadosFile$new(treeNode$name, treeNode$relativePath)
            else if(treeNode$type == "folder" || treeNode$type == "root")
                subcollection = Subcollection$new(treeNode$name, treeNode$relativePath, children)
            
            subcollection
        },

        getCollectionContent = function()
        {
            #TODO(Fudo): Use proper URL here.
            uri <- URLencode(paste0(private$api$getWebDavHostName(), "c=", self$uuid))

            # fetch directory listing via curl and parse XML response
            h <- curl::new_handle()
            curl::handle_setopt(h, customrequest = "PROPFIND")

            #TODO(Fudo): Use proper token here.
            curl::handle_setheaders(h, "Authorization" = paste("OAuth2", private$api$getToken()))
            response <- curl::curl_fetch_memory(uri, h)

            HttpParser$new()$parseWebDAVResponse(response, uri)
        },

        #Todo(Fudo): Move tree creation to another file.
        generateTree = function(collectionContent)
        {
            treeBranches <- sapply(collectionContent, function(filePath)
            {
                splitPath <- unlist(strsplit(filePath, "/", fixed = TRUE))

                pathEndsWithSlash <- substr(filePath, nchar(filePath), nchar(filePath)) == "/"
                
                branch = private$createBranch(splitPath, pathEndsWithSlash)      
            })

            root <- TreeNode$new("./", "root")
            root$relativePath = ""

            sapply(treeBranches, function(branch)
            {
                private$addNode(root, branch)
            })

            root
        },

        createBranch = function(splitPath, pathEndsWithSlash)
        {
            branch <- NULL
            lastElementIndex <- length(splitPath)
            
            lastElementInPathType = "file"
            if(pathEndsWithSlash)
                lastElementInPathType = "folder"

            for(elementIndex in lastElementIndex:1)
            {
                if(elementIndex == lastElementIndex)
                {
                    branch = TreeNode$new(splitPath[[elementIndex]], lastElementInPathType)
                }
                else
                {
                    newFolder = TreeNode$new(splitPath[[elementIndex]], "folder")
                    newFolder$addChild(branch)
                    branch = newFolder
                }

                branch$relativePath <- paste(unlist(splitPath[1:elementIndex]), collapse = "/")
            }

            branch
        },

        addNode = function(container, node)
        {
            child = container$getChild(node$name)

            if(is.null(child))
            {
                container$addChild(node)
            }
            else
            {
                private$addNode(child, node$getFirstChild())
            }
        },

        traverseInOrder = function(node, predicate)
        {
            if(node$hasChildren())
            {
                result <- predicate(node)

                if(!is.null(result))
                    return(result)               

                for(child in node$children)
                {
                    result <- private$traverseInOrder(child, predicate)

                    if(!is.null(result))
                        return(result)
                }

                return(NULL)
            }
            else
            {
                return(predicate(node))
            }
        }

    ),

    cloneable = FALSE
)

TreeNode <- R6::R6Class(

    "TreeNode",

    public = list(

        name = NULL,
        relativePath = NULL,
        children = NULL,
        parent = NULL,
        type = NULL,

        initialize = function(name, type)
        {
            if(type == "folder")
                name <- paste0(name, "/")

            self$name <- name
            self$type <- type
            self$children <- list()
        },

        addChild = function(node)
        {
            self$children <- c(self$children, node)
            node$setParent(self)
            self
        },

        setParent = function(parent)
        {
            self$parent = parent
        },

        getChild = function(childName)
        {
            for(child in self$children)
            {
                if(childName == child$name)
                    return(child)
            }

            return(NULL)
        },

        hasChildren = function()
        {
            if(length(self$children) != 0)
                return(TRUE)
            else
                return(FALSE)
        },

        getFirstChild = function()
        {
            if(!self$hasChildren())
                return(NULL)
            else
                return(self$children[[1]])
        },

        printContent = function(depth)
        {
            indentation <- paste(rep("....", depth), collapse = "")
            print(paste0(indentation, self$name))
            
            for(child in self$children)
                child$printContent(depth + 1)
        }
    ),

    cloneable = FALSE
)
