# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: Apache-2.0

#' api_clients.get
#'
#' api_clients.get is a method defined in Arvados class.
#'
#' @usage arv$api_clients.get(uuid)
#' @param uuid The UUID of the ApiClient in question.
#' @return ApiClient object.
#' @name api_clients.get
NULL

#' api_clients.create
#'
#' api_clients.create is a method defined in Arvados class.
#'
#' @usage arv$api_clients.create(apiclient,
#' 	ensure_unique_name = "false", cluster_id = NULL)
#' @param apiClient ApiClient object.
#' @param ensure_unique_name Adjust name to ensure uniqueness instead of returning an error on (owner_uuid, name) collision.
#' @param cluster_id Create object on a remote federated cluster instead of the current one.
#' @return ApiClient object.
#' @name api_clients.create
NULL

#' api_clients.update
#'
#' api_clients.update is a method defined in Arvados class.
#'
#' @usage arv$api_clients.update(apiclient,
#' 	uuid)
#' @param apiClient ApiClient object.
#' @param uuid The UUID of the ApiClient in question.
#' @return ApiClient object.
#' @name api_clients.update
NULL

#' api_clients.delete
#'
#' api_clients.delete is a method defined in Arvados class.
#'
#' @usage arv$api_clients.delete(uuid)
#' @param uuid The UUID of the ApiClient in question.
#' @return ApiClient object.
#' @name api_clients.delete
NULL

#' api_clients.list
#'
#' api_clients.list is a method defined in Arvados class.
#'
#' @usage arv$api_clients.list(filters = NULL,
#' 	where = NULL, order = NULL, select = NULL,
#' 	distinct = NULL, limit = "100", offset = "0",
#' 	count = "exact", cluster_id = NULL, bypass_federation = NULL)
#' @param filters
#' @param where
#' @param order
#' @param select
#' @param distinct
#' @param limit
#' @param offset
#' @param count
#' @param cluster_id List objects on a remote federated cluster instead of the current one.
#' @param bypass_federation bypass federation behavior, list items from local instance database only
#' @return ApiClientList object.
#' @name api_clients.list
NULL

#' api_client_authorizations.get
#'
#' api_client_authorizations.get is a method defined in Arvados class.
#'
#' @usage arv$api_client_authorizations.get(uuid)
#' @param uuid The UUID of the ApiClientAuthorization in question.
#' @return ApiClientAuthorization object.
#' @name api_client_authorizations.get
NULL

#' api_client_authorizations.create
#'
#' api_client_authorizations.create is a method defined in Arvados class.
#'
#' @usage arv$api_client_authorizations.create(apiclientauthorization,
#' 	ensure_unique_name = "false", cluster_id = NULL)
#' @param apiClientAuthorization ApiClientAuthorization object.
#' @param ensure_unique_name Adjust name to ensure uniqueness instead of returning an error on (owner_uuid, name) collision.
#' @param cluster_id Create object on a remote federated cluster instead of the current one.
#' @return ApiClientAuthorization object.
#' @name api_client_authorizations.create
NULL

#' api_client_authorizations.update
#'
#' api_client_authorizations.update is a method defined in Arvados class.
#'
#' @usage arv$api_client_authorizations.update(apiclientauthorization,
#' 	uuid)
#' @param apiClientAuthorization ApiClientAuthorization object.
#' @param uuid The UUID of the ApiClientAuthorization in question.
#' @return ApiClientAuthorization object.
#' @name api_client_authorizations.update
NULL

#' api_client_authorizations.delete
#'
#' api_client_authorizations.delete is a method defined in Arvados class.
#'
#' @usage arv$api_client_authorizations.delete(uuid)
#' @param uuid The UUID of the ApiClientAuthorization in question.
#' @return ApiClientAuthorization object.
#' @name api_client_authorizations.delete
NULL

#' api_client_authorizations.create_system_auth
#'
#' api_client_authorizations.create_system_auth is a method defined in Arvados class.
#'
#' @usage arv$api_client_authorizations.create_system_auth(api_client_id = NULL,
#' 	scopes = NULL)
#' @param api_client_id
#' @param scopes
#' @return ApiClientAuthorization object.
#' @name api_client_authorizations.create_system_auth
NULL

#' api_client_authorizations.current
#'
#' api_client_authorizations.current is a method defined in Arvados class.
#'
#' @usage arv$api_client_authorizations.current(NULL)
#' @return ApiClientAuthorization object.
#' @name api_client_authorizations.current
NULL

#' api_client_authorizations.list
#'
#' api_client_authorizations.list is a method defined in Arvados class.
#'
#' @usage arv$api_client_authorizations.list(filters = NULL,
#' 	where = NULL, order = NULL, select = NULL,
#' 	distinct = NULL, limit = "100", offset = "0",
#' 	count = "exact", cluster_id = NULL, bypass_federation = NULL)
#' @param filters
#' @param where
#' @param order
#' @param select
#' @param distinct
#' @param limit
#' @param offset
#' @param count
#' @param cluster_id List objects on a remote federated cluster instead of the current one.
#' @param bypass_federation bypass federation behavior, list items from local instance database only
#' @return ApiClientAuthorizationList object.
#' @name api_client_authorizations.list
NULL

#' authorized_keys.get
#'
#' authorized_keys.get is a method defined in Arvados class.
#'
#' @usage arv$authorized_keys.get(uuid)
#' @param uuid The UUID of the AuthorizedKey in question.
#' @return AuthorizedKey object.
#' @name authorized_keys.get
NULL

#' authorized_keys.create
#'
#' authorized_keys.create is a method defined in Arvados class.
#'
#' @usage arv$authorized_keys.create(authorizedkey,
#' 	ensure_unique_name = "false", cluster_id = NULL)
#' @param authorizedKey AuthorizedKey object.
#' @param ensure_unique_name Adjust name to ensure uniqueness instead of returning an error on (owner_uuid, name) collision.
#' @param cluster_id Create object on a remote federated cluster instead of the current one.
#' @return AuthorizedKey object.
#' @name authorized_keys.create
NULL

#' authorized_keys.update
#'
#' authorized_keys.update is a method defined in Arvados class.
#'
#' @usage arv$authorized_keys.update(authorizedkey,
#' 	uuid)
#' @param authorizedKey AuthorizedKey object.
#' @param uuid The UUID of the AuthorizedKey in question.
#' @return AuthorizedKey object.
#' @name authorized_keys.update
NULL

#' authorized_keys.delete
#'
#' authorized_keys.delete is a method defined in Arvados class.
#'
#' @usage arv$authorized_keys.delete(uuid)
#' @param uuid The UUID of the AuthorizedKey in question.
#' @return AuthorizedKey object.
#' @name authorized_keys.delete
NULL

#' authorized_keys.list
#'
#' authorized_keys.list is a method defined in Arvados class.
#'
#' @usage arv$authorized_keys.list(filters = NULL,
#' 	where = NULL, order = NULL, select = NULL,
#' 	distinct = NULL, limit = "100", offset = "0",
#' 	count = "exact", cluster_id = NULL, bypass_federation = NULL)
#' @param filters
#' @param where
#' @param order
#' @param select
#' @param distinct
#' @param limit
#' @param offset
#' @param count
#' @param cluster_id List objects on a remote federated cluster instead of the current one.
#' @param bypass_federation bypass federation behavior, list items from local instance database only
#' @return AuthorizedKeyList object.
#' @name authorized_keys.list
NULL

#' collections.get
#'
#' collections.get is a method defined in Arvados class.
#'
#' @usage arv$collections.get(uuid)
#' @param uuid The UUID of the Collection in question.
#' @return Collection object.
#' @name collections.get
NULL

#' collections.create
#'
#' collections.create is a method defined in Arvados class.
#'
#' @usage arv$collections.create(collection,
#' 	ensure_unique_name = "false", cluster_id = NULL)
#' @param collection Collection object.
#' @param ensure_unique_name Adjust name to ensure uniqueness instead of returning an error on (owner_uuid, name) collision.
#' @param cluster_id Create object on a remote federated cluster instead of the current one.
#' @return Collection object.
#' @name collections.create
NULL

#' collections.update
#'
#' collections.update is a method defined in Arvados class.
#'
#' @usage arv$collections.update(collection,
#' 	uuid)
#' @param collection Collection object.
#' @param uuid The UUID of the Collection in question.
#' @return Collection object.
#' @name collections.update
NULL

#' collections.delete
#'
#' collections.delete is a method defined in Arvados class.
#'
#' @usage arv$collections.delete(uuid)
#' @param uuid The UUID of the Collection in question.
#' @return Collection object.
#' @name collections.delete
NULL

#' collections.provenance
#'
#' collections.provenance is a method defined in Arvados class.
#'
#' @usage arv$collections.provenance(uuid)
#' @param uuid
#' @return Collection object.
#' @name collections.provenance
NULL

#' collections.used_by
#'
#' collections.used_by is a method defined in Arvados class.
#'
#' @usage arv$collections.used_by(uuid)
#' @param uuid
#' @return Collection object.
#' @name collections.used_by
NULL

#' collections.trash
#'
#' collections.trash is a method defined in Arvados class.
#'
#' @usage arv$collections.trash(uuid)
#' @param uuid
#' @return Collection object.
#' @name collections.trash
NULL

#' collections.untrash
#'
#' collections.untrash is a method defined in Arvados class.
#'
#' @usage arv$collections.untrash(uuid)
#' @param uuid
#' @return Collection object.
#' @name collections.untrash
NULL

#' collections.list
#'
#' collections.list is a method defined in Arvados class.
#'
#' @usage arv$collections.list(filters = NULL,
#' 	where = NULL, order = NULL, select = NULL,
#' 	distinct = NULL, limit = "100", offset = "0",
#' 	count = "exact", cluster_id = NULL, bypass_federation = NULL,
#' 	include_trash = NULL, include_old_versions = NULL)
#' @param filters
#' @param where
#' @param order
#' @param select
#' @param distinct
#' @param limit
#' @param offset
#' @param count
#' @param cluster_id List objects on a remote federated cluster instead of the current one.
#' @param bypass_federation bypass federation behavior, list items from local instance database only
#' @param include_trash Include collections whose is_trashed attribute is true.
#' @param include_old_versions Include past collection versions.
#' @return CollectionList object.
#' @name collections.list
NULL

#' containers.get
#'
#' containers.get is a method defined in Arvados class.
#'
#' @usage arv$containers.get(uuid)
#' @param uuid The UUID of the Container in question.
#' @return Container object.
#' @name containers.get
NULL

#' containers.create
#'
#' containers.create is a method defined in Arvados class.
#'
#' @usage arv$containers.create(container,
#' 	ensure_unique_name = "false", cluster_id = NULL)
#' @param container Container object.
#' @param ensure_unique_name Adjust name to ensure uniqueness instead of returning an error on (owner_uuid, name) collision.
#' @param cluster_id Create object on a remote federated cluster instead of the current one.
#' @return Container object.
#' @name containers.create
NULL

#' containers.update
#'
#' containers.update is a method defined in Arvados class.
#'
#' @usage arv$containers.update(container,
#' 	uuid)
#' @param container Container object.
#' @param uuid The UUID of the Container in question.
#' @return Container object.
#' @name containers.update
NULL

#' containers.delete
#'
#' containers.delete is a method defined in Arvados class.
#'
#' @usage arv$containers.delete(uuid)
#' @param uuid The UUID of the Container in question.
#' @return Container object.
#' @name containers.delete
NULL

#' containers.auth
#'
#' containers.auth is a method defined in Arvados class.
#'
#' @usage arv$containers.auth(uuid)
#' @param uuid
#' @return Container object.
#' @name containers.auth
NULL

#' containers.lock
#'
#' containers.lock is a method defined in Arvados class.
#'
#' @usage arv$containers.lock(uuid)
#' @param uuid
#' @return Container object.
#' @name containers.lock
NULL

#' containers.unlock
#'
#' containers.unlock is a method defined in Arvados class.
#'
#' @usage arv$containers.unlock(uuid)
#' @param uuid
#' @return Container object.
#' @name containers.unlock
NULL

#' containers.secret_mounts
#'
#' containers.secret_mounts is a method defined in Arvados class.
#'
#' @usage arv$containers.secret_mounts(uuid)
#' @param uuid
#' @return Container object.
#' @name containers.secret_mounts
NULL

#' containers.current
#'
#' containers.current is a method defined in Arvados class.
#'
#' @usage arv$containers.current(NULL)
#' @return Container object.
#' @name containers.current
NULL

#' containers.list
#'
#' containers.list is a method defined in Arvados class.
#'
#' @usage arv$containers.list(filters = NULL,
#' 	where = NULL, order = NULL, select = NULL,
#' 	distinct = NULL, limit = "100", offset = "0",
#' 	count = "exact", cluster_id = NULL, bypass_federation = NULL)
#' @param filters
#' @param where
#' @param order
#' @param select
#' @param distinct
#' @param limit
#' @param offset
#' @param count
#' @param cluster_id List objects on a remote federated cluster instead of the current one.
#' @param bypass_federation bypass federation behavior, list items from local instance database only
#' @return ContainerList object.
#' @name containers.list
NULL

#' container_requests.get
#'
#' container_requests.get is a method defined in Arvados class.
#'
#' @usage arv$container_requests.get(uuid)
#' @param uuid The UUID of the ContainerRequest in question.
#' @return ContainerRequest object.
#' @name container_requests.get
NULL

#' container_requests.create
#'
#' container_requests.create is a method defined in Arvados class.
#'
#' @usage arv$container_requests.create(containerrequest,
#' 	ensure_unique_name = "false", cluster_id = NULL)
#' @param containerRequest ContainerRequest object.
#' @param ensure_unique_name Adjust name to ensure uniqueness instead of returning an error on (owner_uuid, name) collision.
#' @param cluster_id Create object on a remote federated cluster instead of the current one.
#' @return ContainerRequest object.
#' @name container_requests.create
NULL

#' container_requests.update
#'
#' container_requests.update is a method defined in Arvados class.
#'
#' @usage arv$container_requests.update(containerrequest,
#' 	uuid)
#' @param containerRequest ContainerRequest object.
#' @param uuid The UUID of the ContainerRequest in question.
#' @return ContainerRequest object.
#' @name container_requests.update
NULL

#' container_requests.delete
#'
#' container_requests.delete is a method defined in Arvados class.
#'
#' @usage arv$container_requests.delete(uuid)
#' @param uuid The UUID of the ContainerRequest in question.
#' @return ContainerRequest object.
#' @name container_requests.delete
NULL

#' container_requests.list
#'
#' container_requests.list is a method defined in Arvados class.
#'
#' @usage arv$container_requests.list(filters = NULL,
#' 	where = NULL, order = NULL, select = NULL,
#' 	distinct = NULL, limit = "100", offset = "0",
#' 	count = "exact", cluster_id = NULL, bypass_federation = NULL,
#' 	include_trash = NULL)
#' @param filters
#' @param where
#' @param order
#' @param select
#' @param distinct
#' @param limit
#' @param offset
#' @param count
#' @param cluster_id List objects on a remote federated cluster instead of the current one.
#' @param bypass_federation bypass federation behavior, list items from local instance database only
#' @param include_trash Include container requests whose owner project is trashed.
#' @return ContainerRequestList object.
#' @name container_requests.list
NULL

#' groups.get
#'
#' groups.get is a method defined in Arvados class.
#'
#' @usage arv$groups.get(uuid)
#' @param uuid The UUID of the Group in question.
#' @return Group object.
#' @name groups.get
NULL

#' groups.create
#'
#' groups.create is a method defined in Arvados class.
#'
#' @usage arv$groups.create(group, ensure_unique_name = "false",
#' 	cluster_id = NULL, async = "false")
#' @param group Group object.
#' @param ensure_unique_name Adjust name to ensure uniqueness instead of returning an error on (owner_uuid, name) collision.
#' @param cluster_id Create object on a remote federated cluster instead of the current one.
#' @param async defer permissions update
#' @return Group object.
#' @name groups.create
NULL

#' groups.update
#'
#' groups.update is a method defined in Arvados class.
#'
#' @usage arv$groups.update(group, uuid,
#' 	async = "false")
#' @param group Group object.
#' @param uuid The UUID of the Group in question.
#' @param async defer permissions update
#' @return Group object.
#' @name groups.update
NULL

#' groups.delete
#'
#' groups.delete is a method defined in Arvados class.
#'
#' @usage arv$groups.delete(uuid)
#' @param uuid The UUID of the Group in question.
#' @return Group object.
#' @name groups.delete
NULL

#' groups.contents
#'
#' groups.contents is a method defined in Arvados class.
#'
#' @usage arv$groups.contents(filters = NULL,
#' 	where = NULL, order = NULL, distinct = NULL,
#' 	limit = "100", offset = "0", count = "exact",
#' 	cluster_id = NULL, bypass_federation = NULL,
#' 	include_trash = NULL, uuid = NULL, recursive = NULL,
#' 	include = NULL)
#' @param filters
#' @param where
#' @param order
#' @param distinct
#' @param limit
#' @param offset
#' @param count
#' @param cluster_id List objects on a remote federated cluster instead of the current one.
#' @param bypass_federation bypass federation behavior, list items from local instance database only
#' @param include_trash Include items whose is_trashed attribute is true.
#' @param uuid
#' @param recursive Include contents from child groups recursively.
#' @param include Include objects referred to by listed field in "included" (only owner_uuid)
#' @return Group object.
#' @name groups.contents
NULL

#' groups.shared
#'
#' groups.shared is a method defined in Arvados class.
#'
#' @usage arv$groups.shared(filters = NULL,
#' 	where = NULL, order = NULL, select = NULL,
#' 	distinct = NULL, limit = "100", offset = "0",
#' 	count = "exact", cluster_id = NULL, bypass_federation = NULL,
#' 	include_trash = NULL, include = NULL)
#' @param filters
#' @param where
#' @param order
#' @param select
#' @param distinct
#' @param limit
#' @param offset
#' @param count
#' @param cluster_id List objects on a remote federated cluster instead of the current one.
#' @param bypass_federation bypass federation behavior, list items from local instance database only
#' @param include_trash Include items whose is_trashed attribute is true.
#' @param include
#' @return Group object.
#' @name groups.shared
NULL

#' groups.trash
#'
#' groups.trash is a method defined in Arvados class.
#'
#' @usage arv$groups.trash(uuid)
#' @param uuid
#' @return Group object.
#' @name groups.trash
NULL

#' groups.untrash
#'
#' groups.untrash is a method defined in Arvados class.
#'
#' @usage arv$groups.untrash(uuid)
#' @param uuid
#' @return Group object.
#' @name groups.untrash
NULL

#' groups.list
#'
#' groups.list is a method defined in Arvados class.
#'
#' @usage arv$groups.list(filters = NULL,
#' 	where = NULL, order = NULL, select = NULL,
#' 	distinct = NULL, limit = "100", offset = "0",
#' 	count = "exact", cluster_id = NULL, bypass_federation = NULL,
#' 	include_trash = NULL)
#' @param filters
#' @param where
#' @param order
#' @param select
#' @param distinct
#' @param limit
#' @param offset
#' @param count
#' @param cluster_id List objects on a remote federated cluster instead of the current one.
#' @param bypass_federation bypass federation behavior, list items from local instance database only
#' @param include_trash Include items whose is_trashed attribute is true.
#' @return GroupList object.
#' @name groups.list
NULL

#' keep_services.get
#'
#' keep_services.get is a method defined in Arvados class.
#'
#' @usage arv$keep_services.get(uuid)
#' @param uuid The UUID of the KeepService in question.
#' @return KeepService object.
#' @name keep_services.get
NULL

#' keep_services.create
#'
#' keep_services.create is a method defined in Arvados class.
#'
#' @usage arv$keep_services.create(keepservice,
#' 	ensure_unique_name = "false", cluster_id = NULL)
#' @param keepService KeepService object.
#' @param ensure_unique_name Adjust name to ensure uniqueness instead of returning an error on (owner_uuid, name) collision.
#' @param cluster_id Create object on a remote federated cluster instead of the current one.
#' @return KeepService object.
#' @name keep_services.create
NULL

#' keep_services.update
#'
#' keep_services.update is a method defined in Arvados class.
#'
#' @usage arv$keep_services.update(keepservice,
#' 	uuid)
#' @param keepService KeepService object.
#' @param uuid The UUID of the KeepService in question.
#' @return KeepService object.
#' @name keep_services.update
NULL

#' keep_services.delete
#'
#' keep_services.delete is a method defined in Arvados class.
#'
#' @usage arv$keep_services.delete(uuid)
#' @param uuid The UUID of the KeepService in question.
#' @return KeepService object.
#' @name keep_services.delete
NULL

#' keep_services.accessible
#'
#' keep_services.accessible is a method defined in Arvados class.
#'
#' @usage arv$keep_services.accessible(NULL)
#' @return KeepService object.
#' @name keep_services.accessible
NULL

#' keep_services.list
#'
#' keep_services.list is a method defined in Arvados class.
#'
#' @usage arv$keep_services.list(filters = NULL,
#' 	where = NULL, order = NULL, select = NULL,
#' 	distinct = NULL, limit = "100", offset = "0",
#' 	count = "exact", cluster_id = NULL, bypass_federation = NULL)
#' @param filters
#' @param where
#' @param order
#' @param select
#' @param distinct
#' @param limit
#' @param offset
#' @param count
#' @param cluster_id List objects on a remote federated cluster instead of the current one.
#' @param bypass_federation bypass federation behavior, list items from local instance database only
#' @return KeepServiceList object.
#' @name keep_services.list
NULL

#' links.get
#'
#' links.get is a method defined in Arvados class.
#'
#' @usage arv$links.get(uuid)
#' @param uuid The UUID of the Link in question.
#' @return Link object.
#' @name links.get
NULL

#' links.create
#'
#' links.create is a method defined in Arvados class.
#'
#' @usage arv$links.create(link, ensure_unique_name = "false",
#' 	cluster_id = NULL)
#' @param link Link object.
#' @param ensure_unique_name Adjust name to ensure uniqueness instead of returning an error on (owner_uuid, name) collision.
#' @param cluster_id Create object on a remote federated cluster instead of the current one.
#' @return Link object.
#' @name links.create
NULL

#' links.update
#'
#' links.update is a method defined in Arvados class.
#'
#' @usage arv$links.update(link, uuid)
#' @param link Link object.
#' @param uuid The UUID of the Link in question.
#' @return Link object.
#' @name links.update
NULL

#' links.delete
#'
#' links.delete is a method defined in Arvados class.
#'
#' @usage arv$links.delete(uuid)
#' @param uuid The UUID of the Link in question.
#' @return Link object.
#' @name links.delete
NULL

#' links.list
#'
#' links.list is a method defined in Arvados class.
#'
#' @usage arv$links.list(filters = NULL,
#' 	where = NULL, order = NULL, select = NULL,
#' 	distinct = NULL, limit = "100", offset = "0",
#' 	count = "exact", cluster_id = NULL, bypass_federation = NULL)
#' @param filters
#' @param where
#' @param order
#' @param select
#' @param distinct
#' @param limit
#' @param offset
#' @param count
#' @param cluster_id List objects on a remote federated cluster instead of the current one.
#' @param bypass_federation bypass federation behavior, list items from local instance database only
#' @return LinkList object.
#' @name links.list
NULL

#' links.get_permissions
#'
#' links.get_permissions is a method defined in Arvados class.
#'
#' @usage arv$links.get_permissions(uuid)
#' @param uuid
#' @return Link object.
#' @name links.get_permissions
NULL

#' logs.get
#'
#' logs.get is a method defined in Arvados class.
#'
#' @usage arv$logs.get(uuid)
#' @param uuid The UUID of the Log in question.
#' @return Log object.
#' @name logs.get
NULL

#' logs.create
#'
#' logs.create is a method defined in Arvados class.
#'
#' @usage arv$logs.create(log, ensure_unique_name = "false",
#' 	cluster_id = NULL)
#' @param log Log object.
#' @param ensure_unique_name Adjust name to ensure uniqueness instead of returning an error on (owner_uuid, name) collision.
#' @param cluster_id Create object on a remote federated cluster instead of the current one.
#' @return Log object.
#' @name logs.create
NULL

#' logs.update
#'
#' logs.update is a method defined in Arvados class.
#'
#' @usage arv$logs.update(log, uuid)
#' @param log Log object.
#' @param uuid The UUID of the Log in question.
#' @return Log object.
#' @name logs.update
NULL

#' logs.delete
#'
#' logs.delete is a method defined in Arvados class.
#'
#' @usage arv$logs.delete(uuid)
#' @param uuid The UUID of the Log in question.
#' @return Log object.
#' @name logs.delete
NULL

#' logs.list
#'
#' logs.list is a method defined in Arvados class.
#'
#' @usage arv$logs.list(filters = NULL, where = NULL,
#' 	order = NULL, select = NULL, distinct = NULL,
#' 	limit = "100", offset = "0", count = "exact",
#' 	cluster_id = NULL, bypass_federation = NULL)
#' @param filters
#' @param where
#' @param order
#' @param select
#' @param distinct
#' @param limit
#' @param offset
#' @param count
#' @param cluster_id List objects on a remote federated cluster instead of the current one.
#' @param bypass_federation bypass federation behavior, list items from local instance database only
#' @return LogList object.
#' @name logs.list
NULL

#' users.get
#'
#' users.get is a method defined in Arvados class.
#'
#' @usage arv$users.get(uuid)
#' @param uuid The UUID of the User in question.
#' @return User object.
#' @name users.get
NULL

#' users.create
#'
#' users.create is a method defined in Arvados class.
#'
#' @usage arv$users.create(user, ensure_unique_name = "false",
#' 	cluster_id = NULL)
#' @param user User object.
#' @param ensure_unique_name Adjust name to ensure uniqueness instead of returning an error on (owner_uuid, name) collision.
#' @param cluster_id Create object on a remote federated cluster instead of the current one.
#' @return User object.
#' @name users.create
NULL

#' users.update
#'
#' users.update is a method defined in Arvados class.
#'
#' @usage arv$users.update(user, uuid, bypass_federation = NULL)
#' @param user User object.
#' @param uuid The UUID of the User in question.
#' @param bypass_federation
#' @return User object.
#' @name users.update
NULL

#' users.delete
#'
#' users.delete is a method defined in Arvados class.
#'
#' @usage arv$users.delete(uuid)
#' @param uuid The UUID of the User in question.
#' @return User object.
#' @name users.delete
NULL

#' users.current
#'
#' users.current is a method defined in Arvados class.
#'
#' @usage arv$users.current(NULL)
#' @return User object.
#' @name users.current
NULL

#' users.system
#'
#' users.system is a method defined in Arvados class.
#'
#' @usage arv$users.system(NULL)
#' @return User object.
#' @name users.system
NULL

#' users.activate
#'
#' users.activate is a method defined in Arvados class.
#'
#' @usage arv$users.activate(uuid)
#' @param uuid
#' @return User object.
#' @name users.activate
NULL

#' users.setup
#'
#' users.setup is a method defined in Arvados class.
#'
#' @usage arv$users.setup(uuid = NULL, user = NULL,
#' 	repo_name = NULL, vm_uuid = NULL, send_notification_email = "false")
#' @param uuid
#' @param user
#' @param repo_name
#' @param vm_uuid
#' @param send_notification_email
#' @return User object.
#' @name users.setup
NULL

#' users.unsetup
#'
#' users.unsetup is a method defined in Arvados class.
#'
#' @usage arv$users.unsetup(uuid)
#' @param uuid
#' @return User object.
#' @name users.unsetup
NULL

#' users.merge
#'
#' users.merge is a method defined in Arvados class.
#'
#' @usage arv$users.merge(new_owner_uuid,
#' 	new_user_token = NULL, redirect_to_new_user = NULL,
#' 	old_user_uuid = NULL, new_user_uuid = NULL)
#' @param new_owner_uuid
#' @param new_user_token
#' @param redirect_to_new_user
#' @param old_user_uuid
#' @param new_user_uuid
#' @return User object.
#' @name users.merge
NULL

#' users.list
#'
#' users.list is a method defined in Arvados class.
#'
#' @usage arv$users.list(filters = NULL,
#' 	where = NULL, order = NULL, select = NULL,
#' 	distinct = NULL, limit = "100", offset = "0",
#' 	count = "exact", cluster_id = NULL, bypass_federation = NULL)
#' @param filters
#' @param where
#' @param order
#' @param select
#' @param distinct
#' @param limit
#' @param offset
#' @param count
#' @param cluster_id List objects on a remote federated cluster instead of the current one.
#' @param bypass_federation bypass federation behavior, list items from local instance database only
#' @return UserList object.
#' @name users.list
NULL

#' repositories.get
#'
#' repositories.get is a method defined in Arvados class.
#'
#' @usage arv$repositories.get(uuid)
#' @param uuid The UUID of the Repository in question.
#' @return Repository object.
#' @name repositories.get
NULL

#' repositories.create
#'
#' repositories.create is a method defined in Arvados class.
#'
#' @usage arv$repositories.create(repository,
#' 	ensure_unique_name = "false", cluster_id = NULL)
#' @param repository Repository object.
#' @param ensure_unique_name Adjust name to ensure uniqueness instead of returning an error on (owner_uuid, name) collision.
#' @param cluster_id Create object on a remote federated cluster instead of the current one.
#' @return Repository object.
#' @name repositories.create
NULL

#' repositories.update
#'
#' repositories.update is a method defined in Arvados class.
#'
#' @usage arv$repositories.update(repository,
#' 	uuid)
#' @param repository Repository object.
#' @param uuid The UUID of the Repository in question.
#' @return Repository object.
#' @name repositories.update
NULL

#' repositories.delete
#'
#' repositories.delete is a method defined in Arvados class.
#'
#' @usage arv$repositories.delete(uuid)
#' @param uuid The UUID of the Repository in question.
#' @return Repository object.
#' @name repositories.delete
NULL

#' repositories.get_all_permissions
#'
#' repositories.get_all_permissions is a method defined in Arvados class.
#'
#' @usage arv$repositories.get_all_permissions(NULL)
#' @return Repository object.
#' @name repositories.get_all_permissions
NULL

#' repositories.list
#'
#' repositories.list is a method defined in Arvados class.
#'
#' @usage arv$repositories.list(filters = NULL,
#' 	where = NULL, order = NULL, select = NULL,
#' 	distinct = NULL, limit = "100", offset = "0",
#' 	count = "exact", cluster_id = NULL, bypass_federation = NULL)
#' @param filters
#' @param where
#' @param order
#' @param select
#' @param distinct
#' @param limit
#' @param offset
#' @param count
#' @param cluster_id List objects on a remote federated cluster instead of the current one.
#' @param bypass_federation bypass federation behavior, list items from local instance database only
#' @return RepositoryList object.
#' @name repositories.list
NULL

#' virtual_machines.get
#'
#' virtual_machines.get is a method defined in Arvados class.
#'
#' @usage arv$virtual_machines.get(uuid)
#' @param uuid The UUID of the VirtualMachine in question.
#' @return VirtualMachine object.
#' @name virtual_machines.get
NULL

#' virtual_machines.create
#'
#' virtual_machines.create is a method defined in Arvados class.
#'
#' @usage arv$virtual_machines.create(virtualmachine,
#' 	ensure_unique_name = "false", cluster_id = NULL)
#' @param virtualMachine VirtualMachine object.
#' @param ensure_unique_name Adjust name to ensure uniqueness instead of returning an error on (owner_uuid, name) collision.
#' @param cluster_id Create object on a remote federated cluster instead of the current one.
#' @return VirtualMachine object.
#' @name virtual_machines.create
NULL

#' virtual_machines.update
#'
#' virtual_machines.update is a method defined in Arvados class.
#'
#' @usage arv$virtual_machines.update(virtualmachine,
#' 	uuid)
#' @param virtualMachine VirtualMachine object.
#' @param uuid The UUID of the VirtualMachine in question.
#' @return VirtualMachine object.
#' @name virtual_machines.update
NULL

#' virtual_machines.delete
#'
#' virtual_machines.delete is a method defined in Arvados class.
#'
#' @usage arv$virtual_machines.delete(uuid)
#' @param uuid The UUID of the VirtualMachine in question.
#' @return VirtualMachine object.
#' @name virtual_machines.delete
NULL

#' virtual_machines.logins
#'
#' virtual_machines.logins is a method defined in Arvados class.
#'
#' @usage arv$virtual_machines.logins(uuid)
#' @param uuid
#' @return VirtualMachine object.
#' @name virtual_machines.logins
NULL

#' virtual_machines.get_all_logins
#'
#' virtual_machines.get_all_logins is a method defined in Arvados class.
#'
#' @usage arv$virtual_machines.get_all_logins(NULL)
#' @return VirtualMachine object.
#' @name virtual_machines.get_all_logins
NULL

#' virtual_machines.list
#'
#' virtual_machines.list is a method defined in Arvados class.
#'
#' @usage arv$virtual_machines.list(filters = NULL,
#' 	where = NULL, order = NULL, select = NULL,
#' 	distinct = NULL, limit = "100", offset = "0",
#' 	count = "exact", cluster_id = NULL, bypass_federation = NULL)
#' @param filters
#' @param where
#' @param order
#' @param select
#' @param distinct
#' @param limit
#' @param offset
#' @param count
#' @param cluster_id List objects on a remote federated cluster instead of the current one.
#' @param bypass_federation bypass federation behavior, list items from local instance database only
#' @return VirtualMachineList object.
#' @name virtual_machines.list
NULL

#' workflows.get
#'
#' workflows.get is a method defined in Arvados class.
#'
#' @usage arv$workflows.get(uuid)
#' @param uuid The UUID of the Workflow in question.
#' @return Workflow object.
#' @name workflows.get
NULL

#' workflows.create
#'
#' workflows.create is a method defined in Arvados class.
#'
#' @usage arv$workflows.create(workflow,
#' 	ensure_unique_name = "false", cluster_id = NULL)
#' @param workflow Workflow object.
#' @param ensure_unique_name Adjust name to ensure uniqueness instead of returning an error on (owner_uuid, name) collision.
#' @param cluster_id Create object on a remote federated cluster instead of the current one.
#' @return Workflow object.
#' @name workflows.create
NULL

#' workflows.update
#'
#' workflows.update is a method defined in Arvados class.
#'
#' @usage arv$workflows.update(workflow,
#' 	uuid)
#' @param workflow Workflow object.
#' @param uuid The UUID of the Workflow in question.
#' @return Workflow object.
#' @name workflows.update
NULL

#' workflows.delete
#'
#' workflows.delete is a method defined in Arvados class.
#'
#' @usage arv$workflows.delete(uuid)
#' @param uuid The UUID of the Workflow in question.
#' @return Workflow object.
#' @name workflows.delete
NULL

#' workflows.list
#'
#' workflows.list is a method defined in Arvados class.
#'
#' @usage arv$workflows.list(filters = NULL,
#' 	where = NULL, order = NULL, select = NULL,
#' 	distinct = NULL, limit = "100", offset = "0",
#' 	count = "exact", cluster_id = NULL, bypass_federation = NULL)
#' @param filters
#' @param where
#' @param order
#' @param select
#' @param distinct
#' @param limit
#' @param offset
#' @param count
#' @param cluster_id List objects on a remote federated cluster instead of the current one.
#' @param bypass_federation bypass federation behavior, list items from local instance database only
#' @return WorkflowList object.
#' @name workflows.list
NULL

#' user_agreements.get
#'
#' user_agreements.get is a method defined in Arvados class.
#'
#' @usage arv$user_agreements.get(uuid)
#' @param uuid The UUID of the UserAgreement in question.
#' @return UserAgreement object.
#' @name user_agreements.get
NULL

#' user_agreements.create
#'
#' user_agreements.create is a method defined in Arvados class.
#'
#' @usage arv$user_agreements.create(useragreement,
#' 	ensure_unique_name = "false", cluster_id = NULL)
#' @param userAgreement UserAgreement object.
#' @param ensure_unique_name Adjust name to ensure uniqueness instead of returning an error on (owner_uuid, name) collision.
#' @param cluster_id Create object on a remote federated cluster instead of the current one.
#' @return UserAgreement object.
#' @name user_agreements.create
NULL

#' user_agreements.update
#'
#' user_agreements.update is a method defined in Arvados class.
#'
#' @usage arv$user_agreements.update(useragreement,
#' 	uuid)
#' @param userAgreement UserAgreement object.
#' @param uuid The UUID of the UserAgreement in question.
#' @return UserAgreement object.
#' @name user_agreements.update
NULL

#' user_agreements.delete
#'
#' user_agreements.delete is a method defined in Arvados class.
#'
#' @usage arv$user_agreements.delete(uuid)
#' @param uuid The UUID of the UserAgreement in question.
#' @return UserAgreement object.
#' @name user_agreements.delete
NULL

#' user_agreements.signatures
#'
#' user_agreements.signatures is a method defined in Arvados class.
#'
#' @usage arv$user_agreements.signatures(NULL)
#' @return UserAgreement object.
#' @name user_agreements.signatures
NULL

#' user_agreements.sign
#'
#' user_agreements.sign is a method defined in Arvados class.
#'
#' @usage arv$user_agreements.sign(NULL)
#' @return UserAgreement object.
#' @name user_agreements.sign
NULL

#' user_agreements.list
#'
#' user_agreements.list is a method defined in Arvados class.
#'
#' @usage arv$user_agreements.list(filters = NULL,
#' 	where = NULL, order = NULL, select = NULL,
#' 	distinct = NULL, limit = "100", offset = "0",
#' 	count = "exact", cluster_id = NULL, bypass_federation = NULL)
#' @param filters
#' @param where
#' @param order
#' @param select
#' @param distinct
#' @param limit
#' @param offset
#' @param count
#' @param cluster_id List objects on a remote federated cluster instead of the current one.
#' @param bypass_federation bypass federation behavior, list items from local instance database only
#' @return UserAgreementList object.
#' @name user_agreements.list
NULL

#' user_agreements.new
#'
#' user_agreements.new is a method defined in Arvados class.
#'
#' @usage arv$user_agreements.new(NULL)
#' @return UserAgreement object.
#' @name user_agreements.new
NULL

#' configs.get
#'
#' configs.get is a method defined in Arvados class.
#'
#' @usage arv$configs.get(NULL)
#' @return  object.
#' @name configs.get
NULL

#' project.get
#'
#' projects.get is equivalent to groups.get method.
#'
#' @usage arv$projects.get(uuid)
#' @param uuid The UUID of the Group in question.
#' @return Group object.
#' @name projects.get
NULL

#' project.create
#'
#' projects.create wrapps groups.create method by setting group_class attribute to "project".
#'
#' @usage arv$projects.create(group, ensure_unique_name = "false")
#' @param group Group object.
#' @param ensure_unique_name Adjust name to ensure uniqueness instead of returning an error on (owner_uuid, name) collision.
#' @return Group object.
#' @name projects.create
NULL

#' project.update
#'
#' projects.update wrapps groups.update method by setting group_class attribute to "project".
#'
#' @usage arv$projects.update(group, uuid)
#' @param group Group object.
#' @param uuid The UUID of the Group in question.
#' @return Group object.
#' @name projects.update
NULL

#' project.delete
#'
#' projects.delete is equivalent to groups.delete method.
#'
#' @usage arv$project.delete(uuid)
#' @param uuid The UUID of the Group in question.
#' @return Group object.
#' @name projects.delete
NULL

#' project.list
#'
#' projects.list wrapps groups.list method by setting group_class attribute to "project".
#'
#' @usage arv$projects.list(filters = NULL,
#' 	where = NULL, order = NULL, distinct = NULL,
#' 	limit = "100", offset = "0", count = "exact",
#' 	include_trash = NULL, uuid = NULL, recursive = NULL)
#' @param filters
#' @param where
#' @param order
#' @param distinct
#' @param limit
#' @param offset
#' @param count
#' @param include_trash Include items whose is_trashed attribute is true.
#' @param uuid
#' @param recursive Include contents from child groups recursively.
#' @return Group object.
#' @name projects.list
NULL

#' Arvados
#'
#' Arvados class gives users ability to access Arvados REST API.
#'
#' @section Usage:
#' \preformatted{arv = Arvados$new(authToken = NULL, hostName = NULL, numRetries = 0)}
#'
#' @section Arguments:
#' \describe{
#' 	\item{authToken}{Authentification token. If not specified ARVADOS_API_TOKEN environment variable will be used.}
#' 	\item{hostName}{Host name. If not specified ARVADOS_API_HOST environment variable will be used.}
#' 	\item{numRetries}{Number which specifies how many times to retry failed service requests.}
#' }
#'
#' @section Methods:
#' \describe{
#' 	\item{}{\code{\link{api_client_authorizations.create}}}
#' 	\item{}{\code{\link{api_client_authorizations.create_system_auth}}}
#' 	\item{}{\code{\link{api_client_authorizations.current}}}
#' 	\item{}{\code{\link{api_client_authorizations.delete}}}
#' 	\item{}{\code{\link{api_client_authorizations.get}}}
#' 	\item{}{\code{\link{api_client_authorizations.list}}}
#' 	\item{}{\code{\link{api_client_authorizations.update}}}
#' 	\item{}{\code{\link{api_clients.create}}}
#' 	\item{}{\code{\link{api_clients.delete}}}
#' 	\item{}{\code{\link{api_clients.get}}}
#' 	\item{}{\code{\link{api_clients.list}}}
#' 	\item{}{\code{\link{api_clients.update}}}
#' 	\item{}{\code{\link{authorized_keys.create}}}
#' 	\item{}{\code{\link{authorized_keys.delete}}}
#' 	\item{}{\code{\link{authorized_keys.get}}}
#' 	\item{}{\code{\link{authorized_keys.list}}}
#' 	\item{}{\code{\link{authorized_keys.update}}}
#' 	\item{}{\code{\link{collections.create}}}
#' 	\item{}{\code{\link{collections.delete}}}
#' 	\item{}{\code{\link{collections.get}}}
#' 	\item{}{\code{\link{collections.list}}}
#' 	\item{}{\code{\link{collections.provenance}}}
#' 	\item{}{\code{\link{collections.trash}}}
#' 	\item{}{\code{\link{collections.untrash}}}
#' 	\item{}{\code{\link{collections.update}}}
#' 	\item{}{\code{\link{collections.used_by}}}
#' 	\item{}{\code{\link{configs.get}}}
#' 	\item{}{\code{\link{container_requests.create}}}
#' 	\item{}{\code{\link{container_requests.delete}}}
#' 	\item{}{\code{\link{container_requests.get}}}
#' 	\item{}{\code{\link{container_requests.list}}}
#' 	\item{}{\code{\link{container_requests.update}}}
#' 	\item{}{\code{\link{containers.auth}}}
#' 	\item{}{\code{\link{containers.create}}}
#' 	\item{}{\code{\link{containers.current}}}
#' 	\item{}{\code{\link{containers.delete}}}
#' 	\item{}{\code{\link{containers.get}}}
#' 	\item{}{\code{\link{containers.list}}}
#' 	\item{}{\code{\link{containers.lock}}}
#' 	\item{}{\code{\link{containers.secret_mounts}}}
#' 	\item{}{\code{\link{containers.unlock}}}
#' 	\item{}{\code{\link{containers.update}}}
#' 	\item{}{\code{\link{groups.contents}}}
#' 	\item{}{\code{\link{groups.create}}}
#' 	\item{}{\code{\link{groups.delete}}}
#' 	\item{}{\code{\link{groups.get}}}
#' 	\item{}{\code{\link{groups.list}}}
#' 	\item{}{\code{\link{groups.shared}}}
#' 	\item{}{\code{\link{groups.trash}}}
#' 	\item{}{\code{\link{groups.untrash}}}
#' 	\item{}{\code{\link{groups.update}}}
#' 	\item{}{\code{\link{keep_services.accessible}}}
#' 	\item{}{\code{\link{keep_services.create}}}
#' 	\item{}{\code{\link{keep_services.delete}}}
#' 	\item{}{\code{\link{keep_services.get}}}
#' 	\item{}{\code{\link{keep_services.list}}}
#' 	\item{}{\code{\link{keep_services.update}}}
#' 	\item{}{\code{\link{links.create}}}
#' 	\item{}{\code{\link{links.delete}}}
#' 	\item{}{\code{\link{links.get}}}
#' 	\item{}{\code{\link{links.get_permissions}}}
#' 	\item{}{\code{\link{links.list}}}
#' 	\item{}{\code{\link{links.update}}}
#' 	\item{}{\code{\link{logs.create}}}
#' 	\item{}{\code{\link{logs.delete}}}
#' 	\item{}{\code{\link{logs.get}}}
#' 	\item{}{\code{\link{logs.list}}}
#' 	\item{}{\code{\link{logs.update}}}
#' 	\item{}{\code{\link{projects.create}}}
#' 	\item{}{\code{\link{projects.delete}}}
#' 	\item{}{\code{\link{projects.get}}}
#' 	\item{}{\code{\link{projects.list}}}
#' 	\item{}{\code{\link{projects.update}}}
#' 	\item{}{\code{\link{repositories.create}}}
#' 	\item{}{\code{\link{repositories.delete}}}
#' 	\item{}{\code{\link{repositories.get}}}
#' 	\item{}{\code{\link{repositories.get_all_permissions}}}
#' 	\item{}{\code{\link{repositories.list}}}
#' 	\item{}{\code{\link{repositories.update}}}
#' 	\item{}{\code{\link{user_agreements.create}}}
#' 	\item{}{\code{\link{user_agreements.delete}}}
#' 	\item{}{\code{\link{user_agreements.get}}}
#' 	\item{}{\code{\link{user_agreements.list}}}
#' 	\item{}{\code{\link{user_agreements.new}}}
#' 	\item{}{\code{\link{user_agreements.sign}}}
#' 	\item{}{\code{\link{user_agreements.signatures}}}
#' 	\item{}{\code{\link{user_agreements.update}}}
#' 	\item{}{\code{\link{users.activate}}}
#' 	\item{}{\code{\link{users.create}}}
#' 	\item{}{\code{\link{users.current}}}
#' 	\item{}{\code{\link{users.delete}}}
#' 	\item{}{\code{\link{users.get}}}
#' 	\item{}{\code{\link{users.list}}}
#' 	\item{}{\code{\link{users.merge}}}
#' 	\item{}{\code{\link{users.setup}}}
#' 	\item{}{\code{\link{users.system}}}
#' 	\item{}{\code{\link{users.unsetup}}}
#' 	\item{}{\code{\link{users.update}}}
#' 	\item{}{\code{\link{virtual_machines.create}}}
#' 	\item{}{\code{\link{virtual_machines.delete}}}
#' 	\item{}{\code{\link{virtual_machines.get}}}
#' 	\item{}{\code{\link{virtual_machines.get_all_logins}}}
#' 	\item{}{\code{\link{virtual_machines.list}}}
#' 	\item{}{\code{\link{virtual_machines.logins}}}
#' 	\item{}{\code{\link{virtual_machines.update}}}
#' 	\item{}{\code{\link{workflows.create}}}
#' 	\item{}{\code{\link{workflows.delete}}}
#' 	\item{}{\code{\link{workflows.get}}}
#' 	\item{}{\code{\link{workflows.list}}}
#' 	\item{}{\code{\link{workflows.update}}}
#' }
#'
#' @name Arvados
#' @examples
#' \dontrun{
#' arv <- Arvados$new("your Arvados token", "example.arvadosapi.com")
#'
#' collection <- arv$collections.get("uuid")
#'
#' collectionList <- arv$collections.list(list(list("name", "like", "Test%")))
#' collectionList <- listAll(arv$collections.list, list(list("name", "like", "Test%")))
#'
#' deletedCollection <- arv$collections.delete("uuid")
#'
#' updatedCollection <- arv$collections.update(list(name = "New name", description = "New description"),
#'                                             "uuid")
#'
#' createdCollection <- arv$collections.create(list(name = "Example",
#'                                                  description = "This is a test collection"))
#' }
NULL

#' @export
Arvados <- R6::R6Class(

	"Arvados",

	public = list(

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

		projects.get = function(uuid)
		{
			self$groups.get(uuid)
		},

		projects.create = function(group, ensure_unique_name = "false")
		{
			group <- c("group_class" = "project", group)
			self$groups.create(group, ensure_unique_name)
		},

		projects.update = function(group, uuid)
		{
			group <- c("group_class" = "project", group)
			self$groups.update(group, uuid)
		},

		projects.list = function(filters = NULL, where = NULL,
			order = NULL, select = NULL, distinct = NULL,
			limit = "100", offset = "0", count = "exact",
			include_trash = NULL)
		{
			filters[[length(filters) + 1]] <- list("group_class", "=", "project")
			self$groups.list(filters, where, order, select, distinct,
			                 limit, offset, count, include_trash)
		},

		projects.delete = function(uuid)
		{
			self$groups.delete(uuid)
		},

		api_clients.get = function(uuid)
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

		api_clients.create = function(apiclient,
			ensure_unique_name = "false", cluster_id = NULL)
		{
			endPoint <- stringr::str_interp("api_clients")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("Bearer", private$token),
			                "Content-Type" = "application/json")
			queryArgs <- list(ensure_unique_name = ensure_unique_name,
							  cluster_id = cluster_id)

			if(length(apiclient) > 0)
				body <- jsonlite::toJSON(list(apiclient = apiclient),
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

		api_clients.update = function(apiclient, uuid)
		{
			endPoint <- stringr::str_interp("api_clients/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("Bearer", private$token),
			                "Content-Type" = "application/json")
			queryArgs <- NULL

			if(length(apiclient) > 0)
				body <- jsonlite::toJSON(list(apiclient = apiclient),
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

		api_clients.delete = function(uuid)
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

		api_clients.list = function(filters = NULL,
			where = NULL, order = NULL, select = NULL,
			distinct = NULL, limit = "100", offset = "0",
			count = "exact", cluster_id = NULL, bypass_federation = NULL)
		{
			endPoint <- stringr::str_interp("api_clients")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("Bearer", private$token),
			                "Content-Type" = "application/json")
			queryArgs <- list(filters = filters, where = where,
							  order = order, select = select, distinct = distinct,
							  limit = limit, offset = offset, count = count,
							  cluster_id = cluster_id, bypass_federation = bypass_federation)

			body <- NULL

			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)

			if(!is.null(resource$errors))
				stop(resource$errors)

			resource
		},

		api_client_authorizations.get = function(uuid)
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

		api_client_authorizations.create = function(apiclientauthorization,
			ensure_unique_name = "false", cluster_id = NULL)
		{
			endPoint <- stringr::str_interp("api_client_authorizations")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("Bearer", private$token),
			                "Content-Type" = "application/json")
			queryArgs <- list(ensure_unique_name = ensure_unique_name,
							  cluster_id = cluster_id)

			if(length(apiclientauthorization) > 0)
				body <- jsonlite::toJSON(list(apiclientauthorization = apiclientauthorization),
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

		api_client_authorizations.update = function(apiclientauthorization, uuid)
		{
			endPoint <- stringr::str_interp("api_client_authorizations/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("Bearer", private$token),
			                "Content-Type" = "application/json")
			queryArgs <- NULL

			if(length(apiclientauthorization) > 0)
				body <- jsonlite::toJSON(list(apiclientauthorization = apiclientauthorization),
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

		api_client_authorizations.delete = function(uuid)
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

		api_client_authorizations.create_system_auth = function(api_client_id = NULL, scopes = NULL)
		{
			endPoint <- stringr::str_interp("api_client_authorizations/create_system_auth")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("Bearer", private$token),
			                "Content-Type" = "application/json")
			queryArgs <- list(api_client_id = api_client_id,
							  scopes = scopes)

			body <- NULL

			response <- private$REST$http$exec("POST", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)

			if(!is.null(resource$errors))
				stop(resource$errors)

			resource
		},

		api_client_authorizations.current = function()
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

		api_client_authorizations.list = function(filters = NULL,
			where = NULL, order = NULL, select = NULL,
			distinct = NULL, limit = "100", offset = "0",
			count = "exact", cluster_id = NULL, bypass_federation = NULL)
		{
			endPoint <- stringr::str_interp("api_client_authorizations")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("Bearer", private$token),
			                "Content-Type" = "application/json")
			queryArgs <- list(filters = filters, where = where,
							  order = order, select = select, distinct = distinct,
							  limit = limit, offset = offset, count = count,
							  cluster_id = cluster_id, bypass_federation = bypass_federation)

			body <- NULL

			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)

			if(!is.null(resource$errors))
				stop(resource$errors)

			resource
		},

		authorized_keys.get = function(uuid)
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

		authorized_keys.create = function(authorizedkey,
			ensure_unique_name = "false", cluster_id = NULL)
		{
			endPoint <- stringr::str_interp("authorized_keys")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("Bearer", private$token),
			                "Content-Type" = "application/json")
			queryArgs <- list(ensure_unique_name = ensure_unique_name,
							  cluster_id = cluster_id)

			if(length(authorizedkey) > 0)
				body <- jsonlite::toJSON(list(authorizedkey = authorizedkey),
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

		authorized_keys.update = function(authorizedkey, uuid)
		{
			endPoint <- stringr::str_interp("authorized_keys/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("Bearer", private$token),
			                "Content-Type" = "application/json")
			queryArgs <- NULL

			if(length(authorizedkey) > 0)
				body <- jsonlite::toJSON(list(authorizedkey = authorizedkey),
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

		authorized_keys.delete = function(uuid)
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

		authorized_keys.list = function(filters = NULL,
			where = NULL, order = NULL, select = NULL,
			distinct = NULL, limit = "100", offset = "0",
			count = "exact", cluster_id = NULL, bypass_federation = NULL)
		{
			endPoint <- stringr::str_interp("authorized_keys")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("Bearer", private$token),
			                "Content-Type" = "application/json")
			queryArgs <- list(filters = filters, where = where,
							  order = order, select = select, distinct = distinct,
							  limit = limit, offset = offset, count = count,
							  cluster_id = cluster_id, bypass_federation = bypass_federation)

			body <- NULL

			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)

			if(!is.null(resource$errors))
				stop(resource$errors)

			resource
		},

		collections.get = function(uuid)
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

		collections.create = function(collection,
			ensure_unique_name = "false", cluster_id = NULL)
		{
			endPoint <- stringr::str_interp("collections")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("Bearer", private$token),
			                "Content-Type" = "application/json")
			queryArgs <- list(ensure_unique_name = ensure_unique_name,
							  cluster_id = cluster_id)

			if(length(collection) > 0)
				body <- jsonlite::toJSON(list(collection = collection),
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

		collections.update = function(collection, uuid)
		{
			endPoint <- stringr::str_interp("collections/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("Bearer", private$token),
			                "Content-Type" = "application/json")
			queryArgs <- NULL

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

		collections.delete = function(uuid)
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

		collections.provenance = function(uuid)
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

		collections.used_by = function(uuid)
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

		collections.trash = function(uuid)
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

		collections.untrash = function(uuid)
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

		collections.list = function(filters = NULL,
			where = NULL, order = NULL, select = NULL,
			distinct = NULL, limit = "100", offset = "0",
			count = "exact", cluster_id = NULL, bypass_federation = NULL,
			include_trash = NULL, include_old_versions = NULL)
		{
			endPoint <- stringr::str_interp("collections")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("Bearer", private$token),
			                "Content-Type" = "application/json")
			queryArgs <- list(filters = filters, where = where,
							  order = order, select = select, distinct = distinct,
							  limit = limit, offset = offset, count = count,
							  cluster_id = cluster_id, bypass_federation = bypass_federation,
							  include_trash = include_trash, include_old_versions = include_old_versions)

			body <- NULL

			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)

			if(!is.null(resource$errors))
				stop(resource$errors)

			resource
		},

		containers.get = function(uuid)
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

		containers.create = function(container, ensure_unique_name = "false",
			cluster_id = NULL)
		{
			endPoint <- stringr::str_interp("containers")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("Bearer", private$token),
			                "Content-Type" = "application/json")
			queryArgs <- list(ensure_unique_name = ensure_unique_name,
							  cluster_id = cluster_id)

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

		containers.update = function(container, uuid)
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

		containers.delete = function(uuid)
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

		containers.auth = function(uuid)
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

		containers.lock = function(uuid)
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

		containers.unlock = function(uuid)
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

		containers.secret_mounts = function(uuid)
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

		containers.current = function()
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

		containers.list = function(filters = NULL,
			where = NULL, order = NULL, select = NULL,
			distinct = NULL, limit = "100", offset = "0",
			count = "exact", cluster_id = NULL, bypass_federation = NULL)
		{
			endPoint <- stringr::str_interp("containers")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("Bearer", private$token),
			                "Content-Type" = "application/json")
			queryArgs <- list(filters = filters, where = where,
							  order = order, select = select, distinct = distinct,
							  limit = limit, offset = offset, count = count,
							  cluster_id = cluster_id, bypass_federation = bypass_federation)

			body <- NULL

			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)

			if(!is.null(resource$errors))
				stop(resource$errors)

			resource
		},

		container_requests.get = function(uuid)
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

		container_requests.create = function(containerrequest,
			ensure_unique_name = "false", cluster_id = NULL)
		{
			endPoint <- stringr::str_interp("container_requests")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("Bearer", private$token),
			                "Content-Type" = "application/json")
			queryArgs <- list(ensure_unique_name = ensure_unique_name,
							  cluster_id = cluster_id)

			if(length(containerrequest) > 0)
				body <- jsonlite::toJSON(list(containerrequest = containerrequest),
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

		container_requests.update = function(containerrequest, uuid)
		{
			endPoint <- stringr::str_interp("container_requests/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("Bearer", private$token),
			                "Content-Type" = "application/json")
			queryArgs <- NULL

			if(length(containerrequest) > 0)
				body <- jsonlite::toJSON(list(containerrequest = containerrequest),
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

		container_requests.delete = function(uuid)
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

		container_requests.list = function(filters = NULL,
			where = NULL, order = NULL, select = NULL,
			distinct = NULL, limit = "100", offset = "0",
			count = "exact", cluster_id = NULL, bypass_federation = NULL,
			include_trash = NULL)
		{
			endPoint <- stringr::str_interp("container_requests")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("Bearer", private$token),
			                "Content-Type" = "application/json")
			queryArgs <- list(filters = filters, where = where,
							  order = order, select = select, distinct = distinct,
							  limit = limit, offset = offset, count = count,
							  cluster_id = cluster_id, bypass_federation = bypass_federation,
							  include_trash = include_trash)

			body <- NULL

			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)

			if(!is.null(resource$errors))
				stop(resource$errors)

			resource
		},

		groups.get = function(uuid)
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

		groups.create = function(group, ensure_unique_name = "false",
			cluster_id = NULL, async = "false")
		{
			endPoint <- stringr::str_interp("groups")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("Bearer", private$token),
			                "Content-Type" = "application/json")
			queryArgs <- list(ensure_unique_name = ensure_unique_name,
							  cluster_id = cluster_id, async = async)

			if(length(group) > 0)
				body <- jsonlite::toJSON(list(group = group),
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

		groups.update = function(group, uuid, async = "false")
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

		groups.delete = function(uuid)
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

			resource
		},

		groups.contents = function(filters = NULL,
			where = NULL, order = NULL, distinct = NULL,
			limit = "100", offset = "0", count = "exact",
			cluster_id = NULL, bypass_federation = NULL,
			include_trash = NULL, uuid = NULL, recursive = NULL,
			include = NULL)
		{
			endPoint <- stringr::str_interp("groups/contents")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("Bearer", private$token),
			                "Content-Type" = "application/json")
			queryArgs <- list(filters = filters, where = where,
							  order = order, distinct = distinct, limit = limit,
							  offset = offset, count = count, cluster_id = cluster_id,
							  bypass_federation = bypass_federation, include_trash = include_trash,
							  uuid = uuid, recursive = recursive, include = include)

			body <- NULL

			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)

			if(!is.null(resource$errors))
				stop(resource$errors)

			resource
		},

		groups.shared = function(filters = NULL,
			where = NULL, order = NULL, select = NULL,
			distinct = NULL, limit = "100", offset = "0",
			count = "exact", cluster_id = NULL, bypass_federation = NULL,
			include_trash = NULL, include = NULL)
		{
			endPoint <- stringr::str_interp("groups/shared")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("Bearer", private$token),
			                "Content-Type" = "application/json")
			queryArgs <- list(filters = filters, where = where,
							  order = order, select = select, distinct = distinct,
							  limit = limit, offset = offset, count = count,
							  cluster_id = cluster_id, bypass_federation = bypass_federation,
							  include_trash = include_trash, include = include)

			body <- NULL

			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)

			if(!is.null(resource$errors))
				stop(resource$errors)

			resource
		},

		groups.trash = function(uuid)
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

		groups.untrash = function(uuid)
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

		groups.list = function(filters = NULL, where = NULL,
			order = NULL, select = NULL, distinct = NULL,
			limit = "100", offset = "0", count = "exact",
			cluster_id = NULL, bypass_federation = NULL,
			include_trash = NULL)
		{
			endPoint <- stringr::str_interp("groups")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("Bearer", private$token),
			                "Content-Type" = "application/json")
			queryArgs <- list(filters = filters, where = where,
							  order = order, select = select, distinct = distinct,
							  limit = limit, offset = offset, count = count,
							  cluster_id = cluster_id, bypass_federation = bypass_federation,
							  include_trash = include_trash)

			body <- NULL

			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)

			if(!is.null(resource$errors))
				stop(resource$errors)

			resource
		},

		keep_services.get = function(uuid)
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

		keep_services.create = function(keepservice,
			ensure_unique_name = "false", cluster_id = NULL)
		{
			endPoint <- stringr::str_interp("keep_services")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("Bearer", private$token),
			                "Content-Type" = "application/json")
			queryArgs <- list(ensure_unique_name = ensure_unique_name,
							  cluster_id = cluster_id)

			if(length(keepservice) > 0)
				body <- jsonlite::toJSON(list(keepservice = keepservice),
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

		keep_services.update = function(keepservice, uuid)
		{
			endPoint <- stringr::str_interp("keep_services/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("Bearer", private$token),
			                "Content-Type" = "application/json")
			queryArgs <- NULL

			if(length(keepservice) > 0)
				body <- jsonlite::toJSON(list(keepservice = keepservice),
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

		keep_services.delete = function(uuid)
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

		keep_services.accessible = function()
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

		keep_services.list = function(filters = NULL,
			where = NULL, order = NULL, select = NULL,
			distinct = NULL, limit = "100", offset = "0",
			count = "exact", cluster_id = NULL, bypass_federation = NULL)
		{
			endPoint <- stringr::str_interp("keep_services")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("Bearer", private$token),
			                "Content-Type" = "application/json")
			queryArgs <- list(filters = filters, where = where,
							  order = order, select = select, distinct = distinct,
							  limit = limit, offset = offset, count = count,
							  cluster_id = cluster_id, bypass_federation = bypass_federation)

			body <- NULL

			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)

			if(!is.null(resource$errors))
				stop(resource$errors)

			resource
		},

		links.get = function(uuid)
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

		links.create = function(link, ensure_unique_name = "false",
			cluster_id = NULL)
		{
			endPoint <- stringr::str_interp("links")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("Bearer", private$token),
			                "Content-Type" = "application/json")
			queryArgs <- list(ensure_unique_name = ensure_unique_name,
							  cluster_id = cluster_id)

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

		links.update = function(link, uuid)
		{
			endPoint <- stringr::str_interp("links/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("Bearer", private$token),
			                "Content-Type" = "application/json")
			queryArgs <- NULL

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

		links.delete = function(uuid)
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

		links.list = function(filters = NULL, where = NULL,
			order = NULL, select = NULL, distinct = NULL,
			limit = "100", offset = "0", count = "exact",
			cluster_id = NULL, bypass_federation = NULL)
		{
			endPoint <- stringr::str_interp("links")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("Bearer", private$token),
			                "Content-Type" = "application/json")
			queryArgs <- list(filters = filters, where = where,
							  order = order, select = select, distinct = distinct,
							  limit = limit, offset = offset, count = count,
							  cluster_id = cluster_id, bypass_federation = bypass_federation)

			body <- NULL

			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)

			if(!is.null(resource$errors))
				stop(resource$errors)

			resource
		},

		links.get_permissions = function(uuid)
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

		logs.get = function(uuid)
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

		logs.create = function(log, ensure_unique_name = "false",
			cluster_id = NULL)
		{
			endPoint <- stringr::str_interp("logs")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("Bearer", private$token),
			                "Content-Type" = "application/json")
			queryArgs <- list(ensure_unique_name = ensure_unique_name,
							  cluster_id = cluster_id)

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

		logs.update = function(log, uuid)
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

		logs.delete = function(uuid)
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

		logs.list = function(filters = NULL, where = NULL,
			order = NULL, select = NULL, distinct = NULL,
			limit = "100", offset = "0", count = "exact",
			cluster_id = NULL, bypass_federation = NULL)
		{
			endPoint <- stringr::str_interp("logs")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("Bearer", private$token),
			                "Content-Type" = "application/json")
			queryArgs <- list(filters = filters, where = where,
							  order = order, select = select, distinct = distinct,
							  limit = limit, offset = offset, count = count,
							  cluster_id = cluster_id, bypass_federation = bypass_federation)

			body <- NULL

			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)

			if(!is.null(resource$errors))
				stop(resource$errors)

			resource
		},

		users.get = function(uuid)
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

		users.create = function(user, ensure_unique_name = "false",
			cluster_id = NULL)
		{
			endPoint <- stringr::str_interp("users")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("Bearer", private$token),
			                "Content-Type" = "application/json")
			queryArgs <- list(ensure_unique_name = ensure_unique_name,
							  cluster_id = cluster_id)

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

		users.update = function(user, uuid, bypass_federation = NULL)
		{
			endPoint <- stringr::str_interp("users/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("Bearer", private$token),
			                "Content-Type" = "application/json")
			queryArgs <- list(bypass_federation = bypass_federation)

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

		users.delete = function(uuid)
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

		users.current = function()
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

		users.system = function()
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

		users.activate = function(uuid)
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

		users.setup = function(uuid = NULL, user = NULL,
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

		users.unsetup = function(uuid)
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

		users.merge = function(new_owner_uuid, new_user_token = NULL,
			redirect_to_new_user = NULL, old_user_uuid = NULL,
			new_user_uuid = NULL)
		{
			endPoint <- stringr::str_interp("users/merge")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("Bearer", private$token),
			                "Content-Type" = "application/json")
			queryArgs <- list(new_owner_uuid = new_owner_uuid,
							  new_user_token = new_user_token, redirect_to_new_user = redirect_to_new_user,
							  old_user_uuid = old_user_uuid, new_user_uuid = new_user_uuid)

			body <- NULL

			response <- private$REST$http$exec("POST", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)

			if(!is.null(resource$errors))
				stop(resource$errors)

			resource
		},

		users.list = function(filters = NULL, where = NULL,
			order = NULL, select = NULL, distinct = NULL,
			limit = "100", offset = "0", count = "exact",
			cluster_id = NULL, bypass_federation = NULL)
		{
			endPoint <- stringr::str_interp("users")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("Bearer", private$token),
			                "Content-Type" = "application/json")
			queryArgs <- list(filters = filters, where = where,
							  order = order, select = select, distinct = distinct,
							  limit = limit, offset = offset, count = count,
							  cluster_id = cluster_id, bypass_federation = bypass_federation)

			body <- NULL

			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)

			if(!is.null(resource$errors))
				stop(resource$errors)

			resource
		},

		repositories.get = function(uuid)
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

		repositories.create = function(repository,
			ensure_unique_name = "false", cluster_id = NULL)
		{
			endPoint <- stringr::str_interp("repositories")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("Bearer", private$token),
			                "Content-Type" = "application/json")
			queryArgs <- list(ensure_unique_name = ensure_unique_name,
							  cluster_id = cluster_id)

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

		repositories.update = function(repository, uuid)
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

		repositories.delete = function(uuid)
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

		repositories.get_all_permissions = function()
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

		repositories.list = function(filters = NULL,
			where = NULL, order = NULL, select = NULL,
			distinct = NULL, limit = "100", offset = "0",
			count = "exact", cluster_id = NULL, bypass_federation = NULL)
		{
			endPoint <- stringr::str_interp("repositories")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("Bearer", private$token),
			                "Content-Type" = "application/json")
			queryArgs <- list(filters = filters, where = where,
							  order = order, select = select, distinct = distinct,
							  limit = limit, offset = offset, count = count,
							  cluster_id = cluster_id, bypass_federation = bypass_federation)

			body <- NULL

			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)

			if(!is.null(resource$errors))
				stop(resource$errors)

			resource
		},

		virtual_machines.get = function(uuid)
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

		virtual_machines.create = function(virtualmachine,
			ensure_unique_name = "false", cluster_id = NULL)
		{
			endPoint <- stringr::str_interp("virtual_machines")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("Bearer", private$token),
			                "Content-Type" = "application/json")
			queryArgs <- list(ensure_unique_name = ensure_unique_name,
							  cluster_id = cluster_id)

			if(length(virtualmachine) > 0)
				body <- jsonlite::toJSON(list(virtualmachine = virtualmachine),
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

		virtual_machines.update = function(virtualmachine, uuid)
		{
			endPoint <- stringr::str_interp("virtual_machines/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("Bearer", private$token),
			                "Content-Type" = "application/json")
			queryArgs <- NULL

			if(length(virtualmachine) > 0)
				body <- jsonlite::toJSON(list(virtualmachine = virtualmachine),
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

		virtual_machines.delete = function(uuid)
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

		virtual_machines.logins = function(uuid)
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

		virtual_machines.get_all_logins = function()
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

		virtual_machines.list = function(filters = NULL,
			where = NULL, order = NULL, select = NULL,
			distinct = NULL, limit = "100", offset = "0",
			count = "exact", cluster_id = NULL, bypass_federation = NULL)
		{
			endPoint <- stringr::str_interp("virtual_machines")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("Bearer", private$token),
			                "Content-Type" = "application/json")
			queryArgs <- list(filters = filters, where = where,
							  order = order, select = select, distinct = distinct,
							  limit = limit, offset = offset, count = count,
							  cluster_id = cluster_id, bypass_federation = bypass_federation)

			body <- NULL

			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)

			if(!is.null(resource$errors))
				stop(resource$errors)

			resource
		},

		workflows.get = function(uuid)
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

		workflows.create = function(workflow, ensure_unique_name = "false",
			cluster_id = NULL)
		{
			endPoint <- stringr::str_interp("workflows")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("Bearer", private$token),
			                "Content-Type" = "application/json")
			queryArgs <- list(ensure_unique_name = ensure_unique_name,
							  cluster_id = cluster_id)

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

		workflows.update = function(workflow, uuid)
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

		workflows.delete = function(uuid)
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

		workflows.list = function(filters = NULL,
			where = NULL, order = NULL, select = NULL,
			distinct = NULL, limit = "100", offset = "0",
			count = "exact", cluster_id = NULL, bypass_federation = NULL)
		{
			endPoint <- stringr::str_interp("workflows")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("Bearer", private$token),
			                "Content-Type" = "application/json")
			queryArgs <- list(filters = filters, where = where,
							  order = order, select = select, distinct = distinct,
							  limit = limit, offset = offset, count = count,
							  cluster_id = cluster_id, bypass_federation = bypass_federation)

			body <- NULL

			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)

			if(!is.null(resource$errors))
				stop(resource$errors)

			resource
		},

		user_agreements.get = function(uuid)
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

		user_agreements.create = function(useragreement,
			ensure_unique_name = "false", cluster_id = NULL)
		{
			endPoint <- stringr::str_interp("user_agreements")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("Bearer", private$token),
			                "Content-Type" = "application/json")
			queryArgs <- list(ensure_unique_name = ensure_unique_name,
							  cluster_id = cluster_id)

			if(length(useragreement) > 0)
				body <- jsonlite::toJSON(list(useragreement = useragreement),
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

		user_agreements.update = function(useragreement, uuid)
		{
			endPoint <- stringr::str_interp("user_agreements/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("Bearer", private$token),
			                "Content-Type" = "application/json")
			queryArgs <- NULL

			if(length(useragreement) > 0)
				body <- jsonlite::toJSON(list(useragreement = useragreement),
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

		user_agreements.delete = function(uuid)
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

		user_agreements.signatures = function()
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

		user_agreements.sign = function()
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

		user_agreements.list = function(filters = NULL,
			where = NULL, order = NULL, select = NULL,
			distinct = NULL, limit = "100", offset = "0",
			count = "exact", cluster_id = NULL, bypass_federation = NULL)
		{
			endPoint <- stringr::str_interp("user_agreements")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("Bearer", private$token),
			                "Content-Type" = "application/json")
			queryArgs <- list(filters = filters, where = where,
							  order = order, select = select, distinct = distinct,
							  limit = limit, offset = offset, count = count,
							  cluster_id = cluster_id, bypass_federation = bypass_federation)

			body <- NULL

			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)

			if(!is.null(resource$errors))
				stop(resource$errors)

			resource
		},

		user_agreements.new = function()
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

		configs.get = function()
		{
			endPoint <- stringr::str_interp("config")
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
