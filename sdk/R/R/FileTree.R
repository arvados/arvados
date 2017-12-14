FileTree <- R6::R6Class(
    "FileTree",
    public = list(
        initialize = function(collectionContent)
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
                private$addBranch(root, branch)
            })

            private$tree <- root
        },

        getRoot = function() private$tree,

        printContent = function(node, depth)
        {
            indentation <- paste(rep("....", depth), collapse = "")
            if(node$type == "folder")
                print(paste0(indentation, node$name, "/"))
            else
                print(paste0(indentation, node$name))
            
            for(child in node$children)
                self$printContent(child, depth + 1)
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
                    result <- self$traverseInOrder(child, predicate)

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
                
                node <- private$tree
                for(pathFragment in splitPath)
                {
                    child = node$getChild(pathFragment)

                    if(is.null(child))
                        return(NULL)

                    node = child
                }

                node
            })
        },

        addNode = function(relativePathToNode, size)
        {
            splitPath <- unlist(strsplit(relativePathToNode, "/", fixed = TRUE))

            branch <- private$createBranch(splitPath, size)
            private$addBranch(private$tree, branch)
        }
    ),

    private = list(
        tree = NULL,

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

        addBranch = function(container, node)
        {
            child = container$getChild(node$name)

            if(is.null(child))
            {
                container$addChild(node)
            }
            else
            {
                child$type = "folder"
                private$addBranch(child, node$getFirstChild())
            }
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
        }

    ),

    cloneable = FALSE
)
