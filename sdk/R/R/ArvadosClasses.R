#' @export
UserList <- R6::R6Class(

	"UserList",

	public = list(
		kind = NULL,
		etag = NULL,
		items = NULL,
		next_link = NULL,
		next_page_token = NULL,
		selfLink = NULL,

		initialize = function(
				kind = NULL, etag = NULL, items = NULL, next_link = NULL,
				next_page_token = NULL, selfLink = NULL)
		{
			self$kind <- kind
			self$etag <- etag
			self$items <- items
			self$next_link <- next_link
			self$next_page_token <- next_page_token
			self$selfLink <- selfLink
			
			private$classFields <- c(
				kind, etag, items, next_link, next_page_token,
				selfLink
			)
		},

		toJSON = function() {
			fields <- sapply(private$classFields, function(field)
			{
				self[[field]]
			}, USE.NAMES = TRUE)
			
			jsonlite::toJSON(list("userlist" = 
                     Filter(Negate(is.null), fields)), auto_unbox = TRUE)
		}
	),

	private = list(
		classFields = NULL
	),

	cloneable = FALSE
)

#' @export
User <- R6::R6Class(

	"User",

	public = list(
		uuid = NULL,
		etag = NULL,
		owner_uuid = NULL,
		created_at = NULL,
		modified_by_client_uuid = NULL,
		modified_by_user_uuid = NULL,
		modified_at = NULL,
		email = NULL,
		first_name = NULL,
		last_name = NULL,
		identity_url = NULL,
		is_admin = NULL,
		prefs = NULL,
		updated_at = NULL,
		default_owner_uuid = NULL,
		is_active = NULL,
		username = NULL,

		initialize = function(
				uuid = NULL, etag = NULL, owner_uuid = NULL,
				created_at = NULL, modified_by_client_uuid = NULL,
				modified_by_user_uuid = NULL, modified_at = NULL,
				email = NULL, first_name = NULL, last_name = NULL,
				identity_url = NULL, is_admin = NULL, prefs = NULL,
				updated_at = NULL, default_owner_uuid = NULL,
				is_active = NULL, username = NULL)
		{
			self$uuid <- uuid
			self$etag <- etag
			self$owner_uuid <- owner_uuid
			self$created_at <- created_at
			self$modified_by_client_uuid <- modified_by_client_uuid
			self$modified_by_user_uuid <- modified_by_user_uuid
			self$modified_at <- modified_at
			self$email <- email
			self$first_name <- first_name
			self$last_name <- last_name
			self$identity_url <- identity_url
			self$is_admin <- is_admin
			self$prefs <- prefs
			self$updated_at <- updated_at
			self$default_owner_uuid <- default_owner_uuid
			self$is_active <- is_active
			self$username <- username
			
			private$classFields <- c(
				uuid, etag, owner_uuid, created_at, modified_by_client_uuid,
				modified_by_user_uuid, modified_at, email,
				first_name, last_name, identity_url, is_admin,
				prefs, updated_at, default_owner_uuid, is_active,
				username
			)
		},

		toJSON = function() {
			fields <- sapply(private$classFields, function(field)
			{
				self[[field]]
			}, USE.NAMES = TRUE)
			
			jsonlite::toJSON(list("user" = 
                     Filter(Negate(is.null), fields)), auto_unbox = TRUE)
		}
	),

	private = list(
		classFields = NULL
	),

	cloneable = FALSE
)

#' @export
ApiClientAuthorizationList <- R6::R6Class(

	"ApiClientAuthorizationList",

	public = list(
		kind = NULL,
		etag = NULL,
		items = NULL,
		next_link = NULL,
		next_page_token = NULL,
		selfLink = NULL,

		initialize = function(
				kind = NULL, etag = NULL, items = NULL, next_link = NULL,
				next_page_token = NULL, selfLink = NULL)
		{
			self$kind <- kind
			self$etag <- etag
			self$items <- items
			self$next_link <- next_link
			self$next_page_token <- next_page_token
			self$selfLink <- selfLink
			
			private$classFields <- c(
				kind, etag, items, next_link, next_page_token,
				selfLink
			)
		},

		toJSON = function() {
			fields <- sapply(private$classFields, function(field)
			{
				self[[field]]
			}, USE.NAMES = TRUE)
			
			jsonlite::toJSON(list("apiclientauthorizationlist" = 
                     Filter(Negate(is.null), fields)), auto_unbox = TRUE)
		}
	),

	private = list(
		classFields = NULL
	),

	cloneable = FALSE
)

#' @export
ApiClientAuthorization <- R6::R6Class(

	"ApiClientAuthorization",

	public = list(
		uuid = NULL,
		etag = NULL,
		api_token = NULL,
		api_client_id = NULL,
		user_id = NULL,
		created_by_ip_address = NULL,
		last_used_by_ip_address = NULL,
		last_used_at = NULL,
		expires_at = NULL,
		created_at = NULL,
		updated_at = NULL,
		default_owner_uuid = NULL,
		scopes = NULL,

		initialize = function(
				uuid = NULL, etag = NULL, api_token = NULL,
				api_client_id = NULL, user_id = NULL, created_by_ip_address = NULL,
				last_used_by_ip_address = NULL, last_used_at = NULL,
				expires_at = NULL, created_at = NULL, updated_at = NULL,
				default_owner_uuid = NULL, scopes = NULL)
		{
			self$uuid <- uuid
			self$etag <- etag
			self$api_token <- api_token
			self$api_client_id <- api_client_id
			self$user_id <- user_id
			self$created_by_ip_address <- created_by_ip_address
			self$last_used_by_ip_address <- last_used_by_ip_address
			self$last_used_at <- last_used_at
			self$expires_at <- expires_at
			self$created_at <- created_at
			self$updated_at <- updated_at
			self$default_owner_uuid <- default_owner_uuid
			self$scopes <- scopes
			
			private$classFields <- c(
				uuid, etag, api_token, api_client_id, user_id,
				created_by_ip_address, last_used_by_ip_address,
				last_used_at, expires_at, created_at, updated_at,
				default_owner_uuid, scopes
			)
		},

		toJSON = function() {
			fields <- sapply(private$classFields, function(field)
			{
				self[[field]]
			}, USE.NAMES = TRUE)
			
			jsonlite::toJSON(list("apiclientauthorization" = 
                     Filter(Negate(is.null), fields)), auto_unbox = TRUE)
		}
	),

	private = list(
		classFields = NULL
	),

	cloneable = FALSE
)

#' @export
ApiClientList <- R6::R6Class(

	"ApiClientList",

	public = list(
		kind = NULL,
		etag = NULL,
		items = NULL,
		next_link = NULL,
		next_page_token = NULL,
		selfLink = NULL,

		initialize = function(
				kind = NULL, etag = NULL, items = NULL, next_link = NULL,
				next_page_token = NULL, selfLink = NULL)
		{
			self$kind <- kind
			self$etag <- etag
			self$items <- items
			self$next_link <- next_link
			self$next_page_token <- next_page_token
			self$selfLink <- selfLink
			
			private$classFields <- c(
				kind, etag, items, next_link, next_page_token,
				selfLink
			)
		},

		toJSON = function() {
			fields <- sapply(private$classFields, function(field)
			{
				self[[field]]
			}, USE.NAMES = TRUE)
			
			jsonlite::toJSON(list("apiclientlist" = 
                     Filter(Negate(is.null), fields)), auto_unbox = TRUE)
		}
	),

	private = list(
		classFields = NULL
	),

	cloneable = FALSE
)

#' @export
ApiClient <- R6::R6Class(

	"ApiClient",

	public = list(
		uuid = NULL,
		etag = NULL,
		owner_uuid = NULL,
		modified_by_client_uuid = NULL,
		modified_by_user_uuid = NULL,
		modified_at = NULL,
		name = NULL,
		url_prefix = NULL,
		created_at = NULL,
		updated_at = NULL,
		is_trusted = NULL,

		initialize = function(
				uuid = NULL, etag = NULL, owner_uuid = NULL,
				modified_by_client_uuid = NULL, modified_by_user_uuid = NULL,
				modified_at = NULL, name = NULL, url_prefix = NULL,
				created_at = NULL, updated_at = NULL, is_trusted = NULL)
		{
			self$uuid <- uuid
			self$etag <- etag
			self$owner_uuid <- owner_uuid
			self$modified_by_client_uuid <- modified_by_client_uuid
			self$modified_by_user_uuid <- modified_by_user_uuid
			self$modified_at <- modified_at
			self$name <- name
			self$url_prefix <- url_prefix
			self$created_at <- created_at
			self$updated_at <- updated_at
			self$is_trusted <- is_trusted
			
			private$classFields <- c(
				uuid, etag, owner_uuid, modified_by_client_uuid,
				modified_by_user_uuid, modified_at, name,
				url_prefix, created_at, updated_at, is_trusted
			)
		},

		toJSON = function() {
			fields <- sapply(private$classFields, function(field)
			{
				self[[field]]
			}, USE.NAMES = TRUE)
			
			jsonlite::toJSON(list("apiclient" = 
                     Filter(Negate(is.null), fields)), auto_unbox = TRUE)
		}
	),

	private = list(
		classFields = NULL
	),

	cloneable = FALSE
)

#' @export
ContainerRequestList <- R6::R6Class(

	"ContainerRequestList",

	public = list(
		kind = NULL,
		etag = NULL,
		items = NULL,
		next_link = NULL,
		next_page_token = NULL,
		selfLink = NULL,

		initialize = function(
				kind = NULL, etag = NULL, items = NULL, next_link = NULL,
				next_page_token = NULL, selfLink = NULL)
		{
			self$kind <- kind
			self$etag <- etag
			self$items <- items
			self$next_link <- next_link
			self$next_page_token <- next_page_token
			self$selfLink <- selfLink
			
			private$classFields <- c(
				kind, etag, items, next_link, next_page_token,
				selfLink
			)
		},

		toJSON = function() {
			fields <- sapply(private$classFields, function(field)
			{
				self[[field]]
			}, USE.NAMES = TRUE)
			
			jsonlite::toJSON(list("containerrequestlist" = 
                     Filter(Negate(is.null), fields)), auto_unbox = TRUE)
		}
	),

	private = list(
		classFields = NULL
	),

	cloneable = FALSE
)

#' @export
ContainerRequest <- R6::R6Class(

	"ContainerRequest",

	public = list(
		uuid = NULL,
		etag = NULL,
		owner_uuid = NULL,
		created_at = NULL,
		modified_at = NULL,
		modified_by_client_uuid = NULL,
		modified_by_user_uuid = NULL,
		name = NULL,
		description = NULL,
		properties = NULL,
		state = NULL,
		requesting_container_uuid = NULL,
		container_uuid = NULL,
		container_count_max = NULL,
		mounts = NULL,
		runtime_constraints = NULL,
		container_image = NULL,
		environment = NULL,
		cwd = NULL,
		command = NULL,
		output_path = NULL,
		priority = NULL,
		expires_at = NULL,
		filters = NULL,
		updated_at = NULL,
		container_count = NULL,
		use_existing = NULL,
		scheduling_parameters = NULL,
		output_uuid = NULL,
		log_uuid = NULL,
		output_name = NULL,
		output_ttl = NULL,

		initialize = function(
				uuid = NULL, etag = NULL, owner_uuid = NULL,
				created_at = NULL, modified_at = NULL, modified_by_client_uuid = NULL,
				modified_by_user_uuid = NULL, name = NULL,
				description = NULL, properties = NULL, state = NULL,
				requesting_container_uuid = NULL, container_uuid = NULL,
				container_count_max = NULL, mounts = NULL,
				runtime_constraints = NULL, container_image = NULL,
				environment = NULL, cwd = NULL, command = NULL,
				output_path = NULL, priority = NULL, expires_at = NULL,
				filters = NULL, updated_at = NULL, container_count = NULL,
				use_existing = NULL, scheduling_parameters = NULL,
				output_uuid = NULL, log_uuid = NULL, output_name = NULL,
				output_ttl = NULL)
		{
			self$uuid <- uuid
			self$etag <- etag
			self$owner_uuid <- owner_uuid
			self$created_at <- created_at
			self$modified_at <- modified_at
			self$modified_by_client_uuid <- modified_by_client_uuid
			self$modified_by_user_uuid <- modified_by_user_uuid
			self$name <- name
			self$description <- description
			self$properties <- properties
			self$state <- state
			self$requesting_container_uuid <- requesting_container_uuid
			self$container_uuid <- container_uuid
			self$container_count_max <- container_count_max
			self$mounts <- mounts
			self$runtime_constraints <- runtime_constraints
			self$container_image <- container_image
			self$environment <- environment
			self$cwd <- cwd
			self$command <- command
			self$output_path <- output_path
			self$priority <- priority
			self$expires_at <- expires_at
			self$filters <- filters
			self$updated_at <- updated_at
			self$container_count <- container_count
			self$use_existing <- use_existing
			self$scheduling_parameters <- scheduling_parameters
			self$output_uuid <- output_uuid
			self$log_uuid <- log_uuid
			self$output_name <- output_name
			self$output_ttl <- output_ttl
			
			private$classFields <- c(
				uuid, etag, owner_uuid, created_at, modified_at,
				modified_by_client_uuid, modified_by_user_uuid,
				name, description, properties, state, requesting_container_uuid,
				container_uuid, container_count_max, mounts,
				runtime_constraints, container_image, environment,
				cwd, command, output_path, priority, expires_at,
				filters, updated_at, container_count, use_existing,
				scheduling_parameters, output_uuid, log_uuid,
				output_name, output_ttl
			)
		},

		toJSON = function() {
			fields <- sapply(private$classFields, function(field)
			{
				self[[field]]
			}, USE.NAMES = TRUE)
			
			jsonlite::toJSON(list("containerrequest" = 
                     Filter(Negate(is.null), fields)), auto_unbox = TRUE)
		}
	),

	private = list(
		classFields = NULL
	),

	cloneable = FALSE
)

#' @export
AuthorizedKeyList <- R6::R6Class(

	"AuthorizedKeyList",

	public = list(
		kind = NULL,
		etag = NULL,
		items = NULL,
		next_link = NULL,
		next_page_token = NULL,
		selfLink = NULL,

		initialize = function(
				kind = NULL, etag = NULL, items = NULL, next_link = NULL,
				next_page_token = NULL, selfLink = NULL)
		{
			self$kind <- kind
			self$etag <- etag
			self$items <- items
			self$next_link <- next_link
			self$next_page_token <- next_page_token
			self$selfLink <- selfLink
			
			private$classFields <- c(
				kind, etag, items, next_link, next_page_token,
				selfLink
			)
		},

		toJSON = function() {
			fields <- sapply(private$classFields, function(field)
			{
				self[[field]]
			}, USE.NAMES = TRUE)
			
			jsonlite::toJSON(list("authorizedkeylist" = 
                     Filter(Negate(is.null), fields)), auto_unbox = TRUE)
		}
	),

	private = list(
		classFields = NULL
	),

	cloneable = FALSE
)

#' @export
AuthorizedKey <- R6::R6Class(

	"AuthorizedKey",

	public = list(
		uuid = NULL,
		etag = NULL,
		owner_uuid = NULL,
		modified_by_client_uuid = NULL,
		modified_by_user_uuid = NULL,
		modified_at = NULL,
		name = NULL,
		key_type = NULL,
		authorized_user_uuid = NULL,
		public_key = NULL,
		expires_at = NULL,
		created_at = NULL,
		updated_at = NULL,

		initialize = function(
				uuid = NULL, etag = NULL, owner_uuid = NULL,
				modified_by_client_uuid = NULL, modified_by_user_uuid = NULL,
				modified_at = NULL, name = NULL, key_type = NULL,
				authorized_user_uuid = NULL, public_key = NULL,
				expires_at = NULL, created_at = NULL, updated_at = NULL)
		{
			self$uuid <- uuid
			self$etag <- etag
			self$owner_uuid <- owner_uuid
			self$modified_by_client_uuid <- modified_by_client_uuid
			self$modified_by_user_uuid <- modified_by_user_uuid
			self$modified_at <- modified_at
			self$name <- name
			self$key_type <- key_type
			self$authorized_user_uuid <- authorized_user_uuid
			self$public_key <- public_key
			self$expires_at <- expires_at
			self$created_at <- created_at
			self$updated_at <- updated_at
			
			private$classFields <- c(
				uuid, etag, owner_uuid, modified_by_client_uuid,
				modified_by_user_uuid, modified_at, name,
				key_type, authorized_user_uuid, public_key,
				expires_at, created_at, updated_at
			)
		},

		toJSON = function() {
			fields <- sapply(private$classFields, function(field)
			{
				self[[field]]
			}, USE.NAMES = TRUE)
			
			jsonlite::toJSON(list("authorizedkey" = 
                     Filter(Negate(is.null), fields)), auto_unbox = TRUE)
		}
	),

	private = list(
		classFields = NULL
	),

	cloneable = FALSE
)

#' @export
CollectionList <- R6::R6Class(

	"CollectionList",

	public = list(
		kind = NULL,
		etag = NULL,
		items = NULL,
		next_link = NULL,
		next_page_token = NULL,
		selfLink = NULL,

		initialize = function(
				kind = NULL, etag = NULL, items = NULL, next_link = NULL,
				next_page_token = NULL, selfLink = NULL)
		{
			self$kind <- kind
			self$etag <- etag
			self$items <- items
			self$next_link <- next_link
			self$next_page_token <- next_page_token
			self$selfLink <- selfLink
			
			private$classFields <- c(
				kind, etag, items, next_link, next_page_token,
				selfLink
			)
		},

		toJSON = function() {
			fields <- sapply(private$classFields, function(field)
			{
				self[[field]]
			}, USE.NAMES = TRUE)
			
			jsonlite::toJSON(list("collectionlist" = 
                     Filter(Negate(is.null), fields)), auto_unbox = TRUE)
		}
	),

	private = list(
		classFields = NULL
	),

	cloneable = FALSE
)

#' @export
ContainerList <- R6::R6Class(

	"ContainerList",

	public = list(
		kind = NULL,
		etag = NULL,
		items = NULL,
		next_link = NULL,
		next_page_token = NULL,
		selfLink = NULL,

		initialize = function(
				kind = NULL, etag = NULL, items = NULL, next_link = NULL,
				next_page_token = NULL, selfLink = NULL)
		{
			self$kind <- kind
			self$etag <- etag
			self$items <- items
			self$next_link <- next_link
			self$next_page_token <- next_page_token
			self$selfLink <- selfLink
			
			private$classFields <- c(
				kind, etag, items, next_link, next_page_token,
				selfLink
			)
		},

		toJSON = function() {
			fields <- sapply(private$classFields, function(field)
			{
				self[[field]]
			}, USE.NAMES = TRUE)
			
			jsonlite::toJSON(list("containerlist" = 
                     Filter(Negate(is.null), fields)), auto_unbox = TRUE)
		}
	),

	private = list(
		classFields = NULL
	),

	cloneable = FALSE
)

#' @export
Container <- R6::R6Class(

	"Container",

	public = list(
		uuid = NULL,
		etag = NULL,
		owner_uuid = NULL,
		created_at = NULL,
		modified_at = NULL,
		modified_by_client_uuid = NULL,
		modified_by_user_uuid = NULL,
		state = NULL,
		started_at = NULL,
		finished_at = NULL,
		log = NULL,
		environment = NULL,
		cwd = NULL,
		command = NULL,
		output_path = NULL,
		mounts = NULL,
		runtime_constraints = NULL,
		output = NULL,
		container_image = NULL,
		progress = NULL,
		priority = NULL,
		updated_at = NULL,
		exit_code = NULL,
		auth_uuid = NULL,
		locked_by_uuid = NULL,
		scheduling_parameters = NULL,

		initialize = function(
				uuid = NULL, etag = NULL, owner_uuid = NULL,
				created_at = NULL, modified_at = NULL, modified_by_client_uuid = NULL,
				modified_by_user_uuid = NULL, state = NULL,
				started_at = NULL, finished_at = NULL, log = NULL,
				environment = NULL, cwd = NULL, command = NULL,
				output_path = NULL, mounts = NULL, runtime_constraints = NULL,
				output = NULL, container_image = NULL, progress = NULL,
				priority = NULL, updated_at = NULL, exit_code = NULL,
				auth_uuid = NULL, locked_by_uuid = NULL,
				scheduling_parameters = NULL)
		{
			self$uuid <- uuid
			self$etag <- etag
			self$owner_uuid <- owner_uuid
			self$created_at <- created_at
			self$modified_at <- modified_at
			self$modified_by_client_uuid <- modified_by_client_uuid
			self$modified_by_user_uuid <- modified_by_user_uuid
			self$state <- state
			self$started_at <- started_at
			self$finished_at <- finished_at
			self$log <- log
			self$environment <- environment
			self$cwd <- cwd
			self$command <- command
			self$output_path <- output_path
			self$mounts <- mounts
			self$runtime_constraints <- runtime_constraints
			self$output <- output
			self$container_image <- container_image
			self$progress <- progress
			self$priority <- priority
			self$updated_at <- updated_at
			self$exit_code <- exit_code
			self$auth_uuid <- auth_uuid
			self$locked_by_uuid <- locked_by_uuid
			self$scheduling_parameters <- scheduling_parameters
			
			private$classFields <- c(
				uuid, etag, owner_uuid, created_at, modified_at,
				modified_by_client_uuid, modified_by_user_uuid,
				state, started_at, finished_at, log, environment,
				cwd, command, output_path, mounts, runtime_constraints,
				output, container_image, progress, priority,
				updated_at, exit_code, auth_uuid, locked_by_uuid,
				scheduling_parameters
			)
		},

		toJSON = function() {
			fields <- sapply(private$classFields, function(field)
			{
				self[[field]]
			}, USE.NAMES = TRUE)
			
			jsonlite::toJSON(list("container" = 
                     Filter(Negate(is.null), fields)), auto_unbox = TRUE)
		}
	),

	private = list(
		classFields = NULL
	),

	cloneable = FALSE
)

#' @export
HumanList <- R6::R6Class(

	"HumanList",

	public = list(
		kind = NULL,
		etag = NULL,
		items = NULL,
		next_link = NULL,
		next_page_token = NULL,
		selfLink = NULL,

		initialize = function(
				kind = NULL, etag = NULL, items = NULL, next_link = NULL,
				next_page_token = NULL, selfLink = NULL)
		{
			self$kind <- kind
			self$etag <- etag
			self$items <- items
			self$next_link <- next_link
			self$next_page_token <- next_page_token
			self$selfLink <- selfLink
			
			private$classFields <- c(
				kind, etag, items, next_link, next_page_token,
				selfLink
			)
		},

		toJSON = function() {
			fields <- sapply(private$classFields, function(field)
			{
				self[[field]]
			}, USE.NAMES = TRUE)
			
			jsonlite::toJSON(list("humanlist" = 
                     Filter(Negate(is.null), fields)), auto_unbox = TRUE)
		}
	),

	private = list(
		classFields = NULL
	),

	cloneable = FALSE
)

#' @export
Human <- R6::R6Class(

	"Human",

	public = list(
		uuid = NULL,
		etag = NULL,
		owner_uuid = NULL,
		modified_by_client_uuid = NULL,
		modified_by_user_uuid = NULL,
		modified_at = NULL,
		properties = NULL,
		created_at = NULL,
		updated_at = NULL,

		initialize = function(
				uuid = NULL, etag = NULL, owner_uuid = NULL,
				modified_by_client_uuid = NULL, modified_by_user_uuid = NULL,
				modified_at = NULL, properties = NULL, created_at = NULL,
				updated_at = NULL)
		{
			self$uuid <- uuid
			self$etag <- etag
			self$owner_uuid <- owner_uuid
			self$modified_by_client_uuid <- modified_by_client_uuid
			self$modified_by_user_uuid <- modified_by_user_uuid
			self$modified_at <- modified_at
			self$properties <- properties
			self$created_at <- created_at
			self$updated_at <- updated_at
			
			private$classFields <- c(
				uuid, etag, owner_uuid, modified_by_client_uuid,
				modified_by_user_uuid, modified_at, properties,
				created_at, updated_at
			)
		},

		toJSON = function() {
			fields <- sapply(private$classFields, function(field)
			{
				self[[field]]
			}, USE.NAMES = TRUE)
			
			jsonlite::toJSON(list("human" = 
                     Filter(Negate(is.null), fields)), auto_unbox = TRUE)
		}
	),

	private = list(
		classFields = NULL
	),

	cloneable = FALSE
)

#' @export
JobTaskList <- R6::R6Class(

	"JobTaskList",

	public = list(
		kind = NULL,
		etag = NULL,
		items = NULL,
		next_link = NULL,
		next_page_token = NULL,
		selfLink = NULL,

		initialize = function(
				kind = NULL, etag = NULL, items = NULL, next_link = NULL,
				next_page_token = NULL, selfLink = NULL)
		{
			self$kind <- kind
			self$etag <- etag
			self$items <- items
			self$next_link <- next_link
			self$next_page_token <- next_page_token
			self$selfLink <- selfLink
			
			private$classFields <- c(
				kind, etag, items, next_link, next_page_token,
				selfLink
			)
		},

		toJSON = function() {
			fields <- sapply(private$classFields, function(field)
			{
				self[[field]]
			}, USE.NAMES = TRUE)
			
			jsonlite::toJSON(list("jobtasklist" = 
                     Filter(Negate(is.null), fields)), auto_unbox = TRUE)
		}
	),

	private = list(
		classFields = NULL
	),

	cloneable = FALSE
)

#' @export
JobTask <- R6::R6Class(

	"JobTask",

	public = list(
		uuid = NULL,
		etag = NULL,
		owner_uuid = NULL,
		modified_by_client_uuid = NULL,
		modified_by_user_uuid = NULL,
		modified_at = NULL,
		job_uuid = NULL,
		sequence = NULL,
		parameters = NULL,
		output = NULL,
		progress = NULL,
		success = NULL,
		created_at = NULL,
		updated_at = NULL,
		created_by_job_task_uuid = NULL,
		qsequence = NULL,
		started_at = NULL,
		finished_at = NULL,

		initialize = function(
				uuid = NULL, etag = NULL, owner_uuid = NULL,
				modified_by_client_uuid = NULL, modified_by_user_uuid = NULL,
				modified_at = NULL, job_uuid = NULL, sequence = NULL,
				parameters = NULL, output = NULL, progress = NULL,
				success = NULL, created_at = NULL, updated_at = NULL,
				created_by_job_task_uuid = NULL, qsequence = NULL,
				started_at = NULL, finished_at = NULL)
		{
			self$uuid <- uuid
			self$etag <- etag
			self$owner_uuid <- owner_uuid
			self$modified_by_client_uuid <- modified_by_client_uuid
			self$modified_by_user_uuid <- modified_by_user_uuid
			self$modified_at <- modified_at
			self$job_uuid <- job_uuid
			self$sequence <- sequence
			self$parameters <- parameters
			self$output <- output
			self$progress <- progress
			self$success <- success
			self$created_at <- created_at
			self$updated_at <- updated_at
			self$created_by_job_task_uuid <- created_by_job_task_uuid
			self$qsequence <- qsequence
			self$started_at <- started_at
			self$finished_at <- finished_at
			
			private$classFields <- c(
				uuid, etag, owner_uuid, modified_by_client_uuid,
				modified_by_user_uuid, modified_at, job_uuid,
				sequence, parameters, output, progress, success,
				created_at, updated_at, created_by_job_task_uuid,
				qsequence, started_at, finished_at
			)
		},

		toJSON = function() {
			fields <- sapply(private$classFields, function(field)
			{
				self[[field]]
			}, USE.NAMES = TRUE)
			
			jsonlite::toJSON(list("jobtask" = 
                     Filter(Negate(is.null), fields)), auto_unbox = TRUE)
		}
	),

	private = list(
		classFields = NULL
	),

	cloneable = FALSE
)

#' @export
LinkList <- R6::R6Class(

	"LinkList",

	public = list(
		kind = NULL,
		etag = NULL,
		items = NULL,
		next_link = NULL,
		next_page_token = NULL,
		selfLink = NULL,

		initialize = function(
				kind = NULL, etag = NULL, items = NULL, next_link = NULL,
				next_page_token = NULL, selfLink = NULL)
		{
			self$kind <- kind
			self$etag <- etag
			self$items <- items
			self$next_link <- next_link
			self$next_page_token <- next_page_token
			self$selfLink <- selfLink
			
			private$classFields <- c(
				kind, etag, items, next_link, next_page_token,
				selfLink
			)
		},

		toJSON = function() {
			fields <- sapply(private$classFields, function(field)
			{
				self[[field]]
			}, USE.NAMES = TRUE)
			
			jsonlite::toJSON(list("linklist" = 
                     Filter(Negate(is.null), fields)), auto_unbox = TRUE)
		}
	),

	private = list(
		classFields = NULL
	),

	cloneable = FALSE
)

#' @export
Link <- R6::R6Class(

	"Link",

	public = list(
		uuid = NULL,
		etag = NULL,
		owner_uuid = NULL,
		created_at = NULL,
		modified_by_client_uuid = NULL,
		modified_by_user_uuid = NULL,
		modified_at = NULL,
		tail_uuid = NULL,
		link_class = NULL,
		name = NULL,
		head_uuid = NULL,
		properties = NULL,
		updated_at = NULL,

		initialize = function(
				uuid = NULL, etag = NULL, owner_uuid = NULL,
				created_at = NULL, modified_by_client_uuid = NULL,
				modified_by_user_uuid = NULL, modified_at = NULL,
				tail_uuid = NULL, link_class = NULL, name = NULL,
				head_uuid = NULL, properties = NULL, updated_at = NULL)
		{
			self$uuid <- uuid
			self$etag <- etag
			self$owner_uuid <- owner_uuid
			self$created_at <- created_at
			self$modified_by_client_uuid <- modified_by_client_uuid
			self$modified_by_user_uuid <- modified_by_user_uuid
			self$modified_at <- modified_at
			self$tail_uuid <- tail_uuid
			self$link_class <- link_class
			self$name <- name
			self$head_uuid <- head_uuid
			self$properties <- properties
			self$updated_at <- updated_at
			
			private$classFields <- c(
				uuid, etag, owner_uuid, created_at, modified_by_client_uuid,
				modified_by_user_uuid, modified_at, tail_uuid,
				link_class, name, head_uuid, properties,
				updated_at
			)
		},

		toJSON = function() {
			fields <- sapply(private$classFields, function(field)
			{
				self[[field]]
			}, USE.NAMES = TRUE)
			
			jsonlite::toJSON(list("link" = 
                     Filter(Negate(is.null), fields)), auto_unbox = TRUE)
		}
	),

	private = list(
		classFields = NULL
	),

	cloneable = FALSE
)

#' @export
JobList <- R6::R6Class(

	"JobList",

	public = list(
		kind = NULL,
		etag = NULL,
		items = NULL,
		next_link = NULL,
		next_page_token = NULL,
		selfLink = NULL,

		initialize = function(
				kind = NULL, etag = NULL, items = NULL, next_link = NULL,
				next_page_token = NULL, selfLink = NULL)
		{
			self$kind <- kind
			self$etag <- etag
			self$items <- items
			self$next_link <- next_link
			self$next_page_token <- next_page_token
			self$selfLink <- selfLink
			
			private$classFields <- c(
				kind, etag, items, next_link, next_page_token,
				selfLink
			)
		},

		toJSON = function() {
			fields <- sapply(private$classFields, function(field)
			{
				self[[field]]
			}, USE.NAMES = TRUE)
			
			jsonlite::toJSON(list("joblist" = 
                     Filter(Negate(is.null), fields)), auto_unbox = TRUE)
		}
	),

	private = list(
		classFields = NULL
	),

	cloneable = FALSE
)

#' @export
Job <- R6::R6Class(

	"Job",

	public = list(
		uuid = NULL,
		etag = NULL,
		owner_uuid = NULL,
		modified_by_client_uuid = NULL,
		modified_by_user_uuid = NULL,
		modified_at = NULL,
		submit_id = NULL,
		script = NULL,
		script_version = NULL,
		script_parameters = NULL,
		cancelled_by_client_uuid = NULL,
		cancelled_by_user_uuid = NULL,
		cancelled_at = NULL,
		started_at = NULL,
		finished_at = NULL,
		running = NULL,
		success = NULL,
		output = NULL,
		created_at = NULL,
		updated_at = NULL,
		is_locked_by_uuid = NULL,
		log = NULL,
		tasks_summary = NULL,
		runtime_constraints = NULL,
		nondeterministic = NULL,
		repository = NULL,
		supplied_script_version = NULL,
		docker_image_locator = NULL,
		priority = NULL,
		description = NULL,
		state = NULL,
		arvados_sdk_version = NULL,
		components = NULL,
		script_parameters_digest = NULL,

		initialize = function(
				uuid = NULL, etag = NULL, owner_uuid = NULL,
				modified_by_client_uuid = NULL, modified_by_user_uuid = NULL,
				modified_at = NULL, submit_id = NULL, script = NULL,
				script_version = NULL, script_parameters = NULL,
				cancelled_by_client_uuid = NULL, cancelled_by_user_uuid = NULL,
				cancelled_at = NULL, started_at = NULL, finished_at = NULL,
				running = NULL, success = NULL, output = NULL,
				created_at = NULL, updated_at = NULL, is_locked_by_uuid = NULL,
				log = NULL, tasks_summary = NULL, runtime_constraints = NULL,
				nondeterministic = NULL, repository = NULL,
				supplied_script_version = NULL, docker_image_locator = NULL,
				priority = NULL, description = NULL, state = NULL,
				arvados_sdk_version = NULL, components = NULL,
				script_parameters_digest = NULL)
		{
			self$uuid <- uuid
			self$etag <- etag
			self$owner_uuid <- owner_uuid
			self$modified_by_client_uuid <- modified_by_client_uuid
			self$modified_by_user_uuid <- modified_by_user_uuid
			self$modified_at <- modified_at
			self$submit_id <- submit_id
			self$script <- script
			self$script_version <- script_version
			self$script_parameters <- script_parameters
			self$cancelled_by_client_uuid <- cancelled_by_client_uuid
			self$cancelled_by_user_uuid <- cancelled_by_user_uuid
			self$cancelled_at <- cancelled_at
			self$started_at <- started_at
			self$finished_at <- finished_at
			self$running <- running
			self$success <- success
			self$output <- output
			self$created_at <- created_at
			self$updated_at <- updated_at
			self$is_locked_by_uuid <- is_locked_by_uuid
			self$log <- log
			self$tasks_summary <- tasks_summary
			self$runtime_constraints <- runtime_constraints
			self$nondeterministic <- nondeterministic
			self$repository <- repository
			self$supplied_script_version <- supplied_script_version
			self$docker_image_locator <- docker_image_locator
			self$priority <- priority
			self$description <- description
			self$state <- state
			self$arvados_sdk_version <- arvados_sdk_version
			self$components <- components
			self$script_parameters_digest <- script_parameters_digest
			
			private$classFields <- c(
				uuid, etag, owner_uuid, modified_by_client_uuid,
				modified_by_user_uuid, modified_at, submit_id,
				script, script_version, script_parameters,
				cancelled_by_client_uuid, cancelled_by_user_uuid,
				cancelled_at, started_at, finished_at, running,
				success, output, created_at, updated_at,
				is_locked_by_uuid, log, tasks_summary, runtime_constraints,
				nondeterministic, repository, supplied_script_version,
				docker_image_locator, priority, description,
				state, arvados_sdk_version, components, script_parameters_digest
			)
		},

		toJSON = function() {
			fields <- sapply(private$classFields, function(field)
			{
				self[[field]]
			}, USE.NAMES = TRUE)
			
			jsonlite::toJSON(list("job" = 
                     Filter(Negate(is.null), fields)), auto_unbox = TRUE)
		}
	),

	private = list(
		classFields = NULL
	),

	cloneable = FALSE
)

#' @export
KeepDiskList <- R6::R6Class(

	"KeepDiskList",

	public = list(
		kind = NULL,
		etag = NULL,
		items = NULL,
		next_link = NULL,
		next_page_token = NULL,
		selfLink = NULL,

		initialize = function(
				kind = NULL, etag = NULL, items = NULL, next_link = NULL,
				next_page_token = NULL, selfLink = NULL)
		{
			self$kind <- kind
			self$etag <- etag
			self$items <- items
			self$next_link <- next_link
			self$next_page_token <- next_page_token
			self$selfLink <- selfLink
			
			private$classFields <- c(
				kind, etag, items, next_link, next_page_token,
				selfLink
			)
		},

		toJSON = function() {
			fields <- sapply(private$classFields, function(field)
			{
				self[[field]]
			}, USE.NAMES = TRUE)
			
			jsonlite::toJSON(list("keepdisklist" = 
                     Filter(Negate(is.null), fields)), auto_unbox = TRUE)
		}
	),

	private = list(
		classFields = NULL
	),

	cloneable = FALSE
)

#' @export
KeepDisk <- R6::R6Class(

	"KeepDisk",

	public = list(
		uuid = NULL,
		etag = NULL,
		owner_uuid = NULL,
		modified_by_client_uuid = NULL,
		modified_by_user_uuid = NULL,
		modified_at = NULL,
		ping_secret = NULL,
		node_uuid = NULL,
		filesystem_uuid = NULL,
		bytes_total = NULL,
		bytes_free = NULL,
		is_readable = NULL,
		is_writable = NULL,
		last_read_at = NULL,
		last_write_at = NULL,
		last_ping_at = NULL,
		created_at = NULL,
		updated_at = NULL,
		keep_service_uuid = NULL,

		initialize = function(
				uuid = NULL, etag = NULL, owner_uuid = NULL,
				modified_by_client_uuid = NULL, modified_by_user_uuid = NULL,
				modified_at = NULL, ping_secret = NULL, node_uuid = NULL,
				filesystem_uuid = NULL, bytes_total = NULL,
				bytes_free = NULL, is_readable = NULL, is_writable = NULL,
				last_read_at = NULL, last_write_at = NULL,
				last_ping_at = NULL, created_at = NULL, updated_at = NULL,
				keep_service_uuid = NULL)
		{
			self$uuid <- uuid
			self$etag <- etag
			self$owner_uuid <- owner_uuid
			self$modified_by_client_uuid <- modified_by_client_uuid
			self$modified_by_user_uuid <- modified_by_user_uuid
			self$modified_at <- modified_at
			self$ping_secret <- ping_secret
			self$node_uuid <- node_uuid
			self$filesystem_uuid <- filesystem_uuid
			self$bytes_total <- bytes_total
			self$bytes_free <- bytes_free
			self$is_readable <- is_readable
			self$is_writable <- is_writable
			self$last_read_at <- last_read_at
			self$last_write_at <- last_write_at
			self$last_ping_at <- last_ping_at
			self$created_at <- created_at
			self$updated_at <- updated_at
			self$keep_service_uuid <- keep_service_uuid
			
			private$classFields <- c(
				uuid, etag, owner_uuid, modified_by_client_uuid,
				modified_by_user_uuid, modified_at, ping_secret,
				node_uuid, filesystem_uuid, bytes_total,
				bytes_free, is_readable, is_writable, last_read_at,
				last_write_at, last_ping_at, created_at,
				updated_at, keep_service_uuid
			)
		},

		toJSON = function() {
			fields <- sapply(private$classFields, function(field)
			{
				self[[field]]
			}, USE.NAMES = TRUE)
			
			jsonlite::toJSON(list("keepdisk" = 
                     Filter(Negate(is.null), fields)), auto_unbox = TRUE)
		}
	),

	private = list(
		classFields = NULL
	),

	cloneable = FALSE
)

#' @export
KeepServiceList <- R6::R6Class(

	"KeepServiceList",

	public = list(
		kind = NULL,
		etag = NULL,
		items = NULL,
		next_link = NULL,
		next_page_token = NULL,
		selfLink = NULL,

		initialize = function(
				kind = NULL, etag = NULL, items = NULL, next_link = NULL,
				next_page_token = NULL, selfLink = NULL)
		{
			self$kind <- kind
			self$etag <- etag
			self$items <- items
			self$next_link <- next_link
			self$next_page_token <- next_page_token
			self$selfLink <- selfLink
			
			private$classFields <- c(
				kind, etag, items, next_link, next_page_token,
				selfLink
			)
		},

		toJSON = function() {
			fields <- sapply(private$classFields, function(field)
			{
				self[[field]]
			}, USE.NAMES = TRUE)
			
			jsonlite::toJSON(list("keepservicelist" = 
                     Filter(Negate(is.null), fields)), auto_unbox = TRUE)
		}
	),

	private = list(
		classFields = NULL
	),

	cloneable = FALSE
)

#' @export
KeepService <- R6::R6Class(

	"KeepService",

	public = list(
		uuid = NULL,
		etag = NULL,
		owner_uuid = NULL,
		modified_by_client_uuid = NULL,
		modified_by_user_uuid = NULL,
		modified_at = NULL,
		service_host = NULL,
		service_port = NULL,
		service_ssl_flag = NULL,
		service_type = NULL,
		created_at = NULL,
		updated_at = NULL,
		read_only = NULL,

		initialize = function(
				uuid = NULL, etag = NULL, owner_uuid = NULL,
				modified_by_client_uuid = NULL, modified_by_user_uuid = NULL,
				modified_at = NULL, service_host = NULL,
				service_port = NULL, service_ssl_flag = NULL,
				service_type = NULL, created_at = NULL, updated_at = NULL,
				read_only = NULL)
		{
			self$uuid <- uuid
			self$etag <- etag
			self$owner_uuid <- owner_uuid
			self$modified_by_client_uuid <- modified_by_client_uuid
			self$modified_by_user_uuid <- modified_by_user_uuid
			self$modified_at <- modified_at
			self$service_host <- service_host
			self$service_port <- service_port
			self$service_ssl_flag <- service_ssl_flag
			self$service_type <- service_type
			self$created_at <- created_at
			self$updated_at <- updated_at
			self$read_only <- read_only
			
			private$classFields <- c(
				uuid, etag, owner_uuid, modified_by_client_uuid,
				modified_by_user_uuid, modified_at, service_host,
				service_port, service_ssl_flag, service_type,
				created_at, updated_at, read_only
			)
		},

		toJSON = function() {
			fields <- sapply(private$classFields, function(field)
			{
				self[[field]]
			}, USE.NAMES = TRUE)
			
			jsonlite::toJSON(list("keepservice" = 
                     Filter(Negate(is.null), fields)), auto_unbox = TRUE)
		}
	),

	private = list(
		classFields = NULL
	),

	cloneable = FALSE
)

#' @export
PipelineTemplateList <- R6::R6Class(

	"PipelineTemplateList",

	public = list(
		kind = NULL,
		etag = NULL,
		items = NULL,
		next_link = NULL,
		next_page_token = NULL,
		selfLink = NULL,

		initialize = function(
				kind = NULL, etag = NULL, items = NULL, next_link = NULL,
				next_page_token = NULL, selfLink = NULL)
		{
			self$kind <- kind
			self$etag <- etag
			self$items <- items
			self$next_link <- next_link
			self$next_page_token <- next_page_token
			self$selfLink <- selfLink
			
			private$classFields <- c(
				kind, etag, items, next_link, next_page_token,
				selfLink
			)
		},

		toJSON = function() {
			fields <- sapply(private$classFields, function(field)
			{
				self[[field]]
			}, USE.NAMES = TRUE)
			
			jsonlite::toJSON(list("pipelinetemplatelist" = 
                     Filter(Negate(is.null), fields)), auto_unbox = TRUE)
		}
	),

	private = list(
		classFields = NULL
	),

	cloneable = FALSE
)

#' @export
PipelineTemplate <- R6::R6Class(

	"PipelineTemplate",

	public = list(
		uuid = NULL,
		etag = NULL,
		owner_uuid = NULL,
		created_at = NULL,
		modified_by_client_uuid = NULL,
		modified_by_user_uuid = NULL,
		modified_at = NULL,
		name = NULL,
		components = NULL,
		updated_at = NULL,
		description = NULL,

		initialize = function(
				uuid = NULL, etag = NULL, owner_uuid = NULL,
				created_at = NULL, modified_by_client_uuid = NULL,
				modified_by_user_uuid = NULL, modified_at = NULL,
				name = NULL, components = NULL, updated_at = NULL,
				description = NULL)
		{
			self$uuid <- uuid
			self$etag <- etag
			self$owner_uuid <- owner_uuid
			self$created_at <- created_at
			self$modified_by_client_uuid <- modified_by_client_uuid
			self$modified_by_user_uuid <- modified_by_user_uuid
			self$modified_at <- modified_at
			self$name <- name
			self$components <- components
			self$updated_at <- updated_at
			self$description <- description
			
			private$classFields <- c(
				uuid, etag, owner_uuid, created_at, modified_by_client_uuid,
				modified_by_user_uuid, modified_at, name,
				components, updated_at, description
			)
		},

		toJSON = function() {
			fields <- sapply(private$classFields, function(field)
			{
				self[[field]]
			}, USE.NAMES = TRUE)
			
			jsonlite::toJSON(list("pipelinetemplate" = 
                     Filter(Negate(is.null), fields)), auto_unbox = TRUE)
		}
	),

	private = list(
		classFields = NULL
	),

	cloneable = FALSE
)

#' @export
PipelineInstanceList <- R6::R6Class(

	"PipelineInstanceList",

	public = list(
		kind = NULL,
		etag = NULL,
		items = NULL,
		next_link = NULL,
		next_page_token = NULL,
		selfLink = NULL,

		initialize = function(
				kind = NULL, etag = NULL, items = NULL, next_link = NULL,
				next_page_token = NULL, selfLink = NULL)
		{
			self$kind <- kind
			self$etag <- etag
			self$items <- items
			self$next_link <- next_link
			self$next_page_token <- next_page_token
			self$selfLink <- selfLink
			
			private$classFields <- c(
				kind, etag, items, next_link, next_page_token,
				selfLink
			)
		},

		toJSON = function() {
			fields <- sapply(private$classFields, function(field)
			{
				self[[field]]
			}, USE.NAMES = TRUE)
			
			jsonlite::toJSON(list("pipelineinstancelist" = 
                     Filter(Negate(is.null), fields)), auto_unbox = TRUE)
		}
	),

	private = list(
		classFields = NULL
	),

	cloneable = FALSE
)

#' @export
PipelineInstance <- R6::R6Class(

	"PipelineInstance",

	public = list(
		uuid = NULL,
		etag = NULL,
		owner_uuid = NULL,
		created_at = NULL,
		modified_by_client_uuid = NULL,
		modified_by_user_uuid = NULL,
		modified_at = NULL,
		pipeline_template_uuid = NULL,
		name = NULL,
		components = NULL,
		updated_at = NULL,
		properties = NULL,
		state = NULL,
		components_summary = NULL,
		started_at = NULL,
		finished_at = NULL,
		description = NULL,

		initialize = function(
				uuid = NULL, etag = NULL, owner_uuid = NULL,
				created_at = NULL, modified_by_client_uuid = NULL,
				modified_by_user_uuid = NULL, modified_at = NULL,
				pipeline_template_uuid = NULL, name = NULL,
				components = NULL, updated_at = NULL, properties = NULL,
				state = NULL, components_summary = NULL,
				started_at = NULL, finished_at = NULL, description = NULL)
		{
			self$uuid <- uuid
			self$etag <- etag
			self$owner_uuid <- owner_uuid
			self$created_at <- created_at
			self$modified_by_client_uuid <- modified_by_client_uuid
			self$modified_by_user_uuid <- modified_by_user_uuid
			self$modified_at <- modified_at
			self$pipeline_template_uuid <- pipeline_template_uuid
			self$name <- name
			self$components <- components
			self$updated_at <- updated_at
			self$properties <- properties
			self$state <- state
			self$components_summary <- components_summary
			self$started_at <- started_at
			self$finished_at <- finished_at
			self$description <- description
			
			private$classFields <- c(
				uuid, etag, owner_uuid, created_at, modified_by_client_uuid,
				modified_by_user_uuid, modified_at, pipeline_template_uuid,
				name, components, updated_at, properties,
				state, components_summary, started_at, finished_at,
				description
			)
		},

		toJSON = function() {
			fields <- sapply(private$classFields, function(field)
			{
				self[[field]]
			}, USE.NAMES = TRUE)
			
			jsonlite::toJSON(list("pipelineinstance" = 
                     Filter(Negate(is.null), fields)), auto_unbox = TRUE)
		}
	),

	private = list(
		classFields = NULL
	),

	cloneable = FALSE
)

#' @export
NodeList <- R6::R6Class(

	"NodeList",

	public = list(
		kind = NULL,
		etag = NULL,
		items = NULL,
		next_link = NULL,
		next_page_token = NULL,
		selfLink = NULL,

		initialize = function(
				kind = NULL, etag = NULL, items = NULL, next_link = NULL,
				next_page_token = NULL, selfLink = NULL)
		{
			self$kind <- kind
			self$etag <- etag
			self$items <- items
			self$next_link <- next_link
			self$next_page_token <- next_page_token
			self$selfLink <- selfLink
			
			private$classFields <- c(
				kind, etag, items, next_link, next_page_token,
				selfLink
			)
		},

		toJSON = function() {
			fields <- sapply(private$classFields, function(field)
			{
				self[[field]]
			}, USE.NAMES = TRUE)
			
			jsonlite::toJSON(list("nodelist" = 
                     Filter(Negate(is.null), fields)), auto_unbox = TRUE)
		}
	),

	private = list(
		classFields = NULL
	),

	cloneable = FALSE
)

#' @export
Node <- R6::R6Class(

	"Node",

	public = list(
		uuid = NULL,
		etag = NULL,
		owner_uuid = NULL,
		created_at = NULL,
		modified_by_client_uuid = NULL,
		modified_by_user_uuid = NULL,
		modified_at = NULL,
		slot_number = NULL,
		hostname = NULL,
		domain = NULL,
		ip_address = NULL,
		first_ping_at = NULL,
		last_ping_at = NULL,
		info = NULL,
		updated_at = NULL,
		properties = NULL,
		job_uuid = NULL,

		initialize = function(
				uuid = NULL, etag = NULL, owner_uuid = NULL,
				created_at = NULL, modified_by_client_uuid = NULL,
				modified_by_user_uuid = NULL, modified_at = NULL,
				slot_number = NULL, hostname = NULL, domain = NULL,
				ip_address = NULL, first_ping_at = NULL,
				last_ping_at = NULL, info = NULL, updated_at = NULL,
				properties = NULL, job_uuid = NULL)
		{
			self$uuid <- uuid
			self$etag <- etag
			self$owner_uuid <- owner_uuid
			self$created_at <- created_at
			self$modified_by_client_uuid <- modified_by_client_uuid
			self$modified_by_user_uuid <- modified_by_user_uuid
			self$modified_at <- modified_at
			self$slot_number <- slot_number
			self$hostname <- hostname
			self$domain <- domain
			self$ip_address <- ip_address
			self$first_ping_at <- first_ping_at
			self$last_ping_at <- last_ping_at
			self$info <- info
			self$updated_at <- updated_at
			self$properties <- properties
			self$job_uuid <- job_uuid
			
			private$classFields <- c(
				uuid, etag, owner_uuid, created_at, modified_by_client_uuid,
				modified_by_user_uuid, modified_at, slot_number,
				hostname, domain, ip_address, first_ping_at,
				last_ping_at, info, updated_at, properties,
				job_uuid
			)
		},

		toJSON = function() {
			fields <- sapply(private$classFields, function(field)
			{
				self[[field]]
			}, USE.NAMES = TRUE)
			
			jsonlite::toJSON(list("node" = 
                     Filter(Negate(is.null), fields)), auto_unbox = TRUE)
		}
	),

	private = list(
		classFields = NULL
	),

	cloneable = FALSE
)

#' @export
RepositoryList <- R6::R6Class(

	"RepositoryList",

	public = list(
		kind = NULL,
		etag = NULL,
		items = NULL,
		next_link = NULL,
		next_page_token = NULL,
		selfLink = NULL,

		initialize = function(
				kind = NULL, etag = NULL, items = NULL, next_link = NULL,
				next_page_token = NULL, selfLink = NULL)
		{
			self$kind <- kind
			self$etag <- etag
			self$items <- items
			self$next_link <- next_link
			self$next_page_token <- next_page_token
			self$selfLink <- selfLink
			
			private$classFields <- c(
				kind, etag, items, next_link, next_page_token,
				selfLink
			)
		},

		toJSON = function() {
			fields <- sapply(private$classFields, function(field)
			{
				self[[field]]
			}, USE.NAMES = TRUE)
			
			jsonlite::toJSON(list("repositorylist" = 
                     Filter(Negate(is.null), fields)), auto_unbox = TRUE)
		}
	),

	private = list(
		classFields = NULL
	),

	cloneable = FALSE
)

#' @export
Repository <- R6::R6Class(

	"Repository",

	public = list(
		uuid = NULL,
		etag = NULL,
		owner_uuid = NULL,
		modified_by_client_uuid = NULL,
		modified_by_user_uuid = NULL,
		modified_at = NULL,
		name = NULL,
		created_at = NULL,
		updated_at = NULL,

		initialize = function(
				uuid = NULL, etag = NULL, owner_uuid = NULL,
				modified_by_client_uuid = NULL, modified_by_user_uuid = NULL,
				modified_at = NULL, name = NULL, created_at = NULL,
				updated_at = NULL)
		{
			self$uuid <- uuid
			self$etag <- etag
			self$owner_uuid <- owner_uuid
			self$modified_by_client_uuid <- modified_by_client_uuid
			self$modified_by_user_uuid <- modified_by_user_uuid
			self$modified_at <- modified_at
			self$name <- name
			self$created_at <- created_at
			self$updated_at <- updated_at
			
			private$classFields <- c(
				uuid, etag, owner_uuid, modified_by_client_uuid,
				modified_by_user_uuid, modified_at, name,
				created_at, updated_at
			)
		},

		toJSON = function() {
			fields <- sapply(private$classFields, function(field)
			{
				self[[field]]
			}, USE.NAMES = TRUE)
			
			jsonlite::toJSON(list("repository" = 
                     Filter(Negate(is.null), fields)), auto_unbox = TRUE)
		}
	),

	private = list(
		classFields = NULL
	),

	cloneable = FALSE
)

#' @export
SpecimenList <- R6::R6Class(

	"SpecimenList",

	public = list(
		kind = NULL,
		etag = NULL,
		items = NULL,
		next_link = NULL,
		next_page_token = NULL,
		selfLink = NULL,

		initialize = function(
				kind = NULL, etag = NULL, items = NULL, next_link = NULL,
				next_page_token = NULL, selfLink = NULL)
		{
			self$kind <- kind
			self$etag <- etag
			self$items <- items
			self$next_link <- next_link
			self$next_page_token <- next_page_token
			self$selfLink <- selfLink
			
			private$classFields <- c(
				kind, etag, items, next_link, next_page_token,
				selfLink
			)
		},

		toJSON = function() {
			fields <- sapply(private$classFields, function(field)
			{
				self[[field]]
			}, USE.NAMES = TRUE)
			
			jsonlite::toJSON(list("specimenlist" = 
                     Filter(Negate(is.null), fields)), auto_unbox = TRUE)
		}
	),

	private = list(
		classFields = NULL
	),

	cloneable = FALSE
)

#' @export
Specimen <- R6::R6Class(

	"Specimen",

	public = list(
		uuid = NULL,
		etag = NULL,
		owner_uuid = NULL,
		created_at = NULL,
		modified_by_client_uuid = NULL,
		modified_by_user_uuid = NULL,
		modified_at = NULL,
		material = NULL,
		updated_at = NULL,
		properties = NULL,

		initialize = function(
				uuid = NULL, etag = NULL, owner_uuid = NULL,
				created_at = NULL, modified_by_client_uuid = NULL,
				modified_by_user_uuid = NULL, modified_at = NULL,
				material = NULL, updated_at = NULL, properties = NULL)
		{
			self$uuid <- uuid
			self$etag <- etag
			self$owner_uuid <- owner_uuid
			self$created_at <- created_at
			self$modified_by_client_uuid <- modified_by_client_uuid
			self$modified_by_user_uuid <- modified_by_user_uuid
			self$modified_at <- modified_at
			self$material <- material
			self$updated_at <- updated_at
			self$properties <- properties
			
			private$classFields <- c(
				uuid, etag, owner_uuid, created_at, modified_by_client_uuid,
				modified_by_user_uuid, modified_at, material,
				updated_at, properties
			)
		},

		toJSON = function() {
			fields <- sapply(private$classFields, function(field)
			{
				self[[field]]
			}, USE.NAMES = TRUE)
			
			jsonlite::toJSON(list("specimen" = 
                     Filter(Negate(is.null), fields)), auto_unbox = TRUE)
		}
	),

	private = list(
		classFields = NULL
	),

	cloneable = FALSE
)

#' @export
LogList <- R6::R6Class(

	"LogList",

	public = list(
		kind = NULL,
		etag = NULL,
		items = NULL,
		next_link = NULL,
		next_page_token = NULL,
		selfLink = NULL,

		initialize = function(
				kind = NULL, etag = NULL, items = NULL, next_link = NULL,
				next_page_token = NULL, selfLink = NULL)
		{
			self$kind <- kind
			self$etag <- etag
			self$items <- items
			self$next_link <- next_link
			self$next_page_token <- next_page_token
			self$selfLink <- selfLink
			
			private$classFields <- c(
				kind, etag, items, next_link, next_page_token,
				selfLink
			)
		},

		toJSON = function() {
			fields <- sapply(private$classFields, function(field)
			{
				self[[field]]
			}, USE.NAMES = TRUE)
			
			jsonlite::toJSON(list("loglist" = 
                     Filter(Negate(is.null), fields)), auto_unbox = TRUE)
		}
	),

	private = list(
		classFields = NULL
	),

	cloneable = FALSE
)

#' @export
Log <- R6::R6Class(

	"Log",

	public = list(
		uuid = NULL,
		etag = NULL,
		owner_uuid = NULL,
		modified_by_client_uuid = NULL,
		modified_by_user_uuid = NULL,
		object_uuid = NULL,
		event_at = NULL,
		event_type = NULL,
		summary = NULL,
		properties = NULL,
		created_at = NULL,
		updated_at = NULL,
		modified_at = NULL,
		object_owner_uuid = NULL,

		initialize = function(
				uuid = NULL, etag = NULL, owner_uuid = NULL,
				modified_by_client_uuid = NULL, modified_by_user_uuid = NULL,
				object_uuid = NULL, event_at = NULL, event_type = NULL,
				summary = NULL, properties = NULL, created_at = NULL,
				updated_at = NULL, modified_at = NULL, object_owner_uuid = NULL)
		{
			self$uuid <- uuid
			self$etag <- etag
			self$owner_uuid <- owner_uuid
			self$modified_by_client_uuid <- modified_by_client_uuid
			self$modified_by_user_uuid <- modified_by_user_uuid
			self$object_uuid <- object_uuid
			self$event_at <- event_at
			self$event_type <- event_type
			self$summary <- summary
			self$properties <- properties
			self$created_at <- created_at
			self$updated_at <- updated_at
			self$modified_at <- modified_at
			self$object_owner_uuid <- object_owner_uuid
			
			private$classFields <- c(
				uuid, etag, owner_uuid, modified_by_client_uuid,
				modified_by_user_uuid, object_uuid, event_at,
				event_type, summary, properties, created_at,
				updated_at, modified_at, object_owner_uuid
			)
		},

		toJSON = function() {
			fields <- sapply(private$classFields, function(field)
			{
				self[[field]]
			}, USE.NAMES = TRUE)
			
			jsonlite::toJSON(list("log" = 
                     Filter(Negate(is.null), fields)), auto_unbox = TRUE)
		}
	),

	private = list(
		classFields = NULL
	),

	cloneable = FALSE
)

#' @export
TraitList <- R6::R6Class(

	"TraitList",

	public = list(
		kind = NULL,
		etag = NULL,
		items = NULL,
		next_link = NULL,
		next_page_token = NULL,
		selfLink = NULL,

		initialize = function(
				kind = NULL, etag = NULL, items = NULL, next_link = NULL,
				next_page_token = NULL, selfLink = NULL)
		{
			self$kind <- kind
			self$etag <- etag
			self$items <- items
			self$next_link <- next_link
			self$next_page_token <- next_page_token
			self$selfLink <- selfLink
			
			private$classFields <- c(
				kind, etag, items, next_link, next_page_token,
				selfLink
			)
		},

		toJSON = function() {
			fields <- sapply(private$classFields, function(field)
			{
				self[[field]]
			}, USE.NAMES = TRUE)
			
			jsonlite::toJSON(list("traitlist" = 
                     Filter(Negate(is.null), fields)), auto_unbox = TRUE)
		}
	),

	private = list(
		classFields = NULL
	),

	cloneable = FALSE
)

#' @export
Trait <- R6::R6Class(

	"Trait",

	public = list(
		uuid = NULL,
		etag = NULL,
		owner_uuid = NULL,
		modified_by_client_uuid = NULL,
		modified_by_user_uuid = NULL,
		modified_at = NULL,
		name = NULL,
		properties = NULL,
		created_at = NULL,
		updated_at = NULL,

		initialize = function(
				uuid = NULL, etag = NULL, owner_uuid = NULL,
				modified_by_client_uuid = NULL, modified_by_user_uuid = NULL,
				modified_at = NULL, name = NULL, properties = NULL,
				created_at = NULL, updated_at = NULL)
		{
			self$uuid <- uuid
			self$etag <- etag
			self$owner_uuid <- owner_uuid
			self$modified_by_client_uuid <- modified_by_client_uuid
			self$modified_by_user_uuid <- modified_by_user_uuid
			self$modified_at <- modified_at
			self$name <- name
			self$properties <- properties
			self$created_at <- created_at
			self$updated_at <- updated_at
			
			private$classFields <- c(
				uuid, etag, owner_uuid, modified_by_client_uuid,
				modified_by_user_uuid, modified_at, name,
				properties, created_at, updated_at
			)
		},

		toJSON = function() {
			fields <- sapply(private$classFields, function(field)
			{
				self[[field]]
			}, USE.NAMES = TRUE)
			
			jsonlite::toJSON(list("trait" = 
                     Filter(Negate(is.null), fields)), auto_unbox = TRUE)
		}
	),

	private = list(
		classFields = NULL
	),

	cloneable = FALSE
)

#' @export
VirtualMachineList <- R6::R6Class(

	"VirtualMachineList",

	public = list(
		kind = NULL,
		etag = NULL,
		items = NULL,
		next_link = NULL,
		next_page_token = NULL,
		selfLink = NULL,

		initialize = function(
				kind = NULL, etag = NULL, items = NULL, next_link = NULL,
				next_page_token = NULL, selfLink = NULL)
		{
			self$kind <- kind
			self$etag <- etag
			self$items <- items
			self$next_link <- next_link
			self$next_page_token <- next_page_token
			self$selfLink <- selfLink
			
			private$classFields <- c(
				kind, etag, items, next_link, next_page_token,
				selfLink
			)
		},

		toJSON = function() {
			fields <- sapply(private$classFields, function(field)
			{
				self[[field]]
			}, USE.NAMES = TRUE)
			
			jsonlite::toJSON(list("virtualmachinelist" = 
                     Filter(Negate(is.null), fields)), auto_unbox = TRUE)
		}
	),

	private = list(
		classFields = NULL
	),

	cloneable = FALSE
)

#' @export
VirtualMachine <- R6::R6Class(

	"VirtualMachine",

	public = list(
		uuid = NULL,
		etag = NULL,
		owner_uuid = NULL,
		modified_by_client_uuid = NULL,
		modified_by_user_uuid = NULL,
		modified_at = NULL,
		hostname = NULL,
		created_at = NULL,
		updated_at = NULL,

		initialize = function(
				uuid = NULL, etag = NULL, owner_uuid = NULL,
				modified_by_client_uuid = NULL, modified_by_user_uuid = NULL,
				modified_at = NULL, hostname = NULL, created_at = NULL,
				updated_at = NULL)
		{
			self$uuid <- uuid
			self$etag <- etag
			self$owner_uuid <- owner_uuid
			self$modified_by_client_uuid <- modified_by_client_uuid
			self$modified_by_user_uuid <- modified_by_user_uuid
			self$modified_at <- modified_at
			self$hostname <- hostname
			self$created_at <- created_at
			self$updated_at <- updated_at
			
			private$classFields <- c(
				uuid, etag, owner_uuid, modified_by_client_uuid,
				modified_by_user_uuid, modified_at, hostname,
				created_at, updated_at
			)
		},

		toJSON = function() {
			fields <- sapply(private$classFields, function(field)
			{
				self[[field]]
			}, USE.NAMES = TRUE)
			
			jsonlite::toJSON(list("virtualmachine" = 
                     Filter(Negate(is.null), fields)), auto_unbox = TRUE)
		}
	),

	private = list(
		classFields = NULL
	),

	cloneable = FALSE
)

#' @export
WorkflowList <- R6::R6Class(

	"WorkflowList",

	public = list(
		kind = NULL,
		etag = NULL,
		items = NULL,
		next_link = NULL,
		next_page_token = NULL,
		selfLink = NULL,

		initialize = function(
				kind = NULL, etag = NULL, items = NULL, next_link = NULL,
				next_page_token = NULL, selfLink = NULL)
		{
			self$kind <- kind
			self$etag <- etag
			self$items <- items
			self$next_link <- next_link
			self$next_page_token <- next_page_token
			self$selfLink <- selfLink
			
			private$classFields <- c(
				kind, etag, items, next_link, next_page_token,
				selfLink
			)
		},

		toJSON = function() {
			fields <- sapply(private$classFields, function(field)
			{
				self[[field]]
			}, USE.NAMES = TRUE)
			
			jsonlite::toJSON(list("workflowlist" = 
                     Filter(Negate(is.null), fields)), auto_unbox = TRUE)
		}
	),

	private = list(
		classFields = NULL
	),

	cloneable = FALSE
)

#' @export
Workflow <- R6::R6Class(

	"Workflow",

	public = list(
		uuid = NULL,
		etag = NULL,
		owner_uuid = NULL,
		created_at = NULL,
		modified_at = NULL,
		modified_by_client_uuid = NULL,
		modified_by_user_uuid = NULL,
		name = NULL,
		description = NULL,
		definition = NULL,
		updated_at = NULL,

		initialize = function(
				uuid = NULL, etag = NULL, owner_uuid = NULL,
				created_at = NULL, modified_at = NULL, modified_by_client_uuid = NULL,
				modified_by_user_uuid = NULL, name = NULL,
				description = NULL, definition = NULL, updated_at = NULL)
		{
			self$uuid <- uuid
			self$etag <- etag
			self$owner_uuid <- owner_uuid
			self$created_at <- created_at
			self$modified_at <- modified_at
			self$modified_by_client_uuid <- modified_by_client_uuid
			self$modified_by_user_uuid <- modified_by_user_uuid
			self$name <- name
			self$description <- description
			self$definition <- definition
			self$updated_at <- updated_at
			
			private$classFields <- c(
				uuid, etag, owner_uuid, created_at, modified_at,
				modified_by_client_uuid, modified_by_user_uuid,
				name, description, definition, updated_at
			)
		},

		toJSON = function() {
			fields <- sapply(private$classFields, function(field)
			{
				self[[field]]
			}, USE.NAMES = TRUE)
			
			jsonlite::toJSON(list("workflow" = 
                     Filter(Negate(is.null), fields)), auto_unbox = TRUE)
		}
	),

	private = list(
		classFields = NULL
	),

	cloneable = FALSE
)

#' @export
GroupList <- R6::R6Class(

	"GroupList",

	public = list(
		kind = NULL,
		etag = NULL,
		items = NULL,
		next_link = NULL,
		next_page_token = NULL,
		selfLink = NULL,

		initialize = function(
				kind = NULL, etag = NULL, items = NULL, next_link = NULL,
				next_page_token = NULL, selfLink = NULL)
		{
			self$kind <- kind
			self$etag <- etag
			self$items <- items
			self$next_link <- next_link
			self$next_page_token <- next_page_token
			self$selfLink <- selfLink
			
			private$classFields <- c(
				kind, etag, items, next_link, next_page_token,
				selfLink
			)
		},

		toJSON = function() {
			fields <- sapply(private$classFields, function(field)
			{
				self[[field]]
			}, USE.NAMES = TRUE)
			
			jsonlite::toJSON(list("grouplist" = 
                     Filter(Negate(is.null), fields)), auto_unbox = TRUE)
		}
	),

	private = list(
		classFields = NULL
	),

	cloneable = FALSE
)

#' @export
Group <- R6::R6Class(

	"Group",

	public = list(
		uuid = NULL,
		etag = NULL,
		owner_uuid = NULL,
		created_at = NULL,
		modified_by_client_uuid = NULL,
		modified_by_user_uuid = NULL,
		modified_at = NULL,
		name = NULL,
		description = NULL,
		updated_at = NULL,
		group_class = NULL,
		trash_at = NULL,
		is_trashed = NULL,
		delete_at = NULL,

		initialize = function(
				uuid = NULL, etag = NULL, owner_uuid = NULL,
				created_at = NULL, modified_by_client_uuid = NULL,
				modified_by_user_uuid = NULL, modified_at = NULL,
				name = NULL, description = NULL, updated_at = NULL,
				group_class = NULL, trash_at = NULL, is_trashed = NULL,
				delete_at = NULL)
		{
			self$uuid <- uuid
			self$etag <- etag
			self$owner_uuid <- owner_uuid
			self$created_at <- created_at
			self$modified_by_client_uuid <- modified_by_client_uuid
			self$modified_by_user_uuid <- modified_by_user_uuid
			self$modified_at <- modified_at
			self$name <- name
			self$description <- description
			self$updated_at <- updated_at
			self$group_class <- group_class
			self$trash_at <- trash_at
			self$is_trashed <- is_trashed
			self$delete_at <- delete_at
			
			private$classFields <- c(
				uuid, etag, owner_uuid, created_at, modified_by_client_uuid,
				modified_by_user_uuid, modified_at, name,
				description, updated_at, group_class, trash_at,
				is_trashed, delete_at
			)
		},

		toJSON = function() {
			fields <- sapply(private$classFields, function(field)
			{
				self[[field]]
			}, USE.NAMES = TRUE)
			
			jsonlite::toJSON(list("group" = 
                     Filter(Negate(is.null), fields)), auto_unbox = TRUE)
		}
	),

	private = list(
		classFields = NULL
	),

	cloneable = FALSE
)

#' @export
UserAgreementList <- R6::R6Class(

	"UserAgreementList",

	public = list(
		kind = NULL,
		etag = NULL,
		items = NULL,
		next_link = NULL,
		next_page_token = NULL,
		selfLink = NULL,

		initialize = function(
				kind = NULL, etag = NULL, items = NULL, next_link = NULL,
				next_page_token = NULL, selfLink = NULL)
		{
			self$kind <- kind
			self$etag <- etag
			self$items <- items
			self$next_link <- next_link
			self$next_page_token <- next_page_token
			self$selfLink <- selfLink
			
			private$classFields <- c(
				kind, etag, items, next_link, next_page_token,
				selfLink
			)
		},

		toJSON = function() {
			fields <- sapply(private$classFields, function(field)
			{
				self[[field]]
			}, USE.NAMES = TRUE)
			
			jsonlite::toJSON(list("useragreementlist" = 
                     Filter(Negate(is.null), fields)), auto_unbox = TRUE)
		}
	),

	private = list(
		classFields = NULL
	),

	cloneable = FALSE
)

#' @export
UserAgreement <- R6::R6Class(

	"UserAgreement",

	public = list(
		uuid = NULL,
		etag = NULL,
		owner_uuid = NULL,
		created_at = NULL,
		modified_by_client_uuid = NULL,
		modified_by_user_uuid = NULL,
		modified_at = NULL,
		portable_data_hash = NULL,
		replication_desired = NULL,
		replication_confirmed_at = NULL,
		replication_confirmed = NULL,
		updated_at = NULL,
		manifest_text = NULL,
		name = NULL,
		description = NULL,
		properties = NULL,
		delete_at = NULL,
		file_names = NULL,
		trash_at = NULL,
		is_trashed = NULL,

		initialize = function(
				uuid = NULL, etag = NULL, owner_uuid = NULL,
				created_at = NULL, modified_by_client_uuid = NULL,
				modified_by_user_uuid = NULL, modified_at = NULL,
				portable_data_hash = NULL, replication_desired = NULL,
				replication_confirmed_at = NULL, replication_confirmed = NULL,
				updated_at = NULL, manifest_text = NULL,
				name = NULL, description = NULL, properties = NULL,
				delete_at = NULL, file_names = NULL, trash_at = NULL,
				is_trashed = NULL)
		{
			self$uuid <- uuid
			self$etag <- etag
			self$owner_uuid <- owner_uuid
			self$created_at <- created_at
			self$modified_by_client_uuid <- modified_by_client_uuid
			self$modified_by_user_uuid <- modified_by_user_uuid
			self$modified_at <- modified_at
			self$portable_data_hash <- portable_data_hash
			self$replication_desired <- replication_desired
			self$replication_confirmed_at <- replication_confirmed_at
			self$replication_confirmed <- replication_confirmed
			self$updated_at <- updated_at
			self$manifest_text <- manifest_text
			self$name <- name
			self$description <- description
			self$properties <- properties
			self$delete_at <- delete_at
			self$file_names <- file_names
			self$trash_at <- trash_at
			self$is_trashed <- is_trashed
			
			private$classFields <- c(
				uuid, etag, owner_uuid, created_at, modified_by_client_uuid,
				modified_by_user_uuid, modified_at, portable_data_hash,
				replication_desired, replication_confirmed_at,
				replication_confirmed, updated_at, manifest_text,
				name, description, properties, delete_at,
				file_names, trash_at, is_trashed
			)
		},

		toJSON = function() {
			fields <- sapply(private$classFields, function(field)
			{
				self[[field]]
			}, USE.NAMES = TRUE)
			
			jsonlite::toJSON(list("useragreement" = 
                     Filter(Negate(is.null), fields)), auto_unbox = TRUE)
		}
	),

	private = list(
		classFields = NULL
	),

	cloneable = FALSE
)

