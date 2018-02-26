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
			
			User$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, created_at = resource$created_at,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, email = resource$email,
				first_name = resource$first_name, last_name = resource$last_name,
				identity_url = resource$identity_url, is_admin = resource$is_admin,
				prefs = resource$prefs, updated_at = resource$updated_at,
				default_owner_uuid = resource$default_owner_uuid,
				is_active = resource$is_active, username = resource$username)
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
			
			UserList$new(kind = resource$kind, etag = resource$etag,
				items = resource$items, next_link = resource$next_link,
				next_page_token = resource$next_page_token,
				selfLink = resource$selfLink)
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
			
			User$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, created_at = resource$created_at,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, email = resource$email,
				first_name = resource$first_name, last_name = resource$last_name,
				identity_url = resource$identity_url, is_admin = resource$is_admin,
				prefs = resource$prefs, updated_at = resource$updated_at,
				default_owner_uuid = resource$default_owner_uuid,
				is_active = resource$is_active, username = resource$username)
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
			
			User$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, created_at = resource$created_at,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, email = resource$email,
				first_name = resource$first_name, last_name = resource$last_name,
				identity_url = resource$identity_url, is_admin = resource$is_admin,
				prefs = resource$prefs, updated_at = resource$updated_at,
				default_owner_uuid = resource$default_owner_uuid,
				is_active = resource$is_active, username = resource$username)
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
			
			User$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, created_at = resource$created_at,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, email = resource$email,
				first_name = resource$first_name, last_name = resource$last_name,
				identity_url = resource$identity_url, is_admin = resource$is_admin,
				prefs = resource$prefs, updated_at = resource$updated_at,
				default_owner_uuid = resource$default_owner_uuid,
				is_active = resource$is_active, username = resource$username)
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
			
			User$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, created_at = resource$created_at,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, email = resource$email,
				first_name = resource$first_name, last_name = resource$last_name,
				identity_url = resource$identity_url, is_admin = resource$is_admin,
				prefs = resource$prefs, updated_at = resource$updated_at,
				default_owner_uuid = resource$default_owner_uuid,
				is_active = resource$is_active, username = resource$username)
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
			
			User$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, created_at = resource$created_at,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, email = resource$email,
				first_name = resource$first_name, last_name = resource$last_name,
				identity_url = resource$identity_url, is_admin = resource$is_admin,
				prefs = resource$prefs, updated_at = resource$updated_at,
				default_owner_uuid = resource$default_owner_uuid,
				is_active = resource$is_active, username = resource$username)
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
			
			User$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, created_at = resource$created_at,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, email = resource$email,
				first_name = resource$first_name, last_name = resource$last_name,
				identity_url = resource$identity_url, is_admin = resource$is_admin,
				prefs = resource$prefs, updated_at = resource$updated_at,
				default_owner_uuid = resource$default_owner_uuid,
				is_active = resource$is_active, username = resource$username)
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
			
			User$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, created_at = resource$created_at,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, email = resource$email,
				first_name = resource$first_name, last_name = resource$last_name,
				identity_url = resource$identity_url, is_admin = resource$is_admin,
				prefs = resource$prefs, updated_at = resource$updated_at,
				default_owner_uuid = resource$default_owner_uuid,
				is_active = resource$is_active, username = resource$username)
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
			
			User$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, created_at = resource$created_at,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, email = resource$email,
				first_name = resource$first_name, last_name = resource$last_name,
				identity_url = resource$identity_url, is_admin = resource$is_admin,
				prefs = resource$prefs, updated_at = resource$updated_at,
				default_owner_uuid = resource$default_owner_uuid,
				is_active = resource$is_active, username = resource$username)
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
			
			User$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, created_at = resource$created_at,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, email = resource$email,
				first_name = resource$first_name, last_name = resource$last_name,
				identity_url = resource$identity_url, is_admin = resource$is_admin,
				prefs = resource$prefs, updated_at = resource$updated_at,
				default_owner_uuid = resource$default_owner_uuid,
				is_active = resource$is_active, username = resource$username)
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
			
			UserList$new(kind = resource$kind, etag = resource$etag,
				items = resource$items, next_link = resource$next_link,
				next_page_token = resource$next_page_token,
				selfLink = resource$selfLink)
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
			
			User$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, created_at = resource$created_at,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, email = resource$email,
				first_name = resource$first_name, last_name = resource$last_name,
				identity_url = resource$identity_url, is_admin = resource$is_admin,
				prefs = resource$prefs, updated_at = resource$updated_at,
				default_owner_uuid = resource$default_owner_uuid,
				is_active = resource$is_active, username = resource$username)
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
			
			User$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, created_at = resource$created_at,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, email = resource$email,
				first_name = resource$first_name, last_name = resource$last_name,
				identity_url = resource$identity_url, is_admin = resource$is_admin,
				prefs = resource$prefs, updated_at = resource$updated_at,
				default_owner_uuid = resource$default_owner_uuid,
				is_active = resource$is_active, username = resource$username)
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
			
			ApiClientAuthorization$new(uuid = resource$uuid,
				etag = resource$etag, api_token = resource$api_token,
				api_client_id = resource$api_client_id, user_id = resource$user_id,
				created_by_ip_address = resource$created_by_ip_address,
				last_used_by_ip_address = resource$last_used_by_ip_address,
				last_used_at = resource$last_used_at, expires_at = resource$expires_at,
				created_at = resource$created_at, updated_at = resource$updated_at,
				default_owner_uuid = resource$default_owner_uuid,
				scopes = resource$scopes)
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
			
			ApiClientAuthorizationList$new(kind = resource$kind,
				etag = resource$etag, items = resource$items,
				next_link = resource$next_link, next_page_token = resource$next_page_token,
				selfLink = resource$selfLink)
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
			
			ApiClientAuthorization$new(uuid = resource$uuid,
				etag = resource$etag, api_token = resource$api_token,
				api_client_id = resource$api_client_id, user_id = resource$user_id,
				created_by_ip_address = resource$created_by_ip_address,
				last_used_by_ip_address = resource$last_used_by_ip_address,
				last_used_at = resource$last_used_at, expires_at = resource$expires_at,
				created_at = resource$created_at, updated_at = resource$updated_at,
				default_owner_uuid = resource$default_owner_uuid,
				scopes = resource$scopes)
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
			
			ApiClientAuthorization$new(uuid = resource$uuid,
				etag = resource$etag, api_token = resource$api_token,
				api_client_id = resource$api_client_id, user_id = resource$user_id,
				created_by_ip_address = resource$created_by_ip_address,
				last_used_by_ip_address = resource$last_used_by_ip_address,
				last_used_at = resource$last_used_at, expires_at = resource$expires_at,
				created_at = resource$created_at, updated_at = resource$updated_at,
				default_owner_uuid = resource$default_owner_uuid,
				scopes = resource$scopes)
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
			
			ApiClientAuthorization$new(uuid = resource$uuid,
				etag = resource$etag, api_token = resource$api_token,
				api_client_id = resource$api_client_id, user_id = resource$user_id,
				created_by_ip_address = resource$created_by_ip_address,
				last_used_by_ip_address = resource$last_used_by_ip_address,
				last_used_at = resource$last_used_at, expires_at = resource$expires_at,
				created_at = resource$created_at, updated_at = resource$updated_at,
				default_owner_uuid = resource$default_owner_uuid,
				scopes = resource$scopes)
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
			
			ApiClientAuthorization$new(uuid = resource$uuid,
				etag = resource$etag, api_token = resource$api_token,
				api_client_id = resource$api_client_id, user_id = resource$user_id,
				created_by_ip_address = resource$created_by_ip_address,
				last_used_by_ip_address = resource$last_used_by_ip_address,
				last_used_at = resource$last_used_at, expires_at = resource$expires_at,
				created_at = resource$created_at, updated_at = resource$updated_at,
				default_owner_uuid = resource$default_owner_uuid,
				scopes = resource$scopes)
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
			
			ApiClientAuthorization$new(uuid = resource$uuid,
				etag = resource$etag, api_token = resource$api_token,
				api_client_id = resource$api_client_id, user_id = resource$user_id,
				created_by_ip_address = resource$created_by_ip_address,
				last_used_by_ip_address = resource$last_used_by_ip_address,
				last_used_at = resource$last_used_at, expires_at = resource$expires_at,
				created_at = resource$created_at, updated_at = resource$updated_at,
				default_owner_uuid = resource$default_owner_uuid,
				scopes = resource$scopes)
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
			
			ApiClientAuthorizationList$new(kind = resource$kind,
				etag = resource$etag, items = resource$items,
				next_link = resource$next_link, next_page_token = resource$next_page_token,
				selfLink = resource$selfLink)
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
			
			ApiClientAuthorization$new(uuid = resource$uuid,
				etag = resource$etag, api_token = resource$api_token,
				api_client_id = resource$api_client_id, user_id = resource$user_id,
				created_by_ip_address = resource$created_by_ip_address,
				last_used_by_ip_address = resource$last_used_by_ip_address,
				last_used_at = resource$last_used_at, expires_at = resource$expires_at,
				created_at = resource$created_at, updated_at = resource$updated_at,
				default_owner_uuid = resource$default_owner_uuid,
				scopes = resource$scopes)
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
			
			ApiClientAuthorization$new(uuid = resource$uuid,
				etag = resource$etag, api_token = resource$api_token,
				api_client_id = resource$api_client_id, user_id = resource$user_id,
				created_by_ip_address = resource$created_by_ip_address,
				last_used_by_ip_address = resource$last_used_by_ip_address,
				last_used_at = resource$last_used_at, expires_at = resource$expires_at,
				created_at = resource$created_at, updated_at = resource$updated_at,
				default_owner_uuid = resource$default_owner_uuid,
				scopes = resource$scopes)
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
			
			ApiClient$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, name = resource$name,
				url_prefix = resource$url_prefix, created_at = resource$created_at,
				updated_at = resource$updated_at, is_trusted = resource$is_trusted)
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
			
			ApiClientList$new(kind = resource$kind, etag = resource$etag,
				items = resource$items, next_link = resource$next_link,
				next_page_token = resource$next_page_token,
				selfLink = resource$selfLink)
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
			
			ApiClient$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, name = resource$name,
				url_prefix = resource$url_prefix, created_at = resource$created_at,
				updated_at = resource$updated_at, is_trusted = resource$is_trusted)
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
			
			ApiClient$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, name = resource$name,
				url_prefix = resource$url_prefix, created_at = resource$created_at,
				updated_at = resource$updated_at, is_trusted = resource$is_trusted)
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
			
			ApiClient$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, name = resource$name,
				url_prefix = resource$url_prefix, created_at = resource$created_at,
				updated_at = resource$updated_at, is_trusted = resource$is_trusted)
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
			
			ApiClientList$new(kind = resource$kind, etag = resource$etag,
				items = resource$items, next_link = resource$next_link,
				next_page_token = resource$next_page_token,
				selfLink = resource$selfLink)
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
			
			ApiClient$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, name = resource$name,
				url_prefix = resource$url_prefix, created_at = resource$created_at,
				updated_at = resource$updated_at, is_trusted = resource$is_trusted)
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
			
			ApiClient$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, name = resource$name,
				url_prefix = resource$url_prefix, created_at = resource$created_at,
				updated_at = resource$updated_at, is_trusted = resource$is_trusted)
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
			
			ContainerRequest$new(uuid = resource$uuid,
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
			
			ContainerRequestList$new(kind = resource$kind,
				etag = resource$etag, items = resource$items,
				next_link = resource$next_link, next_page_token = resource$next_page_token,
				selfLink = resource$selfLink)
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
			
			ContainerRequest$new(uuid = resource$uuid,
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
			
			ContainerRequest$new(uuid = resource$uuid,
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
			
			ContainerRequest$new(uuid = resource$uuid,
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
			
			ContainerRequestList$new(kind = resource$kind,
				etag = resource$etag, items = resource$items,
				next_link = resource$next_link, next_page_token = resource$next_page_token,
				selfLink = resource$selfLink)
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
			
			ContainerRequest$new(uuid = resource$uuid,
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
			
			ContainerRequest$new(uuid = resource$uuid,
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
			
			AuthorizedKey$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, name = resource$name,
				key_type = resource$key_type, authorized_user_uuid = resource$authorized_user_uuid,
				public_key = resource$public_key, expires_at = resource$expires_at,
				created_at = resource$created_at, updated_at = resource$updated_at)
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
			
			AuthorizedKeyList$new(kind = resource$kind,
				etag = resource$etag, items = resource$items,
				next_link = resource$next_link, next_page_token = resource$next_page_token,
				selfLink = resource$selfLink)
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
			
			AuthorizedKey$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, name = resource$name,
				key_type = resource$key_type, authorized_user_uuid = resource$authorized_user_uuid,
				public_key = resource$public_key, expires_at = resource$expires_at,
				created_at = resource$created_at, updated_at = resource$updated_at)
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
			
			AuthorizedKey$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, name = resource$name,
				key_type = resource$key_type, authorized_user_uuid = resource$authorized_user_uuid,
				public_key = resource$public_key, expires_at = resource$expires_at,
				created_at = resource$created_at, updated_at = resource$updated_at)
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
			
			AuthorizedKey$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, name = resource$name,
				key_type = resource$key_type, authorized_user_uuid = resource$authorized_user_uuid,
				public_key = resource$public_key, expires_at = resource$expires_at,
				created_at = resource$created_at, updated_at = resource$updated_at)
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
			
			AuthorizedKeyList$new(kind = resource$kind,
				etag = resource$etag, items = resource$items,
				next_link = resource$next_link, next_page_token = resource$next_page_token,
				selfLink = resource$selfLink)
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
			
			AuthorizedKey$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, name = resource$name,
				key_type = resource$key_type, authorized_user_uuid = resource$authorized_user_uuid,
				public_key = resource$public_key, expires_at = resource$expires_at,
				created_at = resource$created_at, updated_at = resource$updated_at)
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
			
			AuthorizedKey$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, name = resource$name,
				key_type = resource$key_type, authorized_user_uuid = resource$authorized_user_uuid,
				public_key = resource$public_key, expires_at = resource$expires_at,
				created_at = resource$created_at, updated_at = resource$updated_at)
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
			
			collection <- Collection$new(uuid = resource$uuid,
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
				is_trashed = resource$is_trashed)
			
			collection$setRESTService(private$REST)
			collection
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
			
			CollectionList$new(kind = resource$kind,
				etag = resource$etag, items = resource$items,
				next_link = resource$next_link, next_page_token = resource$next_page_token,
				selfLink = resource$selfLink)
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
			
			collection <- Collection$new(uuid = resource$uuid,
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
				is_trashed = resource$is_trashed)
			
			collection$setRESTService(private$REST)
			collection
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
			
			collection <- Collection$new(uuid = resource$uuid,
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
				is_trashed = resource$is_trashed)
			
			collection$setRESTService(private$REST)
			collection
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
			
			collection <- Collection$new(uuid = resource$uuid,
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
				is_trashed = resource$is_trashed)
			
			collection$setRESTService(private$REST)
			collection
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
			
			collection <- Collection$new(uuid = resource$uuid,
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
				is_trashed = resource$is_trashed)
			
			collection$setRESTService(private$REST)
			collection
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
			
			collection <- Collection$new(uuid = resource$uuid,
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
				is_trashed = resource$is_trashed)
			
			collection$setRESTService(private$REST)
			collection
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
			
			collection <- Collection$new(uuid = resource$uuid,
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
				is_trashed = resource$is_trashed)
			
			collection$setRESTService(private$REST)
			collection
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
			
			collection <- Collection$new(uuid = resource$uuid,
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
				is_trashed = resource$is_trashed)
			
			collection$setRESTService(private$REST)
			collection
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
			
			CollectionList$new(kind = resource$kind,
				etag = resource$etag, items = resource$items,
				next_link = resource$next_link, next_page_token = resource$next_page_token,
				selfLink = resource$selfLink)
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
			
			collection <- Collection$new(uuid = resource$uuid,
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
				is_trashed = resource$is_trashed)
			
			collection$setRESTService(private$REST)
			collection
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
			
			collection <- Collection$new(uuid = resource$uuid,
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
				is_trashed = resource$is_trashed)
			
			collection$setRESTService(private$REST)
			collection
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
			
			Container$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, created_at = resource$created_at,
				modified_at = resource$modified_at, modified_by_client_uuid = resource$modified_by_client_uuid,
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
			
			ContainerList$new(kind = resource$kind, etag = resource$etag,
				items = resource$items, next_link = resource$next_link,
				next_page_token = resource$next_page_token,
				selfLink = resource$selfLink)
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
			
			Container$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, created_at = resource$created_at,
				modified_at = resource$modified_at, modified_by_client_uuid = resource$modified_by_client_uuid,
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
			
			Container$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, created_at = resource$created_at,
				modified_at = resource$modified_at, modified_by_client_uuid = resource$modified_by_client_uuid,
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
			
			Container$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, created_at = resource$created_at,
				modified_at = resource$modified_at, modified_by_client_uuid = resource$modified_by_client_uuid,
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
			
			Container$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, created_at = resource$created_at,
				modified_at = resource$modified_at, modified_by_client_uuid = resource$modified_by_client_uuid,
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
			
			Container$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, created_at = resource$created_at,
				modified_at = resource$modified_at, modified_by_client_uuid = resource$modified_by_client_uuid,
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
			
			Container$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, created_at = resource$created_at,
				modified_at = resource$modified_at, modified_by_client_uuid = resource$modified_by_client_uuid,
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
			
			Container$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, created_at = resource$created_at,
				modified_at = resource$modified_at, modified_by_client_uuid = resource$modified_by_client_uuid,
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
			
			ContainerList$new(kind = resource$kind, etag = resource$etag,
				items = resource$items, next_link = resource$next_link,
				next_page_token = resource$next_page_token,
				selfLink = resource$selfLink)
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
			
			Container$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, created_at = resource$created_at,
				modified_at = resource$modified_at, modified_by_client_uuid = resource$modified_by_client_uuid,
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
			
			Container$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, created_at = resource$created_at,
				modified_at = resource$modified_at, modified_by_client_uuid = resource$modified_by_client_uuid,
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
			
			Human$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, properties = resource$properties,
				created_at = resource$created_at, updated_at = resource$updated_at)
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
			
			HumanList$new(kind = resource$kind, etag = resource$etag,
				items = resource$items, next_link = resource$next_link,
				next_page_token = resource$next_page_token,
				selfLink = resource$selfLink)
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
			
			Human$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, properties = resource$properties,
				created_at = resource$created_at, updated_at = resource$updated_at)
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
			
			Human$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, properties = resource$properties,
				created_at = resource$created_at, updated_at = resource$updated_at)
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
			
			Human$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, properties = resource$properties,
				created_at = resource$created_at, updated_at = resource$updated_at)
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
			
			HumanList$new(kind = resource$kind, etag = resource$etag,
				items = resource$items, next_link = resource$next_link,
				next_page_token = resource$next_page_token,
				selfLink = resource$selfLink)
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
			
			Human$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, properties = resource$properties,
				created_at = resource$created_at, updated_at = resource$updated_at)
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
			
			Human$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, properties = resource$properties,
				created_at = resource$created_at, updated_at = resource$updated_at)
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
			
			JobTask$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, job_uuid = resource$job_uuid,
				sequence = resource$sequence, parameters = resource$parameters,
				output = resource$output, progress = resource$progress,
				success = resource$success, created_at = resource$created_at,
				updated_at = resource$updated_at, created_by_job_task_uuid = resource$created_by_job_task_uuid,
				qsequence = resource$qsequence, started_at = resource$started_at,
				finished_at = resource$finished_at)
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
			
			JobTaskList$new(kind = resource$kind, etag = resource$etag,
				items = resource$items, next_link = resource$next_link,
				next_page_token = resource$next_page_token,
				selfLink = resource$selfLink)
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
			
			JobTask$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, job_uuid = resource$job_uuid,
				sequence = resource$sequence, parameters = resource$parameters,
				output = resource$output, progress = resource$progress,
				success = resource$success, created_at = resource$created_at,
				updated_at = resource$updated_at, created_by_job_task_uuid = resource$created_by_job_task_uuid,
				qsequence = resource$qsequence, started_at = resource$started_at,
				finished_at = resource$finished_at)
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
			
			JobTask$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, job_uuid = resource$job_uuid,
				sequence = resource$sequence, parameters = resource$parameters,
				output = resource$output, progress = resource$progress,
				success = resource$success, created_at = resource$created_at,
				updated_at = resource$updated_at, created_by_job_task_uuid = resource$created_by_job_task_uuid,
				qsequence = resource$qsequence, started_at = resource$started_at,
				finished_at = resource$finished_at)
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
			
			JobTask$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, job_uuid = resource$job_uuid,
				sequence = resource$sequence, parameters = resource$parameters,
				output = resource$output, progress = resource$progress,
				success = resource$success, created_at = resource$created_at,
				updated_at = resource$updated_at, created_by_job_task_uuid = resource$created_by_job_task_uuid,
				qsequence = resource$qsequence, started_at = resource$started_at,
				finished_at = resource$finished_at)
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
			
			JobTaskList$new(kind = resource$kind, etag = resource$etag,
				items = resource$items, next_link = resource$next_link,
				next_page_token = resource$next_page_token,
				selfLink = resource$selfLink)
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
			
			JobTask$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, job_uuid = resource$job_uuid,
				sequence = resource$sequence, parameters = resource$parameters,
				output = resource$output, progress = resource$progress,
				success = resource$success, created_at = resource$created_at,
				updated_at = resource$updated_at, created_by_job_task_uuid = resource$created_by_job_task_uuid,
				qsequence = resource$qsequence, started_at = resource$started_at,
				finished_at = resource$finished_at)
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
			
			JobTask$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, job_uuid = resource$job_uuid,
				sequence = resource$sequence, parameters = resource$parameters,
				output = resource$output, progress = resource$progress,
				success = resource$success, created_at = resource$created_at,
				updated_at = resource$updated_at, created_by_job_task_uuid = resource$created_by_job_task_uuid,
				qsequence = resource$qsequence, started_at = resource$started_at,
				finished_at = resource$finished_at)
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
			
			Link$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, created_at = resource$created_at,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, tail_uuid = resource$tail_uuid,
				link_class = resource$link_class, name = resource$name,
				head_uuid = resource$head_uuid, properties = resource$properties,
				updated_at = resource$updated_at)
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
			
			LinkList$new(kind = resource$kind, etag = resource$etag,
				items = resource$items, next_link = resource$next_link,
				next_page_token = resource$next_page_token,
				selfLink = resource$selfLink)
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
			
			Link$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, created_at = resource$created_at,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, tail_uuid = resource$tail_uuid,
				link_class = resource$link_class, name = resource$name,
				head_uuid = resource$head_uuid, properties = resource$properties,
				updated_at = resource$updated_at)
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
			
			Link$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, created_at = resource$created_at,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, tail_uuid = resource$tail_uuid,
				link_class = resource$link_class, name = resource$name,
				head_uuid = resource$head_uuid, properties = resource$properties,
				updated_at = resource$updated_at)
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
			
			Link$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, created_at = resource$created_at,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, tail_uuid = resource$tail_uuid,
				link_class = resource$link_class, name = resource$name,
				head_uuid = resource$head_uuid, properties = resource$properties,
				updated_at = resource$updated_at)
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
			
			LinkList$new(kind = resource$kind, etag = resource$etag,
				items = resource$items, next_link = resource$next_link,
				next_page_token = resource$next_page_token,
				selfLink = resource$selfLink)
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
			
			Link$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, created_at = resource$created_at,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, tail_uuid = resource$tail_uuid,
				link_class = resource$link_class, name = resource$name,
				head_uuid = resource$head_uuid, properties = resource$properties,
				updated_at = resource$updated_at)
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
			
			Link$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, created_at = resource$created_at,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, tail_uuid = resource$tail_uuid,
				link_class = resource$link_class, name = resource$name,
				head_uuid = resource$head_uuid, properties = resource$properties,
				updated_at = resource$updated_at)
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
			
			Link$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, created_at = resource$created_at,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, tail_uuid = resource$tail_uuid,
				link_class = resource$link_class, name = resource$name,
				head_uuid = resource$head_uuid, properties = resource$properties,
				updated_at = resource$updated_at)
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
			
			Job$new(uuid = resource$uuid, etag = resource$etag,
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
			
			JobList$new(kind = resource$kind, etag = resource$etag,
				items = resource$items, next_link = resource$next_link,
				next_page_token = resource$next_page_token,
				selfLink = resource$selfLink)
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
			
			Job$new(uuid = resource$uuid, etag = resource$etag,
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
			
			Job$new(uuid = resource$uuid, etag = resource$etag,
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
			
			Job$new(uuid = resource$uuid, etag = resource$etag,
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
			
			Job$new(uuid = resource$uuid, etag = resource$etag,
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
			
			Job$new(uuid = resource$uuid, etag = resource$etag,
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
			
			Job$new(uuid = resource$uuid, etag = resource$etag,
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
			
			Job$new(uuid = resource$uuid, etag = resource$etag,
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
			
			JobList$new(kind = resource$kind, etag = resource$etag,
				items = resource$items, next_link = resource$next_link,
				next_page_token = resource$next_page_token,
				selfLink = resource$selfLink)
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
			
			Job$new(uuid = resource$uuid, etag = resource$etag,
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
			
			Job$new(uuid = resource$uuid, etag = resource$etag,
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
			
			KeepDisk$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, ping_secret = resource$ping_secret,
				node_uuid = resource$node_uuid, filesystem_uuid = resource$filesystem_uuid,
				bytes_total = resource$bytes_total, bytes_free = resource$bytes_free,
				is_readable = resource$is_readable, is_writable = resource$is_writable,
				last_read_at = resource$last_read_at, last_write_at = resource$last_write_at,
				last_ping_at = resource$last_ping_at, created_at = resource$created_at,
				updated_at = resource$updated_at, keep_service_uuid = resource$keep_service_uuid)
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
			
			KeepDiskList$new(kind = resource$kind, etag = resource$etag,
				items = resource$items, next_link = resource$next_link,
				next_page_token = resource$next_page_token,
				selfLink = resource$selfLink)
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
			
			KeepDisk$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, ping_secret = resource$ping_secret,
				node_uuid = resource$node_uuid, filesystem_uuid = resource$filesystem_uuid,
				bytes_total = resource$bytes_total, bytes_free = resource$bytes_free,
				is_readable = resource$is_readable, is_writable = resource$is_writable,
				last_read_at = resource$last_read_at, last_write_at = resource$last_write_at,
				last_ping_at = resource$last_ping_at, created_at = resource$created_at,
				updated_at = resource$updated_at, keep_service_uuid = resource$keep_service_uuid)
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
			
			KeepDisk$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, ping_secret = resource$ping_secret,
				node_uuid = resource$node_uuid, filesystem_uuid = resource$filesystem_uuid,
				bytes_total = resource$bytes_total, bytes_free = resource$bytes_free,
				is_readable = resource$is_readable, is_writable = resource$is_writable,
				last_read_at = resource$last_read_at, last_write_at = resource$last_write_at,
				last_ping_at = resource$last_ping_at, created_at = resource$created_at,
				updated_at = resource$updated_at, keep_service_uuid = resource$keep_service_uuid)
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
			
			KeepDisk$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, ping_secret = resource$ping_secret,
				node_uuid = resource$node_uuid, filesystem_uuid = resource$filesystem_uuid,
				bytes_total = resource$bytes_total, bytes_free = resource$bytes_free,
				is_readable = resource$is_readable, is_writable = resource$is_writable,
				last_read_at = resource$last_read_at, last_write_at = resource$last_write_at,
				last_ping_at = resource$last_ping_at, created_at = resource$created_at,
				updated_at = resource$updated_at, keep_service_uuid = resource$keep_service_uuid)
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
			
			KeepDisk$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, ping_secret = resource$ping_secret,
				node_uuid = resource$node_uuid, filesystem_uuid = resource$filesystem_uuid,
				bytes_total = resource$bytes_total, bytes_free = resource$bytes_free,
				is_readable = resource$is_readable, is_writable = resource$is_writable,
				last_read_at = resource$last_read_at, last_write_at = resource$last_write_at,
				last_ping_at = resource$last_ping_at, created_at = resource$created_at,
				updated_at = resource$updated_at, keep_service_uuid = resource$keep_service_uuid)
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
			
			KeepDiskList$new(kind = resource$kind, etag = resource$etag,
				items = resource$items, next_link = resource$next_link,
				next_page_token = resource$next_page_token,
				selfLink = resource$selfLink)
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
			
			KeepDisk$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, ping_secret = resource$ping_secret,
				node_uuid = resource$node_uuid, filesystem_uuid = resource$filesystem_uuid,
				bytes_total = resource$bytes_total, bytes_free = resource$bytes_free,
				is_readable = resource$is_readable, is_writable = resource$is_writable,
				last_read_at = resource$last_read_at, last_write_at = resource$last_write_at,
				last_ping_at = resource$last_ping_at, created_at = resource$created_at,
				updated_at = resource$updated_at, keep_service_uuid = resource$keep_service_uuid)
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
			
			KeepDisk$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, ping_secret = resource$ping_secret,
				node_uuid = resource$node_uuid, filesystem_uuid = resource$filesystem_uuid,
				bytes_total = resource$bytes_total, bytes_free = resource$bytes_free,
				is_readable = resource$is_readable, is_writable = resource$is_writable,
				last_read_at = resource$last_read_at, last_write_at = resource$last_write_at,
				last_ping_at = resource$last_ping_at, created_at = resource$created_at,
				updated_at = resource$updated_at, keep_service_uuid = resource$keep_service_uuid)
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
			
			KeepService$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, service_host = resource$service_host,
				service_port = resource$service_port, service_ssl_flag = resource$service_ssl_flag,
				service_type = resource$service_type, created_at = resource$created_at,
				updated_at = resource$updated_at, read_only = resource$read_only)
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
			
			KeepServiceList$new(kind = resource$kind,
				etag = resource$etag, items = resource$items,
				next_link = resource$next_link, next_page_token = resource$next_page_token,
				selfLink = resource$selfLink)
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
			
			KeepService$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, service_host = resource$service_host,
				service_port = resource$service_port, service_ssl_flag = resource$service_ssl_flag,
				service_type = resource$service_type, created_at = resource$created_at,
				updated_at = resource$updated_at, read_only = resource$read_only)
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
			
			KeepService$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, service_host = resource$service_host,
				service_port = resource$service_port, service_ssl_flag = resource$service_ssl_flag,
				service_type = resource$service_type, created_at = resource$created_at,
				updated_at = resource$updated_at, read_only = resource$read_only)
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
			
			KeepService$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, service_host = resource$service_host,
				service_port = resource$service_port, service_ssl_flag = resource$service_ssl_flag,
				service_type = resource$service_type, created_at = resource$created_at,
				updated_at = resource$updated_at, read_only = resource$read_only)
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
			
			KeepService$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, service_host = resource$service_host,
				service_port = resource$service_port, service_ssl_flag = resource$service_ssl_flag,
				service_type = resource$service_type, created_at = resource$created_at,
				updated_at = resource$updated_at, read_only = resource$read_only)
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
			
			KeepServiceList$new(kind = resource$kind,
				etag = resource$etag, items = resource$items,
				next_link = resource$next_link, next_page_token = resource$next_page_token,
				selfLink = resource$selfLink)
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
			
			KeepService$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, service_host = resource$service_host,
				service_port = resource$service_port, service_ssl_flag = resource$service_ssl_flag,
				service_type = resource$service_type, created_at = resource$created_at,
				updated_at = resource$updated_at, read_only = resource$read_only)
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
			
			KeepService$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, service_host = resource$service_host,
				service_port = resource$service_port, service_ssl_flag = resource$service_ssl_flag,
				service_type = resource$service_type, created_at = resource$created_at,
				updated_at = resource$updated_at, read_only = resource$read_only)
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
			
			PipelineTemplate$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				created_at = resource$created_at, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, name = resource$name,
				components = resource$components, updated_at = resource$updated_at,
				description = resource$description)
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
			
			PipelineTemplateList$new(kind = resource$kind,
				etag = resource$etag, items = resource$items,
				next_link = resource$next_link, next_page_token = resource$next_page_token,
				selfLink = resource$selfLink)
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
			
			PipelineTemplate$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				created_at = resource$created_at, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, name = resource$name,
				components = resource$components, updated_at = resource$updated_at,
				description = resource$description)
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
			
			PipelineTemplate$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				created_at = resource$created_at, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, name = resource$name,
				components = resource$components, updated_at = resource$updated_at,
				description = resource$description)
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
			
			PipelineTemplate$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				created_at = resource$created_at, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, name = resource$name,
				components = resource$components, updated_at = resource$updated_at,
				description = resource$description)
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
			
			PipelineTemplateList$new(kind = resource$kind,
				etag = resource$etag, items = resource$items,
				next_link = resource$next_link, next_page_token = resource$next_page_token,
				selfLink = resource$selfLink)
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
			
			PipelineTemplate$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				created_at = resource$created_at, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, name = resource$name,
				components = resource$components, updated_at = resource$updated_at,
				description = resource$description)
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
			
			PipelineTemplate$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				created_at = resource$created_at, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, name = resource$name,
				components = resource$components, updated_at = resource$updated_at,
				description = resource$description)
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
			
			PipelineInstance$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				created_at = resource$created_at, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, pipeline_template_uuid = resource$pipeline_template_uuid,
				name = resource$name, components = resource$components,
				updated_at = resource$updated_at, properties = resource$properties,
				state = resource$state, components_summary = resource$components_summary,
				started_at = resource$started_at, finished_at = resource$finished_at,
				description = resource$description)
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
			
			PipelineInstanceList$new(kind = resource$kind,
				etag = resource$etag, items = resource$items,
				next_link = resource$next_link, next_page_token = resource$next_page_token,
				selfLink = resource$selfLink)
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
			
			PipelineInstance$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				created_at = resource$created_at, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, pipeline_template_uuid = resource$pipeline_template_uuid,
				name = resource$name, components = resource$components,
				updated_at = resource$updated_at, properties = resource$properties,
				state = resource$state, components_summary = resource$components_summary,
				started_at = resource$started_at, finished_at = resource$finished_at,
				description = resource$description)
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
			
			PipelineInstance$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				created_at = resource$created_at, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, pipeline_template_uuid = resource$pipeline_template_uuid,
				name = resource$name, components = resource$components,
				updated_at = resource$updated_at, properties = resource$properties,
				state = resource$state, components_summary = resource$components_summary,
				started_at = resource$started_at, finished_at = resource$finished_at,
				description = resource$description)
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
			
			PipelineInstance$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				created_at = resource$created_at, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, pipeline_template_uuid = resource$pipeline_template_uuid,
				name = resource$name, components = resource$components,
				updated_at = resource$updated_at, properties = resource$properties,
				state = resource$state, components_summary = resource$components_summary,
				started_at = resource$started_at, finished_at = resource$finished_at,
				description = resource$description)
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
			
			PipelineInstance$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				created_at = resource$created_at, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, pipeline_template_uuid = resource$pipeline_template_uuid,
				name = resource$name, components = resource$components,
				updated_at = resource$updated_at, properties = resource$properties,
				state = resource$state, components_summary = resource$components_summary,
				started_at = resource$started_at, finished_at = resource$finished_at,
				description = resource$description)
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
			
			PipelineInstanceList$new(kind = resource$kind,
				etag = resource$etag, items = resource$items,
				next_link = resource$next_link, next_page_token = resource$next_page_token,
				selfLink = resource$selfLink)
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
			
			PipelineInstance$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				created_at = resource$created_at, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, pipeline_template_uuid = resource$pipeline_template_uuid,
				name = resource$name, components = resource$components,
				updated_at = resource$updated_at, properties = resource$properties,
				state = resource$state, components_summary = resource$components_summary,
				started_at = resource$started_at, finished_at = resource$finished_at,
				description = resource$description)
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
			
			PipelineInstance$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				created_at = resource$created_at, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, pipeline_template_uuid = resource$pipeline_template_uuid,
				name = resource$name, components = resource$components,
				updated_at = resource$updated_at, properties = resource$properties,
				state = resource$state, components_summary = resource$components_summary,
				started_at = resource$started_at, finished_at = resource$finished_at,
				description = resource$description)
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
			
			Node$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, created_at = resource$created_at,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, slot_number = resource$slot_number,
				hostname = resource$hostname, domain = resource$domain,
				ip_address = resource$ip_address, first_ping_at = resource$first_ping_at,
				last_ping_at = resource$last_ping_at, info = resource$info,
				updated_at = resource$updated_at, properties = resource$properties,
				job_uuid = resource$job_uuid)
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
			
			NodeList$new(kind = resource$kind, etag = resource$etag,
				items = resource$items, next_link = resource$next_link,
				next_page_token = resource$next_page_token,
				selfLink = resource$selfLink)
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
			
			Node$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, created_at = resource$created_at,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, slot_number = resource$slot_number,
				hostname = resource$hostname, domain = resource$domain,
				ip_address = resource$ip_address, first_ping_at = resource$first_ping_at,
				last_ping_at = resource$last_ping_at, info = resource$info,
				updated_at = resource$updated_at, properties = resource$properties,
				job_uuid = resource$job_uuid)
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
			
			Node$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, created_at = resource$created_at,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, slot_number = resource$slot_number,
				hostname = resource$hostname, domain = resource$domain,
				ip_address = resource$ip_address, first_ping_at = resource$first_ping_at,
				last_ping_at = resource$last_ping_at, info = resource$info,
				updated_at = resource$updated_at, properties = resource$properties,
				job_uuid = resource$job_uuid)
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
			
			Node$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, created_at = resource$created_at,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, slot_number = resource$slot_number,
				hostname = resource$hostname, domain = resource$domain,
				ip_address = resource$ip_address, first_ping_at = resource$first_ping_at,
				last_ping_at = resource$last_ping_at, info = resource$info,
				updated_at = resource$updated_at, properties = resource$properties,
				job_uuid = resource$job_uuid)
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
			
			Node$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, created_at = resource$created_at,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, slot_number = resource$slot_number,
				hostname = resource$hostname, domain = resource$domain,
				ip_address = resource$ip_address, first_ping_at = resource$first_ping_at,
				last_ping_at = resource$last_ping_at, info = resource$info,
				updated_at = resource$updated_at, properties = resource$properties,
				job_uuid = resource$job_uuid)
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
			
			NodeList$new(kind = resource$kind, etag = resource$etag,
				items = resource$items, next_link = resource$next_link,
				next_page_token = resource$next_page_token,
				selfLink = resource$selfLink)
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
			
			Node$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, created_at = resource$created_at,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, slot_number = resource$slot_number,
				hostname = resource$hostname, domain = resource$domain,
				ip_address = resource$ip_address, first_ping_at = resource$first_ping_at,
				last_ping_at = resource$last_ping_at, info = resource$info,
				updated_at = resource$updated_at, properties = resource$properties,
				job_uuid = resource$job_uuid)
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
			
			Node$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, created_at = resource$created_at,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, slot_number = resource$slot_number,
				hostname = resource$hostname, domain = resource$domain,
				ip_address = resource$ip_address, first_ping_at = resource$first_ping_at,
				last_ping_at = resource$last_ping_at, info = resource$info,
				updated_at = resource$updated_at, properties = resource$properties,
				job_uuid = resource$job_uuid)
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
			
			Repository$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, name = resource$name,
				created_at = resource$created_at, updated_at = resource$updated_at)
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
			
			RepositoryList$new(kind = resource$kind,
				etag = resource$etag, items = resource$items,
				next_link = resource$next_link, next_page_token = resource$next_page_token,
				selfLink = resource$selfLink)
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
			
			Repository$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, name = resource$name,
				created_at = resource$created_at, updated_at = resource$updated_at)
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
			
			Repository$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, name = resource$name,
				created_at = resource$created_at, updated_at = resource$updated_at)
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
			
			Repository$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, name = resource$name,
				created_at = resource$created_at, updated_at = resource$updated_at)
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
			
			Repository$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, name = resource$name,
				created_at = resource$created_at, updated_at = resource$updated_at)
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
			
			RepositoryList$new(kind = resource$kind,
				etag = resource$etag, items = resource$items,
				next_link = resource$next_link, next_page_token = resource$next_page_token,
				selfLink = resource$selfLink)
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
			
			Repository$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, name = resource$name,
				created_at = resource$created_at, updated_at = resource$updated_at)
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
			
			Repository$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, name = resource$name,
				created_at = resource$created_at, updated_at = resource$updated_at)
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
			
			Specimen$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, created_at = resource$created_at,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, material = resource$material,
				updated_at = resource$updated_at, properties = resource$properties)
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
			
			SpecimenList$new(kind = resource$kind, etag = resource$etag,
				items = resource$items, next_link = resource$next_link,
				next_page_token = resource$next_page_token,
				selfLink = resource$selfLink)
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
			
			Specimen$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, created_at = resource$created_at,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, material = resource$material,
				updated_at = resource$updated_at, properties = resource$properties)
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
			
			Specimen$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, created_at = resource$created_at,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, material = resource$material,
				updated_at = resource$updated_at, properties = resource$properties)
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
			
			Specimen$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, created_at = resource$created_at,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, material = resource$material,
				updated_at = resource$updated_at, properties = resource$properties)
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
			
			SpecimenList$new(kind = resource$kind, etag = resource$etag,
				items = resource$items, next_link = resource$next_link,
				next_page_token = resource$next_page_token,
				selfLink = resource$selfLink)
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
			
			Specimen$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, created_at = resource$created_at,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, material = resource$material,
				updated_at = resource$updated_at, properties = resource$properties)
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
			
			Specimen$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, created_at = resource$created_at,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, material = resource$material,
				updated_at = resource$updated_at, properties = resource$properties)
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
			
			Log$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				object_uuid = resource$object_uuid, event_at = resource$event_at,
				event_type = resource$event_type, summary = resource$summary,
				properties = resource$properties, created_at = resource$created_at,
				updated_at = resource$updated_at, modified_at = resource$modified_at,
				object_owner_uuid = resource$object_owner_uuid)
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
			
			LogList$new(kind = resource$kind, etag = resource$etag,
				items = resource$items, next_link = resource$next_link,
				next_page_token = resource$next_page_token,
				selfLink = resource$selfLink)
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
			
			Log$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				object_uuid = resource$object_uuid, event_at = resource$event_at,
				event_type = resource$event_type, summary = resource$summary,
				properties = resource$properties, created_at = resource$created_at,
				updated_at = resource$updated_at, modified_at = resource$modified_at,
				object_owner_uuid = resource$object_owner_uuid)
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
			
			Log$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				object_uuid = resource$object_uuid, event_at = resource$event_at,
				event_type = resource$event_type, summary = resource$summary,
				properties = resource$properties, created_at = resource$created_at,
				updated_at = resource$updated_at, modified_at = resource$modified_at,
				object_owner_uuid = resource$object_owner_uuid)
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
			
			Log$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				object_uuid = resource$object_uuid, event_at = resource$event_at,
				event_type = resource$event_type, summary = resource$summary,
				properties = resource$properties, created_at = resource$created_at,
				updated_at = resource$updated_at, modified_at = resource$modified_at,
				object_owner_uuid = resource$object_owner_uuid)
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
			
			LogList$new(kind = resource$kind, etag = resource$etag,
				items = resource$items, next_link = resource$next_link,
				next_page_token = resource$next_page_token,
				selfLink = resource$selfLink)
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
			
			Log$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				object_uuid = resource$object_uuid, event_at = resource$event_at,
				event_type = resource$event_type, summary = resource$summary,
				properties = resource$properties, created_at = resource$created_at,
				updated_at = resource$updated_at, modified_at = resource$modified_at,
				object_owner_uuid = resource$object_owner_uuid)
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
			
			Log$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				object_uuid = resource$object_uuid, event_at = resource$event_at,
				event_type = resource$event_type, summary = resource$summary,
				properties = resource$properties, created_at = resource$created_at,
				updated_at = resource$updated_at, modified_at = resource$modified_at,
				object_owner_uuid = resource$object_owner_uuid)
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
			
			Trait$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, name = resource$name,
				properties = resource$properties, created_at = resource$created_at,
				updated_at = resource$updated_at)
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
			
			TraitList$new(kind = resource$kind, etag = resource$etag,
				items = resource$items, next_link = resource$next_link,
				next_page_token = resource$next_page_token,
				selfLink = resource$selfLink)
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
			
			Trait$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, name = resource$name,
				properties = resource$properties, created_at = resource$created_at,
				updated_at = resource$updated_at)
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
			
			Trait$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, name = resource$name,
				properties = resource$properties, created_at = resource$created_at,
				updated_at = resource$updated_at)
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
			
			Trait$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, name = resource$name,
				properties = resource$properties, created_at = resource$created_at,
				updated_at = resource$updated_at)
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
			
			TraitList$new(kind = resource$kind, etag = resource$etag,
				items = resource$items, next_link = resource$next_link,
				next_page_token = resource$next_page_token,
				selfLink = resource$selfLink)
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
			
			Trait$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, name = resource$name,
				properties = resource$properties, created_at = resource$created_at,
				updated_at = resource$updated_at)
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
			
			Trait$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, name = resource$name,
				properties = resource$properties, created_at = resource$created_at,
				updated_at = resource$updated_at)
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
			
			VirtualMachine$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, hostname = resource$hostname,
				created_at = resource$created_at, updated_at = resource$updated_at)
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
			
			VirtualMachineList$new(kind = resource$kind,
				etag = resource$etag, items = resource$items,
				next_link = resource$next_link, next_page_token = resource$next_page_token,
				selfLink = resource$selfLink)
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
			
			VirtualMachine$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, hostname = resource$hostname,
				created_at = resource$created_at, updated_at = resource$updated_at)
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
			
			VirtualMachine$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, hostname = resource$hostname,
				created_at = resource$created_at, updated_at = resource$updated_at)
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
			
			VirtualMachine$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, hostname = resource$hostname,
				created_at = resource$created_at, updated_at = resource$updated_at)
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
			
			VirtualMachine$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, hostname = resource$hostname,
				created_at = resource$created_at, updated_at = resource$updated_at)
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
			
			VirtualMachine$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, hostname = resource$hostname,
				created_at = resource$created_at, updated_at = resource$updated_at)
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
			
			VirtualMachineList$new(kind = resource$kind,
				etag = resource$etag, items = resource$items,
				next_link = resource$next_link, next_page_token = resource$next_page_token,
				selfLink = resource$selfLink)
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
			
			VirtualMachine$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, hostname = resource$hostname,
				created_at = resource$created_at, updated_at = resource$updated_at)
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
			
			VirtualMachine$new(uuid = resource$uuid,
				etag = resource$etag, owner_uuid = resource$owner_uuid,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, hostname = resource$hostname,
				created_at = resource$created_at, updated_at = resource$updated_at)
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
			
			Workflow$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, created_at = resource$created_at,
				modified_at = resource$modified_at, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				name = resource$name, description = resource$description,
				definition = resource$definition, updated_at = resource$updated_at)
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
			
			WorkflowList$new(kind = resource$kind, etag = resource$etag,
				items = resource$items, next_link = resource$next_link,
				next_page_token = resource$next_page_token,
				selfLink = resource$selfLink)
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
			
			Workflow$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, created_at = resource$created_at,
				modified_at = resource$modified_at, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				name = resource$name, description = resource$description,
				definition = resource$definition, updated_at = resource$updated_at)
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
			
			Workflow$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, created_at = resource$created_at,
				modified_at = resource$modified_at, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				name = resource$name, description = resource$description,
				definition = resource$definition, updated_at = resource$updated_at)
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
			
			Workflow$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, created_at = resource$created_at,
				modified_at = resource$modified_at, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				name = resource$name, description = resource$description,
				definition = resource$definition, updated_at = resource$updated_at)
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
			
			WorkflowList$new(kind = resource$kind, etag = resource$etag,
				items = resource$items, next_link = resource$next_link,
				next_page_token = resource$next_page_token,
				selfLink = resource$selfLink)
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
			
			Workflow$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, created_at = resource$created_at,
				modified_at = resource$modified_at, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				name = resource$name, description = resource$description,
				definition = resource$definition, updated_at = resource$updated_at)
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
			
			Workflow$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, created_at = resource$created_at,
				modified_at = resource$modified_at, modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				name = resource$name, description = resource$description,
				definition = resource$definition, updated_at = resource$updated_at)
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
			
			Group$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, created_at = resource$created_at,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, name = resource$name,
				description = resource$description, updated_at = resource$updated_at,
				group_class = resource$group_class, trash_at = resource$trash_at,
				is_trashed = resource$is_trashed, delete_at = resource$delete_at)
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
			
			GroupList$new(kind = resource$kind, etag = resource$etag,
				items = resource$items, next_link = resource$next_link,
				next_page_token = resource$next_page_token,
				selfLink = resource$selfLink)
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
			
			Group$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, created_at = resource$created_at,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, name = resource$name,
				description = resource$description, updated_at = resource$updated_at,
				group_class = resource$group_class, trash_at = resource$trash_at,
				is_trashed = resource$is_trashed, delete_at = resource$delete_at)
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
			
			Group$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, created_at = resource$created_at,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, name = resource$name,
				description = resource$description, updated_at = resource$updated_at,
				group_class = resource$group_class, trash_at = resource$trash_at,
				is_trashed = resource$is_trashed, delete_at = resource$delete_at)
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
			
			Group$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, created_at = resource$created_at,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, name = resource$name,
				description = resource$description, updated_at = resource$updated_at,
				group_class = resource$group_class, trash_at = resource$trash_at,
				is_trashed = resource$is_trashed, delete_at = resource$delete_at)
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
			
			Group$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, created_at = resource$created_at,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, name = resource$name,
				description = resource$description, updated_at = resource$updated_at,
				group_class = resource$group_class, trash_at = resource$trash_at,
				is_trashed = resource$is_trashed, delete_at = resource$delete_at)
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
			
			Group$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, created_at = resource$created_at,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, name = resource$name,
				description = resource$description, updated_at = resource$updated_at,
				group_class = resource$group_class, trash_at = resource$trash_at,
				is_trashed = resource$is_trashed, delete_at = resource$delete_at)
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
			
			Group$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, created_at = resource$created_at,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, name = resource$name,
				description = resource$description, updated_at = resource$updated_at,
				group_class = resource$group_class, trash_at = resource$trash_at,
				is_trashed = resource$is_trashed, delete_at = resource$delete_at)
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
			
			GroupList$new(kind = resource$kind, etag = resource$etag,
				items = resource$items, next_link = resource$next_link,
				next_page_token = resource$next_page_token,
				selfLink = resource$selfLink)
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
			
			Group$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, created_at = resource$created_at,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, name = resource$name,
				description = resource$description, updated_at = resource$updated_at,
				group_class = resource$group_class, trash_at = resource$trash_at,
				is_trashed = resource$is_trashed, delete_at = resource$delete_at)
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
			
			Group$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, created_at = resource$created_at,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, name = resource$name,
				description = resource$description, updated_at = resource$updated_at,
				group_class = resource$group_class, trash_at = resource$trash_at,
				is_trashed = resource$is_trashed, delete_at = resource$delete_at)
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
			
			UserAgreement$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, created_at = resource$created_at,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, portable_data_hash = resource$portable_data_hash,
				replication_desired = resource$replication_desired,
				replication_confirmed_at = resource$replication_confirmed_at,
				replication_confirmed = resource$replication_confirmed,
				updated_at = resource$updated_at, manifest_text = resource$manifest_text,
				name = resource$name, description = resource$description,
				properties = resource$properties, delete_at = resource$delete_at,
				file_names = resource$file_names, trash_at = resource$trash_at,
				is_trashed = resource$is_trashed)
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
			
			UserAgreementList$new(kind = resource$kind,
				etag = resource$etag, items = resource$items,
				next_link = resource$next_link, next_page_token = resource$next_page_token,
				selfLink = resource$selfLink)
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
			
			UserAgreement$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, created_at = resource$created_at,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, portable_data_hash = resource$portable_data_hash,
				replication_desired = resource$replication_desired,
				replication_confirmed_at = resource$replication_confirmed_at,
				replication_confirmed = resource$replication_confirmed,
				updated_at = resource$updated_at, manifest_text = resource$manifest_text,
				name = resource$name, description = resource$description,
				properties = resource$properties, delete_at = resource$delete_at,
				file_names = resource$file_names, trash_at = resource$trash_at,
				is_trashed = resource$is_trashed)
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
			
			UserAgreement$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, created_at = resource$created_at,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, portable_data_hash = resource$portable_data_hash,
				replication_desired = resource$replication_desired,
				replication_confirmed_at = resource$replication_confirmed_at,
				replication_confirmed = resource$replication_confirmed,
				updated_at = resource$updated_at, manifest_text = resource$manifest_text,
				name = resource$name, description = resource$description,
				properties = resource$properties, delete_at = resource$delete_at,
				file_names = resource$file_names, trash_at = resource$trash_at,
				is_trashed = resource$is_trashed)
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
			
			UserAgreement$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, created_at = resource$created_at,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, portable_data_hash = resource$portable_data_hash,
				replication_desired = resource$replication_desired,
				replication_confirmed_at = resource$replication_confirmed_at,
				replication_confirmed = resource$replication_confirmed,
				updated_at = resource$updated_at, manifest_text = resource$manifest_text,
				name = resource$name, description = resource$description,
				properties = resource$properties, delete_at = resource$delete_at,
				file_names = resource$file_names, trash_at = resource$trash_at,
				is_trashed = resource$is_trashed)
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
			
			UserAgreement$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, created_at = resource$created_at,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, portable_data_hash = resource$portable_data_hash,
				replication_desired = resource$replication_desired,
				replication_confirmed_at = resource$replication_confirmed_at,
				replication_confirmed = resource$replication_confirmed,
				updated_at = resource$updated_at, manifest_text = resource$manifest_text,
				name = resource$name, description = resource$description,
				properties = resource$properties, delete_at = resource$delete_at,
				file_names = resource$file_names, trash_at = resource$trash_at,
				is_trashed = resource$is_trashed)
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
			
			UserAgreement$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, created_at = resource$created_at,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, portable_data_hash = resource$portable_data_hash,
				replication_desired = resource$replication_desired,
				replication_confirmed_at = resource$replication_confirmed_at,
				replication_confirmed = resource$replication_confirmed,
				updated_at = resource$updated_at, manifest_text = resource$manifest_text,
				name = resource$name, description = resource$description,
				properties = resource$properties, delete_at = resource$delete_at,
				file_names = resource$file_names, trash_at = resource$trash_at,
				is_trashed = resource$is_trashed)
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
			
			UserAgreementList$new(kind = resource$kind,
				etag = resource$etag, items = resource$items,
				next_link = resource$next_link, next_page_token = resource$next_page_token,
				selfLink = resource$selfLink)
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
			
			UserAgreement$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, created_at = resource$created_at,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, portable_data_hash = resource$portable_data_hash,
				replication_desired = resource$replication_desired,
				replication_confirmed_at = resource$replication_confirmed_at,
				replication_confirmed = resource$replication_confirmed,
				updated_at = resource$updated_at, manifest_text = resource$manifest_text,
				name = resource$name, description = resource$description,
				properties = resource$properties, delete_at = resource$delete_at,
				file_names = resource$file_names, trash_at = resource$trash_at,
				is_trashed = resource$is_trashed)
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
			
			UserAgreement$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, created_at = resource$created_at,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, portable_data_hash = resource$portable_data_hash,
				replication_desired = resource$replication_desired,
				replication_confirmed_at = resource$replication_confirmed_at,
				replication_confirmed = resource$replication_confirmed,
				updated_at = resource$updated_at, manifest_text = resource$manifest_text,
				name = resource$name, description = resource$description,
				properties = resource$properties, delete_at = resource$delete_at,
				file_names = resource$file_names, trash_at = resource$trash_at,
				is_trashed = resource$is_trashed)
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
			
			UserAgreement$new(uuid = resource$uuid, etag = resource$etag,
				owner_uuid = resource$owner_uuid, created_at = resource$created_at,
				modified_by_client_uuid = resource$modified_by_client_uuid,
				modified_by_user_uuid = resource$modified_by_user_uuid,
				modified_at = resource$modified_at, portable_data_hash = resource$portable_data_hash,
				replication_desired = resource$replication_desired,
				replication_confirmed_at = resource$replication_confirmed_at,
				replication_confirmed = resource$replication_confirmed,
				updated_at = resource$updated_at, manifest_text = resource$manifest_text,
				name = resource$name, description = resource$description,
				properties = resource$properties, delete_at = resource$delete_at,
				file_names = resource$file_names, trash_at = resource$trash_at,
				is_trashed = resource$is_trashed)
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
