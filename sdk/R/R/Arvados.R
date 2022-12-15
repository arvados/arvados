# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

#' R6 Class Representing a Arvados
#'
#' @description
#' Arvados class gives users ability to access Arvados REST API. It also allowes user to manipulate collections (and projects?)

#' @export Arvados
Arvados <- R6::R6Class(

    "Arvados",

    public = list(

        #' @description
        #' Initialize new enviroment.
        #' @param authToken ARVADOS_API_TOKEN from 'Get API Token' on Arvados.
        #' @param hostName ARVADOS_API_HOST from 'Get API Token' on Arvados.
        #' @param numRetries Specify number of times to retry failed service requests.
        #' @return A new `Arvados` object.
        #' @examples
        #' arv <- Arvados$new(authToken = "ARVADOS_API_TOKEN", hostName = "ARVADOS_API_HOST", numRetries = 3)
        initialize = function(authToken = NULL, hostName = NULL, numRetries = 0)
        {
            if(!is.null(hostName))
                Sys.setenv(ARVADOS_API_HOST = hostName)

            if(!is.null(authToken))
                Sys.setenv(ARVADOS_API_TOKEN = authToken)

            hostName <- Sys.getenv("ARVADOS_API_HOST")
            token    <- Sys.getenv("ARVADOS_API_TOKEN")

            if(hostName == "" | token == "")
                stop(paste("Please provide host name and authentification token",
                           "or set ARVADOS_API_HOST and ARVADOS_API_TOKEN",
                           "environment variables."))

            private$token <- token
            private$host  <- paste0("https://", hostName, "/arvados/v1/")
            private$numRetries <- numRetries
            private$REST <- RESTService$new(token, hostName,
                                            HttpRequest$new(), HttpParser$new(),
                                            numRetries)

        },

        #' @description
        #' project_exist enables checking if the project with such a UUID exist.
        #' @param uuid The UUID of a project or a file.
        #' @examples
        #' arv$project_exist(uuid = projectUUID)
        project_exist = function(uuid)
        {
            proj <- self$project_list(list(list("uuid", '=', uuid)))
            value <- length(proj$items)

            if (value == 1){
                cat(format('TRUE'))
            } else {
                cat(format('FALSE'))
            }
        },

        #' @description
        #' project_get returns the demanded project.
        #' @param uuid The UUID of the Group in question.
        #' @examples
        #' project <- arv$project_get(uuid = projectUUID)
        project_get = function(uuid)
        {
            self$groups_get(uuid)
        },

        #' @description
        #' project_create creates a new project of a given name and description.
        #' @param name Name of the project.
        #' @param description Description of the project.
        #' @param ownerUUID The UUID of the maternal project to created one.
        #' @param properties List of the properties of the project.
        #' @param ensureUniqueName Adjust name to ensure uniqueness instead of returning an error.
        #' @examples
        #' Properties <- list() # should contain a list of new properties to be added
        #' new_project <- arv$project_create(name = "project name", description = "project description", owner_uuid = "project UUID", properties = NULL, ensureUniqueName = "false")
        project_create = function(name, description, ownerUUID, properties = NULL, ensureUniqueName = "false")
        {
            group <- list(name = name, description = description, owner_uuid = ownerUUID, properties = properties)
            group <- c("group_class" = "project", group)
            self$groups_create(group, ensureUniqueName =  ensureUniqueName)
        },

        #' @description
        #' project_properties_set is a method defined in Arvados class that enables setting properties. Allows to set or overwrite the properties. In case there are set already it overwrites them.
        #' @param listProperties List of new properties.
        #' @param uuid The UUID of a project or a file.
        #' @examples
        #' Properties <- list() # should contain a list of new properties to be added
        #' arv$project_properties_set(Properties, uuid)
        project_properties_set = function(listProperties, uuid)
        {
            group <- c("group_class" = "project", list("properties" = listProperties))
            self$groups_update(group, uuid)

        },

        #' @description
        #' project_properties_append is a method defined in Arvados class that enables appending properties. Allows to add new properties.
        #' @param listOfNewProperties List of new properties.
        #' @param uuid The UUID of a project or a file.
        #' @examples
        #' newProperties <- list() # should contain a list of new properties to be added
        #' arv$project_properties_append(properties = newProperties, uuid)
        project_properties_append = function(properties, uuid)
        {
            proj <- self$project_list(list(list('uuid', '=', uuid)))
            projProp <- proj$items[[1]]$properties

            newListOfProperties <- c(projProp, properties)
            uniqueProperties <- unique(unlist(newListOfProperties))
            newListOfProperties <- suppressWarnings(newListOfProperties[which(newListOfProperties == uniqueProperties)])

            group <- c("group_class" = "project", list("properties" = newListOfProperties))
            self$groups_update(group, uuid);

        },

        #' @description
        #' project_properties_get is a method defined in Arvados class that returns properties.
        #' @param uuid The UUID of a project or a file.
        #' @examples
        #' arv$project_properties_get(projectUUID)
        project_properties_get = function(uuid)
        {
            proj <- self$project_list(list(list('uuid', '=', uuid)))
            proj$items[[1]]$properties
        },

        #' @description
        #' project_properties_delete is a method defined in Arvados class that deletes list of properties.
        #' @param oneProp Property to be deleted.
        #' @param uuid The UUID of a project or a file.
        #' @examples
        #' Properties <- list() # should contain a list of new properties to be added
        #' arv$project_properties_delete(Properties,  projectUUID)
        project_properties_delete = function(oneProp, uuid)
        {
            proj <- self$project_list(list(list('uuid', '=', uuid))) # find project
            projProp <- proj$items[[1]]$properties
            for (i in 1:length(projProp)){
                solution <- identical(projProp[i],oneProp)
                if (solution == TRUE) {
                    projProp <- projProp[names(projProp) != names(oneProp)]
                    self$project_properties_set(projProp, uuid)
                }
            }
        },

        #' @description
        #' project_update enables updating project. New name, description and properties may be given.
        #' @param ... Feature to be updated (name, description, properties).
        #' @param uuid The UUID of a project in question.
        #' @examples
        #' newProperties <- list() # should contain a list of new properties to be added
        #' arv$project_update(name = "new project name", properties = newProperties, uuid = projectUUID)
        project_update = function(..., uuid) {
            vec <- list(...)
            for (i in 1:length(vec))
            {
                if (names(vec[i]) == 'properties') {
                    solution <- self$project_properties_append(vec$properties, uuid = uuid)
                }
            }
            vecNew <- vec[names(vec) != "properties"]
            vecNew <- c("group_class" = "project", vecNew)
            z <- self$groups_update(vecNew, uuid)
        },

        #' @description
        #' project_list enables listing project by its name, uuid, properties, permissions.
        #' @param filters
        #' @param where
        #' @param order
        #' @param distinct
        #' @param limit
        #' @param offset
        #' @param count
        #' @param includeTrash Include items whose is_trashed attribute is true.
        #' @param uuid The UUID of a project in question.
        #' @param recursive Include contents from child groups recursively.
        #' @examples
        #' listOfprojects <- arv$project_list(list(list("owner_uuid", "=", projectUUID))) # Sample query which show projects within the project of a given UUID
        project_list = function(filters = NULL, where = NULL,
                                order = NULL, select = NULL, distinct = NULL,
                                limit = "100", offset = "0", count = "exact",
                                includeTrash = NULL)
        {
            filters[[length(filters) + 1]] <- list("group_class", "=", "project")
            self$groups_list(filters, where, order, select, distinct,
                             limit, offset, count, includeTrash)
        },

        #' @description
        #' project_delete trashes project of a given uuid. It can be restored from trash or deleted permanently.
        #' @param uuid The UUID of the Group in question.
        project_delete = function(uuid)
        {
            self$groups_delete(uuid)
        },

        #' @description
        #' api_clients_get is a method defined in Arvados class.
        #' @param uuid The UUID of the apiClient in question.
        api_clients_get = function(uuid)
        {
            endPoint <- stringr::str_interp("api_clients/${uuid}")
            url <- paste0(private$host, endPoint)
            headers <- list(Authorization = paste("Bearer", private$token),
                            "Content-Type" = "application/json")
            queryArgs <- NULL

            body <- NULL

            response <- private$REST$http$exec("GET", url, headers, body,
                                               queryArgs, private$numRetries)
            resource <- private$REST$httpParser$parseJSONResponse(response)

            if(!is.null(resource$errors))
                stop(resource$errors)

            resource
        },

        #' @description
        #' api_clients_create is a method defined in Arvados class.
        #' @param apiClient apiClient object.
        #' @param ensureUniqueName Adjust name to ensure uniqueness instead of returning an error.
        #' @param clusterID Create object on a remote federated cluster instead of the current one.
        api_clients_create = function(apiClient,
                                      ensureUniqueName = "false", clusterID = NULL)
        {
            endPoint <- stringr::str_interp("api_clients")
            url <- paste0(private$host, endPoint)
            headers <- list(Authorization = paste("Bearer", private$token),
                            "Content-Type" = "application/json")
            queryArgs <- list(ensureUniqueName = ensureUniqueName,
                              clusterID = clusterID)

            if(length(apiClient) > 0)
                body <- jsonlite::toJSON(list(apiClient = apiClient),
                                         auto_unbox = TRUE)
            else
                body <- NULL

            response <- private$REST$http$exec("POST", url, headers, body,
                                               queryArgs, private$numRetries)
            resource <- private$REST$httpParser$parseJSONResponse(response)

            if(!is.null(resource$errors))
                stop(resource$errors)

            resource
        },

        #' @description
        #' api_clients_update is a method defined in Arvados class.
        #' @param apiClient apiClient object.
        #' @param uuid The UUID of the apiClient in question.
        api_clients_update = function(apiClient, uuid)
        {
            endPoint <- stringr::str_interp("api_clients/${uuid}")
            url <- paste0(private$host, endPoint)
            headers <- list(Authorization = paste("Bearer", private$token),
                            "Content-Type" = "application/json")
            queryArgs <- NULL

            if(length(apiClient) > 0)
                body <- jsonlite::toJSON(list(apiClient = apiClient),
                                         auto_unbox = TRUE)
            else
                body <- NULL

            response <- private$REST$http$exec("PUT", url, headers, body,
                                               queryArgs, private$numRetries)
            resource <- private$REST$httpParser$parseJSONResponse(response)

            if(!is.null(resource$errors))
                stop(resource$errors)

            resource
        },

        #' @description
        #' api_clients_delete is a method defined in Arvados class.
        #' @param uuid The UUID of the apiClient in question.
        api_clients_delete = function(uuid)
        {
            endPoint <- stringr::str_interp("api_clients/${uuid}")
            url <- paste0(private$host, endPoint)
            headers <- list(Authorization = paste("Bearer", private$token),
                            "Content-Type" = "application/json")
            queryArgs <- NULL

            body <- NULL

            response <- private$REST$http$exec("DELETE", url, headers, body,
                                               queryArgs, private$numRetries)
            resource <- private$REST$httpParser$parseJSONResponse(response)

            if(!is.null(resource$errors))
                stop(resource$errors)

            resource
        },

        #' @description
        #' api_clients_list is a method defined in Arvados class.
        #' @param filters
        #' @param where
        #' @param order
        #' @param select
        #' @param distinct
        #' @param limit
        #' @param offset
        #' @param count
        #' @param clusterID List objects on a remote federated cluster instead of the current one.
        #' @param bypassFederation Bypass federation behavior, list items from local instance database only.
        api_clients_list = function(filters = NULL,
                                    where = NULL, order = NULL, select = NULL,
                                    distinct = NULL, limit = "100", offset = "0",
                                    count = "exact", clusterID = NULL, bypassFederation = NULL)
        {
            endPoint <- stringr::str_interp("api_clients")
            url <- paste0(private$host, endPoint)
            headers <- list(Authorization = paste("Bearer", private$token),
                            "Content-Type" = "application/json")
            queryArgs <- list(filters = filters, where = where,
                              order = order, select = select, distinct = distinct,
                              limit = limit, offset = offset, count = count,
                              clusterID = clusterID, bypassFederation = bypassFederation)

            body <- NULL

            response <- private$REST$http$exec("GET", url, headers, body,
                                               queryArgs, private$numRetries)
            resource <- private$REST$httpParser$parseJSONResponse(response)

            if(!is.null(resource$errors))
                stop(resource$errors)

            resource
        },

        #' @description
        #' api_client_authorizations_get is a method defined in Arvados class.
        #' @param uuid The UUID of the apiClientAuthorization in question.
        api_client_authorizations_get = function(uuid)
        {
            endPoint <- stringr::str_interp("api_client_authorizations/${uuid}")
            url <- paste0(private$host, endPoint)
            headers <- list(Authorization = paste("Bearer", private$token),
                            "Content-Type" = "application/json")
            queryArgs <- NULL

            body <- NULL

            response <- private$REST$http$exec("GET", url, headers, body,
                                               queryArgs, private$numRetries)
            resource <- private$REST$httpParser$parseJSONResponse(response)

            if(!is.null(resource$errors))
                stop(resource$errors)

            resource
        },

        #' @description
        #' api_client_authorizations_create is a method defined in Arvados class.
        #' @param apiClientAuthorization apiClientAuthorization object.
        #' @param ensureUniqueName Adjust name to ensure uniqueness instead of returning an error on (ownerUUID, name) collision_
        #' @param clusterID Create object on a remote federated cluster instead of the current one.
        api_client_authorizations_create = function(apiClientAuthorization,
                                                    ensureUniqueName = "false", clusterID = NULL)
        {
            endPoint <- stringr::str_interp("api_client_authorizations")
            url <- paste0(private$host, endPoint)
            headers <- list(Authorization = paste("Bearer", private$token),
                            "Content-Type" = "application/json")
            queryArgs <- list(ensureUniqueName = ensureUniqueName,
                              clusterID = clusterID)

            if(length(apiClientAuthorization) > 0)
                body <- jsonlite::toJSON(list(apiClientAuthorization = apiClientAuthorization),
                                         auto_unbox = TRUE)
            else
                body <- NULL

            response <- private$REST$http$exec("POST", url, headers, body,
                                               queryArgs, private$numRetries)
            resource <- private$REST$httpParser$parseJSONResponse(response)

            if(!is.null(resource$errors))
                stop(resource$errors)

            resource
        },

        #' @description
        #' api_client_authorizations_update is a method defined in Arvados class.
        #' @param apiClientAuthorization apiClientAuthorization object.
        #' @param uuid The UUID of the apiClientAuthorization in question.
        api_client_authorizations_update = function(apiClientAuthorization, uuid)
        {
            endPoint <- stringr::str_interp("api_client_authorizations/${uuid}")
            url <- paste0(private$host, endPoint)
            headers <- list(Authorization = paste("Bearer", private$token),
                            "Content-Type" = "application/json")
            queryArgs <- NULL

            if(length(apiClientAuthorization) > 0)
                body <- jsonlite::toJSON(list(apiClientAuthorization = apiClientAuthorization),
                                         auto_unbox = TRUE)
            else
                body <- NULL

            response <- private$REST$http$exec("PUT", url, headers, body,
                                               queryArgs, private$numRetries)
            resource <- private$REST$httpParser$parseJSONResponse(response)

            if(!is.null(resource$errors))
                stop(resource$errors)

            resource
        },

        #' @description
        #' api_client_authorizations_delete is a method defined in Arvados class.
        #' @param uuid The UUID of the apiClientAuthorization in question.
        api_client_authorizations_delete = function(uuid)
        {
            endPoint <- stringr::str_interp("api_client_authorizations/${uuid}")
            url <- paste0(private$host, endPoint)
            headers <- list(Authorization = paste("Bearer", private$token),
                            "Content-Type" = "application/json")
            queryArgs <- NULL

            body <- NULL

            response <- private$REST$http$exec("DELETE", url, headers, body,
                                               queryArgs, private$numRetries)
            resource <- private$REST$httpParser$parseJSONResponse(response)

            if(!is.null(resource$errors))
                stop(resource$errors)

            resource
        },

        #' @description
        #' api_client_authorizations_create_system_auth is a method defined in Arvados class.
        #' @param apiClientID
        #' @param scopes
        api_client_authorizations_create_system_auth = function(apiClientID = NULL, scopes = NULL)
        {
            endPoint <- stringr::str_interp("api_client_authorizations/create_system_auth")
            url <- paste0(private$host, endPoint)
            headers <- list(Authorization = paste("Bearer", private$token),
                            "Content-Type" = "application/json")
            queryArgs <- list(apiClientID = apiClientID,
                              scopes = scopes)

            body <- NULL

            response <- private$REST$http$exec("POST", url, headers, body,
                                               queryArgs, private$numRetries)
            resource <- private$REST$httpParser$parseJSONResponse(response)

            if(!is.null(resource$errors))
                stop(resource$errors)

            resource
        },

        #' @description
        #' api_client_authorizations_current is a method defined in Arvados class.
        api_client_authorizations_current = function()
        {
            endPoint <- stringr::str_interp("api_client_authorizations/current")
            url <- paste0(private$host, endPoint)
            headers <- list(Authorization = paste("Bearer", private$token),
                            "Content-Type" = "application/json")
            queryArgs <- NULL

            body <- NULL

            response <- private$REST$http$exec("GET", url, headers, body,
                                               queryArgs, private$numRetries)
            resource <- private$REST$httpParser$parseJSONResponse(response)

            if(!is.null(resource$errors))
                stop(resource$errors)

            resource
        },

        #' @description
        #' api_client_authorizations_list is a method defined in Arvados class.
        #' @param filters
        #' @param where
        #' @param order
        #' @param select
        #' @param distinct
        #' @param limit
        #' @param offset
        #' @param count
        #' @param clusterID List objects on a remote federated cluster instead of the current one.
        #' @param bypassFederation Bypass federation behavior, list items from local instance database only.
        api_client_authorizations_list = function(filters = NULL,
                                                  where = NULL, order = NULL, select = NULL,
                                                  distinct = NULL, limit = "100", offset = "0",
                                                  count = "exact", clusterID = NULL, bypassFederation = NULL)
        {
            endPoint <- stringr::str_interp("api_client_authorizations")
            url <- paste0(private$host, endPoint)
            headers <- list(Authorization = paste("Bearer", private$token),
                            "Content-Type" = "application/json")
            queryArgs <- list(filters = filters, where = where,
                              order = order, select = select, distinct = distinct,
                              limit = limit, offset = offset, count = count,
                              clusterID = clusterID, bypassFederation = bypassFederation)

            body <- NULL

            response <- private$REST$http$exec("GET", url, headers, body,
                                               queryArgs, private$numRetries)
            resource <- private$REST$httpParser$parseJSONResponse(response)

            if(!is.null(resource$errors))
                stop(resource$errors)

            resource
        },

        #' @description
        #' authorized_keys_get is a method defined in Arvados class.
        #' @param uuid The UUID of the authorizedKey in question.
        authorized_keys_get = function(uuid)
        {
            endPoint <- stringr::str_interp("authorized_keys/${uuid}")
            url <- paste0(private$host, endPoint)
            headers <- list(Authorization = paste("Bearer", private$token),
                            "Content-Type" = "application/json")
            queryArgs <- NULL

            body <- NULL

            response <- private$REST$http$exec("GET", url, headers, body,
                                               queryArgs, private$numRetries)
            resource <- private$REST$httpParser$parseJSONResponse(response)

            if(!is.null(resource$errors))
                stop(resource$errors)

            resource
        },

        #' @description
        #' authorized_keys_create is a method defined in Arvados class.
        #' @param authorizedKey authorizedKey object.
        #' @param ensureUniqueName Adjust name to ensure uniqueness instead of returning an error.
        #' @param clusterID Create object on a remote federated cluster instead of the current one.
        authorized_keys_create = function(authorizedKey,
                                          ensureUniqueName = "false", clusterID = NULL)
        {
            endPoint <- stringr::str_interp("authorized_keys")
            url <- paste0(private$host, endPoint)
            headers <- list(Authorization = paste("Bearer", private$token),
                            "Content-Type" = "application/json")
            queryArgs <- list(ensureUniqueName = ensureUniqueName,
                              clusterID = clusterID)

            if(length(authorizedKey) > 0)
                body <- jsonlite::toJSON(list(authorizedKey = authorizedKey),
                                         auto_unbox = TRUE)
            else
                body <- NULL

            response <- private$REST$http$exec("POST", url, headers, body,
                                               queryArgs, private$numRetries)
            resource <- private$REST$httpParser$parseJSONResponse(response)

            if(!is.null(resource$errors))
                stop(resource$errors)

            resource
        },

        #' @description
        #' authorized_keys_update is a method defined in Arvados class.
        #' @param authorizedKey authorizedKey object.
        #' @param uuid The UUID of the authorizedKey in question.
        authorized_keys_update = function(authorizedKey, uuid)
        {
            endPoint <- stringr::str_interp("authorized_keys/${uuid}")
            url <- paste0(private$host, endPoint)
            headers <- list(Authorization = paste("Bearer", private$token),
                            "Content-Type" = "application/json")
            queryArgs <- NULL

            if(length(authorizedKey) > 0)
                body <- jsonlite::toJSON(list(authorizedKey = authorizedKey),
                                         auto_unbox = TRUE)
            else
                body <- NULL

            response <- private$REST$http$exec("PUT", url, headers, body,
                                               queryArgs, private$numRetries)
            resource <- private$REST$httpParser$parseJSONResponse(response)

            if(!is.null(resource$errors))
                stop(resource$errors)

            resource
        },

        #' @description
        #' authorized_keys_delete is a method defined in Arvados class.
        #' @param uuid The UUID of the authorizedKey in question.
        authorized_keys_delete = function(uuid)
        {
            endPoint <- stringr::str_interp("authorized_keys/${uuid}")
            url <- paste0(private$host, endPoint)
            headers <- list(Authorization = paste("Bearer", private$token),
                            "Content-Type" = "application/json")
            queryArgs <- NULL

            body <- NULL

            response <- private$REST$http$exec("DELETE", url, headers, body,
                                               queryArgs, private$numRetries)
            resource <- private$REST$httpParser$parseJSONResponse(response)

            if(!is.null(resource$errors))
                stop(resource$errors)

            resource
        },

        #' @description
        #' authorized_keys_list is a method defined in Arvados class.
        #' @param filters
        #' @param where
        #' @param order
        #' @param select
        #' @param distinct
        #' @param limit
        #' @param offset
        #' @param count
        #' @param clusterID List objects on a remote federated cluster instead of the current one.
        #' @param bypassFederation Bypass federation behavior, list items from local instance database only.
        authorized_keys_list = function(filters = NULL,
                                        where = NULL, order = NULL, select = NULL,
                                        distinct = NULL, limit = "100", offset = "0",
                                        count = "exact", clusterID = NULL, bypassFederation = NULL)
        {
            endPoint <- stringr::str_interp("authorized_keys")
            url <- paste0(private$host, endPoint)
            headers <- list(Authorization = paste("Bearer", private$token),
                            "Content-Type" = "application/json")
            queryArgs <- list(filters = filters, where = where,
                              order = order, select = select, distinct = distinct,
                              limit = limit, offset = offset, count = count,
                              clusterID = clusterID, bypassFederation = bypassFederation)

            body <- NULL

            response <- private$REST$http$exec("GET", url, headers, body,
                                               queryArgs, private$numRetries)
            resource <- private$REST$httpParser$parseJSONResponse(response)

            if(!is.null(resource$errors))
                stop(resource$errors)

            resource
        },

        #' @description
        #' collections_get is a method defined in Arvados class.
        #' @param uuid The UUID of the Collection in question.
        #' collection <- arv$collections_get(uuid = collectionUUID)
        collections_get = function(uuid)
        {
            endPoint <- stringr::str_interp("collections/${uuid}")
            url <- paste0(private$host, endPoint)
            headers <- list(Authorization = paste("Bearer", private$token),
                            "Content-Type" = "application/json")
            queryArgs <- NULL

            body <- NULL

            response <- private$REST$http$exec("GET", url, headers, body,
                                               queryArgs, private$numRetries)
            resource <- private$REST$httpParser$parseJSONResponse(response)

            if(!is.null(resource$errors))
                stop(resource$errors)

            resource
        },

        #' @description
        #' collections_create is a method defined in Arvados class that enables collections creation.
        #' @param name Name of the collection.
        #' @param description Description of the collection.
        #' @param ownerUUID UUID of the maternal project to created one.
        #' @param properties Properties of the collection.
        #' @param ensureUniqueName Adjust name to ensure uniqueness instead of returning an error.
        #' @param clusterID Create object on a remote federated cluster instead of the current one.
        #' @examples
        #' Properties <- list() # should contain a list of new properties to be added
        #' arv$collections_create(name = "collectionTitle", description = "collectionDescription", ownerUUID = "collectionOwner", properties = Properties)
        collections_create = function(name, description, ownerUUID = NULL, properties = NULL, # name and description are obligatory
                                      ensureUniqueName = "false", clusterID = NULL)
        {
            endPoint <- stringr::str_interp("collections")
            url <- paste0(private$host, endPoint)
            headers <- list(Authorization = paste("Bearer", private$token),
                            "Content-Type" = "application/json")
            queryArgs <- list(ensureUniqueName = ensureUniqueName,
                              clusterID = clusterID)

            collection <- list(name = name, description = description, owner_uuid = ownerUUID, properties = properties)
            if(length(collection) > 0)
                body <- jsonlite::toJSON(list(collection = collection),
                                         auto_unbox = TRUE)
            else
                body <- NULL

            response <- private$REST$http$exec("POST", url, headers, body,
                                               queryArgs, private$numRetries)
            resource <- private$REST$httpParser$parseJSONResponse(response)

            if(!is.null(resource$errors)){
                if(identical(sub('Entity:.*',"", resource$errors), "//railsapi.internal/arvados/v1/collections: 422 Unprocessable ")){
                    resource <- cat(format("A collection with the given name already exists in this projects. If you want to update it use collections_update() instead"))
                }else{
                    stop(resource$errors)
                }
            }

            resource
        },

        #' @description
        #' collections_update is a method defined in Arvados class.
        #' @param name New name of the collection.
        #' @param description New description of the collection.
        #' @param ownerUUID UUID of the maternal project to created one.
        #' @param properties New list of properties of the collection.
        #' @param uuid The UUID of the Collection in question.
        #' @examples
        #' collection <- arv$collections_update(name = "newCollectionTitle", description = "newCollectionDescription", ownerUUID = "collectionOwner", properties = NULL, uuid = "collectionUUID")
        collections_update = function(name, description, ownerUUID = NULL, properties = NULL, uuid)
        {
            endPoint <- stringr::str_interp("collections/${uuid}")
            url <- paste0(private$host, endPoint)
            headers <- list(Authorization = paste("Bearer", private$token),
                            "Content-Type" = "application/json")
            queryArgs <- NULL

            collection <- list(name = name, description = description, ownerUUID = ownerUUID, properties = properties)
            if(length(collection) > 0)
                body <- jsonlite::toJSON(list(collection = collection),
                                         auto_unbox = TRUE)
            else
                body <- NULL

            response <- private$REST$http$exec("PUT", url, headers, body,
                                               queryArgs, private$numRetries)
            resource <- private$REST$httpParser$parseJSONResponse(response)

            if(!is.null(resource$errors))
                stop(resource$errors)

            resource
        },

        #' @description
        #' collections_delete is a method defined in Arvados class.
        #' @param uuid The UUID of the Collection in question.
        #' @examples
        #' arv$collection_delete(collectionUUID)
        collections_delete = function(uuid)
        {
            endPoint <- stringr::str_interp("collections/${uuid}")
            url <- paste0(private$host, endPoint)
            headers <- list(Authorization = paste("Bearer", private$token),
                            "Content-Type" = "application/json")
            queryArgs <- NULL

            body <- NULL

            response <- private$REST$http$exec("DELETE", url, headers, body,
                                               queryArgs, private$numRetries)
            resource <- private$REST$httpParser$parseJSONResponse(response)

            if(!is.null(resource$errors))
                stop(resource$errors)

            resource
        },

        #' @description
        #' collections_provenance is a method defined in Arvados class, it returns the collection by uuid.
        #' @param uuid The UUID of the Collection in question.
        #' @examples
        #' collection <- arv$collections_provenance(collectionUUID)
        collections_provenance = function(uuid)
        {
            endPoint <- stringr::str_interp("collections/${uuid}/provenance")
            url <- paste0(private$host, endPoint)
            headers <- list(Authorization = paste("Bearer", private$token),
                            "Content-Type" = "application/json")
            queryArgs <- NULL

            body <- NULL

            response <- private$REST$http$exec("GET", url, headers, body,
                                               queryArgs, private$numRetries)
            resource <- private$REST$httpParser$parseJSONResponse(response)

            if(!is.null(resource$errors))
                stop(resource$errors)

            resource
        },

        #' @description
        #' collections_used_by is a method defined in Arvados class, it returns collection by portable_data_hash.
        #' @param uuid The UUID of the Collection in question.
        collections_used_by = function(uuid)
        {
            endPoint <- stringr::str_interp("collections/${uuid}/used_by")
            url <- paste0(private$host, endPoint)
            headers <- list(Authorization = paste("Bearer", private$token),
                            "Content-Type" = "application/json")
            queryArgs <- NULL

            body <- NULL

            response <- private$REST$http$exec("GET", url, headers, body,
                                               queryArgs, private$numRetries)
            resource <- private$REST$httpParser$parseJSONResponse(response)

            if(!is.null(resource$errors))
                stop(resource$errors)

            resource
        },

        #' @description
        #' collections_trash is a method defined in Arvados class, it moves collection to trash.
        #' @param uuid The UUID of the Collection in question.
        #' @examples
        #' arv$collections_trash(collectionUUID)
        collections_trash = function(uuid)
        {
            endPoint <- stringr::str_interp("collections/${uuid}/trash")
            url <- paste0(private$host, endPoint)
            headers <- list(Authorization = paste("Bearer", private$token),
                            "Content-Type" = "application/json")
            queryArgs <- NULL

            body <- NULL

            response <- private$REST$http$exec("POST", url, headers, body,
                                               queryArgs, private$numRetries)
            resource <- private$REST$httpParser$parseJSONResponse(response)

            if(!is.null(resource$errors))
                stop(resource$errors)

            resource
        },

        #' @description
        #' collections_untrash is a method defined in Arvados class, it moves collection from trash to project.
        #' @param uuid The UUID of the Collection in question.
        #' @examples
        #' arv$collections_untrash(collectionUUID)
        collections_untrash = function(uuid)
        {
            endPoint <- stringr::str_interp("collections/${uuid}/untrash")
            url <- paste0(private$host, endPoint)
            headers <- list(Authorization = paste("Bearer", private$token),
                            "Content-Type" = "application/json")
            queryArgs <- NULL

            body <- NULL

            response <- private$REST$http$exec("POST", url, headers, body,
                                               queryArgs, private$numRetries)
            resource <- private$REST$httpParser$parseJSONResponse(response)

            if(!is.null(resource$errors))
                stop(resource$errors)

            resource
        },

        #' @description
        #' collections_list is a method defined in Arvados class.
        #' @param filters
        #' @param where
        #' @param order
        #' @param select
        #' @param distinct
        #' @param limit
        #' @param offset
        #' @param count
        #' @param clusterID List objects on a remote federated cluster instead of the current one.
        #' @param bypassFederation Bypass federation behavior, list items from local instance database only.
        #' @param includeTrash Include collections whose is_trashed attribute is true.
        #' @param includeOldVersions Include past collection versions.
        #' @examples
        #' collectionList <- arv$collections_list(list(list("name", "=", "Example")))
        collections_list = function(filters = NULL,
                                    where = NULL, order = NULL, select = NULL,
                                    distinct = NULL, limit = "100", offset = "0",
                                    count = "exact", clusterID = NULL, bypassFederation = NULL,
                                    includeTrash = NULL, includeOldVersions = NULL)
        {
            endPoint <- stringr::str_interp("collections")
            url <- paste0(private$host, endPoint)
            headers <- list(Authorization = paste("Bearer", private$token),
                            "Content-Type" = "application/json")
            queryArgs <- list(filters = filters, where = where,
                              order = order, select = select, distinct = distinct,
                              limit = limit, offset = offset, count = count,
                              clusterID = clusterID, bypassFederation = bypassFederation,
                              includeTrash = includeTrash, includeOldVersions = includeOldVersions)

            body <- NULL

            response <- private$REST$http$exec("GET", url, headers, body,
                                               queryArgs, private$numRetries)
            resource <- private$REST$httpParser$parseJSONResponse(response)

            if(!is.null(resource$errors))
                stop(resource$errors)

            resource
        },

        #' @description
        #' containers_get is a method defined in Arvados class.
        #' @param uuid The UUID of the Container in question.
        containers_get = function(uuid)
        {
            endPoint <- stringr::str_interp("containers/${uuid}")
            url <- paste0(private$host, endPoint)
            headers <- list(Authorization = paste("Bearer", private$token),
                            "Content-Type" = "application/json")
            queryArgs <- NULL

            body <- NULL

            response <- private$REST$http$exec("GET", url, headers, body,
                                               queryArgs, private$numRetries)
            resource <- private$REST$httpParser$parseJSONResponse(response)

            if(!is.null(resource$errors))
                stop(resource$errors)

            resource
        },

        #' @description
        #' containers_create is a method defined in Arvados class.
        #' @param container Container object.
        #' @param ensureUniqueName Adjust name to ensure uniqueness instead of returning an error.
        #' @param clusterID Create object on a remote federated cluster instead of the current one.
        containers_create = function(container, ensureUniqueName = "false",
                                     clusterID = NULL)
        {
            endPoint <- stringr::str_interp("containers")
            url <- paste0(private$host, endPoint)
            headers <- list(Authorization = paste("Bearer", private$token),
                            "Content-Type" = "application/json")
            queryArgs <- list(ensureUniqueName = ensureUniqueName,
                              clusterID = clusterID)

            if(length(container) > 0)
                body <- jsonlite::toJSON(list(container = container),
                                         auto_unbox = TRUE)
            else
                body <- NULL

            response <- private$REST$http$exec("POST", url, headers, body,
                                               queryArgs, private$numRetries)
            resource <- private$REST$httpParser$parseJSONResponse(response)

            if(!is.null(resource$errors))
                stop(resource$errors)

            resource
        },

        #' @description
        #' containers_update is a method defined in Arvados class.
        #' @param container Container object.
        #' @param uuid The UUID of the Container in question.
        containers_update = function(container, uuid)
        {
            endPoint <- stringr::str_interp("containers/${uuid}")
            url <- paste0(private$host, endPoint)
            headers <- list(Authorization = paste("Bearer", private$token),
                            "Content-Type" = "application/json")
            queryArgs <- NULL

            if(length(container) > 0)
                body <- jsonlite::toJSON(list(container = container),
                                         auto_unbox = TRUE)
            else
                body <- NULL

            response <- private$REST$http$exec("PUT", url, headers, body,
                                               queryArgs, private$numRetries)
            resource <- private$REST$httpParser$parseJSONResponse(response)

            if(!is.null(resource$errors))
                stop(resource$errors)

            resource
        },

        #' @description
        #' containers_delete is a method defined in Arvados class.
        #' @param uuid The UUID of the Container in question.
        containers_delete = function(uuid)
        {
            endPoint <- stringr::str_interp("containers/${uuid}")
            url <- paste0(private$host, endPoint)
            headers <- list(Authorization = paste("Bearer", private$token),
                            "Content-Type" = "application/json")
            queryArgs <- NULL

            body <- NULL

            response <- private$REST$http$exec("DELETE", url, headers, body,
                                               queryArgs, private$numRetries)
            resource <- private$REST$httpParser$parseJSONResponse(response)

            if(!is.null(resource$errors))
                stop(resource$errors)

            resource
        },

        #' @description
        #' containers_auth is a method defined in Arvados class.
        #' @param uuid The UUID of the Container in question.
        containers_auth = function(uuid)
        {
            endPoint <- stringr::str_interp("containers/${uuid}/auth")
            url <- paste0(private$host, endPoint)
            headers <- list(Authorization = paste("Bearer", private$token),
                            "Content-Type" = "application/json")
            queryArgs <- NULL

            body <- NULL

            response <- private$REST$http$exec("GET", url, headers, body,
                                               queryArgs, private$numRetries)
            resource <- private$REST$httpParser$parseJSONResponse(response)

            if(!is.null(resource$errors))
                stop(resource$errors)

            resource
        },

        #' @description
        #' containers_lock is a method defined in Arvados class.
        #' @param uuid The UUID of the Container in question.
        containers_lock = function(uuid)
        {
            endPoint <- stringr::str_interp("containers/${uuid}/lock")
            url <- paste0(private$host, endPoint)
            headers <- list(Authorization = paste("Bearer", private$token),
                            "Content-Type" = "application/json")
            queryArgs <- NULL

            body <- NULL

            response <- private$REST$http$exec("POST", url, headers, body,
                                               queryArgs, private$numRetries)
            resource <- private$REST$httpParser$parseJSONResponse(response)

            if(!is.null(resource$errors))
                stop(resource$errors)

            resource
        },

        #' @description
        #' containers_unlock is a method defined in Arvados class.
        #' @param uuid The UUID of the Container in question.
        containers_unlock = function(uuid)
        {
            endPoint <- stringr::str_interp("containers/${uuid}/unlock")
            url <- paste0(private$host, endPoint)
            headers <- list(Authorization = paste("Bearer", private$token),
                            "Content-Type" = "application/json")
            queryArgs <- NULL

            body <- NULL

            response <- private$REST$http$exec("POST", url, headers, body,
                                               queryArgs, private$numRetries)
            resource <- private$REST$httpParser$parseJSONResponse(response)

            if(!is.null(resource$errors))
                stop(resource$errors)

            resource
        },

        #' @description
        #' containers_secret_mounts is a method defined in Arvados class.
        #' @param uuid The UUID of the Container in question.
        containers_secret_mounts = function(uuid)
        {
            endPoint <- stringr::str_interp("containers/${uuid}/secret_mounts")
            url <- paste0(private$host, endPoint)
            headers <- list(Authorization = paste("Bearer", private$token),
                            "Content-Type" = "application/json")
            queryArgs <- NULL

            body <- NULL

            response <- private$REST$http$exec("GET", url, headers, body,
                                               queryArgs, private$numRetries)
            resource <- private$REST$httpParser$parseJSONResponse(response)

            if(!is.null(resource$errors))
                stop(resource$errors)

            resource
        },

        #' @description
        #' containers_current is a method defined in Arvados class.
        containers_current = function()
        {
            endPoint <- stringr::str_interp("containers/current")
            url <- paste0(private$host, endPoint)
            headers <- list(Authorization = paste("Bearer", private$token),
                            "Content-Type" = "application/json")
            queryArgs <- NULL

            body <- NULL

            response <- private$REST$http$exec("GET", url, headers, body,
                                               queryArgs, private$numRetries)
            resource <- private$REST$httpParser$parseJSONResponse(response)

            if(!is.null(resource$errors))
                stop(resource$errors)

            resource
        },

        #' @description
        #' containers_list is a method defined in Arvados class.
        #' @param filters
        #' @param where
        #' @param order
        #' @param select
        #' @param distinct
        #' @param limit
        #' @param offset
        #' @param count
        #' @param clusterID List objects on a remote federated cluster instead of the current one.
        #' @param bypassFederation Bypass federation behavior, list items from local instance database only.
        containers_list = function(filters = NULL,
                                   where = NULL, order = NULL, select = NULL,
                                   distinct = NULL, limit = "100", offset = "0",
                                   count = "exact", clusterID = NULL, bypassFederation = NULL)
        {
            endPoint <- stringr::str_interp("containers")
            url <- paste0(private$host, endPoint)
            headers <- list(Authorization = paste("Bearer", private$token),
                            "Content-Type" = "application/json")
            queryArgs <- list(filters = filters, where = where,
                              order = order, select = select, distinct = distinct,
                              limit = limit, offset = offset, count = count,
                              clusterID = clusterID, bypassFederation = bypassFederation)

            body <- NULL

            response <- private$REST$http$exec("GET", url, headers, body,
                                               queryArgs, private$numRetries)
            resource <- private$REST$httpParser$parseJSONResponse(response)

            if(!is.null(resource$errors))
                stop(resource$errors)

            resource
        },

        #' @description
        #' container_requests_get is a method defined in Arvados class.
        #' @param uuid The UUID of the containerRequest in question.
        container_requests_get = function(uuid)
        {
            endPoint <- stringr::str_interp("container_requests/${uuid}")
            url <- paste0(private$host, endPoint)
            headers <- list(Authorization = paste("Bearer", private$token),
                            "Content-Type" = "application/json")
            queryArgs <- NULL

            body <- NULL

            response <- private$REST$http$exec("GET", url, headers, body,
                                               queryArgs, private$numRetries)
            resource <- private$REST$httpParser$parseJSONResponse(response)

            if(!is.null(resource$errors))
                stop(resource$errors)

            resource
        },

        #' @description
        #' container_requests_create is a method defined in Arvados class.
        #' @param containerRequest containerRequest object.
        #' @param ensureUniqueName Adjust name to ensure uniqueness instead of returning an error.
        #' @param clusterID Create object on a remote federated cluster instead of the current one.
        container_requests_create = function(containerRequest,
                                             ensureUniqueName = "false", clusterID = NULL)
        {
            endPoint <- stringr::str_interp("container_requests")
            url <- paste0(private$host, endPoint)
            headers <- list(Authorization = paste("Bearer", private$token),
                            "Content-Type" = "application/json")
            queryArgs <- list(ensureUniqueName = ensureUniqueName,
                              clusterID = clusterID)

            if(length(containerRequest) > 0)
                body <- jsonlite::toJSON(list(containerRequest = containerRequest),
                                         auto_unbox = TRUE)
            else
                body <- NULL

            response <- private$REST$http$exec("POST", url, headers, body,
                                               queryArgs, private$numRetries)
            resource <- private$REST$httpParser$parseJSONResponse(response)

            if(!is.null(resource$errors))
                stop(resource$errors)

            resource
        },

        #' @description
        #' container_requests_update is a method defined in Arvados class.
        #' @param containerRequest containerRequest object.
        #' @param uuid The UUID of the containerRequest in question.
        container_requests_update = function(containerRequest, uuid)
        {
            endPoint <- stringr::str_interp("container_requests/${uuid}")
            url <- paste0(private$host, endPoint)
            headers <- list(Authorization = paste("Bearer", private$token),
                            "Content-Type" = "application/json")
            queryArgs <- NULL

            if(length(containerRequest) > 0)
                body <- jsonlite::toJSON(list(containerRequest = containerRequest),
                                         auto_unbox = TRUE)
            else
                body <- NULL

            response <- private$REST$http$exec("PUT", url, headers, body,
                                               queryArgs, private$numRetries)
            resource <- private$REST$httpParser$parseJSONResponse(response)

            if(!is.null(resource$errors))
                stop(resource$errors)

            resource
        },

        #' @description
        #' container_requests_delete is a method defined in Arvados class.
        #' @param uuid The UUID of the containerRequest in question.
        container_requests_delete = function(uuid)
        {
            endPoint <- stringr::str_interp("container_requests/${uuid}")
            url <- paste0(private$host, endPoint)
            headers <- list(Authorization = paste("Bearer", private$token),
                            "Content-Type" = "application/json")
            queryArgs <- NULL

            body <- NULL

            response <- private$REST$http$exec("DELETE", url, headers, body,
                                               queryArgs, private$numRetries)
            resource <- private$REST$httpParser$parseJSONResponse(response)

            if(!is.null(resource$errors))
                stop(resource$errors)

            resource
        },

        #' @description
        #' container_requests_list is a method defined in Arvados class.
        #' @param filters
        #' @param where
        #' @param order
        #' @param select
        #' @param distinct
        #' @param limit
        #' @param offset
        #' @param count
        #' @param clusterID List objects on a remote federated cluster instead of the current one.
        #' @param bypassFederation bypass federation behavior, list items from local instance database only
        #' @param includeTrash Include container requests whose owner project is trashed.
        container_requests_list = function(filters = NULL,
                                           where = NULL, order = NULL, select = NULL,
                                           distinct = NULL, limit = "100", offset = "0",
                                           count = "exact", clusterID = NULL, bypassFederation = NULL,
                                           includeTrash = NULL)
        {
            endPoint <- stringr::str_interp("container_requests")
            url <- paste0(private$host, endPoint)
            headers <- list(Authorization = paste("Bearer", private$token),
                            "Content-Type" = "application/json")
            queryArgs <- list(filters = filters, where = where,
                              order = order, select = select, distinct = distinct,
                              limit = limit, offset = offset, count = count,
                              clusterID = clusterID, bypassFederation = bypassFederation,
                              includeTrash = includeTrash)

            body <- NULL

            response <- private$REST$http$exec("GET", url, headers, body,
                                               queryArgs, private$numRetries)
            resource <- private$REST$httpParser$parseJSONResponse(response)

            if(!is.null(resource$errors))
                stop(resource$errors)

            resource
        },

        #' @description
        #' groups_get is a method defined in Arvados class.
        #' @param uuid The UUID of the Group in question.
        groups_get = function(uuid)
        {
            endPoint <- stringr::str_interp("groups/${uuid}")
            url <- paste0(private$host, endPoint)
            headers <- list(Authorization = paste("Bearer", private$token),
                            "Content-Type" = "application/json")

            queryArgs <- NULL

            body <- NULL

            response <- private$REST$http$exec("GET", url, headers, body,
                                               queryArgs, private$numRetries)
            resource <- private$REST$httpParser$parseJSONResponse(response)

            if(!is.null(resource$errors))
                stop(resource$errors)

            resource
        },

        #' @description
        #' groups_create is a method defined in Arvados class that supports project creation.
        #' @param group Group object.
        #' @param ensureUniqueName Adjust name to ensure uniqueness instead of returning an error.
        #' @param clusterID Create object on a remote federated cluster instead of the current one.
        #' @param async Defer permissions update.
        groups_create = function(group, ensureUniqueName = "false",
                                 clusterID = NULL, async = "false")
        {
            endPoint <- stringr::str_interp("groups")
            url <- paste0(private$host, endPoint)
            headers <- list(Authorization = paste("Bearer", private$token),
                            "Content-Type" = "application/json")

            queryArgs <- list(ensureUniqueName = ensureUniqueName,
                              clusterID = clusterID, async = async)

            if(length(group) > 0)
                body <- jsonlite::toJSON(list(group = group),
                                         auto_unbox = TRUE)
            else
                body <- NULL

            response <- private$REST$http$exec("POST", url, headers, body,
                                               queryArgs, private$numRetries)

            resource <- private$REST$httpParser$parseJSONResponse(response)

            if(!is.null(resource$errors)){
                if (identical(sub('#.*', "", resource$errors), "//railsapi.internal/arvados/v1/groups: 422 Unprocessable Entity: ")) {
                #if (identical(sub('P.*', "", resource$errors), "//railsapi.internal/arvados/v1/groups: 422 Unprocessable Entity: #\u003cActiveRecord::RecordNotUnique: ")) {
                    resource <- cat(format("Project of that name already exist. If you want to update it use project_update() instead"))
                }else{
                    stop(resource$errors)
                }
            }

            return(resource)
        },

        #' @description
        #' groups_update is a method defined in Arvados class.
        #' @param group Group object.
        #' @param uuid The UUID of the Group in question.
        #' @param async Defer permissions update.
        groups_update = function(group, uuid, async = "false")
        {
            endPoint <- stringr::str_interp("groups/${uuid}")
            url <- paste0(private$host, endPoint)
            headers <- list(Authorization = paste("Bearer", private$token),
                            "Content-Type" = "application/json")

            queryArgs <- list(async = async)

            if(length(group) > 0)
                body <- jsonlite::toJSON(list(group = group),
                                         auto_unbox = TRUE)
            else
                body <- NULL

            response <- private$REST$http$exec("PUT", url, headers, body,
                                               queryArgs, private$numRetries)
            resource <- private$REST$httpParser$parseJSONResponse(response)

            if(!is.null(resource$errors))
                stop(resource$errors)

            resource
        },

        #' @description
        #' groups_delete is a method defined in Arvados class.
        #' @param uuid The UUID of the Group in question.
        groups_delete = function(uuid)
        {
            endPoint <- stringr::str_interp("groups/${uuid}")
            url <- paste0(private$host, endPoint)
            headers <- list(Authorization = paste("Bearer", private$token),
                            "Content-Type" = "application/json")

            queryArgs <- NULL

            body <- NULL

            response <- private$REST$http$exec("DELETE", url, headers, body,
                                               queryArgs, private$numRetries)
            resource <- private$REST$httpParser$parseJSONResponse(response)

            if(!is.null(resource$errors))
                stop(resource$errors)

            dataTime <- gsub("T.*", "", resource$delete_at)
            cat("The content will be deleted permanently at", dataTime)

            resource
        },

        #' @description
        #' groups_contents is a method defined in Arvados class.
        #' @param filters
        #' @param where
        #' @param order
        #' @param distinct
        #' @param limit
        #' @param offset
        #' @param count
        #' @param clusterID List objects on a remote federated cluster instead of the current one.
        #' @param bypassFederation Bypass federation behavior, list items from local instance database only.
        #' @param includeTrash Include items whose is_trashed attribute is true.
        #' @param uuid
        #' @param recursive Include contents from child groups recursively.
        #' @param include Include objects referred to by listed field in "included" (only ownerUUID).
        groups_contents = function(filters = NULL,
                                   where = NULL, order = NULL, distinct = NULL,
                                   limit = "100", offset = "0", count = "exact",
                                   clusterID = NULL, bypassFederation = NULL,
                                   includeTrash = NULL, uuid = NULL, recursive = NULL,
                                   include = NULL)
        {
            endPoint <- stringr::str_interp("groups/contents")
            url <- paste0(private$host, endPoint)
            headers <- list(Authorization = paste("Bearer", private$token),
                            "Content-Type" = "application/json")

            queryArgs <- list(filters = filters, where = where,
                              order = order, distinct = distinct, limit = limit,
                              offset = offset, count = count, clusterID = clusterID,
                              bypassFederation = bypassFederation, includeTrash = includeTrash,
                              uuid = uuid, recursive = recursive, include = include)

            body <- NULL

            response <- private$REST$http$exec("GET", url, headers, body,
                                               queryArgs, private$numRetries)
            resource <- private$REST$httpParser$parseJSONResponse(response)

            if(!is.null(resource$errors))
                stop(resource$errors)

            resource
        },

        #' @description
        #' groups_shared is a method defined in Arvados class.
        #' @param filters
        #' @param where
        #' @param order
        #' @param select
        #' @param distinct
        #' @param limit
        #' @param offset
        #' @param count
        #' @param clusterID List objects on a remote federated cluster instead of the current one.
        #' @param bypassFederation Bypass federation behavior, list items from local instance database only.
        #' @param includeTrash Include items whose is_trashed attribute is true.
        #' @param include
        groups_shared = function(filters = NULL,
                                 where = NULL, order = NULL, select = NULL,
                                 distinct = NULL, limit = "100", offset = "0",
                                 count = "exact", clusterID = NULL, bypassFederation = NULL,
                                 includeTrash = NULL, include = NULL)
        {
            endPoint <- stringr::str_interp("groups/shared")
            url <- paste0(private$host, endPoint)
            headers <- list(Authorization = paste("Bearer", private$token),
                            "Content-Type" = "application/json")

            queryArgs <- list(filters = filters, where = where,
                              order = order, select = select, distinct = distinct,
                              limit = limit, offset = offset, count = count,
                              clusterID = clusterID, bypassFederation = bypassFederation,
                              includeTrash = includeTrash, include = include)

            body <- NULL

            response <- private$REST$http$exec("GET", url, headers, body,
                                               queryArgs, private$numRetries)
            resource <- private$REST$httpParser$parseJSONResponse(response)

            if(!is.null(resource$errors))
                stop(resource$errors)

            resource
        },

        #' @description
        #' groups_trash is a method defined in Arvados class.
        #' @param uuid The UUID of the Group in question.
        groups_trash = function(uuid)
        {
            endPoint <- stringr::str_interp("groups/${uuid}/trash")
            url <- paste0(private$host, endPoint)
            headers <- list(Authorization = paste("Bearer", private$token),
                            "Content-Type" = "application/json")

            queryArgs <- NULL

            body <- NULL

            response <- private$REST$http$exec("POST", url, headers, body,
                                               queryArgs, private$numRetries)
            resource <- private$REST$httpParser$parseJSONResponse(response)

            if(!is.null(resource$errors))
                stop(resource$errors)

            resource
        },

        #' @description
        #' groups_untrash is a method defined in Arvados class.
        #' @param uuid The UUID of the Group in question.
        groups_untrash = function(uuid)
        {
            endPoint <- stringr::str_interp("groups/${uuid}/untrash")
            url <- paste0(private$host, endPoint)
            headers <- list(Authorization = paste("Bearer", private$token),
                            "Content-Type" = "application/json")

            queryArgs <- NULL

            body <- NULL

            response <- private$REST$http$exec("POST", url, headers, body,
                                               queryArgs, private$numRetries)
            resource <- private$REST$httpParser$parseJSONResponse(response)

            if(!is.null(resource$errors))
                stop(resource$errors)

            resource
        },

        #' @description
        #' groups_list is a method defined in Arvados class.
        #' @param filters
        #' @param where
        #' @param order
        #' @param select
        #' @param distinct
        #' @param limit
        #' @param offset
        #' @param count
        #' @param clusterID List objects on a remote federated cluster instead of the current one.
        #' @param bypassFederation Bypass federation behavior, list items from local instance database only.
        #' @param includeTrash Include items whose is_trashed attribute is true.
        groups_list = function(filters = NULL, where = NULL,
                               order = NULL, select = NULL, distinct = NULL,
                               limit = "100", offset = "0", count = "exact",
                               clusterID = NULL, bypassFederation = NULL,
                               includeTrash = NULL)
        {
            endPoint <- stringr::str_interp("groups")
            url <- paste0(private$host, endPoint)
            headers <- list(Authorization = paste("Bearer", private$token),
                            "Content-Type" = "application/json")

            queryArgs <- list(filters = filters, where = where,
                              order = order, select = select, distinct = distinct,
                              limit = limit, offset = offset, count = count,
                              clusterID = clusterID, bypassFederation = bypassFederation,
                              includeTrash = includeTrash)

            body <- NULL

            response <- private$REST$http$exec("GET", url, headers, body,
                                               queryArgs, private$numRetries)
            resource <- private$REST$httpParser$parseJSONResponse(response)

            if(!is.null(resource$errors))
                stop(resource$errors)

            resource
        },

        #' @description
        #' keep_services_get is a method defined in Arvados class.
        #' @param uuid The UUID of the keepService in question.
        keep_services_get = function(uuid)
        {
            endPoint <- stringr::str_interp("keep_services/${uuid}")
            url <- paste0(private$host, endPoint)
            headers <- list(Authorization = paste("Bearer", private$token),
                            "Content-Type" = "application/json")
            queryArgs <- NULL

            body <- NULL

            response <- private$REST$http$exec("GET", url, headers, body,
                                               queryArgs, private$numRetries)
            resource <- private$REST$httpParser$parseJSONResponse(response)

            if(!is.null(resource$errors))
                stop(resource$errors)

            resource
        },

        #' @description
        #' keep_services_create is a method defined in Arvados class.
        #' @param keepService keepService object.
        #' @param ensureUniqueName Adjust name to ensure uniqueness instead of returning an error.
        #' @param clusterID Create object on a remote federated cluster instead of the current one.
        keep_services_create = function(keepService,
                                        ensureUniqueName = "false", clusterID = NULL)
        {
            endPoint <- stringr::str_interp("keep_services")
            url <- paste0(private$host, endPoint)
            headers <- list(Authorization = paste("Bearer", private$token),
                            "Content-Type" = "application/json")
            queryArgs <- list(ensureUniqueName = ensureUniqueName,
                              clusterID = clusterID)

            if(length(keepService) > 0)
                body <- jsonlite::toJSON(list(keepService = keepService),
                                         auto_unbox = TRUE)
            else
                body <- NULL

            response <- private$REST$http$exec("POST", url, headers, body,
                                               queryArgs, private$numRetries)
            resource <- private$REST$httpParser$parseJSONResponse(response)

            if(!is.null(resource$errors))
                stop(resource$errors)

            resource
        },

        #' @description
        #' keep_services_update is a method defined in Arvados class.
        #' @param keepService keepService object.
        #' @param uuid The UUID of the keepService in question.
        keep_services_update = function(keepService, uuid)
        {
            endPoint <- stringr::str_interp("keep_services/${uuid}")
            url <- paste0(private$host, endPoint)
            headers <- list(Authorization = paste("Bearer", private$token),
                            "Content-Type" = "application/json")
            queryArgs <- NULL

            if(length(keepService) > 0)
                body <- jsonlite::toJSON(list(keepService = keepService),
                                         auto_unbox = TRUE)
            else
                body <- NULL

            response <- private$REST$http$exec("PUT", url, headers, body,
                                               queryArgs, private$numRetries)
            resource <- private$REST$httpParser$parseJSONResponse(response)

            if(!is.null(resource$errors))
                stop(resource$errors)

            resource
        },

        #' @description
        #' keep_services_delete is a method defined in Arvados class.
        #' @param uuid The UUID of the keepService in question.
        keep_services_delete = function(uuid)
        {
            endPoint <- stringr::str_interp("keep_services/${uuid}")
            url <- paste0(private$host, endPoint)
            headers <- list(Authorization = paste("Bearer", private$token),
                            "Content-Type" = "application/json")
            queryArgs <- NULL

            body <- NULL

            response <- private$REST$http$exec("DELETE", url, headers, body,
                                               queryArgs, private$numRetries)
            resource <- private$REST$httpParser$parseJSONResponse(response)

            if(!is.null(resource$errors))
                stop(resource$errors)

            resource
        },

        #' @description
        #' keep_services_accessible is a method defined in Arvados class.
        keep_services_accessible = function()
        {
            endPoint <- stringr::str_interp("keep_services/accessible")
            url <- paste0(private$host, endPoint)
            headers <- list(Authorization = paste("Bearer", private$token),
                            "Content-Type" = "application/json")
            queryArgs <- NULL

            body <- NULL

            response <- private$REST$http$exec("GET", url, headers, body,
                                               queryArgs, private$numRetries)
            resource <- private$REST$httpParser$parseJSONResponse(response)

            if(!is.null(resource$errors))
                stop(resource$errors)

            resource
        },

        #' @description
        #' keep_services_list is a method defined in Arvados class.
        #' @param filters
        #' @param where
        #' @param order
        #' @param select
        #' @param distinct
        #' @param limit
        #' @param offset
        #' @param count
        #' @param clusterID List objects on a remote federated cluster instead of the current one.
        #' @param bypassFederation Bypass federation behavior, list items from local instance database only.
        keep_services_list = function(filters = NULL,
                                      where = NULL, order = NULL, select = NULL,
                                      distinct = NULL, limit = "100", offset = "0",
                                      count = "exact", clusterID = NULL, bypassFederation = NULL)
        {
            endPoint <- stringr::str_interp("keep_services")
            url <- paste0(private$host, endPoint)
            headers <- list(Authorization = paste("Bearer", private$token),
                            "Content-Type" = "application/json")
            queryArgs <- list(filters = filters, where = where,
                              order = order, select = select, distinct = distinct,
                              limit = limit, offset = offset, count = count,
                              clusterID = clusterID, bypassFederation = bypassFederation)

            body <- NULL

            response <- private$REST$http$exec("GET", url, headers, body,
                                               queryArgs, private$numRetries)
            resource <- private$REST$httpParser$parseJSONResponse(response)

            if(!is.null(resource$errors))
                stop(resource$errors)

            resource
        },

        #' @description
        #' project_permission_give is a method defined in Arvados class that enables sharing files with another users.
        #' @param type Possible options are can_read or can_write or can_manage.
        #' @param uuid The UUID of a project or a file.
        #' @param user The UUID of the person that gets the permission.
        #' @examples
        #' arv$project_permission_give(type = "can_read", uuid = objectUUID, user = userUUID)
        project_permission_give = function(type, uuid, user)
        {
            endPoint <- stringr::str_interp("links")
            url <- paste0(private$host, endPoint)
            headers <- list(Authorization = paste("Bearer", private$token),
                            "Content-Type" = "application/json")
            queryArgs <- NULL

            # it is possible to make it as pasting a list to function, not a 3 arg. What's better?
            link <- list("link_class" = "permission", "name" = type, "head_uuid" = uuid, "tail_uuid" = user)

            if(length(link) > 0)
                body <- jsonlite::toJSON(list(link = link),
                                         auto_unbox = TRUE)
            else
                body <- NULL

            response <- private$REST$http$exec("POST", url, headers, body,
                                               queryArgs, private$numRetries)
            resource <- private$REST$httpParser$parseJSONResponse(response)

            if(!is.null(resource$errors))
                stop(resource$errors)

            resource
        },

        #' @description
        #' project_permission_refuse is a method defined in Arvados class that unables sharing files with another users.
        #' @param type Possible options are can_read or can_write or can_manage.
        #' @param uuid The UUID of a project or a file.
        #' @param user The UUID of a person that permissions are taken from.
        #' @examples
        #' arv$project_permission_refuse(type = "can_read", uuid = objectUUID, user = userUUID)
        project_permission_refuse = function(type, uuid, user)
        {
            examples <- self$links_list(list(list("head_uuid","=", uuid)))

            theUser <- examples[which(sapply(examples$items, "[[", "tail_uuid") == user)]
            theType <- theUser$items[which(sapply(theUser$items, "[[", "name") == type)]
            solution <- theType[which(sapply(theType, "[[", "link_class") == 'permission')]

            if (length(solution) == 0) {
                cat(format('No permission granted'))
            } else {
                self$links_delete(solution[[1]]$uuid)
            }

        },

        #' @description
        #' project_permission_update is a method defined in Arvados class that enables updating permissions.
        #' @param typeNew New option like can_read or can_write or can_manage.
        #' @param typeOld Old option.
        #' @param uuid The UUID of a project or a file.
        #' @param user The UUID of the person that the permission is being updated.
        #' @examples
        #' arv$project_permission_update(typeOld = "can_read", typeNew = "can_write", uuid = objectUUID, user = userUUID)
        project_permission_update = function(typeOld, typeNew, uuid, user)
        {
            link <- list("name" = typeNew)

            examples <- self$links_list(list(list("head_uuid","=", uuid)))

            theUser <- examples[which(sapply(examples$items, "[[", "tail_uuid") == user)]
            theType <- theUser$items[which(sapply(theUser$items, "[[", "name") == typeOld)]
            solution <- theType[which(sapply(theType, "[[", "link_class") == 'permission')]

            if (length(solution) == 0) {
                cat(format('No permission granted'))
            } else {
                self$links_update(link, solution[[1]]$uuid)
            }
        },

        #' @description
        #' project_permission_check is a method defined in Arvados class that enables checking file permissions.
        #' @param uuid The UUID of a project or a file.
        #' @param user The UUID of the person that the permission is being updated.
        #' @param type Possible options are can_read or can_write or can_manage.
        #' @examples
        #' arv$project_permission_check(type = "can_read", uuid = objectUUID, user = userUUID)
        project_permission_check = function(uuid, user, type = NULL)
        {
            examples <- self$links_list(list(list("head_uuid","=", uuid)))

            theUser <- examples[which(sapply(examples$items, "[[", "tail_uuid") == user)]

            if (length(type) == 0 ){
                theUser
            } else {
                theType <- theUser$items[which(sapply(theUser$items, "[[", "name") == type)]
                permisions <- theType[which(sapply(theType, "[[", "link_class") == 'permission')]
                print(permisions[[1]]$name)
            }
        },

        #' @description
        #' links_get is a method defined in Arvados class.
        #' @param uuid The UUID of the Link in question.
        links_get = function(uuid)
        {
            endPoint <- stringr::str_interp("links/${uuid}")
            url <- paste0(private$host, endPoint)
            headers <- list(Authorization = paste("Bearer", private$token),
                            "Content-Type" = "application/json")
            queryArgs <- NULL

            body <- NULL

            response <- private$REST$http$exec("GET", url, headers, body,
                                               queryArgs, private$numRetries)
            resource <- private$REST$httpParser$parseJSONResponse(response)

            if(!is.null(resource$errors))
                stop(resource$errors)

            resource
        },

        #' @description
        #' links_create is a method defined in Arvados class.
        #' @param link Link object.
        #' @param ensureUniqueName Adjust name to ensure uniqueness instead of returning an error.
        #' @param clusterID Create object on a remote federated cluster instead of the current one.
        links_create = function(link, ensureUniqueName = "false",
                                clusterID = NULL)
        {
            endPoint <- stringr::str_interp("links")
            url <- paste0(private$host, endPoint)
            headers <- list(Authorization = paste("Bearer", private$token),
                            "Content-Type" = "application/json")
            queryArgs <- list(ensureUniqueName = ensureUniqueName,
                              clusterID = clusterID)

            if(length(link) > 0)
                body <- jsonlite::toJSON(list(link = link),
                                         auto_unbox = TRUE)
            else
                body <- NULL

            response <- private$REST$http$exec("POST", url, headers, body,
                                               queryArgs, private$numRetries)
            resource <- private$REST$httpParser$parseJSONResponse(response)

            if(!is.null(resource$errors))
                stop(resource$errors)

            resource
        },

        #' @description
        #' links_update is a method defined in Arvados class.
        #' @param link Link object.
        #' @param uuid The UUID of the Link in question.
        links_update = function(link, uuid, async = "false")
        {
            endPoint <- stringr::str_interp("links/${uuid}")
            url <- paste0(private$host, endPoint)
            headers <- list(Authorization = paste("Bearer", private$token),
                            "Content-Type" = "application/json")
            queryArgs <- list(async = async)

            if(length(link) > 0)
                body <- jsonlite::toJSON(list(link = link),
                                         auto_unbox = TRUE)
            else
                body <- NULL

            response <- private$REST$http$exec("PUT", url, headers, body,
                                               queryArgs, private$numRetries)
            resource <- private$REST$httpParser$parseJSONResponse(response)

            if(!is.null(resource$errors))
                stop(resource$errors)

            resource
        },

        #' @description
        #' links_delete is a method defined in Arvados class.
        #' @param uuid The UUID of the Link in question.
        links_delete = function(uuid)
        {
            endPoint <- stringr::str_interp("links/${uuid}")
            url <- paste0(private$host, endPoint)
            headers <- list(Authorization = paste("Bearer", private$token),
                            "Content-Type" = "application/json")
            queryArgs <- NULL

            body <- NULL

            response <- private$REST$http$exec("DELETE", url, headers, body,
                                               queryArgs, private$numRetries)
            resource <- private$REST$httpParser$parseJSONResponse(response)

            if(!is.null(resource$errors))
                stop(resource$errors)

            resource
        },

        #' @description
        #' links_list is a method defined in Arvados class.
        #' @param filters
        #' @param where
        #' @param order
        #' @param select
        #' @param distinct
        #' @param limit
        #' @param offset
        #' @param count
        #' @param clusterID List objects on a remote federated cluster instead of the current one.
        #' @param bypassFederation Bypass federation behavior, list items from local instance database only.
        links_list = function(filters = NULL, where = NULL,
                              order = NULL, select = NULL, distinct = NULL,
                              limit = "100", offset = "0", count = "exact",
                              clusterID = NULL, bypassFederation = NULL)
        {
            endPoint <- stringr::str_interp("links")
            url <- paste0(private$host, endPoint)
            headers <- list(Authorization = paste("Bearer", private$token),
                            "Content-Type" = "application/json")
            queryArgs <- list(filters = filters, where = where,
                              order = order, select = select, distinct = distinct,
                              limit = limit, offset = offset, count = count,
                              clusterID = clusterID, bypassFederation = bypassFederation)

            body <- NULL

            response <- private$REST$http$exec("GET", url, headers, body,
                                               queryArgs, private$numRetries)
            resource <- private$REST$httpParser$parseJSONResponse(response)

            if(!is.null(resource$errors))
                stop(resource$errors)

            resource
        },

        #' @description
        #' links_get_permissions is a method defined in Arvados class.
        #' @param uuid The UUID of the Log in question.
        links_get_permissions = function(uuid)
        {
            endPoint <- stringr::str_interp("permissions/${uuid}")
            url <- paste0(private$host, endPoint)
            headers <- list(Authorization = paste("Bearer", private$token),
                            "Content-Type" = "application/json")
            queryArgs <- NULL

            body <- NULL

            response <- private$REST$http$exec("GET", url, headers, body,
                                               queryArgs, private$numRetries)
            resource <- private$REST$httpParser$parseJSONResponse(response)

            if(!is.null(resource$errors))
                stop(resource$errors)

            resource
        },

        #' @description
        #' logs_get is a method defined in Arvados class.
        #' @param uuid The UUID of the Log in question.
        logs_get = function(uuid)
        {
            endPoint <- stringr::str_interp("logs/${uuid}")
            url <- paste0(private$host, endPoint)
            headers <- list(Authorization = paste("Bearer", private$token),
                            "Content-Type" = "application/json")
            queryArgs <- NULL

            body <- NULL

            response <- private$REST$http$exec("GET", url, headers, body,
                                               queryArgs, private$numRetries)
            resource <- private$REST$httpParser$parseJSONResponse(response)

            if(!is.null(resource$errors))
                stop(resource$errors)

            resource
        },

        #' @description
        #' logs_create is a method defined in Arvados class.
        #' @param log Log object.
        #' @param ensureUniqueName Adjust name to ensure uniqueness instead of returning an error.
        #' @param clusterID Create object on a remote federated cluster instead of the current one.
        logs_create = function(log, ensureUniqueName = "false",
                               clusterID = NULL)
        {
            endPoint <- stringr::str_interp("logs")
            url <- paste0(private$host, endPoint)
            headers <- list(Authorization = paste("Bearer", private$token),
                            "Content-Type" = "application/json")
            queryArgs <- list(ensureUniqueName = ensureUniqueName,
                              clusterID = clusterID)

            if(length(log) > 0)
                body <- jsonlite::toJSON(list(log = log),
                                         auto_unbox = TRUE)
            else
                body <- NULL

            response <- private$REST$http$exec("POST", url, headers, body,
                                               queryArgs, private$numRetries)
            resource <- private$REST$httpParser$parseJSONResponse(response)

            if(!is.null(resource$errors))
                stop(resource$errors)

            resource
        },

        #' @description
        #' logs_update is a method defined in Arvados class.
        #' @param log Log object.
        #' @param uuid The UUID of the Log in question.
        logs_update = function(log, uuid)
        {
            endPoint <- stringr::str_interp("logs/${uuid}")
            url <- paste0(private$host, endPoint)
            headers <- list(Authorization = paste("Bearer", private$token),
                            "Content-Type" = "application/json")
            queryArgs <- NULL

            if(length(log) > 0)
                body <- jsonlite::toJSON(list(log = log),
                                         auto_unbox = TRUE)
            else
                body <- NULL

            response <- private$REST$http$exec("PUT", url, headers, body,
                                               queryArgs, private$numRetries)
            resource <- private$REST$httpParser$parseJSONResponse(response)

            if(!is.null(resource$errors))
                stop(resource$errors)

            resource
        },

        #' @description
        #' logs_delete is a method defined in Arvados class.
        #' @param uuid The UUID of the Log in question.
        logs_delete = function(uuid)
        {
            endPoint <- stringr::str_interp("logs/${uuid}")
            url <- paste0(private$host, endPoint)
            headers <- list(Authorization = paste("Bearer", private$token),
                            "Content-Type" = "application/json")
            queryArgs <- NULL

            body <- NULL

            response <- private$REST$http$exec("DELETE", url, headers, body,
                                               queryArgs, private$numRetries)
            resource <- private$REST$httpParser$parseJSONResponse(response)

            if(!is.null(resource$errors))
                stop(resource$errors)

            resource
        },

        #' @description
        #' logs_list is a method defined in Arvados class.
        #' @param filters
        #' @param where
        #' @param order
        #' @param select
        #' @param distinct
        #' @param limit
        #' @param offset
        #' @param count
        #' @param clusterID List objects on a remote federated cluster instead of the current one.
        #' @param bypassFederation Bypass federation behavior, list items from local instance database only.
        logs_list = function(filters = NULL, where = NULL,
                             order = NULL, select = NULL, distinct = NULL,
                             limit = "100", offset = "0", count = "exact",
                             clusterID = NULL, bypassFederation = NULL)
        {
            endPoint <- stringr::str_interp("logs")
            url <- paste0(private$host, endPoint)
            headers <- list(Authorization = paste("Bearer", private$token),
                            "Content-Type" = "application/json")
            queryArgs <- list(filters = filters, where = where,
                              order = order, select = select, distinct = distinct,
                              limit = limit, offset = offset, count = count,
                              clusterID = clusterID, bypassFederation = bypassFederation)

            body <- NULL

            response <- private$REST$http$exec("GET", url, headers, body,
                                               queryArgs, private$numRetries)
            resource <- private$REST$httpParser$parseJSONResponse(response)

            if(!is.null(resource$errors))
                stop(resource$errors)

            resource
        },

        #' @description
        #' users_get is a method defined in Arvados class.
        #' @param uuid The UUID of the User in question.
        users_get = function(uuid)
        {
            endPoint <- stringr::str_interp("users/${uuid}")
            url <- paste0(private$host, endPoint)
            headers <- list(Authorization = paste("Bearer", private$token),
                            "Content-Type" = "application/json")
            queryArgs <- NULL

            body <- NULL

            response <- private$REST$http$exec("GET", url, headers, body,
                                               queryArgs, private$numRetries)
            resource <- private$REST$httpParser$parseJSONResponse(response)

            if(!is.null(resource$errors))
                stop(resource$errors)

            resource
        },

        #' @description
        #' users_create is a method defined in Arvados class.
        #' @param user User object.
        #' @param ensureUniqueName Adjust name to ensure uniqueness instead of returning an error.
        #' @param clusterID Create object on a remote federated cluster instead of the current one.
        users_create = function(user, ensureUniqueName = "false",
                                clusterID = NULL)
        {
            endPoint <- stringr::str_interp("users")
            url <- paste0(private$host, endPoint)
            headers <- list(Authorization = paste("Bearer", private$token),
                            "Content-Type" = "application/json")
            queryArgs <- list(ensureUniqueName = ensureUniqueName,
                              clusterID = clusterID)

            if(length(user) > 0)
                body <- jsonlite::toJSON(list(user = user),
                                         auto_unbox = TRUE)
            else
                body <- NULL

            response <- private$REST$http$exec("POST", url, headers, body,
                                               queryArgs, private$numRetries)
            resource <- private$REST$httpParser$parseJSONResponse(response)

            if(!is.null(resource$errors))
                stop(resource$errors)

            resource
        },

        #' @description
        #' users_update is a method defined in Arvados class.
        #' @param user User object.
        #' @param uuid The UUID of the User in question.
        #' @param bypassFederation
        users_update = function(user, uuid, bypassFederation = NULL)
        {
            endPoint <- stringr::str_interp("users/${uuid}")
            url <- paste0(private$host, endPoint)
            headers <- list(Authorization = paste("Bearer", private$token),
                            "Content-Type" = "application/json")
            queryArgs <- list(bypassFederation = bypassFederation)

            if(length(user) > 0)
                body <- jsonlite::toJSON(list(user = user),
                                         auto_unbox = TRUE)
            else
                body <- NULL

            response <- private$REST$http$exec("PUT", url, headers, body,
                                               queryArgs, private$numRetries)
            resource <- private$REST$httpParser$parseJSONResponse(response)

            if(!is.null(resource$errors))
                stop(resource$errors)

            resource
        },

        #' @description
        #' users_delete is a method defined in Arvados class.
        #' @param uuid The UUID of the User in question.
        users_delete = function(uuid)
        {
            endPoint <- stringr::str_interp("users/${uuid}")
            url <- paste0(private$host, endPoint)
            headers <- list(Authorization = paste("Bearer", private$token),
                            "Content-Type" = "application/json")
            queryArgs <- NULL

            body <- NULL

            response <- private$REST$http$exec("DELETE", url, headers, body,
                                               queryArgs, private$numRetries)
            resource <- private$REST$httpParser$parseJSONResponse(response)

            if(!is.null(resource$errors))
                stop(resource$errors)

            resource
        },

        #' @description
        #' users_current is a method defined in Arvados class.
        users_current = function()
        {
            endPoint <- stringr::str_interp("users/current")
            url <- paste0(private$host, endPoint)
            headers <- list(Authorization = paste("Bearer", private$token),
                            "Content-Type" = "application/json")
            queryArgs <- NULL

            body <- NULL

            response <- private$REST$http$exec("GET", url, headers, body,
                                               queryArgs, private$numRetries)
            resource <- private$REST$httpParser$parseJSONResponse(response)

            if(!is.null(resource$errors))
                stop(resource$errors)

            resource
        },

        #' @description
        #' users_system is a method defined in Arvados class.
        users_system = function()
        {
            endPoint <- stringr::str_interp("users/system")
            url <- paste0(private$host, endPoint)
            headers <- list(Authorization = paste("Bearer", private$token),
                            "Content-Type" = "application/json")
            queryArgs <- NULL

            body <- NULL

            response <- private$REST$http$exec("GET", url, headers, body,
                                               queryArgs, private$numRetries)
            resource <- private$REST$httpParser$parseJSONResponse(response)

            if(!is.null(resource$errors))
                stop(resource$errors)

            resource
        },

        #' @description
        #' users_activate is a method defined in Arvados class.
        #' @param uuid The UUID of the User in question.
        users_activate = function(uuid)
        {
            endPoint <- stringr::str_interp("users/${uuid}/activate")
            url <- paste0(private$host, endPoint)
            headers <- list(Authorization = paste("Bearer", private$token),
                            "Content-Type" = "application/json")
            queryArgs <- NULL

            body <- NULL

            response <- private$REST$http$exec("POST", url, headers, body,
                                               queryArgs, private$numRetries)
            resource <- private$REST$httpParser$parseJSONResponse(response)

            if(!is.null(resource$errors))
                stop(resource$errors)

            resource
        },

        #' @description
        #' users_setup is a method defined in Arvados class.
        #' @param uuid
        #' @param user
        #' @param repo_name
        #' @param vm_uuid
        #' @param send_notification_email
        users_setup = function(uuid = NULL, user = NULL,
                               repo_name = NULL, vm_uuid = NULL, send_notification_email = "false")
        {
            endPoint <- stringr::str_interp("users/setup")
            url <- paste0(private$host, endPoint)
            headers <- list(Authorization = paste("Bearer", private$token),
                            "Content-Type" = "application/json")
            queryArgs <- list(uuid = uuid, user = user,
                              repo_name = repo_name, vm_uuid = vm_uuid,
                              send_notification_email = send_notification_email)

            body <- NULL

            response <- private$REST$http$exec("POST", url, headers, body,
                                               queryArgs, private$numRetries)
            resource <- private$REST$httpParser$parseJSONResponse(response)

            if(!is.null(resource$errors))
                stop(resource$errors)

            resource
        },

        #' @description
        #' users_unsetup is a method defined in Arvados class.
        #' @param uuid The UUID of the User in question.
        users_unsetup = function(uuid)
        {
            endPoint <- stringr::str_interp("users/${uuid}/unsetup")
            url <- paste0(private$host, endPoint)
            headers <- list(Authorization = paste("Bearer", private$token),
                            "Content-Type" = "application/json")
            queryArgs <- NULL

            body <- NULL

            response <- private$REST$http$exec("POST", url, headers, body,
                                               queryArgs, private$numRetries)
            resource <- private$REST$httpParser$parseJSONResponse(response)

            if(!is.null(resource$errors))
                stop(resource$errors)

            resource
        },

        #' @description
        #' users_merge is a method defined in Arvados class.
        #' @param newOwnerUUID
        #' @param newUserToken
        #' @param redirectToNewUser
        #' @param oldUserUUID
        #' @param newUserUUID
        users_merge = function(newOwnerUUID, newUserToken = NULL,
                               redirectToNewUser = NULL, oldUserUUID = NULL,
                               newUserUUID = NULL)
        {
            endPoint <- stringr::str_interp("users/merge")
            url <- paste0(private$host, endPoint)
            headers <- list(Authorization = paste("Bearer", private$token),
                            "Content-Type" = "application/json")
            queryArgs <- list(newOwnerUUID = newOwnerUUID,
                              newUserToken = newUserToken, redirectToNewUser = redirectToNewUser,
                              oldUserUUID = oldUserUUID, newUserUUID = newUserUUID)

            body <- NULL

            response <- private$REST$http$exec("POST", url, headers, body,
                                               queryArgs, private$numRetries)
            resource <- private$REST$httpParser$parseJSONResponse(response)

            if(!is.null(resource$errors))
                stop(resource$errors)

            resource
        },

        #' @description
        #' users_list is a method defined in Arvados class.
        #' @param filters
        #' @param where
        #' @param order
        #' @param select
        #' @param distinct
        #' @param limit
        #' @param offset
        #' @param count
        #' @param clusterID List objects on a remote federated cluster instead of the current one.
        #' @param bypassFederation Bypass federation behavior, list items from local instance database only.
        users_list = function(filters = NULL, where = NULL,
                              order = NULL, select = NULL, distinct = NULL,
                              limit = "100", offset = "0", count = "exact",
                              clusterID = NULL, bypassFederation = NULL)
        {
            endPoint <- stringr::str_interp("users")
            url <- paste0(private$host, endPoint)
            headers <- list(Authorization = paste("Bearer", private$token),
                            "Content-Type" = "application/json")
            queryArgs <- list(filters = filters, where = where,
                              order = order, select = select, distinct = distinct,
                              limit = limit, offset = offset, count = count,
                              clusterID = clusterID, bypassFederation = bypassFederation)

            body <- NULL

            response <- private$REST$http$exec("GET", url, headers, body,
                                               queryArgs, private$numRetries)
            resource <- private$REST$httpParser$parseJSONResponse(response)

            if(!is.null(resource$errors))
                stop(resource$errors)

            resource
        },

        #' @description
        #' repositories_get is a method defined in Arvados class.
        #' @param uuid The UUID of the Repository in question.
        repositories_get = function(uuid)
        {
            endPoint <- stringr::str_interp("repositories/${uuid}")
            url <- paste0(private$host, endPoint)
            headers <- list(Authorization = paste("Bearer", private$token),
                            "Content-Type" = "application/json")
            queryArgs <- NULL

            body <- NULL

            response <- private$REST$http$exec("GET", url, headers, body,
                                               queryArgs, private$numRetries)
            resource <- private$REST$httpParser$parseJSONResponse(response)

            if(!is.null(resource$errors))
                stop(resource$errors)

            resource
        },

        #' @description
        #' repositories_create is a method defined in Arvados class.
        #' @param repository Repository object.
        #' @param ensureUniqueName Adjust name to ensure uniqueness instead of returning an error.
        #' @param clusterID Create object on a remote federated cluster instead of the current one.
        repositories_create = function(repository,
                                       ensureUniqueName = "false", clusterID = NULL)
        {
            endPoint <- stringr::str_interp("repositories")
            url <- paste0(private$host, endPoint)
            headers <- list(Authorization = paste("Bearer", private$token),
                            "Content-Type" = "application/json")
            queryArgs <- list(ensureUniqueName = ensureUniqueName,
                              clusterID = clusterID)

            if(length(repository) > 0)
                body <- jsonlite::toJSON(list(repository = repository),
                                         auto_unbox = TRUE)
            else
                body <- NULL

            response <- private$REST$http$exec("POST", url, headers, body,
                                               queryArgs, private$numRetries)
            resource <- private$REST$httpParser$parseJSONResponse(response)

            if(!is.null(resource$errors))
                stop(resource$errors)

            resource
        },

        #' @description
        #' repositories_update is a method defined in Arvados class.
        #' @param repository Repository object.
        #' @param uuid The UUID of the Repository in question.
        repositories_update = function(repository, uuid)
        {
            endPoint <- stringr::str_interp("repositories/${uuid}")
            url <- paste0(private$host, endPoint)
            headers <- list(Authorization = paste("Bearer", private$token),
                            "Content-Type" = "application/json")
            queryArgs <- NULL

            if(length(repository) > 0)
                body <- jsonlite::toJSON(list(repository = repository),
                                         auto_unbox = TRUE)
            else
                body <- NULL

            response <- private$REST$http$exec("PUT", url, headers, body,
                                               queryArgs, private$numRetries)
            resource <- private$REST$httpParser$parseJSONResponse(response)

            if(!is.null(resource$errors))
                stop(resource$errors)

            resource
        },

        #' @description
        #' repositories_delete is a method defined in Arvados class.
        #' @param uuid The UUID of the Repository in question.
        repositories_delete = function(uuid)
        {
            endPoint <- stringr::str_interp("repositories/${uuid}")
            url <- paste0(private$host, endPoint)
            headers <- list(Authorization = paste("Bearer", private$token),
                            "Content-Type" = "application/json")
            queryArgs <- NULL

            body <- NULL

            response <- private$REST$http$exec("DELETE", url, headers, body,
                                               queryArgs, private$numRetries)
            resource <- private$REST$httpParser$parseJSONResponse(response)

            if(!is.null(resource$errors))
                stop(resource$errors)

            resource
        },

        #' @description
        #' repositories_get_all_permissions is a method defined in Arvados class.
        repositories_get_all_permissions = function()
        {
            endPoint <- stringr::str_interp("repositories/get_all_permissions")
            url <- paste0(private$host, endPoint)
            headers <- list(Authorization = paste("Bearer", private$token),
                            "Content-Type" = "application/json")
            queryArgs <- NULL

            body <- NULL

            response <- private$REST$http$exec("GET", url, headers, body,
                                               queryArgs, private$numRetries)
            resource <- private$REST$httpParser$parseJSONResponse(response)

            if(!is.null(resource$errors))
                stop(resource$errors)

            resource
        },

        #' @description
        #' repositories_list is a method defined in Arvados class.
        #' @param filters
        #' @param where
        #' @param order
        #' @param select
        #' @param distinct
        #' @param limit
        #' @param offset
        #' @param count
        #' @param clusterID List objects on a remote federated cluster instead of the current one.
        #' @param bypassFederation Bypass federation behavior, list items from local instance database only.
        repositories_list = function(filters = NULL,
                                     where = NULL, order = NULL, select = NULL,
                                     distinct = NULL, limit = "100", offset = "0",
                                     count = "exact", clusterID = NULL, bypassFederation = NULL)
        {
            endPoint <- stringr::str_interp("repositories")
            url <- paste0(private$host, endPoint)
            headers <- list(Authorization = paste("Bearer", private$token),
                            "Content-Type" = "application/json")
            queryArgs <- list(filters = filters, where = where,
                              order = order, select = select, distinct = distinct,
                              limit = limit, offset = offset, count = count,
                              clusterID = clusterID, bypassFederation = bypassFederation)

            body <- NULL

            response <- private$REST$http$exec("GET", url, headers, body,
                                               queryArgs, private$numRetries)
            resource <- private$REST$httpParser$parseJSONResponse(response)

            if(!is.null(resource$errors))
                stop(resource$errors)

            resource
        },

        #' @description
        #' virtual_machines_get is a method defined in Arvados class.
        #' @param uuid The UUID of the virtualMachine in question.
        virtual_machines_get = function(uuid)
        {
            endPoint <- stringr::str_interp("virtual_machines/${uuid}")
            url <- paste0(private$host, endPoint)
            headers <- list(Authorization = paste("Bearer", private$token),
                            "Content-Type" = "application/json")
            queryArgs <- NULL

            body <- NULL

            response <- private$REST$http$exec("GET", url, headers, body,
                                               queryArgs, private$numRetries)
            resource <- private$REST$httpParser$parseJSONResponse(response)

            if(!is.null(resource$errors))
                stop(resource$errors)

            resource
        },

        #' @description
        #' virtual_machines_create is a method defined in Arvados class.
        #' @param virtualMachine virtualMachine object.
        #' @param ensureUniqueName Adjust name to ensure uniqueness instead of returning an error.
        #' @param clusterID Create object on a remote federated cluster instead of the current one.
        virtual_machines_create = function(virtualMachine,
                                           ensureUniqueName = "false", clusterID = NULL)
        {
            endPoint <- stringr::str_interp("virtual_machines")
            url <- paste0(private$host, endPoint)
            headers <- list(Authorization = paste("Bearer", private$token),
                            "Content-Type" = "application/json")
            queryArgs <- list(ensureUniqueName = ensureUniqueName,
                              clusterID = clusterID)

            if(length(virtualMachine) > 0)
                body <- jsonlite::toJSON(list(virtualMachine = virtualMachine),
                                         auto_unbox = TRUE)
            else
                body <- NULL

            response <- private$REST$http$exec("POST", url, headers, body,
                                               queryArgs, private$numRetries)
            resource <- private$REST$httpParser$parseJSONResponse(response)

            if(!is.null(resource$errors))
                stop(resource$errors)

            resource
        },

        #' @description
        #' virtual_machines_update is a method defined in Arvados class.
        #' @param virtualMachine virtualMachine object.
        #' @param uuid The UUID of the virtualMachine in question.
        virtual_machines_update = function(virtualMachine, uuid)
        {
            endPoint <- stringr::str_interp("virtual_machines/${uuid}")
            url <- paste0(private$host, endPoint)
            headers <- list(Authorization = paste("Bearer", private$token),
                            "Content-Type" = "application/json")
            queryArgs <- NULL

            if(length(virtualMachine) > 0)
                body <- jsonlite::toJSON(list(virtualMachine = virtualMachine),
                                         auto_unbox = TRUE)
            else
                body <- NULL

            response <- private$REST$http$exec("PUT", url, headers, body,
                                               queryArgs, private$numRetries)
            resource <- private$REST$httpParser$parseJSONResponse(response)

            if(!is.null(resource$errors))
                stop(resource$errors)

            resource
        },

        #' @description
        #' virtual_machines_delete is a method defined in Arvados class.
        #' @param uuid The UUID of the virtualMachine in question.
        virtual_machines_delete = function(uuid)
        {
            endPoint <- stringr::str_interp("virtual_machines/${uuid}")
            url <- paste0(private$host, endPoint)
            headers <- list(Authorization = paste("Bearer", private$token),
                            "Content-Type" = "application/json")
            queryArgs <- NULL

            body <- NULL

            response <- private$REST$http$exec("DELETE", url, headers, body,
                                               queryArgs, private$numRetries)
            resource <- private$REST$httpParser$parseJSONResponse(response)

            if(!is.null(resource$errors))
                stop(resource$errors)

            resource
        },

        #' @description
        #' virtual_machines_logins is a method defined in Arvados class.
        #' @param uuid The UUID of the virtualMachine in question.
        virtual_machines_logins = function(uuid)
        {
            endPoint <- stringr::str_interp("virtual_machines/${uuid}/logins")
            url <- paste0(private$host, endPoint)
            headers <- list(Authorization = paste("Bearer", private$token),
                            "Content-Type" = "application/json")
            queryArgs <- NULL

            body <- NULL

            response <- private$REST$http$exec("GET", url, headers, body,
                                               queryArgs, private$numRetries)
            resource <- private$REST$httpParser$parseJSONResponse(response)

            if(!is.null(resource$errors))
                stop(resource$errors)

            resource
        },

        #' @description
        #' virtual_machines_get_all_logins is a method defined in Arvados class.
        virtual_machines_get_all_logins = function()
        {
            endPoint <- stringr::str_interp("virtual_machines/get_all_logins")
            url <- paste0(private$host, endPoint)
            headers <- list(Authorization = paste("Bearer", private$token),
                            "Content-Type" = "application/json")
            queryArgs <- NULL

            body <- NULL

            response <- private$REST$http$exec("GET", url, headers, body,
                                               queryArgs, private$numRetries)
            resource <- private$REST$httpParser$parseJSONResponse(response)

            if(!is.null(resource$errors))
                stop(resource$errors)

            resource
        },

        #' @description
        #' virtual_machines_list is a method defined in Arvados class.
        #' @param filters
        #' @param where
        #' @param order
        #' @param select
        #' @param distinct
        #' @param limit
        #' @param offset
        #' @param count
        #' @param clusterID List objects on a remote federated cluster instead of the current one.
        #' @param bypassFederation bypass federation behavior, list items from local instance database only
        virtual_machines_list = function(filters = NULL,
                                         where = NULL, order = NULL, select = NULL,
                                         distinct = NULL, limit = "100", offset = "0",
                                         count = "exact", clusterID = NULL, bypassFederation = NULL)
        {
            endPoint <- stringr::str_interp("virtual_machines")
            url <- paste0(private$host, endPoint)
            headers <- list(Authorization = paste("Bearer", private$token),
                            "Content-Type" = "application/json")
            queryArgs <- list(filters = filters, where = where,
                              order = order, select = select, distinct = distinct,
                              limit = limit, offset = offset, count = count,
                              clusterID = clusterID, bypassFederation = bypassFederation)

            body <- NULL

            response <- private$REST$http$exec("GET", url, headers, body,
                                               queryArgs, private$numRetries)
            resource <- private$REST$httpParser$parseJSONResponse(response)

            if(!is.null(resource$errors))
                stop(resource$errors)

            resource
        },

        #' @description
        #' workflows_get is a method defined in Arvados class.
        #' @param uuid The UUID of the Workflow in question.
        workflows_get = function(uuid)
        {
            endPoint <- stringr::str_interp("workflows/${uuid}")
            url <- paste0(private$host, endPoint)
            headers <- list(Authorization = paste("Bearer", private$token),
                            "Content-Type" = "application/json")
            queryArgs <- NULL

            body <- NULL

            response <- private$REST$http$exec("GET", url, headers, body,
                                               queryArgs, private$numRetries)
            resource <- private$REST$httpParser$parseJSONResponse(response)

            if(!is.null(resource$errors))
                stop(resource$errors)

            resource
        },

        #' @description
        #' workflows_create is a method defined in Arvados class.
        #' @param workflow Workflow object.
        #' @param ensureUniqueName Adjust name to ensure uniqueness instead of returning an error.
        #' @param clusterID Create object on a remote federated cluster instead of the current one.
        workflows_create = function(workflow, ensureUniqueName = "false",
                                    clusterID = NULL)
        {
            endPoint <- stringr::str_interp("workflows")
            url <- paste0(private$host, endPoint)
            headers <- list(Authorization = paste("Bearer", private$token),
                            "Content-Type" = "application/json")
            queryArgs <- list(ensureUniqueName = ensureUniqueName,
                              clusterID = clusterID)

            if(length(workflow) > 0)
                body <- jsonlite::toJSON(list(workflow = workflow),
                                         auto_unbox = TRUE)
            else
                body <- NULL

            response <- private$REST$http$exec("POST", url, headers, body,
                                               queryArgs, private$numRetries)
            resource <- private$REST$httpParser$parseJSONResponse(response)

            if(!is.null(resource$errors))
                stop(resource$errors)

            resource
        },

        #' @description
        #' workflows_update is a method defined in Arvados class.
        #' @param workflow Workflow object.
        #' @param uuid The UUID of the Workflow in question.
        workflows_update = function(workflow, uuid)
        {
            endPoint <- stringr::str_interp("workflows/${uuid}")
            url <- paste0(private$host, endPoint)
            headers <- list(Authorization = paste("Bearer", private$token),
                            "Content-Type" = "application/json")
            queryArgs <- NULL

            if(length(workflow) > 0)
                body <- jsonlite::toJSON(list(workflow = workflow),
                                         auto_unbox = TRUE)
            else
                body <- NULL

            response <- private$REST$http$exec("PUT", url, headers, body,
                                               queryArgs, private$numRetries)
            resource <- private$REST$httpParser$parseJSONResponse(response)

            if(!is.null(resource$errors))
                stop(resource$errors)

            resource
        },

        #' @description
        #' workflows_delete is a method defined in Arvados class.
        #' @param uuid The UUID of the Workflow in question.
        workflows_delete = function(uuid)
        {
            endPoint <- stringr::str_interp("workflows/${uuid}")
            url <- paste0(private$host, endPoint)
            headers <- list(Authorization = paste("Bearer", private$token),
                            "Content-Type" = "application/json")
            queryArgs <- NULL

            body <- NULL

            response <- private$REST$http$exec("DELETE", url, headers, body,
                                               queryArgs, private$numRetries)
            resource <- private$REST$httpParser$parseJSONResponse(response)

            if(!is.null(resource$errors))
                stop(resource$errors)

            resource
        },

        #' @description
        #' workflows_list is a method defined in Arvados class.
        #' @param filters
        #' @param where
        #' @param order
        #' @param select
        #' @param distinct
        #' @param limit
        #' @param offset
        #' @param count
        #' @param clusterID List objects on a remote federated cluster instead of the current one.
        #' @param bypassFederation Bypass federation behavior, list items from local instance database only.
        workflows_list = function(filters = NULL,
                                  where = NULL, order = NULL, select = NULL,
                                  distinct = NULL, limit = "100", offset = "0",
                                  count = "exact", clusterID = NULL, bypassFederation = NULL)
        {
            endPoint <- stringr::str_interp("workflows")
            url <- paste0(private$host, endPoint)
            headers <- list(Authorization = paste("Bearer", private$token),
                            "Content-Type" = "application/json")
            queryArgs <- list(filters = filters, where = where,
                              order = order, select = select, distinct = distinct,
                              limit = limit, offset = offset, count = count,
                              clusterID = clusterID, bypassFederation = bypassFederation)

            body <- NULL

            response <- private$REST$http$exec("GET", url, headers, body,
                                               queryArgs, private$numRetries)
            resource <- private$REST$httpParser$parseJSONResponse(response)

            if(!is.null(resource$errors))
                stop(resource$errors)

            resource
        },

        #' @description
        #' user_agreements_get is a method defined in Arvados class.
        #' @param uuid The UUID of the userAgreement in question.
        user_agreements_get = function(uuid)
        {
            endPoint <- stringr::str_interp("user_agreements/${uuid}")
            url <- paste0(private$host, endPoint)
            headers <- list(Authorization = paste("Bearer", private$token),
                            "Content-Type" = "application/json")
            queryArgs <- NULL

            body <- NULL

            response <- private$REST$http$exec("GET", url, headers, body,
                                               queryArgs, private$numRetries)
            resource <- private$REST$httpParser$parseJSONResponse(response)

            if(!is.null(resource$errors))
                stop(resource$errors)

            resource
        },

        #' @description
        #' user_agreements_create is a method defined in Arvados class.
        #' @param userAgreement userAgreement object.
        #' @param ensureUniqueName Adjust name to ensure uniqueness instead of returning an error.
        #' @param clusterID Create object on a remote federated cluster instead of the current one.
        user_agreements_create = function(userAgreement,
                                          ensureUniqueName = "false", clusterID = NULL)
        {
            endPoint <- stringr::str_interp("user_agreements")
            url <- paste0(private$host, endPoint)
            headers <- list(Authorization = paste("Bearer", private$token),
                            "Content-Type" = "application/json")
            queryArgs <- list(ensureUniqueName = ensureUniqueName,
                              clusterID = clusterID)

            if(length(userAgreement) > 0)
                body <- jsonlite::toJSON(list(userAgreement = userAgreement),
                                         auto_unbox = TRUE)
            else
                body <- NULL

            response <- private$REST$http$exec("POST", url, headers, body,
                                               queryArgs, private$numRetries)
            resource <- private$REST$httpParser$parseJSONResponse(response)

            if(!is.null(resource$errors))
                stop(resource$errors)

            resource
        },

        #' @description
        #' user_agreements_update is a method defined in Arvados class.
        #' @param userAgreement userAgreement object.
        #' @param uuid The UUID of the userAgreement in question.
        user_agreements_update = function(userAgreement, uuid)
        {
            endPoint <- stringr::str_interp("user_agreements/${uuid}")
            url <- paste0(private$host, endPoint)
            headers <- list(Authorization = paste("Bearer", private$token),
                            "Content-Type" = "application/json")
            queryArgs <- NULL

            if(length(userAgreement) > 0)
                body <- jsonlite::toJSON(list(userAgreement = userAgreement),
                                         auto_unbox = TRUE)
            else
                body <- NULL

            response <- private$REST$http$exec("PUT", url, headers, body,
                                               queryArgs, private$numRetries)
            resource <- private$REST$httpParser$parseJSONResponse(response)

            if(!is.null(resource$errors))
                stop(resource$errors)

            resource
        },

        #' @description
        #' user_agreements_delete is a method defined in Arvados class.
        #' @param uuid The UUID of the userAgreement in question.
        user_agreements_delete = function(uuid)
        {
            endPoint <- stringr::str_interp("user_agreements/${uuid}")
            url <- paste0(private$host, endPoint)
            headers <- list(Authorization = paste("Bearer", private$token),
                            "Content-Type" = "application/json")
            queryArgs <- NULL

            body <- NULL

            response <- private$REST$http$exec("DELETE", url, headers, body,
                                               queryArgs, private$numRetries)
            resource <- private$REST$httpParser$parseJSONResponse(response)

            if(!is.null(resource$errors))
                stop(resource$errors)

            resource
        },

        #' @description
        #' user_agreements_signatures is a method defined in Arvados class.
        user_agreements_signatures = function()
        {
            endPoint <- stringr::str_interp("user_agreements/signatures")
            url <- paste0(private$host, endPoint)
            headers <- list(Authorization = paste("Bearer", private$token),
                            "Content-Type" = "application/json")
            queryArgs <- NULL

            body <- NULL

            response <- private$REST$http$exec("GET", url, headers, body,
                                               queryArgs, private$numRetries)
            resource <- private$REST$httpParser$parseJSONResponse(response)

            if(!is.null(resource$errors))
                stop(resource$errors)

            resource
        },

        #' @description
        #' user_agreements_sign is a method defined in Arvados class.
        user_agreements_sign = function()
        {
            endPoint <- stringr::str_interp("user_agreements/sign")
            url <- paste0(private$host, endPoint)
            headers <- list(Authorization = paste("Bearer", private$token),
                            "Content-Type" = "application/json")
            queryArgs <- NULL

            body <- NULL

            response <- private$REST$http$exec("POST", url, headers, body,
                                               queryArgs, private$numRetries)
            resource <- private$REST$httpParser$parseJSONResponse(response)

            if(!is.null(resource$errors))
                stop(resource$errors)

            resource
        },

        #' @description
        #' user_agreements_list is a method defined in Arvados class.
        #' @param filters
        #' @param where
        #' @param order
        #' @param select
        #' @param distinct
        #' @param limit
        #' @param offset
        #' @param count
        #' @param clusterID List objects on a remote federated cluster instead of the current one.
        #' @param bypassFederation Bypass federation behavior, list items from local instance database only.
        user_agreements_list = function(filters = NULL,
                                        where = NULL, order = NULL, select = NULL,
                                        distinct = NULL, limit = "100", offset = "0",
                                        count = "exact", clusterID = NULL, bypassFederation = NULL)
        {
            endPoint <- stringr::str_interp("user_agreements")
            url <- paste0(private$host, endPoint)
            headers <- list(Authorization = paste("Bearer", private$token),
                            "Content-Type" = "application/json")
            queryArgs <- list(filters = filters, where = where,
                              order = order, select = select, distinct = distinct,
                              limit = limit, offset = offset, count = count,
                              clusterID = clusterID, bypassFederation = bypassFederation)

            body <- NULL

            response <- private$REST$http$exec("GET", url, headers, body,
                                               queryArgs, private$numRetries)
            resource <- private$REST$httpParser$parseJSONResponse(response)

            if(!is.null(resource$errors))
                stop(resource$errors)

            resource
        },

        #' @description
        #' user_agreements_new is a method defined in Arvados class.
        user_agreements_new = function()
        {
            endPoint <- stringr::str_interp("user_agreements/new")
            url <- paste0(private$host, endPoint)
            headers <- list(Authorization = paste("Bearer", private$token),
                            "Content-Type" = "application/json")
            queryArgs <- NULL

            body <- NULL

            response <- private$REST$http$exec("GET", url, headers, body,
                                               queryArgs, private$numRetries)
            resource <- private$REST$httpParser$parseJSONResponse(response)

            if(!is.null(resource$errors))
                stop(resource$errors)

            resource
        },

        #' @description
        #' configs_get is a method defined in Arvados class.
        configs_get = function()
        {
            endPoint <- stringr::str_interp("config")
            url <- paste0(private$host, endPoint)
            headers <- list(Authorization = paste("Bearer", private$token),
                            "Content-Type" = "application/json")
            queryArgs <- NULL=

                body <- NULL

            response <- private$REST$http$exec("GET", url, headers, body,
                                               queryArgs, private$numRetries)
            resource <- private$REST$httpParser$parseJSONResponse(response)

            if(!is.null(resource$errors))
                stop(resource$errors)

            resource
        },

        getHostName = function() private$host,
        getToken = function() private$token,
        setRESTService = function(newREST) private$REST <- newREST,
        getRESTService = function() private$REST
    ),

    private = list(

        token = NULL,
        host = NULL,
        REST = NULL,
        numRetries = NULL
    ),

    cloneable = FALSE
)


