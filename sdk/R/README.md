[comment]: # (Copyright © The Arvados Authors. All rights reserved.)
[comment]: # ()
[comment]: # (SPDX-License-Identifier: CC-BY-SA-3.0)

# R SDK for Arvados

This SDK focuses on providing support for accessing Arvados projects, collections, and the files within collections. The API is not final and feedback is solicited from users on ways in which it could be improved.

## Key Topics
* Installation
* Usage
  * Initializing API
  * Working with collections
  * Manipulating collection content
  * Working with Arvados projects
  * Help
* Building the ArvadosR package

## Installation

Minimum R version required to run ArvadosR is 3.3.0.

```r
install.packages("ArvadosR", repos=c("https://r.arvados.org", getOption("repos")["CRAN"]), dependencies=TRUE)
library('ArvadosR')
```

> **Note**
> On Linux, you may have to install supporting packages.
>
> On Red Hat, AlmaLinux, and Rocky Linux, this is:
> ```
> yum install libxml2-devel openssl-devel curl-devel
> ```
>
> On Debian and Ubuntu, this is:
> ```
> apt-get install build-essential libxml2-dev libssl-dev libcurl4-gnutls-dev
> ```


## Usage

### Initializing API

```r
# use environment variables ARVADOS_API_TOKEN and ARVADOS_API_HOST
arv <- Arvados$new()

# provide them explicitly
arv <- Arvados$new("your Arvados token", "example.arvadosapi.com")
```

Optionally, add `numRetries` parameter to specify number of times to retry failed service requests. Default is 0.

```r
arv <- Arvados$new("your Arvados token", "example.arvadosapi.com", numRetries = 3)
```

This parameter can be set at any time using `setNumRetries`

```r
arv$setNumRetries(5)
```

### Working with Aravdos projects

##### Create project:

```r
newProject <- arv$project_create(name = "project name", description = "project description", owner_uuid = "project UUID", properties = NULL, ensureUniqueName = "false")
```

##### Update project:

```r
updatedProject <- arv$project_update(name = "new project name", properties = newProperties, uuid = "projectUUID")
```

##### Delete a project:

```r
deletedProject <- arv$project_delete("uuid")
```

#### Find a project:

##### Get a project:

```r
project <- arv$project_get("uuid")
```

##### List projects:

```r
list subprojects of a project
projects <- arv$project_list(list(list("owner_uuid", "=", "aaaaa-j7d0g-ccccccccccccccc")))

list projects which have names beginning with Example
examples <- arv$project_list(list(list("name","like","Example%")))
```

##### List all projects even if the number of items is greater than maximum API limit:

```r
projects <- listAll(arv$project_list, list(list("name","like","Example%")))
```

### Working with collections

#### Create a new collection:

```r
newCollection <- arv$collections_create(name = "collectionTitle", description = "collectionDescription", ownerUUID = "collectionOwner", properties = Properties)
```

#### Update a collection’s metadata:

```r
collection <- arv$collections_update(name = "newCollectionTitle", description = "newCollectionDescription", ownerUUID = "collectionOwner", properties = NULL, uuid =  "collectionUUID")
```

#### Delete a collection:

```r
deletedCollection <- arv$collections_delete("uuid")
```

#### Find a collection:

#### Get a collection:

```r
collection <- arv$collections_get("uuid")
```

Be aware that the result from `collections_get` is not a Collection class. The object returned from this method lets you access collection fields like “name” and “description”. The Collection class lets you access the files in the collection for reading and writing, and is described in the next section.

#### List collections:

```r
# offset of 0 and default limit of 100
collectionList <- arv$collections_list(list(list("name", "like", "Test%")))

collectionList <- arv$collections_list(list(list("name", "like", "Test%")), limit = 10, offset = 2)

# count of total number of items (may be more than returned due to paging)
collectionList$items_available

# items which match the filter criteria
collectionList$items
```

#### List all collections even if the number of items is greater than maximum API limit:

```r
collectionList <- listAll(arv$collections_list, list(list("name", "like", "Test%")))
```

### Manipulating collection content

#### Initialize a collection object:

```r
collection <- Collection$new(arv, "uuid")
```

#### Get list of files:

```r
files <- collection$getFileListing()
```

#### Get ArvadosFile or Subcollection from internal tree-like structure:

```r
arvadosFile <- collection$get("location/to/my/file.cpp")
# or
arvadosSubcollection <- collection$get("location/to/my/directory/")
```

#### Read a table:

```r
arvadosFile   <- collection$get("myinput.txt")
arvConnection <- arvadosFile$connection("r")
mytable       <- read.table(arvConnection)
```

#### Write a table:

```r
arvadosFile   <- collection$create("myoutput.txt")[[1]]
arvConnection <- arvadosFile$connection("w")
write.table(mytable, arvConnection)
arvadosFile$flush()
```

#### Read a table from a tab delimited file:

```r
arvadosFile   <- collection$get("myinput.txt")
arvConnection <- arvadosFile$connection("r")
mytable       <- read.delim(arvConnection)
```

#### Read a gzip compressed R object:

```r
obj <- readRDS(gzcon(coll$get("abc.RDS")$connection("rb")))
```

#### Write to existing file (overwrites current content of the file):

```r
arvadosFile <- collection$get("location/to/my/file.cpp")
arvadosFile$write("This is new file content")
```

#### Read whole file or just a portion of it:

```r
fileContent <- arvadosFile$read()
fileContent <- arvadosFile$read("text")
fileContent <- arvadosFile$read("raw", offset = 1024, length = 512)
```

#### Read various file types:

Chooses file type based on file name extension.  Recognized file extensions: 'txt', 'xlsx', 'csv', 'tsv', 'fasta', 'dat', 'bin', 'rds', 'rdata'.

```r
collection <- Collection$new(arv, collectionUUID)
readFile <- collection$readArvFile(arvadosFile, istable = 'yes')                    # table
readFile <- collection$readArvFile(arvadosFile, istable = 'no')                     # text
readFile <- collection$readArvFile(arvadosFile)                                     # xlsx, csv, tsv, rds, rdata
readFile <- collection$readArvFile(arvadosFile, fileclass = 'fasta')                # fasta
readFile <- collection$readArvFile(arvadosFile, Ncol= 4, Nrow = 32)                 # binary data.frame, only numbers
readFile <- collection$readArvFile(arvadosFile, Ncol = 5, Nrow = 150, istable = "factor") # binary data.frame with factor or text
```

#### Get ArvadosFile or Subcollection size:

```r
size <- arvadosFile$getSizeInBytes()
# or
size <- arvadosSubcollection$getSizeInBytes()
```

#### Create new file in a collection (returns a vector of one or more ArvadosFile objects):

```r
collection$create(files)
```

**Example**

```
mainFile <- collection$create("cpp/src/main.cpp")[[1]]
fileList <- collection$create(c("cpp/src/main.cpp", "cpp/src/util.h"))
```

#### Delete file from a collection:

```r
collection$remove("location/to/my/file.cpp")
```

You can remove both Subcollection and ArvadosFile. If subcollection contains more files or folders they will be removed recursively.

> **Note**
> You can also remove multiple files at once:
> ```
> collection$remove(c("path/to/my/file.cpp", "path/to/other/file.cpp"))
> ```

#### Delete file or folder from a Subcollection:

```r
subcollection <- collection$get("mySubcollection/")
subcollection$remove("fileInsideSubcollection.exe")
subcollection$remove("folderInsideSubcollection/")
```

#### Move or rename a file or folder within a collection (moving between collections is currently not supported):

##### Directly from collection

```r
collection$move("folder/file.cpp", "file.cpp")
```

##### Or from file

```r
file <- collection$get("location/to/my/file.cpp")
file$move("newDestination/file.cpp")
```

##### Or from subcollection

```r
subcollection <- collection$get("location/to/folder")
subcollection$move("newDestination/folder")
```

> **Note**
> Make sure to include new file name in destination. In second example `file$move(“newDestination/”)` will not work.

#### Copy file or folder within a collection (copying between collections is currently not supported):

##### Directly from collection

```r
collection$copy("folder/file.cpp", "file.cpp")
```

##### Or from file

```r
file <- collection$get("location/to/my/file.cpp")
file$copy("destination/file.cpp")
```

##### Or from subcollection

```r
subcollection <- collection$get("location/to/folder")
subcollection$copy("destination/folder")
```


### Help

#### View help page of Arvados classes by puting `?` before class name:

```r
?Arvados
?Collection
?Subcollection
?ArvadosFile
```

#### View help page of any method defined in Arvados class by puting `?` before method name:

```r
?collections_update
?workflows_get
```

 <!-- Taka konwencja USAGE -->

## Building the ArvadosR package

```r
cd arvados/sdk && R CMD build R
```

This will create a tarball of the ArvadosR package in the current directory.

 <!-- Czy dodawać Documentation / Community / Development and Contributing / Licensing? Ale tylko do części Rowej? Wszystko? Wcale? -->

## Documentation

Complete documentation, including the [User Guide](https://doc.arvados.org/user/index.html), [Installation documentation](https://doc.arvados.org/install/index.html), [Administrator documentation](https://doc.arvados.org/admin/index.html) and
[API documentation](https://doc.arvados.org/api/index.html) is available at http://doc.arvados.org/

## Community

Visit [Arvados Community and Getting Help](https://doc.arvados.org/user/getting_started/community.html).

## Reporting bugs

[Report a bug](https://dev.arvados.org/projects/arvados/issues/new) on [dev.arvados.org](https://dev.arvados.org).

## Licensing

Arvados is Free Software.  See [Arvados Free Software Licenses](https://doc.arvados.org/user/copying/copying.html) for information about the open source licenses used in Arvados.
