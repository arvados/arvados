#' users.get is a method defined in Arvados class.
#' 
#' @usage arv$users.get(uuid)
#' @param uuid The UUID of the User in question.
#' @return User object.
#' @name users.get
NULL

#' users.index is a method defined in Arvados class.
#' 
#' @usage arv$users.index(filters = NULL,
#' 	where = NULL, order = NULL, select = NULL,
#' 	distinct = NULL, limit = "100", offset = "0",
#' 	count = "exact")
#' @param filters 
#' @param where 
#' @param order 
#' @param select 
#' @param distinct 
#' @param limit 
#' @param offset 
#' @param count 
#' @return UserList object.
#' @name users.index
NULL

#' users.create is a method defined in Arvados class.
#' 
#' @usage arv$users.create(user, ensure_unique_name = "false")
#' @param user User object.
#' @param ensure_unique_name Adjust name to ensure uniqueness instead of returning an error on (owner_uuid, name) collision.
#' @return User object.
#' @name users.create
NULL

#' users.update is a method defined in Arvados class.
#' 
#' @usage arv$users.update(user, uuid)
#' @param user User object.
#' @param uuid The UUID of the User in question.
#' @return User object.
#' @name users.update
NULL

#' users.delete is a method defined in Arvados class.
#' 
#' @usage arv$users.delete(uuid)
#' @param uuid The UUID of the User in question.
#' @return User object.
#' @name users.delete
NULL

#' users.current is a method defined in Arvados class.
#' 
#' @usage arv$users.current(NULL)
#' @return User object.
#' @name users.current
NULL

#' users.system is a method defined in Arvados class.
#' 
#' @usage arv$users.system(NULL)
#' @return User object.
#' @name users.system
NULL

#' users.activate is a method defined in Arvados class.
#' 
#' @usage arv$users.activate(uuid)
#' @param uuid 
#' @return User object.
#' @name users.activate
NULL

#' users.setup is a method defined in Arvados class.
#' 
#' @usage arv$users.setup(user = NULL, openid_prefix = NULL,
#' 	repo_name = NULL, vm_uuid = NULL, send_notification_email = "false")
#' @param user 
#' @param openid_prefix 
#' @param repo_name 
#' @param vm_uuid 
#' @param send_notification_email 
#' @return User object.
#' @name users.setup
NULL

#' users.unsetup is a method defined in Arvados class.
#' 
#' @usage arv$users.unsetup(uuid)
#' @param uuid 
#' @return User object.
#' @name users.unsetup
NULL

#' users.update_uuid is a method defined in Arvados class.
#' 
#' @usage arv$users.update_uuid(uuid, new_uuid)
#' @param uuid 
#' @param new_uuid 
#' @return User object.
#' @name users.update_uuid
NULL

#' users.list is a method defined in Arvados class.
#' 
#' @usage arv$users.list(filters = NULL,
#' 	where = NULL, order = NULL, select = NULL,
#' 	distinct = NULL, limit = "100", offset = "0",
#' 	count = "exact")
#' @param filters 
#' @param where 
#' @param order 
#' @param select 
#' @param distinct 
#' @param limit 
#' @param offset 
#' @param count 
#' @return UserList object.
#' @name users.list
NULL

#' users.show is a method defined in Arvados class.
#' 
#' @usage arv$users.show(uuid)
#' @param uuid 
#' @return User object.
#' @name users.show
NULL

#' users.destroy is a method defined in Arvados class.
#' 
#' @usage arv$users.destroy(uuid)
#' @param uuid 
#' @return User object.
#' @name users.destroy
NULL

#' api_client_authorizations.get is a method defined in Arvados class.
#' 
#' @usage arv$api_client_authorizations.get(uuid)
#' @param uuid The UUID of the ApiClientAuthorization in question.
#' @return ApiClientAuthorization object.
#' @name api_client_authorizations.get
NULL

#' api_client_authorizations.index is a method defined in Arvados class.
#' 
#' @usage arv$api_client_authorizations.index(filters = NULL,
#' 	where = NULL, order = NULL, select = NULL,
#' 	distinct = NULL, limit = "100", offset = "0",
#' 	count = "exact")
#' @param filters 
#' @param where 
#' @param order 
#' @param select 
#' @param distinct 
#' @param limit 
#' @param offset 
#' @param count 
#' @return ApiClientAuthorizationList object.
#' @name api_client_authorizations.index
NULL

#' api_client_authorizations.create is a method defined in Arvados class.
#' 
#' @usage arv$api_client_authorizations.create(api_client_authorization,
#' 	ensure_unique_name = "false")
#' @param apiClientAuthorization ApiClientAuthorization object.
#' @param ensure_unique_name Adjust name to ensure uniqueness instead of returning an error on (owner_uuid, name) collision.
#' @return ApiClientAuthorization object.
#' @name api_client_authorizations.create
NULL

#' api_client_authorizations.update is a method defined in Arvados class.
#' 
#' @usage arv$api_client_authorizations.update(api_client_authorization,
#' 	uuid)
#' @param apiClientAuthorization ApiClientAuthorization object.
#' @param uuid The UUID of the ApiClientAuthorization in question.
#' @return ApiClientAuthorization object.
#' @name api_client_authorizations.update
NULL

#' api_client_authorizations.delete is a method defined in Arvados class.
#' 
#' @usage arv$api_client_authorizations.delete(uuid)
#' @param uuid The UUID of the ApiClientAuthorization in question.
#' @return ApiClientAuthorization object.
#' @name api_client_authorizations.delete
NULL

#' api_client_authorizations.create_system_auth is a method defined in Arvados class.
#' 
#' @usage arv$api_client_authorizations.create_system_auth(api_client_id = NULL,
#' 	scopes = NULL)
#' @param api_client_id 
#' @param scopes 
#' @return ApiClientAuthorization object.
#' @name api_client_authorizations.create_system_auth
NULL

#' api_client_authorizations.current is a method defined in Arvados class.
#' 
#' @usage arv$api_client_authorizations.current(NULL)
#' @return ApiClientAuthorization object.
#' @name api_client_authorizations.current
NULL

#' api_client_authorizations.list is a method defined in Arvados class.
#' 
#' @usage arv$api_client_authorizations.list(filters = NULL,
#' 	where = NULL, order = NULL, select = NULL,
#' 	distinct = NULL, limit = "100", offset = "0",
#' 	count = "exact")
#' @param filters 
#' @param where 
#' @param order 
#' @param select 
#' @param distinct 
#' @param limit 
#' @param offset 
#' @param count 
#' @return ApiClientAuthorizationList object.
#' @name api_client_authorizations.list
NULL

#' api_client_authorizations.show is a method defined in Arvados class.
#' 
#' @usage arv$api_client_authorizations.show(uuid)
#' @param uuid 
#' @return ApiClientAuthorization object.
#' @name api_client_authorizations.show
NULL

#' api_client_authorizations.destroy is a method defined in Arvados class.
#' 
#' @usage arv$api_client_authorizations.destroy(uuid)
#' @param uuid 
#' @return ApiClientAuthorization object.
#' @name api_client_authorizations.destroy
NULL

#' api_clients.get is a method defined in Arvados class.
#' 
#' @usage arv$api_clients.get(uuid)
#' @param uuid The UUID of the ApiClient in question.
#' @return ApiClient object.
#' @name api_clients.get
NULL

#' api_clients.index is a method defined in Arvados class.
#' 
#' @usage arv$api_clients.index(filters = NULL,
#' 	where = NULL, order = NULL, select = NULL,
#' 	distinct = NULL, limit = "100", offset = "0",
#' 	count = "exact")
#' @param filters 
#' @param where 
#' @param order 
#' @param select 
#' @param distinct 
#' @param limit 
#' @param offset 
#' @param count 
#' @return ApiClientList object.
#' @name api_clients.index
NULL

#' api_clients.create is a method defined in Arvados class.
#' 
#' @usage arv$api_clients.create(api_client,
#' 	ensure_unique_name = "false")
#' @param apiClient ApiClient object.
#' @param ensure_unique_name Adjust name to ensure uniqueness instead of returning an error on (owner_uuid, name) collision.
#' @return ApiClient object.
#' @name api_clients.create
NULL

#' api_clients.update is a method defined in Arvados class.
#' 
#' @usage arv$api_clients.update(api_client,
#' 	uuid)
#' @param apiClient ApiClient object.
#' @param uuid The UUID of the ApiClient in question.
#' @return ApiClient object.
#' @name api_clients.update
NULL

#' api_clients.delete is a method defined in Arvados class.
#' 
#' @usage arv$api_clients.delete(uuid)
#' @param uuid The UUID of the ApiClient in question.
#' @return ApiClient object.
#' @name api_clients.delete
NULL

#' api_clients.list is a method defined in Arvados class.
#' 
#' @usage arv$api_clients.list(filters = NULL,
#' 	where = NULL, order = NULL, select = NULL,
#' 	distinct = NULL, limit = "100", offset = "0",
#' 	count = "exact")
#' @param filters 
#' @param where 
#' @param order 
#' @param select 
#' @param distinct 
#' @param limit 
#' @param offset 
#' @param count 
#' @return ApiClientList object.
#' @name api_clients.list
NULL

#' api_clients.show is a method defined in Arvados class.
#' 
#' @usage arv$api_clients.show(uuid)
#' @param uuid 
#' @return ApiClient object.
#' @name api_clients.show
NULL

#' api_clients.destroy is a method defined in Arvados class.
#' 
#' @usage arv$api_clients.destroy(uuid)
#' @param uuid 
#' @return ApiClient object.
#' @name api_clients.destroy
NULL

#' container_requests.get is a method defined in Arvados class.
#' 
#' @usage arv$container_requests.get(uuid)
#' @param uuid The UUID of the ContainerRequest in question.
#' @return ContainerRequest object.
#' @name container_requests.get
NULL

#' container_requests.index is a method defined in Arvados class.
#' 
#' @usage arv$container_requests.index(filters = NULL,
#' 	where = NULL, order = NULL, select = NULL,
#' 	distinct = NULL, limit = "100", offset = "0",
#' 	count = "exact")
#' @param filters 
#' @param where 
#' @param order 
#' @param select 
#' @param distinct 
#' @param limit 
#' @param offset 
#' @param count 
#' @return ContainerRequestList object.
#' @name container_requests.index
NULL

#' container_requests.create is a method defined in Arvados class.
#' 
#' @usage arv$container_requests.create(container_request,
#' 	ensure_unique_name = "false")
#' @param containerRequest ContainerRequest object.
#' @param ensure_unique_name Adjust name to ensure uniqueness instead of returning an error on (owner_uuid, name) collision.
#' @return ContainerRequest object.
#' @name container_requests.create
NULL

#' container_requests.update is a method defined in Arvados class.
#' 
#' @usage arv$container_requests.update(container_request,
#' 	uuid)
#' @param containerRequest ContainerRequest object.
#' @param uuid The UUID of the ContainerRequest in question.
#' @return ContainerRequest object.
#' @name container_requests.update
NULL

#' container_requests.delete is a method defined in Arvados class.
#' 
#' @usage arv$container_requests.delete(uuid)
#' @param uuid The UUID of the ContainerRequest in question.
#' @return ContainerRequest object.
#' @name container_requests.delete
NULL

#' container_requests.list is a method defined in Arvados class.
#' 
#' @usage arv$container_requests.list(filters = NULL,
#' 	where = NULL, order = NULL, select = NULL,
#' 	distinct = NULL, limit = "100", offset = "0",
#' 	count = "exact")
#' @param filters 
#' @param where 
#' @param order 
#' @param select 
#' @param distinct 
#' @param limit 
#' @param offset 
#' @param count 
#' @return ContainerRequestList object.
#' @name container_requests.list
NULL

#' container_requests.show is a method defined in Arvados class.
#' 
#' @usage arv$container_requests.show(uuid)
#' @param uuid 
#' @return ContainerRequest object.
#' @name container_requests.show
NULL

#' container_requests.destroy is a method defined in Arvados class.
#' 
#' @usage arv$container_requests.destroy(uuid)
#' @param uuid 
#' @return ContainerRequest object.
#' @name container_requests.destroy
NULL

#' authorized_keys.get is a method defined in Arvados class.
#' 
#' @usage arv$authorized_keys.get(uuid)
#' @param uuid The UUID of the AuthorizedKey in question.
#' @return AuthorizedKey object.
#' @name authorized_keys.get
NULL

#' authorized_keys.index is a method defined in Arvados class.
#' 
#' @usage arv$authorized_keys.index(filters = NULL,
#' 	where = NULL, order = NULL, select = NULL,
#' 	distinct = NULL, limit = "100", offset = "0",
#' 	count = "exact")
#' @param filters 
#' @param where 
#' @param order 
#' @param select 
#' @param distinct 
#' @param limit 
#' @param offset 
#' @param count 
#' @return AuthorizedKeyList object.
#' @name authorized_keys.index
NULL

#' authorized_keys.create is a method defined in Arvados class.
#' 
#' @usage arv$authorized_keys.create(authorized_key,
#' 	ensure_unique_name = "false")
#' @param authorizedKey AuthorizedKey object.
#' @param ensure_unique_name Adjust name to ensure uniqueness instead of returning an error on (owner_uuid, name) collision.
#' @return AuthorizedKey object.
#' @name authorized_keys.create
NULL

#' authorized_keys.update is a method defined in Arvados class.
#' 
#' @usage arv$authorized_keys.update(authorized_key,
#' 	uuid)
#' @param authorizedKey AuthorizedKey object.
#' @param uuid The UUID of the AuthorizedKey in question.
#' @return AuthorizedKey object.
#' @name authorized_keys.update
NULL

#' authorized_keys.delete is a method defined in Arvados class.
#' 
#' @usage arv$authorized_keys.delete(uuid)
#' @param uuid The UUID of the AuthorizedKey in question.
#' @return AuthorizedKey object.
#' @name authorized_keys.delete
NULL

#' authorized_keys.list is a method defined in Arvados class.
#' 
#' @usage arv$authorized_keys.list(filters = NULL,
#' 	where = NULL, order = NULL, select = NULL,
#' 	distinct = NULL, limit = "100", offset = "0",
#' 	count = "exact")
#' @param filters 
#' @param where 
#' @param order 
#' @param select 
#' @param distinct 
#' @param limit 
#' @param offset 
#' @param count 
#' @return AuthorizedKeyList object.
#' @name authorized_keys.list
NULL

#' authorized_keys.show is a method defined in Arvados class.
#' 
#' @usage arv$authorized_keys.show(uuid)
#' @param uuid 
#' @return AuthorizedKey object.
#' @name authorized_keys.show
NULL

#' authorized_keys.destroy is a method defined in Arvados class.
#' 
#' @usage arv$authorized_keys.destroy(uuid)
#' @param uuid 
#' @return AuthorizedKey object.
#' @name authorized_keys.destroy
NULL

#' collections.get is a method defined in Arvados class.
#' 
#' @usage arv$collections.get(uuid)
#' @param uuid The UUID of the Collection in question.
#' @return Collection object.
#' @name collections.get
NULL

#' collections.index is a method defined in Arvados class.
#' 
#' @usage arv$collections.index(filters = NULL,
#' 	where = NULL, order = NULL, select = NULL,
#' 	distinct = NULL, limit = "100", offset = "0",
#' 	count = "exact", include_trash = NULL)
#' @param filters 
#' @param where 
#' @param order 
#' @param select 
#' @param distinct 
#' @param limit 
#' @param offset 
#' @param count 
#' @param include_trash Include collections whose is_trashed attribute is true.
#' @return CollectionList object.
#' @name collections.index
NULL

#' collections.create is a method defined in Arvados class.
#' 
#' @usage arv$collections.create(collection,
#' 	ensure_unique_name = "false")
#' @param collection Collection object.
#' @param ensure_unique_name Adjust name to ensure uniqueness instead of returning an error on (owner_uuid, name) collision.
#' @return Collection object.
#' @name collections.create
NULL

#' collections.update is a method defined in Arvados class.
#' 
#' @usage arv$collections.update(collection,
#' 	uuid)
#' @param collection Collection object.
#' @param uuid The UUID of the Collection in question.
#' @return Collection object.
#' @name collections.update
NULL

#' collections.delete is a method defined in Arvados class.
#' 
#' @usage arv$collections.delete(uuid)
#' @param uuid The UUID of the Collection in question.
#' @return Collection object.
#' @name collections.delete
NULL

#' collections.provenance is a method defined in Arvados class.
#' 
#' @usage arv$collections.provenance(uuid)
#' @param uuid 
#' @return Collection object.
#' @name collections.provenance
NULL

#' collections.used_by is a method defined in Arvados class.
#' 
#' @usage arv$collections.used_by(uuid)
#' @param uuid 
#' @return Collection object.
#' @name collections.used_by
NULL

#' collections.trash is a method defined in Arvados class.
#' 
#' @usage arv$collections.trash(uuid)
#' @param uuid 
#' @return Collection object.
#' @name collections.trash
NULL

#' collections.untrash is a method defined in Arvados class.
#' 
#' @usage arv$collections.untrash(uuid)
#' @param uuid 
#' @return Collection object.
#' @name collections.untrash
NULL

#' collections.list is a method defined in Arvados class.
#' 
#' @usage arv$collections.list(filters = NULL,
#' 	where = NULL, order = NULL, select = NULL,
#' 	distinct = NULL, limit = "100", offset = "0",
#' 	count = "exact", include_trash = NULL)
#' @param filters 
#' @param where 
#' @param order 
#' @param select 
#' @param distinct 
#' @param limit 
#' @param offset 
#' @param count 
#' @param include_trash Include collections whose is_trashed attribute is true.
#' @return CollectionList object.
#' @name collections.list
NULL

#' collections.show is a method defined in Arvados class.
#' 
#' @usage arv$collections.show(uuid)
#' @param uuid 
#' @return Collection object.
#' @name collections.show
NULL

#' collections.destroy is a method defined in Arvados class.
#' 
#' @usage arv$collections.destroy(uuid)
#' @param uuid 
#' @return Collection object.
#' @name collections.destroy
NULL

#' containers.get is a method defined in Arvados class.
#' 
#' @usage arv$containers.get(uuid)
#' @param uuid The UUID of the Container in question.
#' @return Container object.
#' @name containers.get
NULL

#' containers.index is a method defined in Arvados class.
#' 
#' @usage arv$containers.index(filters = NULL,
#' 	where = NULL, order = NULL, select = NULL,
#' 	distinct = NULL, limit = "100", offset = "0",
#' 	count = "exact")
#' @param filters 
#' @param where 
#' @param order 
#' @param select 
#' @param distinct 
#' @param limit 
#' @param offset 
#' @param count 
#' @return ContainerList object.
#' @name containers.index
NULL

#' containers.create is a method defined in Arvados class.
#' 
#' @usage arv$containers.create(container,
#' 	ensure_unique_name = "false")
#' @param container Container object.
#' @param ensure_unique_name Adjust name to ensure uniqueness instead of returning an error on (owner_uuid, name) collision.
#' @return Container object.
#' @name containers.create
NULL

#' containers.update is a method defined in Arvados class.
#' 
#' @usage arv$containers.update(container,
#' 	uuid)
#' @param container Container object.
#' @param uuid The UUID of the Container in question.
#' @return Container object.
#' @name containers.update
NULL

#' containers.delete is a method defined in Arvados class.
#' 
#' @usage arv$containers.delete(uuid)
#' @param uuid The UUID of the Container in question.
#' @return Container object.
#' @name containers.delete
NULL

#' containers.auth is a method defined in Arvados class.
#' 
#' @usage arv$containers.auth(uuid)
#' @param uuid 
#' @return Container object.
#' @name containers.auth
NULL

#' containers.lock is a method defined in Arvados class.
#' 
#' @usage arv$containers.lock(uuid)
#' @param uuid 
#' @return Container object.
#' @name containers.lock
NULL

#' containers.unlock is a method defined in Arvados class.
#' 
#' @usage arv$containers.unlock(uuid)
#' @param uuid 
#' @return Container object.
#' @name containers.unlock
NULL

#' containers.current is a method defined in Arvados class.
#' 
#' @usage arv$containers.current(NULL)
#' @return Container object.
#' @name containers.current
NULL

#' containers.list is a method defined in Arvados class.
#' 
#' @usage arv$containers.list(filters = NULL,
#' 	where = NULL, order = NULL, select = NULL,
#' 	distinct = NULL, limit = "100", offset = "0",
#' 	count = "exact")
#' @param filters 
#' @param where 
#' @param order 
#' @param select 
#' @param distinct 
#' @param limit 
#' @param offset 
#' @param count 
#' @return ContainerList object.
#' @name containers.list
NULL

#' containers.show is a method defined in Arvados class.
#' 
#' @usage arv$containers.show(uuid)
#' @param uuid 
#' @return Container object.
#' @name containers.show
NULL

#' containers.destroy is a method defined in Arvados class.
#' 
#' @usage arv$containers.destroy(uuid)
#' @param uuid 
#' @return Container object.
#' @name containers.destroy
NULL

#' humans.get is a method defined in Arvados class.
#' 
#' @usage arv$humans.get(uuid)
#' @param uuid The UUID of the Human in question.
#' @return Human object.
#' @name humans.get
NULL

#' humans.index is a method defined in Arvados class.
#' 
#' @usage arv$humans.index(filters = NULL,
#' 	where = NULL, order = NULL, select = NULL,
#' 	distinct = NULL, limit = "100", offset = "0",
#' 	count = "exact")
#' @param filters 
#' @param where 
#' @param order 
#' @param select 
#' @param distinct 
#' @param limit 
#' @param offset 
#' @param count 
#' @return HumanList object.
#' @name humans.index
NULL

#' humans.create is a method defined in Arvados class.
#' 
#' @usage arv$humans.create(human, ensure_unique_name = "false")
#' @param human Human object.
#' @param ensure_unique_name Adjust name to ensure uniqueness instead of returning an error on (owner_uuid, name) collision.
#' @return Human object.
#' @name humans.create
NULL

#' humans.update is a method defined in Arvados class.
#' 
#' @usage arv$humans.update(human, uuid)
#' @param human Human object.
#' @param uuid The UUID of the Human in question.
#' @return Human object.
#' @name humans.update
NULL

#' humans.delete is a method defined in Arvados class.
#' 
#' @usage arv$humans.delete(uuid)
#' @param uuid The UUID of the Human in question.
#' @return Human object.
#' @name humans.delete
NULL

#' humans.list is a method defined in Arvados class.
#' 
#' @usage arv$humans.list(filters = NULL,
#' 	where = NULL, order = NULL, select = NULL,
#' 	distinct = NULL, limit = "100", offset = "0",
#' 	count = "exact")
#' @param filters 
#' @param where 
#' @param order 
#' @param select 
#' @param distinct 
#' @param limit 
#' @param offset 
#' @param count 
#' @return HumanList object.
#' @name humans.list
NULL

#' humans.show is a method defined in Arvados class.
#' 
#' @usage arv$humans.show(uuid)
#' @param uuid 
#' @return Human object.
#' @name humans.show
NULL

#' humans.destroy is a method defined in Arvados class.
#' 
#' @usage arv$humans.destroy(uuid)
#' @param uuid 
#' @return Human object.
#' @name humans.destroy
NULL

#' job_tasks.get is a method defined in Arvados class.
#' 
#' @usage arv$job_tasks.get(uuid)
#' @param uuid The UUID of the JobTask in question.
#' @return JobTask object.
#' @name job_tasks.get
NULL

#' job_tasks.index is a method defined in Arvados class.
#' 
#' @usage arv$job_tasks.index(filters = NULL,
#' 	where = NULL, order = NULL, select = NULL,
#' 	distinct = NULL, limit = "100", offset = "0",
#' 	count = "exact")
#' @param filters 
#' @param where 
#' @param order 
#' @param select 
#' @param distinct 
#' @param limit 
#' @param offset 
#' @param count 
#' @return JobTaskList object.
#' @name job_tasks.index
NULL

#' job_tasks.create is a method defined in Arvados class.
#' 
#' @usage arv$job_tasks.create(job_task,
#' 	ensure_unique_name = "false")
#' @param jobTask JobTask object.
#' @param ensure_unique_name Adjust name to ensure uniqueness instead of returning an error on (owner_uuid, name) collision.
#' @return JobTask object.
#' @name job_tasks.create
NULL

#' job_tasks.update is a method defined in Arvados class.
#' 
#' @usage arv$job_tasks.update(job_task,
#' 	uuid)
#' @param jobTask JobTask object.
#' @param uuid The UUID of the JobTask in question.
#' @return JobTask object.
#' @name job_tasks.update
NULL

#' job_tasks.delete is a method defined in Arvados class.
#' 
#' @usage arv$job_tasks.delete(uuid)
#' @param uuid The UUID of the JobTask in question.
#' @return JobTask object.
#' @name job_tasks.delete
NULL

#' job_tasks.list is a method defined in Arvados class.
#' 
#' @usage arv$job_tasks.list(filters = NULL,
#' 	where = NULL, order = NULL, select = NULL,
#' 	distinct = NULL, limit = "100", offset = "0",
#' 	count = "exact")
#' @param filters 
#' @param where 
#' @param order 
#' @param select 
#' @param distinct 
#' @param limit 
#' @param offset 
#' @param count 
#' @return JobTaskList object.
#' @name job_tasks.list
NULL

#' job_tasks.show is a method defined in Arvados class.
#' 
#' @usage arv$job_tasks.show(uuid)
#' @param uuid 
#' @return JobTask object.
#' @name job_tasks.show
NULL

#' job_tasks.destroy is a method defined in Arvados class.
#' 
#' @usage arv$job_tasks.destroy(uuid)
#' @param uuid 
#' @return JobTask object.
#' @name job_tasks.destroy
NULL

#' links.get is a method defined in Arvados class.
#' 
#' @usage arv$links.get(uuid)
#' @param uuid The UUID of the Link in question.
#' @return Link object.
#' @name links.get
NULL

#' links.index is a method defined in Arvados class.
#' 
#' @usage arv$links.index(filters = NULL,
#' 	where = NULL, order = NULL, select = NULL,
#' 	distinct = NULL, limit = "100", offset = "0",
#' 	count = "exact")
#' @param filters 
#' @param where 
#' @param order 
#' @param select 
#' @param distinct 
#' @param limit 
#' @param offset 
#' @param count 
#' @return LinkList object.
#' @name links.index
NULL

#' links.create is a method defined in Arvados class.
#' 
#' @usage arv$links.create(link, ensure_unique_name = "false")
#' @param link Link object.
#' @param ensure_unique_name Adjust name to ensure uniqueness instead of returning an error on (owner_uuid, name) collision.
#' @return Link object.
#' @name links.create
NULL

#' links.update is a method defined in Arvados class.
#' 
#' @usage arv$links.update(link, uuid)
#' @param link Link object.
#' @param uuid The UUID of the Link in question.
#' @return Link object.
#' @name links.update
NULL

#' links.delete is a method defined in Arvados class.
#' 
#' @usage arv$links.delete(uuid)
#' @param uuid The UUID of the Link in question.
#' @return Link object.
#' @name links.delete
NULL

#' links.list is a method defined in Arvados class.
#' 
#' @usage arv$links.list(filters = NULL,
#' 	where = NULL, order = NULL, select = NULL,
#' 	distinct = NULL, limit = "100", offset = "0",
#' 	count = "exact")
#' @param filters 
#' @param where 
#' @param order 
#' @param select 
#' @param distinct 
#' @param limit 
#' @param offset 
#' @param count 
#' @return LinkList object.
#' @name links.list
NULL

#' links.show is a method defined in Arvados class.
#' 
#' @usage arv$links.show(uuid)
#' @param uuid 
#' @return Link object.
#' @name links.show
NULL

#' links.destroy is a method defined in Arvados class.
#' 
#' @usage arv$links.destroy(uuid)
#' @param uuid 
#' @return Link object.
#' @name links.destroy
NULL

#' links.get_permissions is a method defined in Arvados class.
#' 
#' @usage arv$links.get_permissions(uuid)
#' @param uuid 
#' @return Link object.
#' @name links.get_permissions
NULL

#' jobs.get is a method defined in Arvados class.
#' 
#' @usage arv$jobs.get(uuid)
#' @param uuid The UUID of the Job in question.
#' @return Job object.
#' @name jobs.get
NULL

#' jobs.index is a method defined in Arvados class.
#' 
#' @usage arv$jobs.index(filters = NULL,
#' 	where = NULL, order = NULL, select = NULL,
#' 	distinct = NULL, limit = "100", offset = "0",
#' 	count = "exact")
#' @param filters 
#' @param where 
#' @param order 
#' @param select 
#' @param distinct 
#' @param limit 
#' @param offset 
#' @param count 
#' @return JobList object.
#' @name jobs.index
NULL

#' jobs.create is a method defined in Arvados class.
#' 
#' @usage arv$jobs.create(job, ensure_unique_name = "false",
#' 	find_or_create = "false", filters = NULL,
#' 	minimum_script_version = NULL, exclude_script_versions = NULL)
#' @param job Job object.
#' @param ensure_unique_name Adjust name to ensure uniqueness instead of returning an error on (owner_uuid, name) collision.
#' @param find_or_create 
#' @param filters 
#' @param minimum_script_version 
#' @param exclude_script_versions 
#' @return Job object.
#' @name jobs.create
NULL

#' jobs.update is a method defined in Arvados class.
#' 
#' @usage arv$jobs.update(job, uuid)
#' @param job Job object.
#' @param uuid The UUID of the Job in question.
#' @return Job object.
#' @name jobs.update
NULL

#' jobs.delete is a method defined in Arvados class.
#' 
#' @usage arv$jobs.delete(uuid)
#' @param uuid The UUID of the Job in question.
#' @return Job object.
#' @name jobs.delete
NULL

#' jobs.queue is a method defined in Arvados class.
#' 
#' @usage arv$jobs.queue(filters = NULL,
#' 	where = NULL, order = NULL, select = NULL,
#' 	distinct = NULL, limit = "100", offset = "0",
#' 	count = "exact")
#' @param filters 
#' @param where 
#' @param order 
#' @param select 
#' @param distinct 
#' @param limit 
#' @param offset 
#' @param count 
#' @return Job object.
#' @name jobs.queue
NULL

#' jobs.queue_size is a method defined in Arvados class.
#' 
#' @usage arv$jobs.queue_size(NULL)
#' @return Job object.
#' @name jobs.queue_size
NULL

#' jobs.cancel is a method defined in Arvados class.
#' 
#' @usage arv$jobs.cancel(uuid)
#' @param uuid 
#' @return Job object.
#' @name jobs.cancel
NULL

#' jobs.lock is a method defined in Arvados class.
#' 
#' @usage arv$jobs.lock(uuid)
#' @param uuid 
#' @return Job object.
#' @name jobs.lock
NULL

#' jobs.list is a method defined in Arvados class.
#' 
#' @usage arv$jobs.list(filters = NULL, where = NULL,
#' 	order = NULL, select = NULL, distinct = NULL,
#' 	limit = "100", offset = "0", count = "exact")
#' @param filters 
#' @param where 
#' @param order 
#' @param select 
#' @param distinct 
#' @param limit 
#' @param offset 
#' @param count 
#' @return JobList object.
#' @name jobs.list
NULL

#' jobs.show is a method defined in Arvados class.
#' 
#' @usage arv$jobs.show(uuid)
#' @param uuid 
#' @return Job object.
#' @name jobs.show
NULL

#' jobs.destroy is a method defined in Arvados class.
#' 
#' @usage arv$jobs.destroy(uuid)
#' @param uuid 
#' @return Job object.
#' @name jobs.destroy
NULL

#' keep_disks.get is a method defined in Arvados class.
#' 
#' @usage arv$keep_disks.get(uuid)
#' @param uuid The UUID of the KeepDisk in question.
#' @return KeepDisk object.
#' @name keep_disks.get
NULL

#' keep_disks.index is a method defined in Arvados class.
#' 
#' @usage arv$keep_disks.index(filters = NULL,
#' 	where = NULL, order = NULL, select = NULL,
#' 	distinct = NULL, limit = "100", offset = "0",
#' 	count = "exact")
#' @param filters 
#' @param where 
#' @param order 
#' @param select 
#' @param distinct 
#' @param limit 
#' @param offset 
#' @param count 
#' @return KeepDiskList object.
#' @name keep_disks.index
NULL

#' keep_disks.create is a method defined in Arvados class.
#' 
#' @usage arv$keep_disks.create(keep_disk,
#' 	ensure_unique_name = "false")
#' @param keepDisk KeepDisk object.
#' @param ensure_unique_name Adjust name to ensure uniqueness instead of returning an error on (owner_uuid, name) collision.
#' @return KeepDisk object.
#' @name keep_disks.create
NULL

#' keep_disks.update is a method defined in Arvados class.
#' 
#' @usage arv$keep_disks.update(keep_disk,
#' 	uuid)
#' @param keepDisk KeepDisk object.
#' @param uuid The UUID of the KeepDisk in question.
#' @return KeepDisk object.
#' @name keep_disks.update
NULL

#' keep_disks.delete is a method defined in Arvados class.
#' 
#' @usage arv$keep_disks.delete(uuid)
#' @param uuid The UUID of the KeepDisk in question.
#' @return KeepDisk object.
#' @name keep_disks.delete
NULL

#' keep_disks.ping is a method defined in Arvados class.
#' 
#' @usage arv$keep_disks.ping(uuid = NULL,
#' 	ping_secret, node_uuid = NULL, filesystem_uuid = NULL,
#' 	service_host = NULL, service_port, service_ssl_flag)
#' @param uuid 
#' @param ping_secret 
#' @param node_uuid 
#' @param filesystem_uuid 
#' @param service_host 
#' @param service_port 
#' @param service_ssl_flag 
#' @return KeepDisk object.
#' @name keep_disks.ping
NULL

#' keep_disks.list is a method defined in Arvados class.
#' 
#' @usage arv$keep_disks.list(filters = NULL,
#' 	where = NULL, order = NULL, select = NULL,
#' 	distinct = NULL, limit = "100", offset = "0",
#' 	count = "exact")
#' @param filters 
#' @param where 
#' @param order 
#' @param select 
#' @param distinct 
#' @param limit 
#' @param offset 
#' @param count 
#' @return KeepDiskList object.
#' @name keep_disks.list
NULL

#' keep_disks.show is a method defined in Arvados class.
#' 
#' @usage arv$keep_disks.show(uuid)
#' @param uuid 
#' @return KeepDisk object.
#' @name keep_disks.show
NULL

#' keep_disks.destroy is a method defined in Arvados class.
#' 
#' @usage arv$keep_disks.destroy(uuid)
#' @param uuid 
#' @return KeepDisk object.
#' @name keep_disks.destroy
NULL

#' keep_services.get is a method defined in Arvados class.
#' 
#' @usage arv$keep_services.get(uuid)
#' @param uuid The UUID of the KeepService in question.
#' @return KeepService object.
#' @name keep_services.get
NULL

#' keep_services.index is a method defined in Arvados class.
#' 
#' @usage arv$keep_services.index(filters = NULL,
#' 	where = NULL, order = NULL, select = NULL,
#' 	distinct = NULL, limit = "100", offset = "0",
#' 	count = "exact")
#' @param filters 
#' @param where 
#' @param order 
#' @param select 
#' @param distinct 
#' @param limit 
#' @param offset 
#' @param count 
#' @return KeepServiceList object.
#' @name keep_services.index
NULL

#' keep_services.create is a method defined in Arvados class.
#' 
#' @usage arv$keep_services.create(keep_service,
#' 	ensure_unique_name = "false")
#' @param keepService KeepService object.
#' @param ensure_unique_name Adjust name to ensure uniqueness instead of returning an error on (owner_uuid, name) collision.
#' @return KeepService object.
#' @name keep_services.create
NULL

#' keep_services.update is a method defined in Arvados class.
#' 
#' @usage arv$keep_services.update(keep_service,
#' 	uuid)
#' @param keepService KeepService object.
#' @param uuid The UUID of the KeepService in question.
#' @return KeepService object.
#' @name keep_services.update
NULL

#' keep_services.delete is a method defined in Arvados class.
#' 
#' @usage arv$keep_services.delete(uuid)
#' @param uuid The UUID of the KeepService in question.
#' @return KeepService object.
#' @name keep_services.delete
NULL

#' keep_services.accessible is a method defined in Arvados class.
#' 
#' @usage arv$keep_services.accessible(NULL)
#' @return KeepService object.
#' @name keep_services.accessible
NULL

#' keep_services.list is a method defined in Arvados class.
#' 
#' @usage arv$keep_services.list(filters = NULL,
#' 	where = NULL, order = NULL, select = NULL,
#' 	distinct = NULL, limit = "100", offset = "0",
#' 	count = "exact")
#' @param filters 
#' @param where 
#' @param order 
#' @param select 
#' @param distinct 
#' @param limit 
#' @param offset 
#' @param count 
#' @return KeepServiceList object.
#' @name keep_services.list
NULL

#' keep_services.show is a method defined in Arvados class.
#' 
#' @usage arv$keep_services.show(uuid)
#' @param uuid 
#' @return KeepService object.
#' @name keep_services.show
NULL

#' keep_services.destroy is a method defined in Arvados class.
#' 
#' @usage arv$keep_services.destroy(uuid)
#' @param uuid 
#' @return KeepService object.
#' @name keep_services.destroy
NULL

#' pipeline_templates.get is a method defined in Arvados class.
#' 
#' @usage arv$pipeline_templates.get(uuid)
#' @param uuid The UUID of the PipelineTemplate in question.
#' @return PipelineTemplate object.
#' @name pipeline_templates.get
NULL

#' pipeline_templates.index is a method defined in Arvados class.
#' 
#' @usage arv$pipeline_templates.index(filters = NULL,
#' 	where = NULL, order = NULL, select = NULL,
#' 	distinct = NULL, limit = "100", offset = "0",
#' 	count = "exact")
#' @param filters 
#' @param where 
#' @param order 
#' @param select 
#' @param distinct 
#' @param limit 
#' @param offset 
#' @param count 
#' @return PipelineTemplateList object.
#' @name pipeline_templates.index
NULL

#' pipeline_templates.create is a method defined in Arvados class.
#' 
#' @usage arv$pipeline_templates.create(pipeline_template,
#' 	ensure_unique_name = "false")
#' @param pipelineTemplate PipelineTemplate object.
#' @param ensure_unique_name Adjust name to ensure uniqueness instead of returning an error on (owner_uuid, name) collision.
#' @return PipelineTemplate object.
#' @name pipeline_templates.create
NULL

#' pipeline_templates.update is a method defined in Arvados class.
#' 
#' @usage arv$pipeline_templates.update(pipeline_template,
#' 	uuid)
#' @param pipelineTemplate PipelineTemplate object.
#' @param uuid The UUID of the PipelineTemplate in question.
#' @return PipelineTemplate object.
#' @name pipeline_templates.update
NULL

#' pipeline_templates.delete is a method defined in Arvados class.
#' 
#' @usage arv$pipeline_templates.delete(uuid)
#' @param uuid The UUID of the PipelineTemplate in question.
#' @return PipelineTemplate object.
#' @name pipeline_templates.delete
NULL

#' pipeline_templates.list is a method defined in Arvados class.
#' 
#' @usage arv$pipeline_templates.list(filters = NULL,
#' 	where = NULL, order = NULL, select = NULL,
#' 	distinct = NULL, limit = "100", offset = "0",
#' 	count = "exact")
#' @param filters 
#' @param where 
#' @param order 
#' @param select 
#' @param distinct 
#' @param limit 
#' @param offset 
#' @param count 
#' @return PipelineTemplateList object.
#' @name pipeline_templates.list
NULL

#' pipeline_templates.show is a method defined in Arvados class.
#' 
#' @usage arv$pipeline_templates.show(uuid)
#' @param uuid 
#' @return PipelineTemplate object.
#' @name pipeline_templates.show
NULL

#' pipeline_templates.destroy is a method defined in Arvados class.
#' 
#' @usage arv$pipeline_templates.destroy(uuid)
#' @param uuid 
#' @return PipelineTemplate object.
#' @name pipeline_templates.destroy
NULL

#' pipeline_instances.get is a method defined in Arvados class.
#' 
#' @usage arv$pipeline_instances.get(uuid)
#' @param uuid The UUID of the PipelineInstance in question.
#' @return PipelineInstance object.
#' @name pipeline_instances.get
NULL

#' pipeline_instances.index is a method defined in Arvados class.
#' 
#' @usage arv$pipeline_instances.index(filters = NULL,
#' 	where = NULL, order = NULL, select = NULL,
#' 	distinct = NULL, limit = "100", offset = "0",
#' 	count = "exact")
#' @param filters 
#' @param where 
#' @param order 
#' @param select 
#' @param distinct 
#' @param limit 
#' @param offset 
#' @param count 
#' @return PipelineInstanceList object.
#' @name pipeline_instances.index
NULL

#' pipeline_instances.create is a method defined in Arvados class.
#' 
#' @usage arv$pipeline_instances.create(pipeline_instance,
#' 	ensure_unique_name = "false")
#' @param pipelineInstance PipelineInstance object.
#' @param ensure_unique_name Adjust name to ensure uniqueness instead of returning an error on (owner_uuid, name) collision.
#' @return PipelineInstance object.
#' @name pipeline_instances.create
NULL

#' pipeline_instances.update is a method defined in Arvados class.
#' 
#' @usage arv$pipeline_instances.update(pipeline_instance,
#' 	uuid)
#' @param pipelineInstance PipelineInstance object.
#' @param uuid The UUID of the PipelineInstance in question.
#' @return PipelineInstance object.
#' @name pipeline_instances.update
NULL

#' pipeline_instances.delete is a method defined in Arvados class.
#' 
#' @usage arv$pipeline_instances.delete(uuid)
#' @param uuid The UUID of the PipelineInstance in question.
#' @return PipelineInstance object.
#' @name pipeline_instances.delete
NULL

#' pipeline_instances.cancel is a method defined in Arvados class.
#' 
#' @usage arv$pipeline_instances.cancel(uuid)
#' @param uuid 
#' @return PipelineInstance object.
#' @name pipeline_instances.cancel
NULL

#' pipeline_instances.list is a method defined in Arvados class.
#' 
#' @usage arv$pipeline_instances.list(filters = NULL,
#' 	where = NULL, order = NULL, select = NULL,
#' 	distinct = NULL, limit = "100", offset = "0",
#' 	count = "exact")
#' @param filters 
#' @param where 
#' @param order 
#' @param select 
#' @param distinct 
#' @param limit 
#' @param offset 
#' @param count 
#' @return PipelineInstanceList object.
#' @name pipeline_instances.list
NULL

#' pipeline_instances.show is a method defined in Arvados class.
#' 
#' @usage arv$pipeline_instances.show(uuid)
#' @param uuid 
#' @return PipelineInstance object.
#' @name pipeline_instances.show
NULL

#' pipeline_instances.destroy is a method defined in Arvados class.
#' 
#' @usage arv$pipeline_instances.destroy(uuid)
#' @param uuid 
#' @return PipelineInstance object.
#' @name pipeline_instances.destroy
NULL

#' nodes.get is a method defined in Arvados class.
#' 
#' @usage arv$nodes.get(uuid)
#' @param uuid The UUID of the Node in question.
#' @return Node object.
#' @name nodes.get
NULL

#' nodes.index is a method defined in Arvados class.
#' 
#' @usage arv$nodes.index(filters = NULL,
#' 	where = NULL, order = NULL, select = NULL,
#' 	distinct = NULL, limit = "100", offset = "0",
#' 	count = "exact")
#' @param filters 
#' @param where 
#' @param order 
#' @param select 
#' @param distinct 
#' @param limit 
#' @param offset 
#' @param count 
#' @return NodeList object.
#' @name nodes.index
NULL

#' nodes.create is a method defined in Arvados class.
#' 
#' @usage arv$nodes.create(node, ensure_unique_name = "false",
#' 	assign_slot = NULL)
#' @param node Node object.
#' @param ensure_unique_name Adjust name to ensure uniqueness instead of returning an error on (owner_uuid, name) collision.
#' @param assign_slot assign slot and hostname
#' @return Node object.
#' @name nodes.create
NULL

#' nodes.update is a method defined in Arvados class.
#' 
#' @usage arv$nodes.update(node, uuid, assign_slot = NULL)
#' @param node Node object.
#' @param uuid The UUID of the Node in question.
#' @param assign_slot assign slot and hostname
#' @return Node object.
#' @name nodes.update
NULL

#' nodes.delete is a method defined in Arvados class.
#' 
#' @usage arv$nodes.delete(uuid)
#' @param uuid The UUID of the Node in question.
#' @return Node object.
#' @name nodes.delete
NULL

#' nodes.ping is a method defined in Arvados class.
#' 
#' @usage arv$nodes.ping(uuid, ping_secret)
#' @param uuid 
#' @param ping_secret 
#' @return Node object.
#' @name nodes.ping
NULL

#' nodes.list is a method defined in Arvados class.
#' 
#' @usage arv$nodes.list(filters = NULL,
#' 	where = NULL, order = NULL, select = NULL,
#' 	distinct = NULL, limit = "100", offset = "0",
#' 	count = "exact")
#' @param filters 
#' @param where 
#' @param order 
#' @param select 
#' @param distinct 
#' @param limit 
#' @param offset 
#' @param count 
#' @return NodeList object.
#' @name nodes.list
NULL

#' nodes.show is a method defined in Arvados class.
#' 
#' @usage arv$nodes.show(uuid)
#' @param uuid 
#' @return Node object.
#' @name nodes.show
NULL

#' nodes.destroy is a method defined in Arvados class.
#' 
#' @usage arv$nodes.destroy(uuid)
#' @param uuid 
#' @return Node object.
#' @name nodes.destroy
NULL

#' repositories.get is a method defined in Arvados class.
#' 
#' @usage arv$repositories.get(uuid)
#' @param uuid The UUID of the Repository in question.
#' @return Repository object.
#' @name repositories.get
NULL

#' repositories.index is a method defined in Arvados class.
#' 
#' @usage arv$repositories.index(filters = NULL,
#' 	where = NULL, order = NULL, select = NULL,
#' 	distinct = NULL, limit = "100", offset = "0",
#' 	count = "exact")
#' @param filters 
#' @param where 
#' @param order 
#' @param select 
#' @param distinct 
#' @param limit 
#' @param offset 
#' @param count 
#' @return RepositoryList object.
#' @name repositories.index
NULL

#' repositories.create is a method defined in Arvados class.
#' 
#' @usage arv$repositories.create(repository,
#' 	ensure_unique_name = "false")
#' @param repository Repository object.
#' @param ensure_unique_name Adjust name to ensure uniqueness instead of returning an error on (owner_uuid, name) collision.
#' @return Repository object.
#' @name repositories.create
NULL

#' repositories.update is a method defined in Arvados class.
#' 
#' @usage arv$repositories.update(repository,
#' 	uuid)
#' @param repository Repository object.
#' @param uuid The UUID of the Repository in question.
#' @return Repository object.
#' @name repositories.update
NULL

#' repositories.delete is a method defined in Arvados class.
#' 
#' @usage arv$repositories.delete(uuid)
#' @param uuid The UUID of the Repository in question.
#' @return Repository object.
#' @name repositories.delete
NULL

#' repositories.get_all_permissions is a method defined in Arvados class.
#' 
#' @usage arv$repositories.get_all_permissions(NULL)
#' @return Repository object.
#' @name repositories.get_all_permissions
NULL

#' repositories.list is a method defined in Arvados class.
#' 
#' @usage arv$repositories.list(filters = NULL,
#' 	where = NULL, order = NULL, select = NULL,
#' 	distinct = NULL, limit = "100", offset = "0",
#' 	count = "exact")
#' @param filters 
#' @param where 
#' @param order 
#' @param select 
#' @param distinct 
#' @param limit 
#' @param offset 
#' @param count 
#' @return RepositoryList object.
#' @name repositories.list
NULL

#' repositories.show is a method defined in Arvados class.
#' 
#' @usage arv$repositories.show(uuid)
#' @param uuid 
#' @return Repository object.
#' @name repositories.show
NULL

#' repositories.destroy is a method defined in Arvados class.
#' 
#' @usage arv$repositories.destroy(uuid)
#' @param uuid 
#' @return Repository object.
#' @name repositories.destroy
NULL

#' specimens.get is a method defined in Arvados class.
#' 
#' @usage arv$specimens.get(uuid)
#' @param uuid The UUID of the Specimen in question.
#' @return Specimen object.
#' @name specimens.get
NULL

#' specimens.index is a method defined in Arvados class.
#' 
#' @usage arv$specimens.index(filters = NULL,
#' 	where = NULL, order = NULL, select = NULL,
#' 	distinct = NULL, limit = "100", offset = "0",
#' 	count = "exact")
#' @param filters 
#' @param where 
#' @param order 
#' @param select 
#' @param distinct 
#' @param limit 
#' @param offset 
#' @param count 
#' @return SpecimenList object.
#' @name specimens.index
NULL

#' specimens.create is a method defined in Arvados class.
#' 
#' @usage arv$specimens.create(specimen,
#' 	ensure_unique_name = "false")
#' @param specimen Specimen object.
#' @param ensure_unique_name Adjust name to ensure uniqueness instead of returning an error on (owner_uuid, name) collision.
#' @return Specimen object.
#' @name specimens.create
NULL

#' specimens.update is a method defined in Arvados class.
#' 
#' @usage arv$specimens.update(specimen,
#' 	uuid)
#' @param specimen Specimen object.
#' @param uuid The UUID of the Specimen in question.
#' @return Specimen object.
#' @name specimens.update
NULL

#' specimens.delete is a method defined in Arvados class.
#' 
#' @usage arv$specimens.delete(uuid)
#' @param uuid The UUID of the Specimen in question.
#' @return Specimen object.
#' @name specimens.delete
NULL

#' specimens.list is a method defined in Arvados class.
#' 
#' @usage arv$specimens.list(filters = NULL,
#' 	where = NULL, order = NULL, select = NULL,
#' 	distinct = NULL, limit = "100", offset = "0",
#' 	count = "exact")
#' @param filters 
#' @param where 
#' @param order 
#' @param select 
#' @param distinct 
#' @param limit 
#' @param offset 
#' @param count 
#' @return SpecimenList object.
#' @name specimens.list
NULL

#' specimens.show is a method defined in Arvados class.
#' 
#' @usage arv$specimens.show(uuid)
#' @param uuid 
#' @return Specimen object.
#' @name specimens.show
NULL

#' specimens.destroy is a method defined in Arvados class.
#' 
#' @usage arv$specimens.destroy(uuid)
#' @param uuid 
#' @return Specimen object.
#' @name specimens.destroy
NULL

#' logs.get is a method defined in Arvados class.
#' 
#' @usage arv$logs.get(uuid)
#' @param uuid The UUID of the Log in question.
#' @return Log object.
#' @name logs.get
NULL

#' logs.index is a method defined in Arvados class.
#' 
#' @usage arv$logs.index(filters = NULL,
#' 	where = NULL, order = NULL, select = NULL,
#' 	distinct = NULL, limit = "100", offset = "0",
#' 	count = "exact")
#' @param filters 
#' @param where 
#' @param order 
#' @param select 
#' @param distinct 
#' @param limit 
#' @param offset 
#' @param count 
#' @return LogList object.
#' @name logs.index
NULL

#' logs.create is a method defined in Arvados class.
#' 
#' @usage arv$logs.create(log, ensure_unique_name = "false")
#' @param log Log object.
#' @param ensure_unique_name Adjust name to ensure uniqueness instead of returning an error on (owner_uuid, name) collision.
#' @return Log object.
#' @name logs.create
NULL

#' logs.update is a method defined in Arvados class.
#' 
#' @usage arv$logs.update(log, uuid)
#' @param log Log object.
#' @param uuid The UUID of the Log in question.
#' @return Log object.
#' @name logs.update
NULL

#' logs.delete is a method defined in Arvados class.
#' 
#' @usage arv$logs.delete(uuid)
#' @param uuid The UUID of the Log in question.
#' @return Log object.
#' @name logs.delete
NULL

#' logs.list is a method defined in Arvados class.
#' 
#' @usage arv$logs.list(filters = NULL, where = NULL,
#' 	order = NULL, select = NULL, distinct = NULL,
#' 	limit = "100", offset = "0", count = "exact")
#' @param filters 
#' @param where 
#' @param order 
#' @param select 
#' @param distinct 
#' @param limit 
#' @param offset 
#' @param count 
#' @return LogList object.
#' @name logs.list
NULL

#' logs.show is a method defined in Arvados class.
#' 
#' @usage arv$logs.show(uuid)
#' @param uuid 
#' @return Log object.
#' @name logs.show
NULL

#' logs.destroy is a method defined in Arvados class.
#' 
#' @usage arv$logs.destroy(uuid)
#' @param uuid 
#' @return Log object.
#' @name logs.destroy
NULL

#' traits.get is a method defined in Arvados class.
#' 
#' @usage arv$traits.get(uuid)
#' @param uuid The UUID of the Trait in question.
#' @return Trait object.
#' @name traits.get
NULL

#' traits.index is a method defined in Arvados class.
#' 
#' @usage arv$traits.index(filters = NULL,
#' 	where = NULL, order = NULL, select = NULL,
#' 	distinct = NULL, limit = "100", offset = "0",
#' 	count = "exact")
#' @param filters 
#' @param where 
#' @param order 
#' @param select 
#' @param distinct 
#' @param limit 
#' @param offset 
#' @param count 
#' @return TraitList object.
#' @name traits.index
NULL

#' traits.create is a method defined in Arvados class.
#' 
#' @usage arv$traits.create(trait, ensure_unique_name = "false")
#' @param trait Trait object.
#' @param ensure_unique_name Adjust name to ensure uniqueness instead of returning an error on (owner_uuid, name) collision.
#' @return Trait object.
#' @name traits.create
NULL

#' traits.update is a method defined in Arvados class.
#' 
#' @usage arv$traits.update(trait, uuid)
#' @param trait Trait object.
#' @param uuid The UUID of the Trait in question.
#' @return Trait object.
#' @name traits.update
NULL

#' traits.delete is a method defined in Arvados class.
#' 
#' @usage arv$traits.delete(uuid)
#' @param uuid The UUID of the Trait in question.
#' @return Trait object.
#' @name traits.delete
NULL

#' traits.list is a method defined in Arvados class.
#' 
#' @usage arv$traits.list(filters = NULL,
#' 	where = NULL, order = NULL, select = NULL,
#' 	distinct = NULL, limit = "100", offset = "0",
#' 	count = "exact")
#' @param filters 
#' @param where 
#' @param order 
#' @param select 
#' @param distinct 
#' @param limit 
#' @param offset 
#' @param count 
#' @return TraitList object.
#' @name traits.list
NULL

#' traits.show is a method defined in Arvados class.
#' 
#' @usage arv$traits.show(uuid)
#' @param uuid 
#' @return Trait object.
#' @name traits.show
NULL

#' traits.destroy is a method defined in Arvados class.
#' 
#' @usage arv$traits.destroy(uuid)
#' @param uuid 
#' @return Trait object.
#' @name traits.destroy
NULL

#' virtual_machines.get is a method defined in Arvados class.
#' 
#' @usage arv$virtual_machines.get(uuid)
#' @param uuid The UUID of the VirtualMachine in question.
#' @return VirtualMachine object.
#' @name virtual_machines.get
NULL

#' virtual_machines.index is a method defined in Arvados class.
#' 
#' @usage arv$virtual_machines.index(filters = NULL,
#' 	where = NULL, order = NULL, select = NULL,
#' 	distinct = NULL, limit = "100", offset = "0",
#' 	count = "exact")
#' @param filters 
#' @param where 
#' @param order 
#' @param select 
#' @param distinct 
#' @param limit 
#' @param offset 
#' @param count 
#' @return VirtualMachineList object.
#' @name virtual_machines.index
NULL

#' virtual_machines.create is a method defined in Arvados class.
#' 
#' @usage arv$virtual_machines.create(virtual_machine,
#' 	ensure_unique_name = "false")
#' @param virtualMachine VirtualMachine object.
#' @param ensure_unique_name Adjust name to ensure uniqueness instead of returning an error on (owner_uuid, name) collision.
#' @return VirtualMachine object.
#' @name virtual_machines.create
NULL

#' virtual_machines.update is a method defined in Arvados class.
#' 
#' @usage arv$virtual_machines.update(virtual_machine,
#' 	uuid)
#' @param virtualMachine VirtualMachine object.
#' @param uuid The UUID of the VirtualMachine in question.
#' @return VirtualMachine object.
#' @name virtual_machines.update
NULL

#' virtual_machines.delete is a method defined in Arvados class.
#' 
#' @usage arv$virtual_machines.delete(uuid)
#' @param uuid The UUID of the VirtualMachine in question.
#' @return VirtualMachine object.
#' @name virtual_machines.delete
NULL

#' virtual_machines.logins is a method defined in Arvados class.
#' 
#' @usage arv$virtual_machines.logins(uuid)
#' @param uuid 
#' @return VirtualMachine object.
#' @name virtual_machines.logins
NULL

#' virtual_machines.get_all_logins is a method defined in Arvados class.
#' 
#' @usage arv$virtual_machines.get_all_logins(NULL)
#' @return VirtualMachine object.
#' @name virtual_machines.get_all_logins
NULL

#' virtual_machines.list is a method defined in Arvados class.
#' 
#' @usage arv$virtual_machines.list(filters = NULL,
#' 	where = NULL, order = NULL, select = NULL,
#' 	distinct = NULL, limit = "100", offset = "0",
#' 	count = "exact")
#' @param filters 
#' @param where 
#' @param order 
#' @param select 
#' @param distinct 
#' @param limit 
#' @param offset 
#' @param count 
#' @return VirtualMachineList object.
#' @name virtual_machines.list
NULL

#' virtual_machines.show is a method defined in Arvados class.
#' 
#' @usage arv$virtual_machines.show(uuid)
#' @param uuid 
#' @return VirtualMachine object.
#' @name virtual_machines.show
NULL

#' virtual_machines.destroy is a method defined in Arvados class.
#' 
#' @usage arv$virtual_machines.destroy(uuid)
#' @param uuid 
#' @return VirtualMachine object.
#' @name virtual_machines.destroy
NULL

#' workflows.get is a method defined in Arvados class.
#' 
#' @usage arv$workflows.get(uuid)
#' @param uuid The UUID of the Workflow in question.
#' @return Workflow object.
#' @name workflows.get
NULL

#' workflows.index is a method defined in Arvados class.
#' 
#' @usage arv$workflows.index(filters = NULL,
#' 	where = NULL, order = NULL, select = NULL,
#' 	distinct = NULL, limit = "100", offset = "0",
#' 	count = "exact")
#' @param filters 
#' @param where 
#' @param order 
#' @param select 
#' @param distinct 
#' @param limit 
#' @param offset 
#' @param count 
#' @return WorkflowList object.
#' @name workflows.index
NULL

#' workflows.create is a method defined in Arvados class.
#' 
#' @usage arv$workflows.create(workflow,
#' 	ensure_unique_name = "false")
#' @param workflow Workflow object.
#' @param ensure_unique_name Adjust name to ensure uniqueness instead of returning an error on (owner_uuid, name) collision.
#' @return Workflow object.
#' @name workflows.create
NULL

#' workflows.update is a method defined in Arvados class.
#' 
#' @usage arv$workflows.update(workflow,
#' 	uuid)
#' @param workflow Workflow object.
#' @param uuid The UUID of the Workflow in question.
#' @return Workflow object.
#' @name workflows.update
NULL

#' workflows.delete is a method defined in Arvados class.
#' 
#' @usage arv$workflows.delete(uuid)
#' @param uuid The UUID of the Workflow in question.
#' @return Workflow object.
#' @name workflows.delete
NULL

#' workflows.list is a method defined in Arvados class.
#' 
#' @usage arv$workflows.list(filters = NULL,
#' 	where = NULL, order = NULL, select = NULL,
#' 	distinct = NULL, limit = "100", offset = "0",
#' 	count = "exact")
#' @param filters 
#' @param where 
#' @param order 
#' @param select 
#' @param distinct 
#' @param limit 
#' @param offset 
#' @param count 
#' @return WorkflowList object.
#' @name workflows.list
NULL

#' workflows.show is a method defined in Arvados class.
#' 
#' @usage arv$workflows.show(uuid)
#' @param uuid 
#' @return Workflow object.
#' @name workflows.show
NULL

#' workflows.destroy is a method defined in Arvados class.
#' 
#' @usage arv$workflows.destroy(uuid)
#' @param uuid 
#' @return Workflow object.
#' @name workflows.destroy
NULL

#' groups.get is a method defined in Arvados class.
#' 
#' @usage arv$groups.get(uuid)
#' @param uuid The UUID of the Group in question.
#' @return Group object.
#' @name groups.get
NULL

#' groups.index is a method defined in Arvados class.
#' 
#' @usage arv$groups.index(filters = NULL,
#' 	where = NULL, order = NULL, select = NULL,
#' 	distinct = NULL, limit = "100", offset = "0",
#' 	count = "exact", include_trash = NULL)
#' @param filters 
#' @param where 
#' @param order 
#' @param select 
#' @param distinct 
#' @param limit 
#' @param offset 
#' @param count 
#' @param include_trash Include items whose is_trashed attribute is true.
#' @return GroupList object.
#' @name groups.index
NULL

#' groups.create is a method defined in Arvados class.
#' 
#' @usage arv$groups.create(group, ensure_unique_name = "false")
#' @param group Group object.
#' @param ensure_unique_name Adjust name to ensure uniqueness instead of returning an error on (owner_uuid, name) collision.
#' @return Group object.
#' @name groups.create
NULL

#' groups.update is a method defined in Arvados class.
#' 
#' @usage arv$groups.update(group, uuid)
#' @param group Group object.
#' @param uuid The UUID of the Group in question.
#' @return Group object.
#' @name groups.update
NULL

#' groups.delete is a method defined in Arvados class.
#' 
#' @usage arv$groups.delete(uuid)
#' @param uuid The UUID of the Group in question.
#' @return Group object.
#' @name groups.delete
NULL

#' groups.contents is a method defined in Arvados class.
#' 
#' @usage arv$groups.contents(filters = NULL,
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
#' @name groups.contents
NULL

#' groups.trash is a method defined in Arvados class.
#' 
#' @usage arv$groups.trash(uuid)
#' @param uuid 
#' @return Group object.
#' @name groups.trash
NULL

#' groups.untrash is a method defined in Arvados class.
#' 
#' @usage arv$groups.untrash(uuid)
#' @param uuid 
#' @return Group object.
#' @name groups.untrash
NULL

#' groups.list is a method defined in Arvados class.
#' 
#' @usage arv$groups.list(filters = NULL,
#' 	where = NULL, order = NULL, select = NULL,
#' 	distinct = NULL, limit = "100", offset = "0",
#' 	count = "exact", include_trash = NULL)
#' @param filters 
#' @param where 
#' @param order 
#' @param select 
#' @param distinct 
#' @param limit 
#' @param offset 
#' @param count 
#' @param include_trash Include items whose is_trashed attribute is true.
#' @return GroupList object.
#' @name groups.list
NULL

#' groups.show is a method defined in Arvados class.
#' 
#' @usage arv$groups.show(uuid)
#' @param uuid 
#' @return Group object.
#' @name groups.show
NULL

#' groups.destroy is a method defined in Arvados class.
#' 
#' @usage arv$groups.destroy(uuid)
#' @param uuid 
#' @return Group object.
#' @name groups.destroy
NULL

#' user_agreements.get is a method defined in Arvados class.
#' 
#' @usage arv$user_agreements.get(uuid)
#' @param uuid The UUID of the UserAgreement in question.
#' @return UserAgreement object.
#' @name user_agreements.get
NULL

#' user_agreements.index is a method defined in Arvados class.
#' 
#' @usage arv$user_agreements.index(filters = NULL,
#' 	where = NULL, order = NULL, select = NULL,
#' 	distinct = NULL, limit = "100", offset = "0",
#' 	count = "exact")
#' @param filters 
#' @param where 
#' @param order 
#' @param select 
#' @param distinct 
#' @param limit 
#' @param offset 
#' @param count 
#' @return UserAgreementList object.
#' @name user_agreements.index
NULL

#' user_agreements.create is a method defined in Arvados class.
#' 
#' @usage arv$user_agreements.create(user_agreement,
#' 	ensure_unique_name = "false")
#' @param userAgreement UserAgreement object.
#' @param ensure_unique_name Adjust name to ensure uniqueness instead of returning an error on (owner_uuid, name) collision.
#' @return UserAgreement object.
#' @name user_agreements.create
NULL

#' user_agreements.update is a method defined in Arvados class.
#' 
#' @usage arv$user_agreements.update(user_agreement,
#' 	uuid)
#' @param userAgreement UserAgreement object.
#' @param uuid The UUID of the UserAgreement in question.
#' @return UserAgreement object.
#' @name user_agreements.update
NULL

#' user_agreements.delete is a method defined in Arvados class.
#' 
#' @usage arv$user_agreements.delete(uuid)
#' @param uuid The UUID of the UserAgreement in question.
#' @return UserAgreement object.
#' @name user_agreements.delete
NULL

#' user_agreements.signatures is a method defined in Arvados class.
#' 
#' @usage arv$user_agreements.signatures(NULL)
#' @return UserAgreement object.
#' @name user_agreements.signatures
NULL

#' user_agreements.sign is a method defined in Arvados class.
#' 
#' @usage arv$user_agreements.sign(NULL)
#' @return UserAgreement object.
#' @name user_agreements.sign
NULL

#' user_agreements.list is a method defined in Arvados class.
#' 
#' @usage arv$user_agreements.list(filters = NULL,
#' 	where = NULL, order = NULL, select = NULL,
#' 	distinct = NULL, limit = "100", offset = "0",
#' 	count = "exact")
#' @param filters 
#' @param where 
#' @param order 
#' @param select 
#' @param distinct 
#' @param limit 
#' @param offset 
#' @param count 
#' @return UserAgreementList object.
#' @name user_agreements.list
NULL

#' user_agreements.new is a method defined in Arvados class.
#' 
#' @usage arv$user_agreements.new(NULL)
#' @return UserAgreement object.
#' @name user_agreements.new
NULL

#' user_agreements.show is a method defined in Arvados class.
#' 
#' @usage arv$user_agreements.show(uuid)
#' @param uuid 
#' @return UserAgreement object.
#' @name user_agreements.show
NULL

#' user_agreements.destroy is a method defined in Arvados class.
#' 
#' @usage arv$user_agreements.destroy(uuid)
#' @param uuid 
#' @return UserAgreement object.
#' @name user_agreements.destroy
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

		users.get = function(uuid)
		{
			endPoint <- stringr::str_interp("users/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- NULL
			
			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- User$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				created_at = resource$created_at, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, email = resource$email,
				first_name = resource$first_name, last_name = resource$last_name,
				identity_url = resource$identity_url, is_admin = resource$is_admin,
				prefs = resource$prefs, updated_at = resource$updated_at,
				default_owner_uuid = resource$default_owner_uuid,
				is_active = resource$is_active, username = resource$username)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		users.index = function(filters = NULL, where = NULL,
			order = NULL, select = NULL, distinct = NULL,
			limit = "100", offset = "0", count = "exact")
		{
			endPoint <- stringr::str_interp("users")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(filters = filters, where = where,
				order = order, select = select, distinct = distinct,
				limit = limit, offset = offset, count = count)
			body <- NULL
			
			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- UserList$new(kind = resource$kind,
				etag = resource$etag, items = resource$items,
				next_link = resource$next_link, next_page_token = resource$next_page_token,
				selfLink = resource$selfLink)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		users.create = function(user, ensure_unique_name = "false")
		{
			endPoint <- stringr::str_interp("users")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(ensure_unique_name = ensure_unique_name)
			body <- user$toJSON()
			
			response <- private$REST$http$exec("POST", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- User$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				created_at = resource$created_at, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, email = resource$email,
				first_name = resource$first_name, last_name = resource$last_name,
				identity_url = resource$identity_url, is_admin = resource$is_admin,
				prefs = resource$prefs, updated_at = resource$updated_at,
				default_owner_uuid = resource$default_owner_uuid,
				is_active = resource$is_active, username = resource$username)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		users.update = function(user, uuid)
		{
			endPoint <- stringr::str_interp("users/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- user$toJSON()
			
			response <- private$REST$http$exec("PUT", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- User$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				created_at = resource$created_at, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, email = resource$email,
				first_name = resource$first_name, last_name = resource$last_name,
				identity_url = resource$identity_url, is_admin = resource$is_admin,
				prefs = resource$prefs, updated_at = resource$updated_at,
				default_owner_uuid = resource$default_owner_uuid,
				is_active = resource$is_active, username = resource$username)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		users.delete = function(uuid)
		{
			endPoint <- stringr::str_interp("users/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- NULL
			
			response <- private$REST$http$exec("DELETE", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- User$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				created_at = resource$created_at, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, email = resource$email,
				first_name = resource$first_name, last_name = resource$last_name,
				identity_url = resource$identity_url, is_admin = resource$is_admin,
				prefs = resource$prefs, updated_at = resource$updated_at,
				default_owner_uuid = resource$default_owner_uuid,
				is_active = resource$is_active, username = resource$username)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		users.current = function()
		{
			endPoint <- stringr::str_interp("users/current")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- NULL
			body <- NULL
			
			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- User$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				created_at = resource$created_at, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, email = resource$email,
				first_name = resource$first_name, last_name = resource$last_name,
				identity_url = resource$identity_url, is_admin = resource$is_admin,
				prefs = resource$prefs, updated_at = resource$updated_at,
				default_owner_uuid = resource$default_owner_uuid,
				is_active = resource$is_active, username = resource$username)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		users.system = function()
		{
			endPoint <- stringr::str_interp("users/system")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- NULL
			body <- NULL
			
			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- User$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				created_at = resource$created_at, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, email = resource$email,
				first_name = resource$first_name, last_name = resource$last_name,
				identity_url = resource$identity_url, is_admin = resource$is_admin,
				prefs = resource$prefs, updated_at = resource$updated_at,
				default_owner_uuid = resource$default_owner_uuid,
				is_active = resource$is_active, username = resource$username)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		users.activate = function(uuid)
		{
			endPoint <- stringr::str_interp("users/${uuid}/activate")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- NULL
			
			response <- private$REST$http$exec("POST", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- User$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				created_at = resource$created_at, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, email = resource$email,
				first_name = resource$first_name, last_name = resource$last_name,
				identity_url = resource$identity_url, is_admin = resource$is_admin,
				prefs = resource$prefs, updated_at = resource$updated_at,
				default_owner_uuid = resource$default_owner_uuid,
				is_active = resource$is_active, username = resource$username)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		users.setup = function(user = NULL, openid_prefix = NULL,
			repo_name = NULL, vm_uuid = NULL, send_notification_email = "false")
		{
			endPoint <- stringr::str_interp("users/setup")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(user = user, openid_prefix = openid_prefix,
				repo_name = repo_name, vm_uuid = vm_uuid,
				send_notification_email = send_notification_email)
			body <- NULL
			
			response <- private$REST$http$exec("POST", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- User$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				created_at = resource$created_at, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, email = resource$email,
				first_name = resource$first_name, last_name = resource$last_name,
				identity_url = resource$identity_url, is_admin = resource$is_admin,
				prefs = resource$prefs, updated_at = resource$updated_at,
				default_owner_uuid = resource$default_owner_uuid,
				is_active = resource$is_active, username = resource$username)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		users.unsetup = function(uuid)
		{
			endPoint <- stringr::str_interp("users/${uuid}/unsetup")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- NULL
			
			response <- private$REST$http$exec("POST", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- User$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				created_at = resource$created_at, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, email = resource$email,
				first_name = resource$first_name, last_name = resource$last_name,
				identity_url = resource$identity_url, is_admin = resource$is_admin,
				prefs = resource$prefs, updated_at = resource$updated_at,
				default_owner_uuid = resource$default_owner_uuid,
				is_active = resource$is_active, username = resource$username)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		users.update_uuid = function(uuid, new_uuid)
		{
			endPoint <- stringr::str_interp("users/${uuid}/update_uuid")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid, new_uuid = new_uuid)
			body <- NULL
			
			response <- private$REST$http$exec("POST", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- User$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				created_at = resource$created_at, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, email = resource$email,
				first_name = resource$first_name, last_name = resource$last_name,
				identity_url = resource$identity_url, is_admin = resource$is_admin,
				prefs = resource$prefs, updated_at = resource$updated_at,
				default_owner_uuid = resource$default_owner_uuid,
				is_active = resource$is_active, username = resource$username)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		users.list = function(filters = NULL, where = NULL,
			order = NULL, select = NULL, distinct = NULL,
			limit = "100", offset = "0", count = "exact")
		{
			endPoint <- stringr::str_interp("users")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(filters = filters, where = where,
				order = order, select = select, distinct = distinct,
				limit = limit, offset = offset, count = count)
			body <- NULL
			
			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- UserList$new(kind = resource$kind,
				etag = resource$etag, items = resource$items,
				next_link = resource$next_link, next_page_token = resource$next_page_token,
				selfLink = resource$selfLink)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		users.show = function(uuid)
		{
			endPoint <- stringr::str_interp("users/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- NULL
			
			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- User$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				created_at = resource$created_at, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, email = resource$email,
				first_name = resource$first_name, last_name = resource$last_name,
				identity_url = resource$identity_url, is_admin = resource$is_admin,
				prefs = resource$prefs, updated_at = resource$updated_at,
				default_owner_uuid = resource$default_owner_uuid,
				is_active = resource$is_active, username = resource$username)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		users.destroy = function(uuid)
		{
			endPoint <- stringr::str_interp("users/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- NULL
			
			response <- private$REST$http$exec("DELETE", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- User$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				created_at = resource$created_at, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, email = resource$email,
				first_name = resource$first_name, last_name = resource$last_name,
				identity_url = resource$identity_url, is_admin = resource$is_admin,
				prefs = resource$prefs, updated_at = resource$updated_at,
				default_owner_uuid = resource$default_owner_uuid,
				is_active = resource$is_active, username = resource$username)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		api_client_authorizations.get = function(uuid)
		{
			endPoint <- stringr::str_interp("api_client_authorizations/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- NULL
			
			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- ApiClientAuthorization$new(uuid = resource$uuid,
				etag = resource$etag, api_token = resource$api_token,
				api_client_id = resource$api_client_id, user_id = resource$user_id,
				created_by_ip_address = resource$created_by_ip_address,
				last_used_by_ip_address = resource$last_used_by_ip_address,
				last_used_at = resource$last_used_at, expires_at = resource$expires_at,
				created_at = resource$created_at, updated_at = resource$updated_at,
				default_owner_uuid = resource$default_owner_uuid,
				scopes = resource$scopes)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		api_client_authorizations.index = function(filters = NULL,
			where = NULL, order = NULL, select = NULL,
			distinct = NULL, limit = "100", offset = "0",
			count = "exact")
		{
			endPoint <- stringr::str_interp("api_client_authorizations")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(filters = filters, where = where,
				order = order, select = select, distinct = distinct,
				limit = limit, offset = offset, count = count)
			body <- NULL
			
			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- ApiClientAuthorizationList$new(kind = resource$kind,
				etag = resource$etag, items = resource$items,
				next_link = resource$next_link, next_page_token = resource$next_page_token,
				selfLink = resource$selfLink)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		api_client_authorizations.create = function(api_client_authorization,
			ensure_unique_name = "false")
		{
			endPoint <- stringr::str_interp("api_client_authorizations")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(ensure_unique_name = ensure_unique_name)
			body <- api_client_authorization$toJSON()
			
			response <- private$REST$http$exec("POST", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- ApiClientAuthorization$new(uuid = resource$uuid,
				etag = resource$etag, api_token = resource$api_token,
				api_client_id = resource$api_client_id, user_id = resource$user_id,
				created_by_ip_address = resource$created_by_ip_address,
				last_used_by_ip_address = resource$last_used_by_ip_address,
				last_used_at = resource$last_used_at, expires_at = resource$expires_at,
				created_at = resource$created_at, updated_at = resource$updated_at,
				default_owner_uuid = resource$default_owner_uuid,
				scopes = resource$scopes)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		api_client_authorizations.update = function(api_client_authorization, uuid)
		{
			endPoint <- stringr::str_interp("api_client_authorizations/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- api_client_authorization$toJSON()
			
			response <- private$REST$http$exec("PUT", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- ApiClientAuthorization$new(uuid = resource$uuid,
				etag = resource$etag, api_token = resource$api_token,
				api_client_id = resource$api_client_id, user_id = resource$user_id,
				created_by_ip_address = resource$created_by_ip_address,
				last_used_by_ip_address = resource$last_used_by_ip_address,
				last_used_at = resource$last_used_at, expires_at = resource$expires_at,
				created_at = resource$created_at, updated_at = resource$updated_at,
				default_owner_uuid = resource$default_owner_uuid,
				scopes = resource$scopes)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		api_client_authorizations.delete = function(uuid)
		{
			endPoint <- stringr::str_interp("api_client_authorizations/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- NULL
			
			response <- private$REST$http$exec("DELETE", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- ApiClientAuthorization$new(uuid = resource$uuid,
				etag = resource$etag, api_token = resource$api_token,
				api_client_id = resource$api_client_id, user_id = resource$user_id,
				created_by_ip_address = resource$created_by_ip_address,
				last_used_by_ip_address = resource$last_used_by_ip_address,
				last_used_at = resource$last_used_at, expires_at = resource$expires_at,
				created_at = resource$created_at, updated_at = resource$updated_at,
				default_owner_uuid = resource$default_owner_uuid,
				scopes = resource$scopes)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		api_client_authorizations.create_system_auth = function(api_client_id = NULL, scopes = NULL)
		{
			endPoint <- stringr::str_interp("api_client_authorizations/create_system_auth")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(api_client_id = api_client_id,
				scopes = scopes)
			body <- NULL
			
			response <- private$REST$http$exec("POST", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- ApiClientAuthorization$new(uuid = resource$uuid,
				etag = resource$etag, api_token = resource$api_token,
				api_client_id = resource$api_client_id, user_id = resource$user_id,
				created_by_ip_address = resource$created_by_ip_address,
				last_used_by_ip_address = resource$last_used_by_ip_address,
				last_used_at = resource$last_used_at, expires_at = resource$expires_at,
				created_at = resource$created_at, updated_at = resource$updated_at,
				default_owner_uuid = resource$default_owner_uuid,
				scopes = resource$scopes)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		api_client_authorizations.current = function()
		{
			endPoint <- stringr::str_interp("api_client_authorizations/current")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- NULL
			body <- NULL
			
			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- ApiClientAuthorization$new(uuid = resource$uuid,
				etag = resource$etag, api_token = resource$api_token,
				api_client_id = resource$api_client_id, user_id = resource$user_id,
				created_by_ip_address = resource$created_by_ip_address,
				last_used_by_ip_address = resource$last_used_by_ip_address,
				last_used_at = resource$last_used_at, expires_at = resource$expires_at,
				created_at = resource$created_at, updated_at = resource$updated_at,
				default_owner_uuid = resource$default_owner_uuid,
				scopes = resource$scopes)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		api_client_authorizations.list = function(filters = NULL,
			where = NULL, order = NULL, select = NULL,
			distinct = NULL, limit = "100", offset = "0",
			count = "exact")
		{
			endPoint <- stringr::str_interp("api_client_authorizations")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(filters = filters, where = where,
				order = order, select = select, distinct = distinct,
				limit = limit, offset = offset, count = count)
			body <- NULL
			
			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- ApiClientAuthorizationList$new(kind = resource$kind,
				etag = resource$etag, items = resource$items,
				next_link = resource$next_link, next_page_token = resource$next_page_token,
				selfLink = resource$selfLink)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		api_client_authorizations.show = function(uuid)
		{
			endPoint <- stringr::str_interp("api_client_authorizations/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- NULL
			
			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- ApiClientAuthorization$new(uuid = resource$uuid,
				etag = resource$etag, api_token = resource$api_token,
				api_client_id = resource$api_client_id, user_id = resource$user_id,
				created_by_ip_address = resource$created_by_ip_address,
				last_used_by_ip_address = resource$last_used_by_ip_address,
				last_used_at = resource$last_used_at, expires_at = resource$expires_at,
				created_at = resource$created_at, updated_at = resource$updated_at,
				default_owner_uuid = resource$default_owner_uuid,
				scopes = resource$scopes)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		api_client_authorizations.destroy = function(uuid)
		{
			endPoint <- stringr::str_interp("api_client_authorizations/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- NULL
			
			response <- private$REST$http$exec("DELETE", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- ApiClientAuthorization$new(uuid = resource$uuid,
				etag = resource$etag, api_token = resource$api_token,
				api_client_id = resource$api_client_id, user_id = resource$user_id,
				created_by_ip_address = resource$created_by_ip_address,
				last_used_by_ip_address = resource$last_used_by_ip_address,
				last_used_at = resource$last_used_at, expires_at = resource$expires_at,
				created_at = resource$created_at, updated_at = resource$updated_at,
				default_owner_uuid = resource$default_owner_uuid,
				scopes = resource$scopes)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		api_clients.get = function(uuid)
		{
			endPoint <- stringr::str_interp("api_clients/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- NULL
			
			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- ApiClient$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, name = resource$name,
				url_prefix = resource$url_prefix, created_at = resource$created_at,
				updated_at = resource$updated_at, is_trusted = resource$is_trusted)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		api_clients.index = function(filters = NULL,
			where = NULL, order = NULL, select = NULL,
			distinct = NULL, limit = "100", offset = "0",
			count = "exact")
		{
			endPoint <- stringr::str_interp("api_clients")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(filters = filters, where = where,
				order = order, select = select, distinct = distinct,
				limit = limit, offset = offset, count = count)
			body <- NULL
			
			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- ApiClientList$new(kind = resource$kind,
				etag = resource$etag, items = resource$items,
				next_link = resource$next_link, next_page_token = resource$next_page_token,
				selfLink = resource$selfLink)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		api_clients.create = function(api_client, ensure_unique_name = "false")
		{
			endPoint <- stringr::str_interp("api_clients")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(ensure_unique_name = ensure_unique_name)
			body <- api_client$toJSON()
			
			response <- private$REST$http$exec("POST", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- ApiClient$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, name = resource$name,
				url_prefix = resource$url_prefix, created_at = resource$created_at,
				updated_at = resource$updated_at, is_trusted = resource$is_trusted)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		api_clients.update = function(api_client, uuid)
		{
			endPoint <- stringr::str_interp("api_clients/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- api_client$toJSON()
			
			response <- private$REST$http$exec("PUT", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- ApiClient$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, name = resource$name,
				url_prefix = resource$url_prefix, created_at = resource$created_at,
				updated_at = resource$updated_at, is_trusted = resource$is_trusted)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		api_clients.delete = function(uuid)
		{
			endPoint <- stringr::str_interp("api_clients/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- NULL
			
			response <- private$REST$http$exec("DELETE", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- ApiClient$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, name = resource$name,
				url_prefix = resource$url_prefix, created_at = resource$created_at,
				updated_at = resource$updated_at, is_trusted = resource$is_trusted)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		api_clients.list = function(filters = NULL,
			where = NULL, order = NULL, select = NULL,
			distinct = NULL, limit = "100", offset = "0",
			count = "exact")
		{
			endPoint <- stringr::str_interp("api_clients")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(filters = filters, where = where,
				order = order, select = select, distinct = distinct,
				limit = limit, offset = offset, count = count)
			body <- NULL
			
			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- ApiClientList$new(kind = resource$kind,
				etag = resource$etag, items = resource$items,
				next_link = resource$next_link, next_page_token = resource$next_page_token,
				selfLink = resource$selfLink)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		api_clients.show = function(uuid)
		{
			endPoint <- stringr::str_interp("api_clients/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- NULL
			
			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- ApiClient$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, name = resource$name,
				url_prefix = resource$url_prefix, created_at = resource$created_at,
				updated_at = resource$updated_at, is_trusted = resource$is_trusted)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		api_clients.destroy = function(uuid)
		{
			endPoint <- stringr::str_interp("api_clients/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- NULL
			
			response <- private$REST$http$exec("DELETE", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- ApiClient$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, name = resource$name,
				url_prefix = resource$url_prefix, created_at = resource$created_at,
				updated_at = resource$updated_at, is_trusted = resource$is_trusted)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		container_requests.get = function(uuid)
		{
			endPoint <- stringr::str_interp("container_requests/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- NULL
			
			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- ContainerRequest$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				created_at = resource$created_at, modified_at = resource$modified_at,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				name = resource$name, description = resource$description,
				properties = resource$properties, state = resource$state,
				requesting_container_uuid = resource$requesting_container_uuid,
				container_uuid = resource$container_uuid,
				container_count_max = resource$container_count_max,
				mounts = resource$mounts, runtime_constraints = resource$runtime_constraints,
				container_image = resource$container_image,
				environment = resource$environment, cwd = resource$cwd,
				command = resource$command, output_path = resource$output_path,
				priority = resource$priority, expires_at = resource$expires_at,
				filters = resource$filters, updated_at = resource$updated_at,
				container_count = resource$container_count,
				use_existing = resource$use_existing, scheduling_parameters = resource$scheduling_parameters,
				output_uuid = resource$output_uuid, log_uuid = resource$log_uuid,
				output_name = resource$output_name, output_ttl = resource$output_ttl)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		container_requests.index = function(filters = NULL,
			where = NULL, order = NULL, select = NULL,
			distinct = NULL, limit = "100", offset = "0",
			count = "exact")
		{
			endPoint <- stringr::str_interp("container_requests")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(filters = filters, where = where,
				order = order, select = select, distinct = distinct,
				limit = limit, offset = offset, count = count)
			body <- NULL
			
			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- ContainerRequestList$new(kind = resource$kind,
				etag = resource$etag, items = resource$items,
				next_link = resource$next_link, next_page_token = resource$next_page_token,
				selfLink = resource$selfLink)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		container_requests.create = function(container_request,
			ensure_unique_name = "false")
		{
			endPoint <- stringr::str_interp("container_requests")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(ensure_unique_name = ensure_unique_name)
			body <- container_request$toJSON()
			
			response <- private$REST$http$exec("POST", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- ContainerRequest$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				created_at = resource$created_at, modified_at = resource$modified_at,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				name = resource$name, description = resource$description,
				properties = resource$properties, state = resource$state,
				requesting_container_uuid = resource$requesting_container_uuid,
				container_uuid = resource$container_uuid,
				container_count_max = resource$container_count_max,
				mounts = resource$mounts, runtime_constraints = resource$runtime_constraints,
				container_image = resource$container_image,
				environment = resource$environment, cwd = resource$cwd,
				command = resource$command, output_path = resource$output_path,
				priority = resource$priority, expires_at = resource$expires_at,
				filters = resource$filters, updated_at = resource$updated_at,
				container_count = resource$container_count,
				use_existing = resource$use_existing, scheduling_parameters = resource$scheduling_parameters,
				output_uuid = resource$output_uuid, log_uuid = resource$log_uuid,
				output_name = resource$output_name, output_ttl = resource$output_ttl)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		container_requests.update = function(container_request, uuid)
		{
			endPoint <- stringr::str_interp("container_requests/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- container_request$toJSON()
			
			response <- private$REST$http$exec("PUT", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- ContainerRequest$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				created_at = resource$created_at, modified_at = resource$modified_at,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				name = resource$name, description = resource$description,
				properties = resource$properties, state = resource$state,
				requesting_container_uuid = resource$requesting_container_uuid,
				container_uuid = resource$container_uuid,
				container_count_max = resource$container_count_max,
				mounts = resource$mounts, runtime_constraints = resource$runtime_constraints,
				container_image = resource$container_image,
				environment = resource$environment, cwd = resource$cwd,
				command = resource$command, output_path = resource$output_path,
				priority = resource$priority, expires_at = resource$expires_at,
				filters = resource$filters, updated_at = resource$updated_at,
				container_count = resource$container_count,
				use_existing = resource$use_existing, scheduling_parameters = resource$scheduling_parameters,
				output_uuid = resource$output_uuid, log_uuid = resource$log_uuid,
				output_name = resource$output_name, output_ttl = resource$output_ttl)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		container_requests.delete = function(uuid)
		{
			endPoint <- stringr::str_interp("container_requests/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- NULL
			
			response <- private$REST$http$exec("DELETE", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- ContainerRequest$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				created_at = resource$created_at, modified_at = resource$modified_at,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				name = resource$name, description = resource$description,
				properties = resource$properties, state = resource$state,
				requesting_container_uuid = resource$requesting_container_uuid,
				container_uuid = resource$container_uuid,
				container_count_max = resource$container_count_max,
				mounts = resource$mounts, runtime_constraints = resource$runtime_constraints,
				container_image = resource$container_image,
				environment = resource$environment, cwd = resource$cwd,
				command = resource$command, output_path = resource$output_path,
				priority = resource$priority, expires_at = resource$expires_at,
				filters = resource$filters, updated_at = resource$updated_at,
				container_count = resource$container_count,
				use_existing = resource$use_existing, scheduling_parameters = resource$scheduling_parameters,
				output_uuid = resource$output_uuid, log_uuid = resource$log_uuid,
				output_name = resource$output_name, output_ttl = resource$output_ttl)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		container_requests.list = function(filters = NULL,
			where = NULL, order = NULL, select = NULL,
			distinct = NULL, limit = "100", offset = "0",
			count = "exact")
		{
			endPoint <- stringr::str_interp("container_requests")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(filters = filters, where = where,
				order = order, select = select, distinct = distinct,
				limit = limit, offset = offset, count = count)
			body <- NULL
			
			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- ContainerRequestList$new(kind = resource$kind,
				etag = resource$etag, items = resource$items,
				next_link = resource$next_link, next_page_token = resource$next_page_token,
				selfLink = resource$selfLink)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		container_requests.show = function(uuid)
		{
			endPoint <- stringr::str_interp("container_requests/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- NULL
			
			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- ContainerRequest$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				created_at = resource$created_at, modified_at = resource$modified_at,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				name = resource$name, description = resource$description,
				properties = resource$properties, state = resource$state,
				requesting_container_uuid = resource$requesting_container_uuid,
				container_uuid = resource$container_uuid,
				container_count_max = resource$container_count_max,
				mounts = resource$mounts, runtime_constraints = resource$runtime_constraints,
				container_image = resource$container_image,
				environment = resource$environment, cwd = resource$cwd,
				command = resource$command, output_path = resource$output_path,
				priority = resource$priority, expires_at = resource$expires_at,
				filters = resource$filters, updated_at = resource$updated_at,
				container_count = resource$container_count,
				use_existing = resource$use_existing, scheduling_parameters = resource$scheduling_parameters,
				output_uuid = resource$output_uuid, log_uuid = resource$log_uuid,
				output_name = resource$output_name, output_ttl = resource$output_ttl)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		container_requests.destroy = function(uuid)
		{
			endPoint <- stringr::str_interp("container_requests/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- NULL
			
			response <- private$REST$http$exec("DELETE", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- ContainerRequest$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				created_at = resource$created_at, modified_at = resource$modified_at,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				name = resource$name, description = resource$description,
				properties = resource$properties, state = resource$state,
				requesting_container_uuid = resource$requesting_container_uuid,
				container_uuid = resource$container_uuid,
				container_count_max = resource$container_count_max,
				mounts = resource$mounts, runtime_constraints = resource$runtime_constraints,
				container_image = resource$container_image,
				environment = resource$environment, cwd = resource$cwd,
				command = resource$command, output_path = resource$output_path,
				priority = resource$priority, expires_at = resource$expires_at,
				filters = resource$filters, updated_at = resource$updated_at,
				container_count = resource$container_count,
				use_existing = resource$use_existing, scheduling_parameters = resource$scheduling_parameters,
				output_uuid = resource$output_uuid, log_uuid = resource$log_uuid,
				output_name = resource$output_name, output_ttl = resource$output_ttl)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		authorized_keys.get = function(uuid)
		{
			endPoint <- stringr::str_interp("authorized_keys/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- NULL
			
			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- AuthorizedKey$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, name = resource$name,
				key_type = resource$key_type, authorized_user_uuid = resource$authorized_user_uuid,
				public_key = resource$public_key, expires_at = resource$expires_at,
				created_at = resource$created_at, updated_at = resource$updated_at)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		authorized_keys.index = function(filters = NULL,
			where = NULL, order = NULL, select = NULL,
			distinct = NULL, limit = "100", offset = "0",
			count = "exact")
		{
			endPoint <- stringr::str_interp("authorized_keys")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(filters = filters, where = where,
				order = order, select = select, distinct = distinct,
				limit = limit, offset = offset, count = count)
			body <- NULL
			
			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- AuthorizedKeyList$new(kind = resource$kind,
				etag = resource$etag, items = resource$items,
				next_link = resource$next_link, next_page_token = resource$next_page_token,
				selfLink = resource$selfLink)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		authorized_keys.create = function(authorized_key,
			ensure_unique_name = "false")
		{
			endPoint <- stringr::str_interp("authorized_keys")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(ensure_unique_name = ensure_unique_name)
			body <- authorized_key$toJSON()
			
			response <- private$REST$http$exec("POST", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- AuthorizedKey$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, name = resource$name,
				key_type = resource$key_type, authorized_user_uuid = resource$authorized_user_uuid,
				public_key = resource$public_key, expires_at = resource$expires_at,
				created_at = resource$created_at, updated_at = resource$updated_at)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		authorized_keys.update = function(authorized_key, uuid)
		{
			endPoint <- stringr::str_interp("authorized_keys/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- authorized_key$toJSON()
			
			response <- private$REST$http$exec("PUT", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- AuthorizedKey$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, name = resource$name,
				key_type = resource$key_type, authorized_user_uuid = resource$authorized_user_uuid,
				public_key = resource$public_key, expires_at = resource$expires_at,
				created_at = resource$created_at, updated_at = resource$updated_at)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		authorized_keys.delete = function(uuid)
		{
			endPoint <- stringr::str_interp("authorized_keys/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- NULL
			
			response <- private$REST$http$exec("DELETE", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- AuthorizedKey$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, name = resource$name,
				key_type = resource$key_type, authorized_user_uuid = resource$authorized_user_uuid,
				public_key = resource$public_key, expires_at = resource$expires_at,
				created_at = resource$created_at, updated_at = resource$updated_at)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		authorized_keys.list = function(filters = NULL,
			where = NULL, order = NULL, select = NULL,
			distinct = NULL, limit = "100", offset = "0",
			count = "exact")
		{
			endPoint <- stringr::str_interp("authorized_keys")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(filters = filters, where = where,
				order = order, select = select, distinct = distinct,
				limit = limit, offset = offset, count = count)
			body <- NULL
			
			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- AuthorizedKeyList$new(kind = resource$kind,
				etag = resource$etag, items = resource$items,
				next_link = resource$next_link, next_page_token = resource$next_page_token,
				selfLink = resource$selfLink)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		authorized_keys.show = function(uuid)
		{
			endPoint <- stringr::str_interp("authorized_keys/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- NULL
			
			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- AuthorizedKey$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, name = resource$name,
				key_type = resource$key_type, authorized_user_uuid = resource$authorized_user_uuid,
				public_key = resource$public_key, expires_at = resource$expires_at,
				created_at = resource$created_at, updated_at = resource$updated_at)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		authorized_keys.destroy = function(uuid)
		{
			endPoint <- stringr::str_interp("authorized_keys/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- NULL
			
			response <- private$REST$http$exec("DELETE", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- AuthorizedKey$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, name = resource$name,
				key_type = resource$key_type, authorized_user_uuid = resource$authorized_user_uuid,
				public_key = resource$public_key, expires_at = resource$expires_at,
				created_at = resource$created_at, updated_at = resource$updated_at)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		collections.get = function(uuid)
		{
			endPoint <- stringr::str_interp("collections/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- NULL
			
			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- Collection$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				created_at = resource$created_at, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, portable_data_hash = resource$portable_data_hash,
				replication_desired = resource$replication_desired,
				replication_confirmed_at = resource$replication_confirmed_at,
				replication_confirmed = resource$replication_confirmed,
				updated_at = resource$updated_at, manifest_text = resource$manifest_text,
				name = resource$name, description = resource$description,
				properties = resource$properties, delete_at = resource$delete_at,
				file_names = resource$file_names, trash_at = resource$trash_at,
				is_trashed = resource$is_trashed, storage_classes_desired = resource$storage_classes_desired,
				storage_classes_confirmed = resource$storage_classes_confirmed,
				storage_classes_confirmed_at = resource$storage_classes_confirmed_at)
			
			result$setRESTService(private$REST)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		collections.index = function(filters = NULL,
			where = NULL, order = NULL, select = NULL,
			distinct = NULL, limit = "100", offset = "0",
			count = "exact", include_trash = NULL)
		{
			endPoint <- stringr::str_interp("collections")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(filters = filters, where = where,
				order = order, select = select, distinct = distinct,
				limit = limit, offset = offset, count = count,
				include_trash = include_trash)
			body <- NULL
			
			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- CollectionList$new(kind = resource$kind,
				etag = resource$etag, items = resource$items,
				next_link = resource$next_link, next_page_token = resource$next_page_token,
				selfLink = resource$selfLink)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		collections.create = function(collection, ensure_unique_name = "false")
		{
			endPoint <- stringr::str_interp("collections")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(ensure_unique_name = ensure_unique_name)
			body <- collection$toJSON()
			
			response <- private$REST$http$exec("POST", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- Collection$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				created_at = resource$created_at, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, portable_data_hash = resource$portable_data_hash,
				replication_desired = resource$replication_desired,
				replication_confirmed_at = resource$replication_confirmed_at,
				replication_confirmed = resource$replication_confirmed,
				updated_at = resource$updated_at, manifest_text = resource$manifest_text,
				name = resource$name, description = resource$description,
				properties = resource$properties, delete_at = resource$delete_at,
				file_names = resource$file_names, trash_at = resource$trash_at,
				is_trashed = resource$is_trashed, storage_classes_desired = resource$storage_classes_desired,
				storage_classes_confirmed = resource$storage_classes_confirmed,
				storage_classes_confirmed_at = resource$storage_classes_confirmed_at)
			
			result$setRESTService(private$REST)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		collections.update = function(collection, uuid)
		{
			endPoint <- stringr::str_interp("collections/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- collection$toJSON()
			
			response <- private$REST$http$exec("PUT", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- Collection$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				created_at = resource$created_at, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, portable_data_hash = resource$portable_data_hash,
				replication_desired = resource$replication_desired,
				replication_confirmed_at = resource$replication_confirmed_at,
				replication_confirmed = resource$replication_confirmed,
				updated_at = resource$updated_at, manifest_text = resource$manifest_text,
				name = resource$name, description = resource$description,
				properties = resource$properties, delete_at = resource$delete_at,
				file_names = resource$file_names, trash_at = resource$trash_at,
				is_trashed = resource$is_trashed, storage_classes_desired = resource$storage_classes_desired,
				storage_classes_confirmed = resource$storage_classes_confirmed,
				storage_classes_confirmed_at = resource$storage_classes_confirmed_at)
			
			result$setRESTService(private$REST)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		collections.delete = function(uuid)
		{
			endPoint <- stringr::str_interp("collections/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- NULL
			
			response <- private$REST$http$exec("DELETE", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- Collection$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				created_at = resource$created_at, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, portable_data_hash = resource$portable_data_hash,
				replication_desired = resource$replication_desired,
				replication_confirmed_at = resource$replication_confirmed_at,
				replication_confirmed = resource$replication_confirmed,
				updated_at = resource$updated_at, manifest_text = resource$manifest_text,
				name = resource$name, description = resource$description,
				properties = resource$properties, delete_at = resource$delete_at,
				file_names = resource$file_names, trash_at = resource$trash_at,
				is_trashed = resource$is_trashed, storage_classes_desired = resource$storage_classes_desired,
				storage_classes_confirmed = resource$storage_classes_confirmed,
				storage_classes_confirmed_at = resource$storage_classes_confirmed_at)
			
			result$setRESTService(private$REST)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		collections.provenance = function(uuid)
		{
			endPoint <- stringr::str_interp("collections/${uuid}/provenance")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- NULL
			
			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- Collection$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				created_at = resource$created_at, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, portable_data_hash = resource$portable_data_hash,
				replication_desired = resource$replication_desired,
				replication_confirmed_at = resource$replication_confirmed_at,
				replication_confirmed = resource$replication_confirmed,
				updated_at = resource$updated_at, manifest_text = resource$manifest_text,
				name = resource$name, description = resource$description,
				properties = resource$properties, delete_at = resource$delete_at,
				file_names = resource$file_names, trash_at = resource$trash_at,
				is_trashed = resource$is_trashed, storage_classes_desired = resource$storage_classes_desired,
				storage_classes_confirmed = resource$storage_classes_confirmed,
				storage_classes_confirmed_at = resource$storage_classes_confirmed_at)
			
			result$setRESTService(private$REST)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		collections.used_by = function(uuid)
		{
			endPoint <- stringr::str_interp("collections/${uuid}/used_by")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- NULL
			
			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- Collection$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				created_at = resource$created_at, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, portable_data_hash = resource$portable_data_hash,
				replication_desired = resource$replication_desired,
				replication_confirmed_at = resource$replication_confirmed_at,
				replication_confirmed = resource$replication_confirmed,
				updated_at = resource$updated_at, manifest_text = resource$manifest_text,
				name = resource$name, description = resource$description,
				properties = resource$properties, delete_at = resource$delete_at,
				file_names = resource$file_names, trash_at = resource$trash_at,
				is_trashed = resource$is_trashed, storage_classes_desired = resource$storage_classes_desired,
				storage_classes_confirmed = resource$storage_classes_confirmed,
				storage_classes_confirmed_at = resource$storage_classes_confirmed_at)
			
			result$setRESTService(private$REST)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		collections.trash = function(uuid)
		{
			endPoint <- stringr::str_interp("collections/${uuid}/trash")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- NULL
			
			response <- private$REST$http$exec("POST", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- Collection$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				created_at = resource$created_at, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, portable_data_hash = resource$portable_data_hash,
				replication_desired = resource$replication_desired,
				replication_confirmed_at = resource$replication_confirmed_at,
				replication_confirmed = resource$replication_confirmed,
				updated_at = resource$updated_at, manifest_text = resource$manifest_text,
				name = resource$name, description = resource$description,
				properties = resource$properties, delete_at = resource$delete_at,
				file_names = resource$file_names, trash_at = resource$trash_at,
				is_trashed = resource$is_trashed, storage_classes_desired = resource$storage_classes_desired,
				storage_classes_confirmed = resource$storage_classes_confirmed,
				storage_classes_confirmed_at = resource$storage_classes_confirmed_at)
			
			result$setRESTService(private$REST)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		collections.untrash = function(uuid)
		{
			endPoint <- stringr::str_interp("collections/${uuid}/untrash")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- NULL
			
			response <- private$REST$http$exec("POST", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- Collection$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				created_at = resource$created_at, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, portable_data_hash = resource$portable_data_hash,
				replication_desired = resource$replication_desired,
				replication_confirmed_at = resource$replication_confirmed_at,
				replication_confirmed = resource$replication_confirmed,
				updated_at = resource$updated_at, manifest_text = resource$manifest_text,
				name = resource$name, description = resource$description,
				properties = resource$properties, delete_at = resource$delete_at,
				file_names = resource$file_names, trash_at = resource$trash_at,
				is_trashed = resource$is_trashed, storage_classes_desired = resource$storage_classes_desired,
				storage_classes_confirmed = resource$storage_classes_confirmed,
				storage_classes_confirmed_at = resource$storage_classes_confirmed_at)
			
			result$setRESTService(private$REST)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		collections.list = function(filters = NULL,
			where = NULL, order = NULL, select = NULL,
			distinct = NULL, limit = "100", offset = "0",
			count = "exact", include_trash = NULL)
		{
			endPoint <- stringr::str_interp("collections")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(filters = filters, where = where,
				order = order, select = select, distinct = distinct,
				limit = limit, offset = offset, count = count,
				include_trash = include_trash)
			body <- NULL
			
			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- CollectionList$new(kind = resource$kind,
				etag = resource$etag, items = resource$items,
				next_link = resource$next_link, next_page_token = resource$next_page_token,
				selfLink = resource$selfLink)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		collections.show = function(uuid)
		{
			endPoint <- stringr::str_interp("collections/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- NULL
			
			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- Collection$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				created_at = resource$created_at, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, portable_data_hash = resource$portable_data_hash,
				replication_desired = resource$replication_desired,
				replication_confirmed_at = resource$replication_confirmed_at,
				replication_confirmed = resource$replication_confirmed,
				updated_at = resource$updated_at, manifest_text = resource$manifest_text,
				name = resource$name, description = resource$description,
				properties = resource$properties, delete_at = resource$delete_at,
				file_names = resource$file_names, trash_at = resource$trash_at,
				is_trashed = resource$is_trashed, storage_classes_desired = resource$storage_classes_desired,
				storage_classes_confirmed = resource$storage_classes_confirmed,
				storage_classes_confirmed_at = resource$storage_classes_confirmed_at)
			
			result$setRESTService(private$REST)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		collections.destroy = function(uuid)
		{
			endPoint <- stringr::str_interp("collections/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- NULL
			
			response <- private$REST$http$exec("DELETE", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- Collection$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				created_at = resource$created_at, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, portable_data_hash = resource$portable_data_hash,
				replication_desired = resource$replication_desired,
				replication_confirmed_at = resource$replication_confirmed_at,
				replication_confirmed = resource$replication_confirmed,
				updated_at = resource$updated_at, manifest_text = resource$manifest_text,
				name = resource$name, description = resource$description,
				properties = resource$properties, delete_at = resource$delete_at,
				file_names = resource$file_names, trash_at = resource$trash_at,
				is_trashed = resource$is_trashed, storage_classes_desired = resource$storage_classes_desired,
				storage_classes_confirmed = resource$storage_classes_confirmed,
				storage_classes_confirmed_at = resource$storage_classes_confirmed_at)
			
			result$setRESTService(private$REST)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		containers.get = function(uuid)
		{
			endPoint <- stringr::str_interp("containers/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- NULL
			
			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- Container$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				created_at = resource$created_at, modified_at = resource$modified_at,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				state = resource$state, started_at = resource$started_at,
				finished_at = resource$finished_at, log = resource$log,
				environment = resource$environment, cwd = resource$cwd,
				command = resource$command, output_path = resource$output_path,
				mounts = resource$mounts, runtime_constraints = resource$runtime_constraints,
				output = resource$output, container_image = resource$container_image,
				progress = resource$progress, priority = resource$priority,
				updated_at = resource$updated_at, exit_code = resource$exit_code,
				auth_uuid = resource$auth_uuid, locked_by_uuid = resource$locked_by_uuid,
				scheduling_parameters = resource$scheduling_parameters)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		containers.index = function(filters = NULL,
			where = NULL, order = NULL, select = NULL,
			distinct = NULL, limit = "100", offset = "0",
			count = "exact")
		{
			endPoint <- stringr::str_interp("containers")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(filters = filters, where = where,
				order = order, select = select, distinct = distinct,
				limit = limit, offset = offset, count = count)
			body <- NULL
			
			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- ContainerList$new(kind = resource$kind,
				etag = resource$etag, items = resource$items,
				next_link = resource$next_link, next_page_token = resource$next_page_token,
				selfLink = resource$selfLink)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		containers.create = function(container, ensure_unique_name = "false")
		{
			endPoint <- stringr::str_interp("containers")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(ensure_unique_name = ensure_unique_name)
			body <- container$toJSON()
			
			response <- private$REST$http$exec("POST", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- Container$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				created_at = resource$created_at, modified_at = resource$modified_at,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				state = resource$state, started_at = resource$started_at,
				finished_at = resource$finished_at, log = resource$log,
				environment = resource$environment, cwd = resource$cwd,
				command = resource$command, output_path = resource$output_path,
				mounts = resource$mounts, runtime_constraints = resource$runtime_constraints,
				output = resource$output, container_image = resource$container_image,
				progress = resource$progress, priority = resource$priority,
				updated_at = resource$updated_at, exit_code = resource$exit_code,
				auth_uuid = resource$auth_uuid, locked_by_uuid = resource$locked_by_uuid,
				scheduling_parameters = resource$scheduling_parameters)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		containers.update = function(container, uuid)
		{
			endPoint <- stringr::str_interp("containers/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- container$toJSON()
			
			response <- private$REST$http$exec("PUT", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- Container$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				created_at = resource$created_at, modified_at = resource$modified_at,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				state = resource$state, started_at = resource$started_at,
				finished_at = resource$finished_at, log = resource$log,
				environment = resource$environment, cwd = resource$cwd,
				command = resource$command, output_path = resource$output_path,
				mounts = resource$mounts, runtime_constraints = resource$runtime_constraints,
				output = resource$output, container_image = resource$container_image,
				progress = resource$progress, priority = resource$priority,
				updated_at = resource$updated_at, exit_code = resource$exit_code,
				auth_uuid = resource$auth_uuid, locked_by_uuid = resource$locked_by_uuid,
				scheduling_parameters = resource$scheduling_parameters)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		containers.delete = function(uuid)
		{
			endPoint <- stringr::str_interp("containers/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- NULL
			
			response <- private$REST$http$exec("DELETE", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- Container$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				created_at = resource$created_at, modified_at = resource$modified_at,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				state = resource$state, started_at = resource$started_at,
				finished_at = resource$finished_at, log = resource$log,
				environment = resource$environment, cwd = resource$cwd,
				command = resource$command, output_path = resource$output_path,
				mounts = resource$mounts, runtime_constraints = resource$runtime_constraints,
				output = resource$output, container_image = resource$container_image,
				progress = resource$progress, priority = resource$priority,
				updated_at = resource$updated_at, exit_code = resource$exit_code,
				auth_uuid = resource$auth_uuid, locked_by_uuid = resource$locked_by_uuid,
				scheduling_parameters = resource$scheduling_parameters)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		containers.auth = function(uuid)
		{
			endPoint <- stringr::str_interp("containers/${uuid}/auth")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- NULL
			
			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- Container$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				created_at = resource$created_at, modified_at = resource$modified_at,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				state = resource$state, started_at = resource$started_at,
				finished_at = resource$finished_at, log = resource$log,
				environment = resource$environment, cwd = resource$cwd,
				command = resource$command, output_path = resource$output_path,
				mounts = resource$mounts, runtime_constraints = resource$runtime_constraints,
				output = resource$output, container_image = resource$container_image,
				progress = resource$progress, priority = resource$priority,
				updated_at = resource$updated_at, exit_code = resource$exit_code,
				auth_uuid = resource$auth_uuid, locked_by_uuid = resource$locked_by_uuid,
				scheduling_parameters = resource$scheduling_parameters)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		containers.lock = function(uuid)
		{
			endPoint <- stringr::str_interp("containers/${uuid}/lock")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- NULL
			
			response <- private$REST$http$exec("POST", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- Container$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				created_at = resource$created_at, modified_at = resource$modified_at,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				state = resource$state, started_at = resource$started_at,
				finished_at = resource$finished_at, log = resource$log,
				environment = resource$environment, cwd = resource$cwd,
				command = resource$command, output_path = resource$output_path,
				mounts = resource$mounts, runtime_constraints = resource$runtime_constraints,
				output = resource$output, container_image = resource$container_image,
				progress = resource$progress, priority = resource$priority,
				updated_at = resource$updated_at, exit_code = resource$exit_code,
				auth_uuid = resource$auth_uuid, locked_by_uuid = resource$locked_by_uuid,
				scheduling_parameters = resource$scheduling_parameters)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		containers.unlock = function(uuid)
		{
			endPoint <- stringr::str_interp("containers/${uuid}/unlock")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- NULL
			
			response <- private$REST$http$exec("POST", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- Container$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				created_at = resource$created_at, modified_at = resource$modified_at,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				state = resource$state, started_at = resource$started_at,
				finished_at = resource$finished_at, log = resource$log,
				environment = resource$environment, cwd = resource$cwd,
				command = resource$command, output_path = resource$output_path,
				mounts = resource$mounts, runtime_constraints = resource$runtime_constraints,
				output = resource$output, container_image = resource$container_image,
				progress = resource$progress, priority = resource$priority,
				updated_at = resource$updated_at, exit_code = resource$exit_code,
				auth_uuid = resource$auth_uuid, locked_by_uuid = resource$locked_by_uuid,
				scheduling_parameters = resource$scheduling_parameters)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		containers.current = function()
		{
			endPoint <- stringr::str_interp("containers/current")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- NULL
			body <- NULL
			
			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- Container$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				created_at = resource$created_at, modified_at = resource$modified_at,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				state = resource$state, started_at = resource$started_at,
				finished_at = resource$finished_at, log = resource$log,
				environment = resource$environment, cwd = resource$cwd,
				command = resource$command, output_path = resource$output_path,
				mounts = resource$mounts, runtime_constraints = resource$runtime_constraints,
				output = resource$output, container_image = resource$container_image,
				progress = resource$progress, priority = resource$priority,
				updated_at = resource$updated_at, exit_code = resource$exit_code,
				auth_uuid = resource$auth_uuid, locked_by_uuid = resource$locked_by_uuid,
				scheduling_parameters = resource$scheduling_parameters)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		containers.list = function(filters = NULL,
			where = NULL, order = NULL, select = NULL,
			distinct = NULL, limit = "100", offset = "0",
			count = "exact")
		{
			endPoint <- stringr::str_interp("containers")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(filters = filters, where = where,
				order = order, select = select, distinct = distinct,
				limit = limit, offset = offset, count = count)
			body <- NULL
			
			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- ContainerList$new(kind = resource$kind,
				etag = resource$etag, items = resource$items,
				next_link = resource$next_link, next_page_token = resource$next_page_token,
				selfLink = resource$selfLink)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		containers.show = function(uuid)
		{
			endPoint <- stringr::str_interp("containers/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- NULL
			
			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- Container$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				created_at = resource$created_at, modified_at = resource$modified_at,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				state = resource$state, started_at = resource$started_at,
				finished_at = resource$finished_at, log = resource$log,
				environment = resource$environment, cwd = resource$cwd,
				command = resource$command, output_path = resource$output_path,
				mounts = resource$mounts, runtime_constraints = resource$runtime_constraints,
				output = resource$output, container_image = resource$container_image,
				progress = resource$progress, priority = resource$priority,
				updated_at = resource$updated_at, exit_code = resource$exit_code,
				auth_uuid = resource$auth_uuid, locked_by_uuid = resource$locked_by_uuid,
				scheduling_parameters = resource$scheduling_parameters)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		containers.destroy = function(uuid)
		{
			endPoint <- stringr::str_interp("containers/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- NULL
			
			response <- private$REST$http$exec("DELETE", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- Container$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				created_at = resource$created_at, modified_at = resource$modified_at,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				state = resource$state, started_at = resource$started_at,
				finished_at = resource$finished_at, log = resource$log,
				environment = resource$environment, cwd = resource$cwd,
				command = resource$command, output_path = resource$output_path,
				mounts = resource$mounts, runtime_constraints = resource$runtime_constraints,
				output = resource$output, container_image = resource$container_image,
				progress = resource$progress, priority = resource$priority,
				updated_at = resource$updated_at, exit_code = resource$exit_code,
				auth_uuid = resource$auth_uuid, locked_by_uuid = resource$locked_by_uuid,
				scheduling_parameters = resource$scheduling_parameters)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		humans.get = function(uuid)
		{
			endPoint <- stringr::str_interp("humans/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- NULL
			
			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- Human$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, properties = resource$properties,
				created_at = resource$created_at, updated_at = resource$updated_at)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		humans.index = function(filters = NULL, where = NULL,
			order = NULL, select = NULL, distinct = NULL,
			limit = "100", offset = "0", count = "exact")
		{
			endPoint <- stringr::str_interp("humans")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(filters = filters, where = where,
				order = order, select = select, distinct = distinct,
				limit = limit, offset = offset, count = count)
			body <- NULL
			
			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- HumanList$new(kind = resource$kind,
				etag = resource$etag, items = resource$items,
				next_link = resource$next_link, next_page_token = resource$next_page_token,
				selfLink = resource$selfLink)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		humans.create = function(human, ensure_unique_name = "false")
		{
			endPoint <- stringr::str_interp("humans")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(ensure_unique_name = ensure_unique_name)
			body <- human$toJSON()
			
			response <- private$REST$http$exec("POST", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- Human$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, properties = resource$properties,
				created_at = resource$created_at, updated_at = resource$updated_at)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		humans.update = function(human, uuid)
		{
			endPoint <- stringr::str_interp("humans/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- human$toJSON()
			
			response <- private$REST$http$exec("PUT", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- Human$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, properties = resource$properties,
				created_at = resource$created_at, updated_at = resource$updated_at)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		humans.delete = function(uuid)
		{
			endPoint <- stringr::str_interp("humans/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- NULL
			
			response <- private$REST$http$exec("DELETE", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- Human$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, properties = resource$properties,
				created_at = resource$created_at, updated_at = resource$updated_at)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		humans.list = function(filters = NULL, where = NULL,
			order = NULL, select = NULL, distinct = NULL,
			limit = "100", offset = "0", count = "exact")
		{
			endPoint <- stringr::str_interp("humans")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(filters = filters, where = where,
				order = order, select = select, distinct = distinct,
				limit = limit, offset = offset, count = count)
			body <- NULL
			
			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- HumanList$new(kind = resource$kind,
				etag = resource$etag, items = resource$items,
				next_link = resource$next_link, next_page_token = resource$next_page_token,
				selfLink = resource$selfLink)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		humans.show = function(uuid)
		{
			endPoint <- stringr::str_interp("humans/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- NULL
			
			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- Human$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, properties = resource$properties,
				created_at = resource$created_at, updated_at = resource$updated_at)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		humans.destroy = function(uuid)
		{
			endPoint <- stringr::str_interp("humans/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- NULL
			
			response <- private$REST$http$exec("DELETE", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- Human$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, properties = resource$properties,
				created_at = resource$created_at, updated_at = resource$updated_at)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		job_tasks.get = function(uuid)
		{
			endPoint <- stringr::str_interp("job_tasks/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- NULL
			
			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- JobTask$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, job_uuid = resource$job_uuid,
				sequence = resource$sequence, parameters = resource$parameters,
				output = resource$output, progress = resource$progress,
				success = resource$success, created_at = resource$created_at,
				updated_at = resource$updated_at, created_by_job_task_uuid = resource$created_by_job_task_uuid,
				qsequence = resource$qsequence, started_at = resource$started_at,
				finished_at = resource$finished_at)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		job_tasks.index = function(filters = NULL,
			where = NULL, order = NULL, select = NULL,
			distinct = NULL, limit = "100", offset = "0",
			count = "exact")
		{
			endPoint <- stringr::str_interp("job_tasks")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(filters = filters, where = where,
				order = order, select = select, distinct = distinct,
				limit = limit, offset = offset, count = count)
			body <- NULL
			
			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- JobTaskList$new(kind = resource$kind,
				etag = resource$etag, items = resource$items,
				next_link = resource$next_link, next_page_token = resource$next_page_token,
				selfLink = resource$selfLink)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		job_tasks.create = function(job_task, ensure_unique_name = "false")
		{
			endPoint <- stringr::str_interp("job_tasks")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(ensure_unique_name = ensure_unique_name)
			body <- job_task$toJSON()
			
			response <- private$REST$http$exec("POST", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- JobTask$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, job_uuid = resource$job_uuid,
				sequence = resource$sequence, parameters = resource$parameters,
				output = resource$output, progress = resource$progress,
				success = resource$success, created_at = resource$created_at,
				updated_at = resource$updated_at, created_by_job_task_uuid = resource$created_by_job_task_uuid,
				qsequence = resource$qsequence, started_at = resource$started_at,
				finished_at = resource$finished_at)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		job_tasks.update = function(job_task, uuid)
		{
			endPoint <- stringr::str_interp("job_tasks/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- job_task$toJSON()
			
			response <- private$REST$http$exec("PUT", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- JobTask$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, job_uuid = resource$job_uuid,
				sequence = resource$sequence, parameters = resource$parameters,
				output = resource$output, progress = resource$progress,
				success = resource$success, created_at = resource$created_at,
				updated_at = resource$updated_at, created_by_job_task_uuid = resource$created_by_job_task_uuid,
				qsequence = resource$qsequence, started_at = resource$started_at,
				finished_at = resource$finished_at)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		job_tasks.delete = function(uuid)
		{
			endPoint <- stringr::str_interp("job_tasks/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- NULL
			
			response <- private$REST$http$exec("DELETE", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- JobTask$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, job_uuid = resource$job_uuid,
				sequence = resource$sequence, parameters = resource$parameters,
				output = resource$output, progress = resource$progress,
				success = resource$success, created_at = resource$created_at,
				updated_at = resource$updated_at, created_by_job_task_uuid = resource$created_by_job_task_uuid,
				qsequence = resource$qsequence, started_at = resource$started_at,
				finished_at = resource$finished_at)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		job_tasks.list = function(filters = NULL,
			where = NULL, order = NULL, select = NULL,
			distinct = NULL, limit = "100", offset = "0",
			count = "exact")
		{
			endPoint <- stringr::str_interp("job_tasks")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(filters = filters, where = where,
				order = order, select = select, distinct = distinct,
				limit = limit, offset = offset, count = count)
			body <- NULL
			
			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- JobTaskList$new(kind = resource$kind,
				etag = resource$etag, items = resource$items,
				next_link = resource$next_link, next_page_token = resource$next_page_token,
				selfLink = resource$selfLink)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		job_tasks.show = function(uuid)
		{
			endPoint <- stringr::str_interp("job_tasks/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- NULL
			
			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- JobTask$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, job_uuid = resource$job_uuid,
				sequence = resource$sequence, parameters = resource$parameters,
				output = resource$output, progress = resource$progress,
				success = resource$success, created_at = resource$created_at,
				updated_at = resource$updated_at, created_by_job_task_uuid = resource$created_by_job_task_uuid,
				qsequence = resource$qsequence, started_at = resource$started_at,
				finished_at = resource$finished_at)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		job_tasks.destroy = function(uuid)
		{
			endPoint <- stringr::str_interp("job_tasks/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- NULL
			
			response <- private$REST$http$exec("DELETE", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- JobTask$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, job_uuid = resource$job_uuid,
				sequence = resource$sequence, parameters = resource$parameters,
				output = resource$output, progress = resource$progress,
				success = resource$success, created_at = resource$created_at,
				updated_at = resource$updated_at, created_by_job_task_uuid = resource$created_by_job_task_uuid,
				qsequence = resource$qsequence, started_at = resource$started_at,
				finished_at = resource$finished_at)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		links.get = function(uuid)
		{
			endPoint <- stringr::str_interp("links/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- NULL
			
			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- Link$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				created_at = resource$created_at, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, tail_uuid = resource$tail_uuid,
				link_class = resource$link_class, name = resource$name,
				head_uuid = resource$head_uuid, properties = resource$properties,
				updated_at = resource$updated_at)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		links.index = function(filters = NULL, where = NULL,
			order = NULL, select = NULL, distinct = NULL,
			limit = "100", offset = "0", count = "exact")
		{
			endPoint <- stringr::str_interp("links")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(filters = filters, where = where,
				order = order, select = select, distinct = distinct,
				limit = limit, offset = offset, count = count)
			body <- NULL
			
			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- LinkList$new(kind = resource$kind,
				etag = resource$etag, items = resource$items,
				next_link = resource$next_link, next_page_token = resource$next_page_token,
				selfLink = resource$selfLink)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		links.create = function(link, ensure_unique_name = "false")
		{
			endPoint <- stringr::str_interp("links")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(ensure_unique_name = ensure_unique_name)
			body <- link$toJSON()
			
			response <- private$REST$http$exec("POST", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- Link$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				created_at = resource$created_at, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, tail_uuid = resource$tail_uuid,
				link_class = resource$link_class, name = resource$name,
				head_uuid = resource$head_uuid, properties = resource$properties,
				updated_at = resource$updated_at)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		links.update = function(link, uuid)
		{
			endPoint <- stringr::str_interp("links/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- link$toJSON()
			
			response <- private$REST$http$exec("PUT", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- Link$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				created_at = resource$created_at, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, tail_uuid = resource$tail_uuid,
				link_class = resource$link_class, name = resource$name,
				head_uuid = resource$head_uuid, properties = resource$properties,
				updated_at = resource$updated_at)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		links.delete = function(uuid)
		{
			endPoint <- stringr::str_interp("links/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- NULL
			
			response <- private$REST$http$exec("DELETE", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- Link$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				created_at = resource$created_at, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, tail_uuid = resource$tail_uuid,
				link_class = resource$link_class, name = resource$name,
				head_uuid = resource$head_uuid, properties = resource$properties,
				updated_at = resource$updated_at)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		links.list = function(filters = NULL, where = NULL,
			order = NULL, select = NULL, distinct = NULL,
			limit = "100", offset = "0", count = "exact")
		{
			endPoint <- stringr::str_interp("links")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(filters = filters, where = where,
				order = order, select = select, distinct = distinct,
				limit = limit, offset = offset, count = count)
			body <- NULL
			
			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- LinkList$new(kind = resource$kind,
				etag = resource$etag, items = resource$items,
				next_link = resource$next_link, next_page_token = resource$next_page_token,
				selfLink = resource$selfLink)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		links.show = function(uuid)
		{
			endPoint <- stringr::str_interp("links/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- NULL
			
			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- Link$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				created_at = resource$created_at, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, tail_uuid = resource$tail_uuid,
				link_class = resource$link_class, name = resource$name,
				head_uuid = resource$head_uuid, properties = resource$properties,
				updated_at = resource$updated_at)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		links.destroy = function(uuid)
		{
			endPoint <- stringr::str_interp("links/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- NULL
			
			response <- private$REST$http$exec("DELETE", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- Link$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				created_at = resource$created_at, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, tail_uuid = resource$tail_uuid,
				link_class = resource$link_class, name = resource$name,
				head_uuid = resource$head_uuid, properties = resource$properties,
				updated_at = resource$updated_at)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		links.get_permissions = function(uuid)
		{
			endPoint <- stringr::str_interp("permissions/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- NULL
			
			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- Link$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				created_at = resource$created_at, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, tail_uuid = resource$tail_uuid,
				link_class = resource$link_class, name = resource$name,
				head_uuid = resource$head_uuid, properties = resource$properties,
				updated_at = resource$updated_at)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		jobs.get = function(uuid)
		{
			endPoint <- stringr::str_interp("jobs/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- NULL
			
			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- Job$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, submit_id = resource$submit_id,
				script = resource$script, script_version = resource$script_version,
				script_parameters = resource$script_parameters,
				cancelled_by_client_uuid = resource$cancelled_by_client_uuid,
				cancelled_by_user_uuid = resource$cancelled_by_user_uuid,
				cancelled_at = resource$cancelled_at, started_at = resource$started_at,
				finished_at = resource$finished_at, running = resource$running,
				success = resource$success, output = resource$output,
				created_at = resource$created_at, updated_at = resource$updated_at,
				is_locked_by_uuid = resource$is_locked_by_uuid,
				log = resource$log, tasks_summary = resource$tasks_summary,
				runtime_constraints = resource$runtime_constraints,
				nondeterministic = resource$nondeterministic,
				repository = resource$repository, supplied_script_version = resource$supplied_script_version,
				docker_image_locator = resource$docker_image_locator,
				priority = resource$priority, description = resource$description,
				state = resource$state, arvados_sdk_version = resource$arvados_sdk_version,
				components = resource$components, script_parameters_digest = resource$script_parameters_digest)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		jobs.index = function(filters = NULL, where = NULL,
			order = NULL, select = NULL, distinct = NULL,
			limit = "100", offset = "0", count = "exact")
		{
			endPoint <- stringr::str_interp("jobs")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(filters = filters, where = where,
				order = order, select = select, distinct = distinct,
				limit = limit, offset = offset, count = count)
			body <- NULL
			
			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- JobList$new(kind = resource$kind,
				etag = resource$etag, items = resource$items,
				next_link = resource$next_link, next_page_token = resource$next_page_token,
				selfLink = resource$selfLink)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		jobs.create = function(job, ensure_unique_name = "false",
			find_or_create = "false", filters = NULL,
			minimum_script_version = NULL, exclude_script_versions = NULL)
		{
			endPoint <- stringr::str_interp("jobs")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(ensure_unique_name = ensure_unique_name,
				find_or_create = find_or_create, filters = filters,
				minimum_script_version = minimum_script_version,
				exclude_script_versions = exclude_script_versions)
			body <- job$toJSON()
			
			response <- private$REST$http$exec("POST", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- Job$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, submit_id = resource$submit_id,
				script = resource$script, script_version = resource$script_version,
				script_parameters = resource$script_parameters,
				cancelled_by_client_uuid = resource$cancelled_by_client_uuid,
				cancelled_by_user_uuid = resource$cancelled_by_user_uuid,
				cancelled_at = resource$cancelled_at, started_at = resource$started_at,
				finished_at = resource$finished_at, running = resource$running,
				success = resource$success, output = resource$output,
				created_at = resource$created_at, updated_at = resource$updated_at,
				is_locked_by_uuid = resource$is_locked_by_uuid,
				log = resource$log, tasks_summary = resource$tasks_summary,
				runtime_constraints = resource$runtime_constraints,
				nondeterministic = resource$nondeterministic,
				repository = resource$repository, supplied_script_version = resource$supplied_script_version,
				docker_image_locator = resource$docker_image_locator,
				priority = resource$priority, description = resource$description,
				state = resource$state, arvados_sdk_version = resource$arvados_sdk_version,
				components = resource$components, script_parameters_digest = resource$script_parameters_digest)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		jobs.update = function(job, uuid)
		{
			endPoint <- stringr::str_interp("jobs/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- job$toJSON()
			
			response <- private$REST$http$exec("PUT", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- Job$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, submit_id = resource$submit_id,
				script = resource$script, script_version = resource$script_version,
				script_parameters = resource$script_parameters,
				cancelled_by_client_uuid = resource$cancelled_by_client_uuid,
				cancelled_by_user_uuid = resource$cancelled_by_user_uuid,
				cancelled_at = resource$cancelled_at, started_at = resource$started_at,
				finished_at = resource$finished_at, running = resource$running,
				success = resource$success, output = resource$output,
				created_at = resource$created_at, updated_at = resource$updated_at,
				is_locked_by_uuid = resource$is_locked_by_uuid,
				log = resource$log, tasks_summary = resource$tasks_summary,
				runtime_constraints = resource$runtime_constraints,
				nondeterministic = resource$nondeterministic,
				repository = resource$repository, supplied_script_version = resource$supplied_script_version,
				docker_image_locator = resource$docker_image_locator,
				priority = resource$priority, description = resource$description,
				state = resource$state, arvados_sdk_version = resource$arvados_sdk_version,
				components = resource$components, script_parameters_digest = resource$script_parameters_digest)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		jobs.delete = function(uuid)
		{
			endPoint <- stringr::str_interp("jobs/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- NULL
			
			response <- private$REST$http$exec("DELETE", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- Job$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, submit_id = resource$submit_id,
				script = resource$script, script_version = resource$script_version,
				script_parameters = resource$script_parameters,
				cancelled_by_client_uuid = resource$cancelled_by_client_uuid,
				cancelled_by_user_uuid = resource$cancelled_by_user_uuid,
				cancelled_at = resource$cancelled_at, started_at = resource$started_at,
				finished_at = resource$finished_at, running = resource$running,
				success = resource$success, output = resource$output,
				created_at = resource$created_at, updated_at = resource$updated_at,
				is_locked_by_uuid = resource$is_locked_by_uuid,
				log = resource$log, tasks_summary = resource$tasks_summary,
				runtime_constraints = resource$runtime_constraints,
				nondeterministic = resource$nondeterministic,
				repository = resource$repository, supplied_script_version = resource$supplied_script_version,
				docker_image_locator = resource$docker_image_locator,
				priority = resource$priority, description = resource$description,
				state = resource$state, arvados_sdk_version = resource$arvados_sdk_version,
				components = resource$components, script_parameters_digest = resource$script_parameters_digest)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		jobs.queue = function(filters = NULL, where = NULL,
			order = NULL, select = NULL, distinct = NULL,
			limit = "100", offset = "0", count = "exact")
		{
			endPoint <- stringr::str_interp("jobs/queue")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(filters = filters, where = where,
				order = order, select = select, distinct = distinct,
				limit = limit, offset = offset, count = count)
			body <- NULL
			
			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- Job$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, submit_id = resource$submit_id,
				script = resource$script, script_version = resource$script_version,
				script_parameters = resource$script_parameters,
				cancelled_by_client_uuid = resource$cancelled_by_client_uuid,
				cancelled_by_user_uuid = resource$cancelled_by_user_uuid,
				cancelled_at = resource$cancelled_at, started_at = resource$started_at,
				finished_at = resource$finished_at, running = resource$running,
				success = resource$success, output = resource$output,
				created_at = resource$created_at, updated_at = resource$updated_at,
				is_locked_by_uuid = resource$is_locked_by_uuid,
				log = resource$log, tasks_summary = resource$tasks_summary,
				runtime_constraints = resource$runtime_constraints,
				nondeterministic = resource$nondeterministic,
				repository = resource$repository, supplied_script_version = resource$supplied_script_version,
				docker_image_locator = resource$docker_image_locator,
				priority = resource$priority, description = resource$description,
				state = resource$state, arvados_sdk_version = resource$arvados_sdk_version,
				components = resource$components, script_parameters_digest = resource$script_parameters_digest)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		jobs.queue_size = function()
		{
			endPoint <- stringr::str_interp("jobs/queue_size")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- NULL
			body <- NULL
			
			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- Job$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, submit_id = resource$submit_id,
				script = resource$script, script_version = resource$script_version,
				script_parameters = resource$script_parameters,
				cancelled_by_client_uuid = resource$cancelled_by_client_uuid,
				cancelled_by_user_uuid = resource$cancelled_by_user_uuid,
				cancelled_at = resource$cancelled_at, started_at = resource$started_at,
				finished_at = resource$finished_at, running = resource$running,
				success = resource$success, output = resource$output,
				created_at = resource$created_at, updated_at = resource$updated_at,
				is_locked_by_uuid = resource$is_locked_by_uuid,
				log = resource$log, tasks_summary = resource$tasks_summary,
				runtime_constraints = resource$runtime_constraints,
				nondeterministic = resource$nondeterministic,
				repository = resource$repository, supplied_script_version = resource$supplied_script_version,
				docker_image_locator = resource$docker_image_locator,
				priority = resource$priority, description = resource$description,
				state = resource$state, arvados_sdk_version = resource$arvados_sdk_version,
				components = resource$components, script_parameters_digest = resource$script_parameters_digest)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		jobs.cancel = function(uuid)
		{
			endPoint <- stringr::str_interp("jobs/${uuid}/cancel")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- NULL
			
			response <- private$REST$http$exec("POST", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- Job$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, submit_id = resource$submit_id,
				script = resource$script, script_version = resource$script_version,
				script_parameters = resource$script_parameters,
				cancelled_by_client_uuid = resource$cancelled_by_client_uuid,
				cancelled_by_user_uuid = resource$cancelled_by_user_uuid,
				cancelled_at = resource$cancelled_at, started_at = resource$started_at,
				finished_at = resource$finished_at, running = resource$running,
				success = resource$success, output = resource$output,
				created_at = resource$created_at, updated_at = resource$updated_at,
				is_locked_by_uuid = resource$is_locked_by_uuid,
				log = resource$log, tasks_summary = resource$tasks_summary,
				runtime_constraints = resource$runtime_constraints,
				nondeterministic = resource$nondeterministic,
				repository = resource$repository, supplied_script_version = resource$supplied_script_version,
				docker_image_locator = resource$docker_image_locator,
				priority = resource$priority, description = resource$description,
				state = resource$state, arvados_sdk_version = resource$arvados_sdk_version,
				components = resource$components, script_parameters_digest = resource$script_parameters_digest)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		jobs.lock = function(uuid)
		{
			endPoint <- stringr::str_interp("jobs/${uuid}/lock")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- NULL
			
			response <- private$REST$http$exec("POST", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- Job$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, submit_id = resource$submit_id,
				script = resource$script, script_version = resource$script_version,
				script_parameters = resource$script_parameters,
				cancelled_by_client_uuid = resource$cancelled_by_client_uuid,
				cancelled_by_user_uuid = resource$cancelled_by_user_uuid,
				cancelled_at = resource$cancelled_at, started_at = resource$started_at,
				finished_at = resource$finished_at, running = resource$running,
				success = resource$success, output = resource$output,
				created_at = resource$created_at, updated_at = resource$updated_at,
				is_locked_by_uuid = resource$is_locked_by_uuid,
				log = resource$log, tasks_summary = resource$tasks_summary,
				runtime_constraints = resource$runtime_constraints,
				nondeterministic = resource$nondeterministic,
				repository = resource$repository, supplied_script_version = resource$supplied_script_version,
				docker_image_locator = resource$docker_image_locator,
				priority = resource$priority, description = resource$description,
				state = resource$state, arvados_sdk_version = resource$arvados_sdk_version,
				components = resource$components, script_parameters_digest = resource$script_parameters_digest)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		jobs.list = function(filters = NULL, where = NULL,
			order = NULL, select = NULL, distinct = NULL,
			limit = "100", offset = "0", count = "exact")
		{
			endPoint <- stringr::str_interp("jobs")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(filters = filters, where = where,
				order = order, select = select, distinct = distinct,
				limit = limit, offset = offset, count = count)
			body <- NULL
			
			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- JobList$new(kind = resource$kind,
				etag = resource$etag, items = resource$items,
				next_link = resource$next_link, next_page_token = resource$next_page_token,
				selfLink = resource$selfLink)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		jobs.show = function(uuid)
		{
			endPoint <- stringr::str_interp("jobs/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- NULL
			
			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- Job$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, submit_id = resource$submit_id,
				script = resource$script, script_version = resource$script_version,
				script_parameters = resource$script_parameters,
				cancelled_by_client_uuid = resource$cancelled_by_client_uuid,
				cancelled_by_user_uuid = resource$cancelled_by_user_uuid,
				cancelled_at = resource$cancelled_at, started_at = resource$started_at,
				finished_at = resource$finished_at, running = resource$running,
				success = resource$success, output = resource$output,
				created_at = resource$created_at, updated_at = resource$updated_at,
				is_locked_by_uuid = resource$is_locked_by_uuid,
				log = resource$log, tasks_summary = resource$tasks_summary,
				runtime_constraints = resource$runtime_constraints,
				nondeterministic = resource$nondeterministic,
				repository = resource$repository, supplied_script_version = resource$supplied_script_version,
				docker_image_locator = resource$docker_image_locator,
				priority = resource$priority, description = resource$description,
				state = resource$state, arvados_sdk_version = resource$arvados_sdk_version,
				components = resource$components, script_parameters_digest = resource$script_parameters_digest)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		jobs.destroy = function(uuid)
		{
			endPoint <- stringr::str_interp("jobs/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- NULL
			
			response <- private$REST$http$exec("DELETE", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- Job$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, submit_id = resource$submit_id,
				script = resource$script, script_version = resource$script_version,
				script_parameters = resource$script_parameters,
				cancelled_by_client_uuid = resource$cancelled_by_client_uuid,
				cancelled_by_user_uuid = resource$cancelled_by_user_uuid,
				cancelled_at = resource$cancelled_at, started_at = resource$started_at,
				finished_at = resource$finished_at, running = resource$running,
				success = resource$success, output = resource$output,
				created_at = resource$created_at, updated_at = resource$updated_at,
				is_locked_by_uuid = resource$is_locked_by_uuid,
				log = resource$log, tasks_summary = resource$tasks_summary,
				runtime_constraints = resource$runtime_constraints,
				nondeterministic = resource$nondeterministic,
				repository = resource$repository, supplied_script_version = resource$supplied_script_version,
				docker_image_locator = resource$docker_image_locator,
				priority = resource$priority, description = resource$description,
				state = resource$state, arvados_sdk_version = resource$arvados_sdk_version,
				components = resource$components, script_parameters_digest = resource$script_parameters_digest)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		keep_disks.get = function(uuid)
		{
			endPoint <- stringr::str_interp("keep_disks/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- NULL
			
			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- KeepDisk$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, ping_secret = resource$ping_secret,
				node_uuid = resource$node_uuid, filesystem_uuid = resource$filesystem_uuid,
				bytes_total = resource$bytes_total, bytes_free = resource$bytes_free,
				is_readable = resource$is_readable, is_writable = resource$is_writable,
				last_read_at = resource$last_read_at, last_write_at = resource$last_write_at,
				last_ping_at = resource$last_ping_at, created_at = resource$created_at,
				updated_at = resource$updated_at, keep_service_uuid = resource$keep_service_uuid)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		keep_disks.index = function(filters = NULL,
			where = NULL, order = NULL, select = NULL,
			distinct = NULL, limit = "100", offset = "0",
			count = "exact")
		{
			endPoint <- stringr::str_interp("keep_disks")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(filters = filters, where = where,
				order = order, select = select, distinct = distinct,
				limit = limit, offset = offset, count = count)
			body <- NULL
			
			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- KeepDiskList$new(kind = resource$kind,
				etag = resource$etag, items = resource$items,
				next_link = resource$next_link, next_page_token = resource$next_page_token,
				selfLink = resource$selfLink)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		keep_disks.create = function(keep_disk, ensure_unique_name = "false")
		{
			endPoint <- stringr::str_interp("keep_disks")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(ensure_unique_name = ensure_unique_name)
			body <- keep_disk$toJSON()
			
			response <- private$REST$http$exec("POST", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- KeepDisk$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, ping_secret = resource$ping_secret,
				node_uuid = resource$node_uuid, filesystem_uuid = resource$filesystem_uuid,
				bytes_total = resource$bytes_total, bytes_free = resource$bytes_free,
				is_readable = resource$is_readable, is_writable = resource$is_writable,
				last_read_at = resource$last_read_at, last_write_at = resource$last_write_at,
				last_ping_at = resource$last_ping_at, created_at = resource$created_at,
				updated_at = resource$updated_at, keep_service_uuid = resource$keep_service_uuid)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		keep_disks.update = function(keep_disk, uuid)
		{
			endPoint <- stringr::str_interp("keep_disks/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- keep_disk$toJSON()
			
			response <- private$REST$http$exec("PUT", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- KeepDisk$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, ping_secret = resource$ping_secret,
				node_uuid = resource$node_uuid, filesystem_uuid = resource$filesystem_uuid,
				bytes_total = resource$bytes_total, bytes_free = resource$bytes_free,
				is_readable = resource$is_readable, is_writable = resource$is_writable,
				last_read_at = resource$last_read_at, last_write_at = resource$last_write_at,
				last_ping_at = resource$last_ping_at, created_at = resource$created_at,
				updated_at = resource$updated_at, keep_service_uuid = resource$keep_service_uuid)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		keep_disks.delete = function(uuid)
		{
			endPoint <- stringr::str_interp("keep_disks/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- NULL
			
			response <- private$REST$http$exec("DELETE", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- KeepDisk$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, ping_secret = resource$ping_secret,
				node_uuid = resource$node_uuid, filesystem_uuid = resource$filesystem_uuid,
				bytes_total = resource$bytes_total, bytes_free = resource$bytes_free,
				is_readable = resource$is_readable, is_writable = resource$is_writable,
				last_read_at = resource$last_read_at, last_write_at = resource$last_write_at,
				last_ping_at = resource$last_ping_at, created_at = resource$created_at,
				updated_at = resource$updated_at, keep_service_uuid = resource$keep_service_uuid)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		keep_disks.ping = function(uuid = NULL, ping_secret,
			node_uuid = NULL, filesystem_uuid = NULL,
			service_host = NULL, service_port, service_ssl_flag)
		{
			endPoint <- stringr::str_interp("keep_disks/ping")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid, ping_secret = ping_secret,
				node_uuid = node_uuid, filesystem_uuid = filesystem_uuid,
				service_host = service_host, service_port = service_port,
				service_ssl_flag = service_ssl_flag)
			body <- NULL
			
			response <- private$REST$http$exec("POST", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- KeepDisk$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, ping_secret = resource$ping_secret,
				node_uuid = resource$node_uuid, filesystem_uuid = resource$filesystem_uuid,
				bytes_total = resource$bytes_total, bytes_free = resource$bytes_free,
				is_readable = resource$is_readable, is_writable = resource$is_writable,
				last_read_at = resource$last_read_at, last_write_at = resource$last_write_at,
				last_ping_at = resource$last_ping_at, created_at = resource$created_at,
				updated_at = resource$updated_at, keep_service_uuid = resource$keep_service_uuid)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		keep_disks.list = function(filters = NULL,
			where = NULL, order = NULL, select = NULL,
			distinct = NULL, limit = "100", offset = "0",
			count = "exact")
		{
			endPoint <- stringr::str_interp("keep_disks")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(filters = filters, where = where,
				order = order, select = select, distinct = distinct,
				limit = limit, offset = offset, count = count)
			body <- NULL
			
			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- KeepDiskList$new(kind = resource$kind,
				etag = resource$etag, items = resource$items,
				next_link = resource$next_link, next_page_token = resource$next_page_token,
				selfLink = resource$selfLink)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		keep_disks.show = function(uuid)
		{
			endPoint <- stringr::str_interp("keep_disks/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- NULL
			
			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- KeepDisk$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, ping_secret = resource$ping_secret,
				node_uuid = resource$node_uuid, filesystem_uuid = resource$filesystem_uuid,
				bytes_total = resource$bytes_total, bytes_free = resource$bytes_free,
				is_readable = resource$is_readable, is_writable = resource$is_writable,
				last_read_at = resource$last_read_at, last_write_at = resource$last_write_at,
				last_ping_at = resource$last_ping_at, created_at = resource$created_at,
				updated_at = resource$updated_at, keep_service_uuid = resource$keep_service_uuid)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		keep_disks.destroy = function(uuid)
		{
			endPoint <- stringr::str_interp("keep_disks/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- NULL
			
			response <- private$REST$http$exec("DELETE", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- KeepDisk$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, ping_secret = resource$ping_secret,
				node_uuid = resource$node_uuid, filesystem_uuid = resource$filesystem_uuid,
				bytes_total = resource$bytes_total, bytes_free = resource$bytes_free,
				is_readable = resource$is_readable, is_writable = resource$is_writable,
				last_read_at = resource$last_read_at, last_write_at = resource$last_write_at,
				last_ping_at = resource$last_ping_at, created_at = resource$created_at,
				updated_at = resource$updated_at, keep_service_uuid = resource$keep_service_uuid)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		keep_services.get = function(uuid)
		{
			endPoint <- stringr::str_interp("keep_services/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- NULL
			
			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- KeepService$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, service_host = resource$service_host,
				service_port = resource$service_port, service_ssl_flag = resource$service_ssl_flag,
				service_type = resource$service_type, created_at = resource$created_at,
				updated_at = resource$updated_at, read_only = resource$read_only)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		keep_services.index = function(filters = NULL,
			where = NULL, order = NULL, select = NULL,
			distinct = NULL, limit = "100", offset = "0",
			count = "exact")
		{
			endPoint <- stringr::str_interp("keep_services")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(filters = filters, where = where,
				order = order, select = select, distinct = distinct,
				limit = limit, offset = offset, count = count)
			body <- NULL
			
			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- KeepServiceList$new(kind = resource$kind,
				etag = resource$etag, items = resource$items,
				next_link = resource$next_link, next_page_token = resource$next_page_token,
				selfLink = resource$selfLink)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		keep_services.create = function(keep_service,
			ensure_unique_name = "false")
		{
			endPoint <- stringr::str_interp("keep_services")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(ensure_unique_name = ensure_unique_name)
			body <- keep_service$toJSON()
			
			response <- private$REST$http$exec("POST", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- KeepService$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, service_host = resource$service_host,
				service_port = resource$service_port, service_ssl_flag = resource$service_ssl_flag,
				service_type = resource$service_type, created_at = resource$created_at,
				updated_at = resource$updated_at, read_only = resource$read_only)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		keep_services.update = function(keep_service, uuid)
		{
			endPoint <- stringr::str_interp("keep_services/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- keep_service$toJSON()
			
			response <- private$REST$http$exec("PUT", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- KeepService$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, service_host = resource$service_host,
				service_port = resource$service_port, service_ssl_flag = resource$service_ssl_flag,
				service_type = resource$service_type, created_at = resource$created_at,
				updated_at = resource$updated_at, read_only = resource$read_only)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		keep_services.delete = function(uuid)
		{
			endPoint <- stringr::str_interp("keep_services/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- NULL
			
			response <- private$REST$http$exec("DELETE", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- KeepService$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, service_host = resource$service_host,
				service_port = resource$service_port, service_ssl_flag = resource$service_ssl_flag,
				service_type = resource$service_type, created_at = resource$created_at,
				updated_at = resource$updated_at, read_only = resource$read_only)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		keep_services.accessible = function()
		{
			endPoint <- stringr::str_interp("keep_services/accessible")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- NULL
			body <- NULL
			
			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- KeepService$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, service_host = resource$service_host,
				service_port = resource$service_port, service_ssl_flag = resource$service_ssl_flag,
				service_type = resource$service_type, created_at = resource$created_at,
				updated_at = resource$updated_at, read_only = resource$read_only)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		keep_services.list = function(filters = NULL,
			where = NULL, order = NULL, select = NULL,
			distinct = NULL, limit = "100", offset = "0",
			count = "exact")
		{
			endPoint <- stringr::str_interp("keep_services")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(filters = filters, where = where,
				order = order, select = select, distinct = distinct,
				limit = limit, offset = offset, count = count)
			body <- NULL
			
			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- KeepServiceList$new(kind = resource$kind,
				etag = resource$etag, items = resource$items,
				next_link = resource$next_link, next_page_token = resource$next_page_token,
				selfLink = resource$selfLink)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		keep_services.show = function(uuid)
		{
			endPoint <- stringr::str_interp("keep_services/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- NULL
			
			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- KeepService$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, service_host = resource$service_host,
				service_port = resource$service_port, service_ssl_flag = resource$service_ssl_flag,
				service_type = resource$service_type, created_at = resource$created_at,
				updated_at = resource$updated_at, read_only = resource$read_only)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		keep_services.destroy = function(uuid)
		{
			endPoint <- stringr::str_interp("keep_services/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- NULL
			
			response <- private$REST$http$exec("DELETE", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- KeepService$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, service_host = resource$service_host,
				service_port = resource$service_port, service_ssl_flag = resource$service_ssl_flag,
				service_type = resource$service_type, created_at = resource$created_at,
				updated_at = resource$updated_at, read_only = resource$read_only)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		pipeline_templates.get = function(uuid)
		{
			endPoint <- stringr::str_interp("pipeline_templates/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- NULL
			
			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- PipelineTemplate$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				created_at = resource$created_at, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, name = resource$name,
				components = resource$components, updated_at = resource$updated_at,
				description = resource$description)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		pipeline_templates.index = function(filters = NULL,
			where = NULL, order = NULL, select = NULL,
			distinct = NULL, limit = "100", offset = "0",
			count = "exact")
		{
			endPoint <- stringr::str_interp("pipeline_templates")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(filters = filters, where = where,
				order = order, select = select, distinct = distinct,
				limit = limit, offset = offset, count = count)
			body <- NULL
			
			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- PipelineTemplateList$new(kind = resource$kind,
				etag = resource$etag, items = resource$items,
				next_link = resource$next_link, next_page_token = resource$next_page_token,
				selfLink = resource$selfLink)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		pipeline_templates.create = function(pipeline_template,
			ensure_unique_name = "false")
		{
			endPoint <- stringr::str_interp("pipeline_templates")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(ensure_unique_name = ensure_unique_name)
			body <- pipeline_template$toJSON()
			
			response <- private$REST$http$exec("POST", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- PipelineTemplate$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				created_at = resource$created_at, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, name = resource$name,
				components = resource$components, updated_at = resource$updated_at,
				description = resource$description)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		pipeline_templates.update = function(pipeline_template, uuid)
		{
			endPoint <- stringr::str_interp("pipeline_templates/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- pipeline_template$toJSON()
			
			response <- private$REST$http$exec("PUT", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- PipelineTemplate$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				created_at = resource$created_at, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, name = resource$name,
				components = resource$components, updated_at = resource$updated_at,
				description = resource$description)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		pipeline_templates.delete = function(uuid)
		{
			endPoint <- stringr::str_interp("pipeline_templates/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- NULL
			
			response <- private$REST$http$exec("DELETE", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- PipelineTemplate$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				created_at = resource$created_at, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, name = resource$name,
				components = resource$components, updated_at = resource$updated_at,
				description = resource$description)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		pipeline_templates.list = function(filters = NULL,
			where = NULL, order = NULL, select = NULL,
			distinct = NULL, limit = "100", offset = "0",
			count = "exact")
		{
			endPoint <- stringr::str_interp("pipeline_templates")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(filters = filters, where = where,
				order = order, select = select, distinct = distinct,
				limit = limit, offset = offset, count = count)
			body <- NULL
			
			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- PipelineTemplateList$new(kind = resource$kind,
				etag = resource$etag, items = resource$items,
				next_link = resource$next_link, next_page_token = resource$next_page_token,
				selfLink = resource$selfLink)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		pipeline_templates.show = function(uuid)
		{
			endPoint <- stringr::str_interp("pipeline_templates/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- NULL
			
			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- PipelineTemplate$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				created_at = resource$created_at, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, name = resource$name,
				components = resource$components, updated_at = resource$updated_at,
				description = resource$description)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		pipeline_templates.destroy = function(uuid)
		{
			endPoint <- stringr::str_interp("pipeline_templates/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- NULL
			
			response <- private$REST$http$exec("DELETE", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- PipelineTemplate$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				created_at = resource$created_at, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, name = resource$name,
				components = resource$components, updated_at = resource$updated_at,
				description = resource$description)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		pipeline_instances.get = function(uuid)
		{
			endPoint <- stringr::str_interp("pipeline_instances/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- NULL
			
			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- PipelineInstance$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				created_at = resource$created_at, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, pipeline_template_uuid = resource$pipeline_template_uuid,
				name = resource$name, components = resource$components,
				updated_at = resource$updated_at, properties = resource$properties,
				state = resource$state, components_summary = resource$components_summary,
				started_at = resource$started_at, finished_at = resource$finished_at,
				description = resource$description)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		pipeline_instances.index = function(filters = NULL,
			where = NULL, order = NULL, select = NULL,
			distinct = NULL, limit = "100", offset = "0",
			count = "exact")
		{
			endPoint <- stringr::str_interp("pipeline_instances")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(filters = filters, where = where,
				order = order, select = select, distinct = distinct,
				limit = limit, offset = offset, count = count)
			body <- NULL
			
			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- PipelineInstanceList$new(kind = resource$kind,
				etag = resource$etag, items = resource$items,
				next_link = resource$next_link, next_page_token = resource$next_page_token,
				selfLink = resource$selfLink)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		pipeline_instances.create = function(pipeline_instance,
			ensure_unique_name = "false")
		{
			endPoint <- stringr::str_interp("pipeline_instances")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(ensure_unique_name = ensure_unique_name)
			body <- pipeline_instance$toJSON()
			
			response <- private$REST$http$exec("POST", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- PipelineInstance$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				created_at = resource$created_at, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, pipeline_template_uuid = resource$pipeline_template_uuid,
				name = resource$name, components = resource$components,
				updated_at = resource$updated_at, properties = resource$properties,
				state = resource$state, components_summary = resource$components_summary,
				started_at = resource$started_at, finished_at = resource$finished_at,
				description = resource$description)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		pipeline_instances.update = function(pipeline_instance, uuid)
		{
			endPoint <- stringr::str_interp("pipeline_instances/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- pipeline_instance$toJSON()
			
			response <- private$REST$http$exec("PUT", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- PipelineInstance$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				created_at = resource$created_at, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, pipeline_template_uuid = resource$pipeline_template_uuid,
				name = resource$name, components = resource$components,
				updated_at = resource$updated_at, properties = resource$properties,
				state = resource$state, components_summary = resource$components_summary,
				started_at = resource$started_at, finished_at = resource$finished_at,
				description = resource$description)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		pipeline_instances.delete = function(uuid)
		{
			endPoint <- stringr::str_interp("pipeline_instances/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- NULL
			
			response <- private$REST$http$exec("DELETE", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- PipelineInstance$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				created_at = resource$created_at, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, pipeline_template_uuid = resource$pipeline_template_uuid,
				name = resource$name, components = resource$components,
				updated_at = resource$updated_at, properties = resource$properties,
				state = resource$state, components_summary = resource$components_summary,
				started_at = resource$started_at, finished_at = resource$finished_at,
				description = resource$description)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		pipeline_instances.cancel = function(uuid)
		{
			endPoint <- stringr::str_interp("pipeline_instances/${uuid}/cancel")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- NULL
			
			response <- private$REST$http$exec("POST", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- PipelineInstance$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				created_at = resource$created_at, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, pipeline_template_uuid = resource$pipeline_template_uuid,
				name = resource$name, components = resource$components,
				updated_at = resource$updated_at, properties = resource$properties,
				state = resource$state, components_summary = resource$components_summary,
				started_at = resource$started_at, finished_at = resource$finished_at,
				description = resource$description)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		pipeline_instances.list = function(filters = NULL,
			where = NULL, order = NULL, select = NULL,
			distinct = NULL, limit = "100", offset = "0",
			count = "exact")
		{
			endPoint <- stringr::str_interp("pipeline_instances")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(filters = filters, where = where,
				order = order, select = select, distinct = distinct,
				limit = limit, offset = offset, count = count)
			body <- NULL
			
			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- PipelineInstanceList$new(kind = resource$kind,
				etag = resource$etag, items = resource$items,
				next_link = resource$next_link, next_page_token = resource$next_page_token,
				selfLink = resource$selfLink)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		pipeline_instances.show = function(uuid)
		{
			endPoint <- stringr::str_interp("pipeline_instances/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- NULL
			
			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- PipelineInstance$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				created_at = resource$created_at, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, pipeline_template_uuid = resource$pipeline_template_uuid,
				name = resource$name, components = resource$components,
				updated_at = resource$updated_at, properties = resource$properties,
				state = resource$state, components_summary = resource$components_summary,
				started_at = resource$started_at, finished_at = resource$finished_at,
				description = resource$description)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		pipeline_instances.destroy = function(uuid)
		{
			endPoint <- stringr::str_interp("pipeline_instances/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- NULL
			
			response <- private$REST$http$exec("DELETE", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- PipelineInstance$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				created_at = resource$created_at, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, pipeline_template_uuid = resource$pipeline_template_uuid,
				name = resource$name, components = resource$components,
				updated_at = resource$updated_at, properties = resource$properties,
				state = resource$state, components_summary = resource$components_summary,
				started_at = resource$started_at, finished_at = resource$finished_at,
				description = resource$description)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		nodes.get = function(uuid)
		{
			endPoint <- stringr::str_interp("nodes/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- NULL
			
			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- Node$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				created_at = resource$created_at, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, slot_number = resource$slot_number,
				hostname = resource$hostname, domain = resource$domain,
				ip_address = resource$ip_address, first_ping_at = resource$first_ping_at,
				last_ping_at = resource$last_ping_at, info = resource$info,
				updated_at = resource$updated_at, properties = resource$properties,
				job_uuid = resource$job_uuid)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		nodes.index = function(filters = NULL, where = NULL,
			order = NULL, select = NULL, distinct = NULL,
			limit = "100", offset = "0", count = "exact")
		{
			endPoint <- stringr::str_interp("nodes")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(filters = filters, where = where,
				order = order, select = select, distinct = distinct,
				limit = limit, offset = offset, count = count)
			body <- NULL
			
			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- NodeList$new(kind = resource$kind,
				etag = resource$etag, items = resource$items,
				next_link = resource$next_link, next_page_token = resource$next_page_token,
				selfLink = resource$selfLink)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		nodes.create = function(node, ensure_unique_name = "false",
			assign_slot = NULL)
		{
			endPoint <- stringr::str_interp("nodes")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(ensure_unique_name = ensure_unique_name,
				assign_slot = assign_slot)
			body <- node$toJSON()
			
			response <- private$REST$http$exec("POST", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- Node$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				created_at = resource$created_at, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, slot_number = resource$slot_number,
				hostname = resource$hostname, domain = resource$domain,
				ip_address = resource$ip_address, first_ping_at = resource$first_ping_at,
				last_ping_at = resource$last_ping_at, info = resource$info,
				updated_at = resource$updated_at, properties = resource$properties,
				job_uuid = resource$job_uuid)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		nodes.update = function(node, uuid, assign_slot = NULL)
		{
			endPoint <- stringr::str_interp("nodes/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid, assign_slot = assign_slot)
			body <- node$toJSON()
			
			response <- private$REST$http$exec("PUT", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- Node$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				created_at = resource$created_at, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, slot_number = resource$slot_number,
				hostname = resource$hostname, domain = resource$domain,
				ip_address = resource$ip_address, first_ping_at = resource$first_ping_at,
				last_ping_at = resource$last_ping_at, info = resource$info,
				updated_at = resource$updated_at, properties = resource$properties,
				job_uuid = resource$job_uuid)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		nodes.delete = function(uuid)
		{
			endPoint <- stringr::str_interp("nodes/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- NULL
			
			response <- private$REST$http$exec("DELETE", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- Node$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				created_at = resource$created_at, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, slot_number = resource$slot_number,
				hostname = resource$hostname, domain = resource$domain,
				ip_address = resource$ip_address, first_ping_at = resource$first_ping_at,
				last_ping_at = resource$last_ping_at, info = resource$info,
				updated_at = resource$updated_at, properties = resource$properties,
				job_uuid = resource$job_uuid)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		nodes.ping = function(uuid, ping_secret)
		{
			endPoint <- stringr::str_interp("nodes/${uuid}/ping")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid, ping_secret = ping_secret)
			body <- NULL
			
			response <- private$REST$http$exec("POST", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- Node$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				created_at = resource$created_at, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, slot_number = resource$slot_number,
				hostname = resource$hostname, domain = resource$domain,
				ip_address = resource$ip_address, first_ping_at = resource$first_ping_at,
				last_ping_at = resource$last_ping_at, info = resource$info,
				updated_at = resource$updated_at, properties = resource$properties,
				job_uuid = resource$job_uuid)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		nodes.list = function(filters = NULL, where = NULL,
			order = NULL, select = NULL, distinct = NULL,
			limit = "100", offset = "0", count = "exact")
		{
			endPoint <- stringr::str_interp("nodes")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(filters = filters, where = where,
				order = order, select = select, distinct = distinct,
				limit = limit, offset = offset, count = count)
			body <- NULL
			
			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- NodeList$new(kind = resource$kind,
				etag = resource$etag, items = resource$items,
				next_link = resource$next_link, next_page_token = resource$next_page_token,
				selfLink = resource$selfLink)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		nodes.show = function(uuid)
		{
			endPoint <- stringr::str_interp("nodes/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- NULL
			
			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- Node$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				created_at = resource$created_at, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, slot_number = resource$slot_number,
				hostname = resource$hostname, domain = resource$domain,
				ip_address = resource$ip_address, first_ping_at = resource$first_ping_at,
				last_ping_at = resource$last_ping_at, info = resource$info,
				updated_at = resource$updated_at, properties = resource$properties,
				job_uuid = resource$job_uuid)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		nodes.destroy = function(uuid)
		{
			endPoint <- stringr::str_interp("nodes/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- NULL
			
			response <- private$REST$http$exec("DELETE", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- Node$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				created_at = resource$created_at, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, slot_number = resource$slot_number,
				hostname = resource$hostname, domain = resource$domain,
				ip_address = resource$ip_address, first_ping_at = resource$first_ping_at,
				last_ping_at = resource$last_ping_at, info = resource$info,
				updated_at = resource$updated_at, properties = resource$properties,
				job_uuid = resource$job_uuid)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		repositories.get = function(uuid)
		{
			endPoint <- stringr::str_interp("repositories/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- NULL
			
			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- Repository$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, name = resource$name,
				created_at = resource$created_at, updated_at = resource$updated_at)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		repositories.index = function(filters = NULL,
			where = NULL, order = NULL, select = NULL,
			distinct = NULL, limit = "100", offset = "0",
			count = "exact")
		{
			endPoint <- stringr::str_interp("repositories")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(filters = filters, where = where,
				order = order, select = select, distinct = distinct,
				limit = limit, offset = offset, count = count)
			body <- NULL
			
			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- RepositoryList$new(kind = resource$kind,
				etag = resource$etag, items = resource$items,
				next_link = resource$next_link, next_page_token = resource$next_page_token,
				selfLink = resource$selfLink)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		repositories.create = function(repository, ensure_unique_name = "false")
		{
			endPoint <- stringr::str_interp("repositories")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(ensure_unique_name = ensure_unique_name)
			body <- repository$toJSON()
			
			response <- private$REST$http$exec("POST", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- Repository$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, name = resource$name,
				created_at = resource$created_at, updated_at = resource$updated_at)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		repositories.update = function(repository, uuid)
		{
			endPoint <- stringr::str_interp("repositories/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- repository$toJSON()
			
			response <- private$REST$http$exec("PUT", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- Repository$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, name = resource$name,
				created_at = resource$created_at, updated_at = resource$updated_at)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		repositories.delete = function(uuid)
		{
			endPoint <- stringr::str_interp("repositories/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- NULL
			
			response <- private$REST$http$exec("DELETE", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- Repository$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, name = resource$name,
				created_at = resource$created_at, updated_at = resource$updated_at)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		repositories.get_all_permissions = function()
		{
			endPoint <- stringr::str_interp("repositories/get_all_permissions")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- NULL
			body <- NULL
			
			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- Repository$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, name = resource$name,
				created_at = resource$created_at, updated_at = resource$updated_at)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		repositories.list = function(filters = NULL,
			where = NULL, order = NULL, select = NULL,
			distinct = NULL, limit = "100", offset = "0",
			count = "exact")
		{
			endPoint <- stringr::str_interp("repositories")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(filters = filters, where = where,
				order = order, select = select, distinct = distinct,
				limit = limit, offset = offset, count = count)
			body <- NULL
			
			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- RepositoryList$new(kind = resource$kind,
				etag = resource$etag, items = resource$items,
				next_link = resource$next_link, next_page_token = resource$next_page_token,
				selfLink = resource$selfLink)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		repositories.show = function(uuid)
		{
			endPoint <- stringr::str_interp("repositories/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- NULL
			
			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- Repository$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, name = resource$name,
				created_at = resource$created_at, updated_at = resource$updated_at)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		repositories.destroy = function(uuid)
		{
			endPoint <- stringr::str_interp("repositories/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- NULL
			
			response <- private$REST$http$exec("DELETE", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- Repository$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, name = resource$name,
				created_at = resource$created_at, updated_at = resource$updated_at)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		specimens.get = function(uuid)
		{
			endPoint <- stringr::str_interp("specimens/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- NULL
			
			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- Specimen$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				created_at = resource$created_at, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, material = resource$material,
				updated_at = resource$updated_at, properties = resource$properties)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		specimens.index = function(filters = NULL,
			where = NULL, order = NULL, select = NULL,
			distinct = NULL, limit = "100", offset = "0",
			count = "exact")
		{
			endPoint <- stringr::str_interp("specimens")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(filters = filters, where = where,
				order = order, select = select, distinct = distinct,
				limit = limit, offset = offset, count = count)
			body <- NULL
			
			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- SpecimenList$new(kind = resource$kind,
				etag = resource$etag, items = resource$items,
				next_link = resource$next_link, next_page_token = resource$next_page_token,
				selfLink = resource$selfLink)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		specimens.create = function(specimen, ensure_unique_name = "false")
		{
			endPoint <- stringr::str_interp("specimens")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(ensure_unique_name = ensure_unique_name)
			body <- specimen$toJSON()
			
			response <- private$REST$http$exec("POST", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- Specimen$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				created_at = resource$created_at, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, material = resource$material,
				updated_at = resource$updated_at, properties = resource$properties)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		specimens.update = function(specimen, uuid)
		{
			endPoint <- stringr::str_interp("specimens/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- specimen$toJSON()
			
			response <- private$REST$http$exec("PUT", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- Specimen$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				created_at = resource$created_at, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, material = resource$material,
				updated_at = resource$updated_at, properties = resource$properties)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		specimens.delete = function(uuid)
		{
			endPoint <- stringr::str_interp("specimens/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- NULL
			
			response <- private$REST$http$exec("DELETE", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- Specimen$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				created_at = resource$created_at, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, material = resource$material,
				updated_at = resource$updated_at, properties = resource$properties)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		specimens.list = function(filters = NULL,
			where = NULL, order = NULL, select = NULL,
			distinct = NULL, limit = "100", offset = "0",
			count = "exact")
		{
			endPoint <- stringr::str_interp("specimens")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(filters = filters, where = where,
				order = order, select = select, distinct = distinct,
				limit = limit, offset = offset, count = count)
			body <- NULL
			
			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- SpecimenList$new(kind = resource$kind,
				etag = resource$etag, items = resource$items,
				next_link = resource$next_link, next_page_token = resource$next_page_token,
				selfLink = resource$selfLink)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		specimens.show = function(uuid)
		{
			endPoint <- stringr::str_interp("specimens/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- NULL
			
			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- Specimen$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				created_at = resource$created_at, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, material = resource$material,
				updated_at = resource$updated_at, properties = resource$properties)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		specimens.destroy = function(uuid)
		{
			endPoint <- stringr::str_interp("specimens/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- NULL
			
			response <- private$REST$http$exec("DELETE", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- Specimen$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				created_at = resource$created_at, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, material = resource$material,
				updated_at = resource$updated_at, properties = resource$properties)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		logs.get = function(uuid)
		{
			endPoint <- stringr::str_interp("logs/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- NULL
			
			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- Log$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				object_uuid = resource$object_uuid, event_at = resource$event_at,
				event_type = resource$event_type, summary = resource$summary,
				properties = resource$properties, created_at = resource$created_at,
				updated_at = resource$updated_at, modified_at = resource$modified_at,
				object_owner_uuid = resource$object_owner_uuid)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		logs.index = function(filters = NULL, where = NULL,
			order = NULL, select = NULL, distinct = NULL,
			limit = "100", offset = "0", count = "exact")
		{
			endPoint <- stringr::str_interp("logs")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(filters = filters, where = where,
				order = order, select = select, distinct = distinct,
				limit = limit, offset = offset, count = count)
			body <- NULL
			
			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- LogList$new(kind = resource$kind,
				etag = resource$etag, items = resource$items,
				next_link = resource$next_link, next_page_token = resource$next_page_token,
				selfLink = resource$selfLink)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		logs.create = function(log, ensure_unique_name = "false")
		{
			endPoint <- stringr::str_interp("logs")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(ensure_unique_name = ensure_unique_name)
			body <- log$toJSON()
			
			response <- private$REST$http$exec("POST", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- Log$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				object_uuid = resource$object_uuid, event_at = resource$event_at,
				event_type = resource$event_type, summary = resource$summary,
				properties = resource$properties, created_at = resource$created_at,
				updated_at = resource$updated_at, modified_at = resource$modified_at,
				object_owner_uuid = resource$object_owner_uuid)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		logs.update = function(log, uuid)
		{
			endPoint <- stringr::str_interp("logs/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- log$toJSON()
			
			response <- private$REST$http$exec("PUT", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- Log$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				object_uuid = resource$object_uuid, event_at = resource$event_at,
				event_type = resource$event_type, summary = resource$summary,
				properties = resource$properties, created_at = resource$created_at,
				updated_at = resource$updated_at, modified_at = resource$modified_at,
				object_owner_uuid = resource$object_owner_uuid)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		logs.delete = function(uuid)
		{
			endPoint <- stringr::str_interp("logs/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- NULL
			
			response <- private$REST$http$exec("DELETE", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- Log$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				object_uuid = resource$object_uuid, event_at = resource$event_at,
				event_type = resource$event_type, summary = resource$summary,
				properties = resource$properties, created_at = resource$created_at,
				updated_at = resource$updated_at, modified_at = resource$modified_at,
				object_owner_uuid = resource$object_owner_uuid)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		logs.list = function(filters = NULL, where = NULL,
			order = NULL, select = NULL, distinct = NULL,
			limit = "100", offset = "0", count = "exact")
		{
			endPoint <- stringr::str_interp("logs")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(filters = filters, where = where,
				order = order, select = select, distinct = distinct,
				limit = limit, offset = offset, count = count)
			body <- NULL
			
			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- LogList$new(kind = resource$kind,
				etag = resource$etag, items = resource$items,
				next_link = resource$next_link, next_page_token = resource$next_page_token,
				selfLink = resource$selfLink)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		logs.show = function(uuid)
		{
			endPoint <- stringr::str_interp("logs/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- NULL
			
			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- Log$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				object_uuid = resource$object_uuid, event_at = resource$event_at,
				event_type = resource$event_type, summary = resource$summary,
				properties = resource$properties, created_at = resource$created_at,
				updated_at = resource$updated_at, modified_at = resource$modified_at,
				object_owner_uuid = resource$object_owner_uuid)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		logs.destroy = function(uuid)
		{
			endPoint <- stringr::str_interp("logs/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- NULL
			
			response <- private$REST$http$exec("DELETE", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- Log$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				object_uuid = resource$object_uuid, event_at = resource$event_at,
				event_type = resource$event_type, summary = resource$summary,
				properties = resource$properties, created_at = resource$created_at,
				updated_at = resource$updated_at, modified_at = resource$modified_at,
				object_owner_uuid = resource$object_owner_uuid)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		traits.get = function(uuid)
		{
			endPoint <- stringr::str_interp("traits/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- NULL
			
			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- Trait$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, name = resource$name,
				properties = resource$properties, created_at = resource$created_at,
				updated_at = resource$updated_at)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		traits.index = function(filters = NULL, where = NULL,
			order = NULL, select = NULL, distinct = NULL,
			limit = "100", offset = "0", count = "exact")
		{
			endPoint <- stringr::str_interp("traits")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(filters = filters, where = where,
				order = order, select = select, distinct = distinct,
				limit = limit, offset = offset, count = count)
			body <- NULL
			
			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- TraitList$new(kind = resource$kind,
				etag = resource$etag, items = resource$items,
				next_link = resource$next_link, next_page_token = resource$next_page_token,
				selfLink = resource$selfLink)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		traits.create = function(trait, ensure_unique_name = "false")
		{
			endPoint <- stringr::str_interp("traits")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(ensure_unique_name = ensure_unique_name)
			body <- trait$toJSON()
			
			response <- private$REST$http$exec("POST", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- Trait$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, name = resource$name,
				properties = resource$properties, created_at = resource$created_at,
				updated_at = resource$updated_at)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		traits.update = function(trait, uuid)
		{
			endPoint <- stringr::str_interp("traits/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- trait$toJSON()
			
			response <- private$REST$http$exec("PUT", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- Trait$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, name = resource$name,
				properties = resource$properties, created_at = resource$created_at,
				updated_at = resource$updated_at)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		traits.delete = function(uuid)
		{
			endPoint <- stringr::str_interp("traits/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- NULL
			
			response <- private$REST$http$exec("DELETE", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- Trait$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, name = resource$name,
				properties = resource$properties, created_at = resource$created_at,
				updated_at = resource$updated_at)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		traits.list = function(filters = NULL, where = NULL,
			order = NULL, select = NULL, distinct = NULL,
			limit = "100", offset = "0", count = "exact")
		{
			endPoint <- stringr::str_interp("traits")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(filters = filters, where = where,
				order = order, select = select, distinct = distinct,
				limit = limit, offset = offset, count = count)
			body <- NULL
			
			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- TraitList$new(kind = resource$kind,
				etag = resource$etag, items = resource$items,
				next_link = resource$next_link, next_page_token = resource$next_page_token,
				selfLink = resource$selfLink)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		traits.show = function(uuid)
		{
			endPoint <- stringr::str_interp("traits/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- NULL
			
			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- Trait$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, name = resource$name,
				properties = resource$properties, created_at = resource$created_at,
				updated_at = resource$updated_at)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		traits.destroy = function(uuid)
		{
			endPoint <- stringr::str_interp("traits/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- NULL
			
			response <- private$REST$http$exec("DELETE", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- Trait$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, name = resource$name,
				properties = resource$properties, created_at = resource$created_at,
				updated_at = resource$updated_at)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		virtual_machines.get = function(uuid)
		{
			endPoint <- stringr::str_interp("virtual_machines/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- NULL
			
			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- VirtualMachine$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, hostname = resource$hostname,
				created_at = resource$created_at, updated_at = resource$updated_at)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		virtual_machines.index = function(filters = NULL,
			where = NULL, order = NULL, select = NULL,
			distinct = NULL, limit = "100", offset = "0",
			count = "exact")
		{
			endPoint <- stringr::str_interp("virtual_machines")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(filters = filters, where = where,
				order = order, select = select, distinct = distinct,
				limit = limit, offset = offset, count = count)
			body <- NULL
			
			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- VirtualMachineList$new(kind = resource$kind,
				etag = resource$etag, items = resource$items,
				next_link = resource$next_link, next_page_token = resource$next_page_token,
				selfLink = resource$selfLink)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		virtual_machines.create = function(virtual_machine,
			ensure_unique_name = "false")
		{
			endPoint <- stringr::str_interp("virtual_machines")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(ensure_unique_name = ensure_unique_name)
			body <- virtual_machine$toJSON()
			
			response <- private$REST$http$exec("POST", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- VirtualMachine$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, hostname = resource$hostname,
				created_at = resource$created_at, updated_at = resource$updated_at)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		virtual_machines.update = function(virtual_machine, uuid)
		{
			endPoint <- stringr::str_interp("virtual_machines/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- virtual_machine$toJSON()
			
			response <- private$REST$http$exec("PUT", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- VirtualMachine$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, hostname = resource$hostname,
				created_at = resource$created_at, updated_at = resource$updated_at)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		virtual_machines.delete = function(uuid)
		{
			endPoint <- stringr::str_interp("virtual_machines/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- NULL
			
			response <- private$REST$http$exec("DELETE", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- VirtualMachine$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, hostname = resource$hostname,
				created_at = resource$created_at, updated_at = resource$updated_at)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		virtual_machines.logins = function(uuid)
		{
			endPoint <- stringr::str_interp("virtual_machines/${uuid}/logins")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- NULL
			
			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- VirtualMachine$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, hostname = resource$hostname,
				created_at = resource$created_at, updated_at = resource$updated_at)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		virtual_machines.get_all_logins = function()
		{
			endPoint <- stringr::str_interp("virtual_machines/get_all_logins")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- NULL
			body <- NULL
			
			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- VirtualMachine$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, hostname = resource$hostname,
				created_at = resource$created_at, updated_at = resource$updated_at)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		virtual_machines.list = function(filters = NULL,
			where = NULL, order = NULL, select = NULL,
			distinct = NULL, limit = "100", offset = "0",
			count = "exact")
		{
			endPoint <- stringr::str_interp("virtual_machines")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(filters = filters, where = where,
				order = order, select = select, distinct = distinct,
				limit = limit, offset = offset, count = count)
			body <- NULL
			
			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- VirtualMachineList$new(kind = resource$kind,
				etag = resource$etag, items = resource$items,
				next_link = resource$next_link, next_page_token = resource$next_page_token,
				selfLink = resource$selfLink)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		virtual_machines.show = function(uuid)
		{
			endPoint <- stringr::str_interp("virtual_machines/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- NULL
			
			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- VirtualMachine$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, hostname = resource$hostname,
				created_at = resource$created_at, updated_at = resource$updated_at)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		virtual_machines.destroy = function(uuid)
		{
			endPoint <- stringr::str_interp("virtual_machines/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- NULL
			
			response <- private$REST$http$exec("DELETE", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- VirtualMachine$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, hostname = resource$hostname,
				created_at = resource$created_at, updated_at = resource$updated_at)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		workflows.get = function(uuid)
		{
			endPoint <- stringr::str_interp("workflows/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- NULL
			
			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- Workflow$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				created_at = resource$created_at, modified_at = resource$modified_at,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				name = resource$name, description = resource$description,
				definition = resource$definition, updated_at = resource$updated_at)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		workflows.index = function(filters = NULL,
			where = NULL, order = NULL, select = NULL,
			distinct = NULL, limit = "100", offset = "0",
			count = "exact")
		{
			endPoint <- stringr::str_interp("workflows")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(filters = filters, where = where,
				order = order, select = select, distinct = distinct,
				limit = limit, offset = offset, count = count)
			body <- NULL
			
			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- WorkflowList$new(kind = resource$kind,
				etag = resource$etag, items = resource$items,
				next_link = resource$next_link, next_page_token = resource$next_page_token,
				selfLink = resource$selfLink)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		workflows.create = function(workflow, ensure_unique_name = "false")
		{
			endPoint <- stringr::str_interp("workflows")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(ensure_unique_name = ensure_unique_name)
			body <- workflow$toJSON()
			
			response <- private$REST$http$exec("POST", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- Workflow$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				created_at = resource$created_at, modified_at = resource$modified_at,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				name = resource$name, description = resource$description,
				definition = resource$definition, updated_at = resource$updated_at)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		workflows.update = function(workflow, uuid)
		{
			endPoint <- stringr::str_interp("workflows/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- workflow$toJSON()
			
			response <- private$REST$http$exec("PUT", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- Workflow$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				created_at = resource$created_at, modified_at = resource$modified_at,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				name = resource$name, description = resource$description,
				definition = resource$definition, updated_at = resource$updated_at)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		workflows.delete = function(uuid)
		{
			endPoint <- stringr::str_interp("workflows/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- NULL
			
			response <- private$REST$http$exec("DELETE", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- Workflow$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				created_at = resource$created_at, modified_at = resource$modified_at,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				name = resource$name, description = resource$description,
				definition = resource$definition, updated_at = resource$updated_at)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		workflows.list = function(filters = NULL,
			where = NULL, order = NULL, select = NULL,
			distinct = NULL, limit = "100", offset = "0",
			count = "exact")
		{
			endPoint <- stringr::str_interp("workflows")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(filters = filters, where = where,
				order = order, select = select, distinct = distinct,
				limit = limit, offset = offset, count = count)
			body <- NULL
			
			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- WorkflowList$new(kind = resource$kind,
				etag = resource$etag, items = resource$items,
				next_link = resource$next_link, next_page_token = resource$next_page_token,
				selfLink = resource$selfLink)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		workflows.show = function(uuid)
		{
			endPoint <- stringr::str_interp("workflows/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- NULL
			
			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- Workflow$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				created_at = resource$created_at, modified_at = resource$modified_at,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				name = resource$name, description = resource$description,
				definition = resource$definition, updated_at = resource$updated_at)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		workflows.destroy = function(uuid)
		{
			endPoint <- stringr::str_interp("workflows/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- NULL
			
			response <- private$REST$http$exec("DELETE", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- Workflow$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				created_at = resource$created_at, modified_at = resource$modified_at,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				name = resource$name, description = resource$description,
				definition = resource$definition, updated_at = resource$updated_at)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		groups.get = function(uuid)
		{
			endPoint <- stringr::str_interp("groups/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- NULL
			
			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- Group$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				created_at = resource$created_at, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, name = resource$name,
				description = resource$description, updated_at = resource$updated_at,
				group_class = resource$group_class, trash_at = resource$trash_at,
				is_trashed = resource$is_trashed, delete_at = resource$delete_at)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		groups.index = function(filters = NULL, where = NULL,
			order = NULL, select = NULL, distinct = NULL,
			limit = "100", offset = "0", count = "exact",
			include_trash = NULL)
		{
			endPoint <- stringr::str_interp("groups")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(filters = filters, where = where,
				order = order, select = select, distinct = distinct,
				limit = limit, offset = offset, count = count,
				include_trash = include_trash)
			body <- NULL
			
			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- GroupList$new(kind = resource$kind,
				etag = resource$etag, items = resource$items,
				next_link = resource$next_link, next_page_token = resource$next_page_token,
				selfLink = resource$selfLink)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		groups.create = function(group, ensure_unique_name = "false")
		{
			endPoint <- stringr::str_interp("groups")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(ensure_unique_name = ensure_unique_name)
			body <- group$toJSON()
			
			response <- private$REST$http$exec("POST", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- Group$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				created_at = resource$created_at, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, name = resource$name,
				description = resource$description, updated_at = resource$updated_at,
				group_class = resource$group_class, trash_at = resource$trash_at,
				is_trashed = resource$is_trashed, delete_at = resource$delete_at)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		groups.update = function(group, uuid)
		{
			endPoint <- stringr::str_interp("groups/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- group$toJSON()
			
			response <- private$REST$http$exec("PUT", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- Group$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				created_at = resource$created_at, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, name = resource$name,
				description = resource$description, updated_at = resource$updated_at,
				group_class = resource$group_class, trash_at = resource$trash_at,
				is_trashed = resource$is_trashed, delete_at = resource$delete_at)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		groups.delete = function(uuid)
		{
			endPoint <- stringr::str_interp("groups/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- NULL
			
			response <- private$REST$http$exec("DELETE", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- Group$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				created_at = resource$created_at, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, name = resource$name,
				description = resource$description, updated_at = resource$updated_at,
				group_class = resource$group_class, trash_at = resource$trash_at,
				is_trashed = resource$is_trashed, delete_at = resource$delete_at)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		groups.contents = function(filters = NULL,
			where = NULL, order = NULL, distinct = NULL,
			limit = "100", offset = "0", count = "exact",
			include_trash = NULL, uuid = NULL, recursive = NULL)
		{
			endPoint <- stringr::str_interp("groups/contents")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(filters = filters, where = where,
				order = order, distinct = distinct, limit = limit,
				offset = offset, count = count, include_trash = include_trash,
				uuid = uuid, recursive = recursive)
			body <- NULL
			
			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- Group$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				created_at = resource$created_at, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, name = resource$name,
				description = resource$description, updated_at = resource$updated_at,
				group_class = resource$group_class, trash_at = resource$trash_at,
				is_trashed = resource$is_trashed, delete_at = resource$delete_at)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		groups.trash = function(uuid)
		{
			endPoint <- stringr::str_interp("groups/${uuid}/trash")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- NULL
			
			response <- private$REST$http$exec("POST", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- Group$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				created_at = resource$created_at, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, name = resource$name,
				description = resource$description, updated_at = resource$updated_at,
				group_class = resource$group_class, trash_at = resource$trash_at,
				is_trashed = resource$is_trashed, delete_at = resource$delete_at)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		groups.untrash = function(uuid)
		{
			endPoint <- stringr::str_interp("groups/${uuid}/untrash")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- NULL
			
			response <- private$REST$http$exec("POST", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- Group$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				created_at = resource$created_at, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, name = resource$name,
				description = resource$description, updated_at = resource$updated_at,
				group_class = resource$group_class, trash_at = resource$trash_at,
				is_trashed = resource$is_trashed, delete_at = resource$delete_at)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		groups.list = function(filters = NULL, where = NULL,
			order = NULL, select = NULL, distinct = NULL,
			limit = "100", offset = "0", count = "exact",
			include_trash = NULL)
		{
			endPoint <- stringr::str_interp("groups")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(filters = filters, where = where,
				order = order, select = select, distinct = distinct,
				limit = limit, offset = offset, count = count,
				include_trash = include_trash)
			body <- NULL
			
			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- GroupList$new(kind = resource$kind,
				etag = resource$etag, items = resource$items,
				next_link = resource$next_link, next_page_token = resource$next_page_token,
				selfLink = resource$selfLink)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		groups.show = function(uuid)
		{
			endPoint <- stringr::str_interp("groups/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- NULL
			
			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- Group$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				created_at = resource$created_at, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, name = resource$name,
				description = resource$description, updated_at = resource$updated_at,
				group_class = resource$group_class, trash_at = resource$trash_at,
				is_trashed = resource$is_trashed, delete_at = resource$delete_at)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		groups.destroy = function(uuid)
		{
			endPoint <- stringr::str_interp("groups/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- NULL
			
			response <- private$REST$http$exec("DELETE", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- Group$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				created_at = resource$created_at, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, name = resource$name,
				description = resource$description, updated_at = resource$updated_at,
				group_class = resource$group_class, trash_at = resource$trash_at,
				is_trashed = resource$is_trashed, delete_at = resource$delete_at)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		user_agreements.get = function(uuid)
		{
			endPoint <- stringr::str_interp("user_agreements/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- NULL
			
			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- UserAgreement$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				created_at = resource$created_at, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, portable_data_hash = resource$portable_data_hash,
				replication_desired = resource$replication_desired,
				replication_confirmed_at = resource$replication_confirmed_at,
				replication_confirmed = resource$replication_confirmed,
				updated_at = resource$updated_at, manifest_text = resource$manifest_text,
				name = resource$name, description = resource$description,
				properties = resource$properties, delete_at = resource$delete_at,
				file_names = resource$file_names, trash_at = resource$trash_at,
				is_trashed = resource$is_trashed, storage_classes_desired = resource$storage_classes_desired,
				storage_classes_confirmed = resource$storage_classes_confirmed,
				storage_classes_confirmed_at = resource$storage_classes_confirmed_at)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		user_agreements.index = function(filters = NULL,
			where = NULL, order = NULL, select = NULL,
			distinct = NULL, limit = "100", offset = "0",
			count = "exact")
		{
			endPoint <- stringr::str_interp("user_agreements")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(filters = filters, where = where,
				order = order, select = select, distinct = distinct,
				limit = limit, offset = offset, count = count)
			body <- NULL
			
			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- UserAgreementList$new(kind = resource$kind,
				etag = resource$etag, items = resource$items,
				next_link = resource$next_link, next_page_token = resource$next_page_token,
				selfLink = resource$selfLink)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		user_agreements.create = function(user_agreement,
			ensure_unique_name = "false")
		{
			endPoint <- stringr::str_interp("user_agreements")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(ensure_unique_name = ensure_unique_name)
			body <- user_agreement$toJSON()
			
			response <- private$REST$http$exec("POST", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- UserAgreement$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				created_at = resource$created_at, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, portable_data_hash = resource$portable_data_hash,
				replication_desired = resource$replication_desired,
				replication_confirmed_at = resource$replication_confirmed_at,
				replication_confirmed = resource$replication_confirmed,
				updated_at = resource$updated_at, manifest_text = resource$manifest_text,
				name = resource$name, description = resource$description,
				properties = resource$properties, delete_at = resource$delete_at,
				file_names = resource$file_names, trash_at = resource$trash_at,
				is_trashed = resource$is_trashed, storage_classes_desired = resource$storage_classes_desired,
				storage_classes_confirmed = resource$storage_classes_confirmed,
				storage_classes_confirmed_at = resource$storage_classes_confirmed_at)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		user_agreements.update = function(user_agreement, uuid)
		{
			endPoint <- stringr::str_interp("user_agreements/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- user_agreement$toJSON()
			
			response <- private$REST$http$exec("PUT", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- UserAgreement$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				created_at = resource$created_at, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, portable_data_hash = resource$portable_data_hash,
				replication_desired = resource$replication_desired,
				replication_confirmed_at = resource$replication_confirmed_at,
				replication_confirmed = resource$replication_confirmed,
				updated_at = resource$updated_at, manifest_text = resource$manifest_text,
				name = resource$name, description = resource$description,
				properties = resource$properties, delete_at = resource$delete_at,
				file_names = resource$file_names, trash_at = resource$trash_at,
				is_trashed = resource$is_trashed, storage_classes_desired = resource$storage_classes_desired,
				storage_classes_confirmed = resource$storage_classes_confirmed,
				storage_classes_confirmed_at = resource$storage_classes_confirmed_at)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		user_agreements.delete = function(uuid)
		{
			endPoint <- stringr::str_interp("user_agreements/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- NULL
			
			response <- private$REST$http$exec("DELETE", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- UserAgreement$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				created_at = resource$created_at, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, portable_data_hash = resource$portable_data_hash,
				replication_desired = resource$replication_desired,
				replication_confirmed_at = resource$replication_confirmed_at,
				replication_confirmed = resource$replication_confirmed,
				updated_at = resource$updated_at, manifest_text = resource$manifest_text,
				name = resource$name, description = resource$description,
				properties = resource$properties, delete_at = resource$delete_at,
				file_names = resource$file_names, trash_at = resource$trash_at,
				is_trashed = resource$is_trashed, storage_classes_desired = resource$storage_classes_desired,
				storage_classes_confirmed = resource$storage_classes_confirmed,
				storage_classes_confirmed_at = resource$storage_classes_confirmed_at)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		user_agreements.signatures = function()
		{
			endPoint <- stringr::str_interp("user_agreements/signatures")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- NULL
			body <- NULL
			
			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- UserAgreement$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				created_at = resource$created_at, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, portable_data_hash = resource$portable_data_hash,
				replication_desired = resource$replication_desired,
				replication_confirmed_at = resource$replication_confirmed_at,
				replication_confirmed = resource$replication_confirmed,
				updated_at = resource$updated_at, manifest_text = resource$manifest_text,
				name = resource$name, description = resource$description,
				properties = resource$properties, delete_at = resource$delete_at,
				file_names = resource$file_names, trash_at = resource$trash_at,
				is_trashed = resource$is_trashed, storage_classes_desired = resource$storage_classes_desired,
				storage_classes_confirmed = resource$storage_classes_confirmed,
				storage_classes_confirmed_at = resource$storage_classes_confirmed_at)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		user_agreements.sign = function()
		{
			endPoint <- stringr::str_interp("user_agreements/sign")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- NULL
			body <- NULL
			
			response <- private$REST$http$exec("POST", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- UserAgreement$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				created_at = resource$created_at, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, portable_data_hash = resource$portable_data_hash,
				replication_desired = resource$replication_desired,
				replication_confirmed_at = resource$replication_confirmed_at,
				replication_confirmed = resource$replication_confirmed,
				updated_at = resource$updated_at, manifest_text = resource$manifest_text,
				name = resource$name, description = resource$description,
				properties = resource$properties, delete_at = resource$delete_at,
				file_names = resource$file_names, trash_at = resource$trash_at,
				is_trashed = resource$is_trashed, storage_classes_desired = resource$storage_classes_desired,
				storage_classes_confirmed = resource$storage_classes_confirmed,
				storage_classes_confirmed_at = resource$storage_classes_confirmed_at)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		user_agreements.list = function(filters = NULL,
			where = NULL, order = NULL, select = NULL,
			distinct = NULL, limit = "100", offset = "0",
			count = "exact")
		{
			endPoint <- stringr::str_interp("user_agreements")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(filters = filters, where = where,
				order = order, select = select, distinct = distinct,
				limit = limit, offset = offset, count = count)
			body <- NULL
			
			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- UserAgreementList$new(kind = resource$kind,
				etag = resource$etag, items = resource$items,
				next_link = resource$next_link, next_page_token = resource$next_page_token,
				selfLink = resource$selfLink)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		user_agreements.new = function()
		{
			endPoint <- stringr::str_interp("user_agreements/new")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- NULL
			body <- NULL
			
			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- UserAgreement$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				created_at = resource$created_at, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, portable_data_hash = resource$portable_data_hash,
				replication_desired = resource$replication_desired,
				replication_confirmed_at = resource$replication_confirmed_at,
				replication_confirmed = resource$replication_confirmed,
				updated_at = resource$updated_at, manifest_text = resource$manifest_text,
				name = resource$name, description = resource$description,
				properties = resource$properties, delete_at = resource$delete_at,
				file_names = resource$file_names, trash_at = resource$trash_at,
				is_trashed = resource$is_trashed, storage_classes_desired = resource$storage_classes_desired,
				storage_classes_confirmed = resource$storage_classes_confirmed,
				storage_classes_confirmed_at = resource$storage_classes_confirmed_at)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		user_agreements.show = function(uuid)
		{
			endPoint <- stringr::str_interp("user_agreements/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- NULL
			
			response <- private$REST$http$exec("GET", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- UserAgreement$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				created_at = resource$created_at, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, portable_data_hash = resource$portable_data_hash,
				replication_desired = resource$replication_desired,
				replication_confirmed_at = resource$replication_confirmed_at,
				replication_confirmed = resource$replication_confirmed,
				updated_at = resource$updated_at, manifest_text = resource$manifest_text,
				name = resource$name, description = resource$description,
				properties = resource$properties, delete_at = resource$delete_at,
				file_names = resource$file_names, trash_at = resource$trash_at,
				is_trashed = resource$is_trashed, storage_classes_desired = resource$storage_classes_desired,
				storage_classes_confirmed = resource$storage_classes_confirmed,
				storage_classes_confirmed_at = resource$storage_classes_confirmed_at)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		user_agreements.destroy = function(uuid)
		{
			endPoint <- stringr::str_interp("user_agreements/${uuid}")
			url <- paste0(private$host, endPoint)
			headers <- list(Authorization = paste("OAuth2", private$token), 
			                "Content-Type" = "application/json")
			queryArgs <- list(uuid = uuid)
			body <- NULL
			
			response <- private$REST$http$exec("DELETE", url, headers, body,
			                                   queryArgs, private$numRetries)
			resource <- private$REST$httpParser$parseJSONResponse(response)
			
			if(!is.null(resource$errors))
				stop(resource$errors)
			
			result <- UserAgreement$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				created_at = resource$created_at, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, portable_data_hash = resource$portable_data_hash,
				replication_desired = resource$replication_desired,
				replication_confirmed_at = resource$replication_confirmed_at,
				replication_confirmed = resource$replication_confirmed,
				updated_at = resource$updated_at, manifest_text = resource$manifest_text,
				name = resource$name, description = resource$description,
				properties = resource$properties, delete_at = resource$delete_at,
				file_names = resource$file_names, trash_at = resource$trash_at,
				is_trashed = resource$is_trashed, storage_classes_desired = resource$storage_classes_desired,
				storage_classes_confirmed = resource$storage_classes_confirmed,
				storage_classes_confirmed_at = resource$storage_classes_confirmed_at)
			
			if(result$isEmpty())
				resource
			else
				result
		},

		getHostName = function() private$host,
		getToken = function() private$token,
		setRESTService = function(newREST) private$REST <- newREST
	),

	private = list(

		token = NULL,
		host = NULL,
		REST = NULL,
		numRetries = NULL
	),

	cloneable = FALSE
)
