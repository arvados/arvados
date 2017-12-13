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

            private$fileItems <- private$getCollectionContent()

            private$fileTree <- private$generateTree(private$fileItems)
        },

        printFileContent = function()
        {
            private$fileTree$printContent(0)
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
            print(paste(filePath, newSize))

            node <- private$getNode(filePath)
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
        },

        #Todo(Fudo): Move tree creation to another file.
        generateTree = function(collectionContent)
        {
            treeBranches <- sapply(collectionContent, function(filePath)
            {
                splitPath <- unlist(strsplit(filePath$name, "/", fixed = TRUE))

                branch = private$createBranch(splitPath, filePath$fileSize)      
            })

            root <- TreeNode$new("./", "root", NULL)
            root$relativePath = ""

            sapply(treeBranches, function(branch)
            {
                private$addNode(root, branch)
            })

            root
        },

        createBranch = function(splitPath, fileSize)
        {
            branch <- NULL
            lastElementIndex <- length(splitPath)

            for(elementIndex in lastElementIndex:1)
            {
                if(elementIndex == lastElementIndex)
                {
                    branch = TreeNode$new(splitPath[[elementIndex]], "file", fileSize)
                }
                else
                {
                    newFolder = TreeNode$new(splitPath[[elementIndex]], "folder", NULL)
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
                child$type = "folder"
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
        },

        getNode = function(relativePathToNode)
        {
            treeBranches <- sapply(relativePathToNode, function(filePath)
            {
                splitPath <- unlist(strsplit(filePath, "/", fixed = TRUE))
                
                node = private$fileTree
                for(pathFragment in splitPath)
                {
                    child = node$getChild(pathFragment)
                    if(is.null(child))
                        stop("Subcollection/ArvadosFile you are looking for doesn't exist.")
                    node = child
                }

                node
            })
        }

    ),

    cloneable = FALSE
)

TreeNode <- R6::R6Class(

    "TreeNode",

    public = list(

        name         = NULL,
        relativePath = NULL,
        size         = NULL,
        children     = NULL,
        parent       = NULL,
        type         = NULL,

        initialize = function(name, type, size)
        {
            self$name <- name
            self$type <- type
            self$size <- size
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
            if(self$type == "folder")
                print(paste0(indentation, self$name, "/"))
            else
                print(paste0(indentation, self$name))
            
            for(child in self$children)
                child$printContent(depth + 1)
        }
    ),

    cloneable = FALSE
)
