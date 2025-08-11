```
Copyright (C) The Arvados Authors. All rights reserved.
 
SPDX-License-Identifier: CC-BY-SA-3.0
```

# Arvados Java SDK

##### About
Arvados Java Client allows to access Arvados servers and uses two APIs:
* lower level [Keep Server API](https://doc.arvados.org/api/index.html)
* higher level [Keep-Web API](https://godoc.org/github.com/arvados/arvados/services/keep-web) (when needed)

##### Required Java version
This SDK requires Java 8+

##### Logging

SLF4J is used for logging. Concrete logging framework and configuration must be provided by a client.

##### Configuration

[TypeSafe Configuration](https://github.com/lightbend/config) is used for configuring this library.

Please, have a look at java/resources/reference.conf for default values provided with this library.

* **keepweb-host** - change to host of your Keep-Web installation
* **keepweb-port** - change to port of your Keep-Web installation
* **host** - change to host of your Arvados installation
* **port** - change to port of your Arvados installation
* **token** - authenticates registered user, one must provide
  [token obtained from Arvados Workbench](https://doc.arvados.org/user/reference/api-tokens.html)
* **protocol** - don't change to unless really needed
* **host-insecure** - insecure communication with Arvados (ignores SSL certificate verification), 
  don't change to *true* unless really needed
* **split-size** - size of chunk files in megabytes
* **temp-dir** - temporary chunk files storage
* **copies** - amount of chunk files duplicates per Keep server
* **retries** - in case of chunk files send failure this should allow to repeat send 
  (*NOTE*: this parameter is not used at the moment but was left for future improvements)

In order to override default settings one can create application.conf file in an application.
Example: src/test/resources/application.conf.

Alternatively ExternalConfigProvider class can be used to pass configuration via code. 
ExternalConfigProvider comes with a builder and all of the above values must be provided in order for it to work properly.

ArvadosFacade has two constructors, one without arguments that uses values from reference.conf and second one 
taking ExternalConfigProvider as an argument.

##### API clients

All API clients inherit from BaseStandardApiClient. This class contains implementation of all 
common methods as described in http://doc.arvados.org/api/methods.html.

Parameters provided to common or specific methods are String UUID or fields wrapped in Java objects. For example:

```java
String uuid = "ardev-4zz18-rxcql7qwyakg1r1";

Collection actual = client.get(uuid);
```

```java
ListArgument listArgument = ListArgument.builder()
        .filters(Arrays.asList(
                Filter.of("owner_uuid", Operator.LIKE, "ardev%"),
                Filter.of("name", Operator.LIKE, "Super%"),
                Filter.of("portable_data_hash", Operator.IN, Lists.newArrayList("54f6d9f59065d3c009d4306660989379+65")
            )))
        .build();

CollectionList actual = client.list(listArgument);
```

Non-standard API clients must inherit from BaseApiClient. 
For example: KeepServerApiClient communicates directly with Keep servers using exclusively non-common methods.

##### Business logic

More advanced API data handling could be implemented as *Facade* classes. 
In current version functionalities provided by SDK are handled by *ArvadosFacade*.
They include:
* **downloading single file from collection** - using Keep-Web
* **downloading whole collection** - using Keep-Web or Keep Server API
* **listing file info from certain collection** - information is returned as list of *FileTokens* providing file details
* **uploading single file** - to either new or existing collection
* **uploading list of files** - to either new or existing collection
* **creating an empty collection**
* **getting current user info**
* **listing current user's collections**
* **creating new project**
* **deleting certain collection**

##### Note regarding Keep-Web

Current version requires both Keep Web and standard Keep Server API configured in order to use Keep-Web functionalities.

##### Integration tests

In order to run integration tests all fields within following configuration file must be provided: 
```java
src/test/resources/integration-test-appliation.conf 
```
Parameter **integration-tests.project-uuid** should contain UUID of one project available to user,
whose token was provided within configuration file. 

Integration tests require connection to real Arvados server.

##### Note regarding file naming

While uploading via this SDK all uploaded files within single collection must have different names.
This applies also to uploading files to already existing collection. 
Renaming files with duplicate names is not implemented in current version.

##### Building with Gradle

The Arvados Java SDK is built with `gradle`. Common development build tasks are:

* `clean`
* `test`
* `jar` (build the jar files, including documentation)
* `install`
