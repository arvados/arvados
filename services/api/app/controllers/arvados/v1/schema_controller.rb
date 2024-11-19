# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class Arvados::V1::SchemaController < ApplicationController
  skip_before_action :catch_redirect_hint
  skip_before_action :find_objects_for_index
  skip_before_action :find_object_by_uuid
  skip_before_action :load_filters_param
  skip_before_action :load_limit_offset_order_params
  skip_before_action :load_select_param
  skip_before_action :load_read_auths
  skip_before_action :load_where_param
  skip_before_action :render_404_if_no_object
  skip_before_action :require_auth_scope

  include DbCurrentTime

  def index
    expires_in 24.hours, public: true
    send_json discovery_doc
  end

  protected

  ActionNameMap = {
    'destroy' => 'delete',
    'index' => 'list',
    'show' => 'get',
  }

  HttpMethodDescriptionMap = {
    "DELETE" => "delete",
    "GET" => "query",
    "POST" => "update",
    "PUT" => "create",
  }

  ModelHumanNameMap = {
    # The discovery document has code to humanize most model names.
    # These are exceptions that require some capitalization.
    "ApiClientAuthorization" => "API client authorization",
    "KeepService" => "Keep service",
  }

  SchemaDescriptionMap = {
    # This hash contains descriptions for everything in the schema.
    # Schemas are looked up by their model name.
    # Schema properties are looked up by "{model_name}.{property_name}"
    # and fall back to just the property name if that doesn't exist.
    "ApiClientAuthorization" => "Arvados API client authorization token

This resource represents an API token a user may use to authenticate an
Arvados API request.",
    "AuthorizedKey" => "Arvados authorized public key

This resource represents a public key a user may use to authenticate themselves
to services on the cluster. Its primary use today is to store SSH keys for
virtual machines (\"shell nodes\"). It may be extended to store other keys in
the future.",
    "Collection" => "Arvados data collection

A collection describes how a set of files is stored in data blocks in Keep,
along with associated metadata.",
    "ComputedPermission" => "Arvados computed permission

Computed permissions do not correspond directly to any Arvados resource, but
provide a simple way to query the entire graph of permissions granted to
users and groups.",
    "ContainerRequest" => "Arvados container request

A container request represents a user's request that Arvados do some compute
work, along with full details about what work should be done. Arvados will
attempt to fulfill the request by mapping it to a matching container record,
running the work on demand if necessary.",
    "Container" => "Arvados container record

A container represents compute work that has been or should be dispatched,
along with its results. A container can satisfy one or more container requests.",
    "Group" => "Arvados group

Groups provide a way to organize users or data together, depending on their
`group_class`.",
    "KeepService" => "Arvados Keep service

This resource stores information about a single Keep service in this Arvados
cluster that clients can contact to retrieve and store data.",
    "Link" => "Arvados object link

A link provides a way to define relationships between Arvados objects,
depending on their `link_class`.",
    "Log" => "Arvados log record

This resource represents a single log record about an event in this Arvados
cluster. Some individual Arvados services create log records. Users can also
create custom logs.",
    "UserAgreement" => "Arvados user agreement

A user agreement is a collection with terms that users must agree to before
they can use this Arvados cluster.",
    "User" => "Arvados user

A user represents a single individual or role who may be authorized to access
this Arvados cluster.",
    "VirtualMachine" => "Arvados virtual machine (\"shell node\")

This resource stores information about a virtual machine or \"shell node\"
hosted on this Arvados cluster where users can log in and use preconfigured
Arvados client tools.",
    "Workflow" => "Arvados workflow

A workflow contains workflow definition source code that Arvados can execute
along with associated metadata for users.",

    # This section contains:
    # * attributes shared across most resources
    # * attributes shared across Collections and UserAgreements
    # * attributes shared across Containers and ContainerRequests
    "command" =>
    "An array of strings that defines the command that the dispatcher should
execute inside this container.",
    "container_image" =>
    "The portable data hash of the Arvados collection that contains the image
to use for this container.",
    "created_at" => "The time this %s was created.",
    "current_version_uuid" => "The UUID of the current version of this %s.",
    "cwd" =>
    "A string that the defines the working directory that the dispatcher should
use when it executes the command inside this container.",
    "delete_at" => "The time this %s will be permanently deleted.",
    "description" =>
    "A longer HTML description of this %s assigned by a user.
Allowed HTML tags are `a`, `b`, `blockquote`, `br`, `code`,
`del`, `dd`, `dl`, `dt`, `em`, `h1`, `h2`, `h3`, `h4`, `h5`, `h6`, `hr`,
`i`, `img`, `kbd`, `li`, `ol`, `p`, `pre`,
`s`, `section`, `span`, `strong`, `sub`, `sup`, and `ul`.",
    "environment" =>
    "A hash of string keys and values that defines the environment variables
for the dispatcher to set when it executes this container.",
    "file_count" =>
    "The number of files represented in this %s's `manifest_text`.
This attribute is read-only.",
    "file_size_total" =>
    "The total size in bytes of files represented in this %s's `manifest_text`.
This attribute is read-only.",
    "is_trashed" => "A boolean flag to indicate whether or not this %s is trashed.",
    "manifest_text" =>
    "The manifest text that describes how files are constructed from data blocks
in this %s. Refer to the [manifest format][] reference for details.

[manifest format]: https://doc.arvados.org/architecture/manifest-format.html

",
    "modified_at" => "The time this %s was last updated.",
    "modified_by_user_uuid" => "The UUID of the user that last updated this %s.",
    "mounts" =>
    "A hash where each key names a directory inside this container, and its
value is an object that defines the mount source for that directory. Refer
to the [mount types reference][] for details.

[mount types reference]: https://doc.arvados.org/api/methods/containers.html#mount_types

",
    "name" => "The name of this %s assigned by a user.",
    "output_glob" =>
    "An array of strings of shell-style glob patterns that define which file(s)
and subdirectory(ies) under the `output_path` directory should be recorded in
the container's final output. Refer to the [glob patterns reference][] for details.

[glob patterns reference]: https://doc.arvados.org/api/methods/containers.html#glob_patterns

",
    "output_path" =>
    "A string that defines the file or directory path where the command
writes output that should be saved from this container.",
    "output_properties" =>
"A hash of arbitrary metadata to set on the output collection of this %s.
Some keys may be reserved by Arvados or defined by a configured vocabulary.
Refer to the [metadata properties reference][] for details.

[metadata properties reference]: https://doc.arvados.org/api/properties.html

",
    "output_storage_classes" =>
    "An array of strings identifying the storage class(es) that should be set
on the output collection of this %s. Storage classes are configured by
the cluster administrator.",
    "owner_uuid" => "The UUID of the user or group that owns this %s.",
    "portable_data_hash" =>
    "The portable data hash of this %s. This string provides a unique
and stable reference to these contents.",
    "preserve_version" =>
    "A boolean flag to indicate whether this specific version of this %s
should be persisted in cluster storage.",
    "priority" =>
    "An integer between 0 and 1000 (inclusive) that represents this %s's
scheduling priority. 0 represents a request to be cancelled. Higher
values represent higher priority. Refer to the [priority reference][] for details.

[priority reference]: https://doc.arvados.org/api/methods/container_requests.html#priority

",
    "properties" =>
    "A hash of arbitrary metadata for this %s.
Some keys may be reserved by Arvados or defined by a configured vocabulary.
Refer to the [metadata properties reference][] for details.

[metadata properties reference]: https://doc.arvados.org/api/properties.html

",
    "replication_confirmed" =>
    "The number of copies of data in this %s that the cluster has confirmed
exist in storage.",
    "replication_confirmed_at" =>
    "The last time the cluster confirmed that it met `replication_confirmed`
for this %s.",
    "replication_desired" =>
    "The number of copies that should be made for data in this %s.",
    "runtime_auth_scopes" =>
    "The `scopes` from the API client authorization token used to run this %s.",
    "runtime_constraints" =>
    "A hash that identifies compute resources this container requires to run
successfully. See the [runtime constraints reference][] for details.

[runtime constraints reference]: https://doc.arvados.org/api/methods/containers.html#runtime_constraints

",
    "runtime_token" =>
    "The `api_token` from an Arvados API client authorization token that a
dispatcher should use to set up this container.",
    "runtime_user_uuid" =>
    "The UUID of the Arvados user associated with the API client authorization
token used to run this container.",
    "secret_mounts" =>
    "A hash like `mounts`, but this attribute is only available through a
dedicated API before the container is run.",
    "scheduling_parameters" =>
    "A hash of scheduling parameters that should be passed to the underlying
dispatcher when this container is run.
See the [scheduling parameters reference][] for details.

[scheduling parameters reference]: https://doc.arvados.org/api/methods/containers.html#scheduling_parameters

",
    "storage_classes_desired" =>
    "An array of strings identifying the storage class(es) that should be used
for data in this %s. Storage classes are configured by the cluster administrator.",
    "storage_classes_confirmed" =>
    "An array of strings identifying the storage class(es) the cluster has
confirmed have a copy of this %s's data.",
    "storage_classes_confirmed_at" =>
    "The last time the cluster confirmed that data was stored on the storage
class(es) in `storage_classes_confirmed`.",
    "trash_at" => "The time this %s will be trashed.",

    "ApiClientAuthorization.api_token" =>
    "The secret token that can be used to authorize Arvados API requests.",
    "ApiClientAuthorization.created_by_ip_address" =>
    "The IP address of the client that created this token.",
    "ApiClientAuthorization.expires_at" =>
    "The time after which this token is no longer valid for authorization.",
    "ApiClientAuthorization.last_used_at" =>
    "The last time this token was used to authorize a request.",
    "ApiClientAuthorization.last_used_by_ip_address" =>
    "The IP address of the client that last used this token.",
    "ApiClientAuthorization.scopes" =>
    "An array of strings identifying HTTP methods and API paths this token is
authorized to use. Refer to the [scopes reference][] for details.

[scopes reference]: https://doc.arvados.org/api/tokens.html#scopes

",
    "version" =>
    "An integer that counts which version of a %s this record
represents. Refer to [collection versioning][] for details. This attribute is
read-only.

[collection versioning]: https://doc.arvados.org/user/topics/collection-versioning.html

",

    "AuthorizedKey.authorized_user_uuid" =>
    "The UUID of the Arvados user that is authorized by this key.",
    "AuthorizedKey.expires_at" =>
    "The time after which this key is no longer valid for authorization.",
    "AuthorizedKey.key_type" =>
    "A string identifying what type of service uses this key. Supported values are:

  * `\"SSH\"`

",
    "AuthorizedKey.public_key" =>
    "The full public key, in the format referenced by `key_type`.",

    "ComputedPermission.user_uuid" =>
    "The UUID of the Arvados user who has this permission.",
    "ComputedPermission.target_uuid" =>
    "The UUID of the Arvados object the user has access to.",
    "ComputedPermission.perm_level" =>
    "A string representing the user's level of access to the target object.
Possible values are:

  * `\"can_read\"`
  * `\"can_write\"`
  * `\"can_manage\"`

",

    "Container.auth_uuid" =>
    "The UUID of the Arvados API client authorization token that a dispatcher
should use to set up this container. This token is automatically created by
Arvados and this attribute automatically assigned unless a container is
created with `runtime_token`.",
    "Container.cost" =>
    "A float with the estimated cost of the cloud instance used to run this
container. The value is `0` if cost estimation is not available on this cluster.",
    "Container.exit_code" =>
    "An integer that records the Unix exit code of the `command` from a
finished container.",
    "Container.gateway_address" =>
    "A string with the address of the Arvados gateway server, in `HOST:PORT`
format. This is for internal use only.",
    "Container.interactive_session_started" =>
    "This flag is set true if any user starts an interactive shell inside the
running container.",
    "Container.lock_count" =>
    "The number of times this container has been locked by a dispatcher. This
may be greater than 1 if a dispatcher locks a container but then execution is
interrupted for any reason.",
    "Container.locked_by_uuid" =>
    "The UUID of the Arvados API client authorization token that successfully
locked this container in preparation to execute it.",
    "Container.log" =>
    "The portable data hash of the Arvados collection that contains this
container's logs.",
    "Container.output" =>
    "The portable data hash of the Arvados collection that contains this
container's output file(s).",
    "Container.progress" =>
    "A float between 0.0 and 1.0 (inclusive) that represents the container's
execution progress. This attribute is not implemented yet.",
    "Container.runtime_status" =>
    "A hash with status updates from a running container.
Refer to the [runtime status reference][] for details.

[runtime status reference]: https://doc.arvados.org/api/methods/containers.html#runtime_status

",
    "Container.subrequests_cost" =>
    "A float with the estimated cost of all cloud instances used to run this
container and all its subrequests. The value is `0` if cost estimation is not
available on this cluster.",
    "Container.state" =>
    "A string representing the container's current execution status. Possible
values are:

  * `\"Queued\"` --- This container has not been dispatched yet.
  * `\"Locked\"` --- A dispatcher has claimed this container in preparation to run it.
  * `\"Running\"` --- A dispatcher is running this container.
  * `\"Cancelled\"` --- Container execution has been cancelled by user request.
  * `\"Complete\"` --- A dispatcher ran this container to completion and recorded the results.

",

    "ContainerRequest.auth_uuid" =>
    "The UUID of the Arvados API client authorization token that a
dispatcher should use to set up a corresponding container. This token is
automatically created by Arvados and this attribute automatically assigned
unless a container request is created with `runtime_token`.",
    "ContainerRequest.container_count" =>
    "An integer that records how many times Arvados has attempted to dispatch
a container to fulfill this container request.",
    "ContainerRequest.container_count_max" =>
    "An integer that defines the maximum number of times Arvados should attempt
to dispatch a container to fulfill this container request.",
    "ContainerRequest.container_uuid" =>
    "The UUID of the container that fulfills this container request, if any.",
    "ContainerRequest.cumulative_cost" =>
    "A float with the estimated cost of all cloud instances used to run
container(s) to fulfill this container request and their subrequests.
The value is `0` if cost estimation is not available on this cluster.",
    "ContainerRequest.expires_at" =>
    "The time after which this %s will no longer be fulfilled.",
    "ContainerRequest.filters" =>
    "Filters that limit which existing containers are eligible to satisfy this
container request. This attribute is not implemented yet and should be null.",
    "ContainerRequest.log_uuid" =>
    "The UUID of the Arvados collection that contains logs for all the
container(s) that were dispatched to fulfill this container request.",
    "ContainerRequest.output_name" =>
    "The name to set on the output collection of this container request.",
    "ContainerRequest.output_ttl" =>
    "An integer in seconds. If greater than zero, when an output collection is
created for this container request, its `expires_at` attribute will be set this
far in the future.",
    "ContainerRequest.output_uuid" =>
    "The UUID of the Arvados collection that contains output for all the
container(s) that were dispatched to fulfill this container request.",
    "ContainerRequest.requesting_container_uuid" =>
    "The UUID of the container that created this container request, if any.",
    "ContainerRequest.state" =>
    "A string indicating where this container request is in its lifecycle.
Possible values are:

  * `\"Uncommitted\"` --- The container request has not been finalized and can still be edited.
  * `\"Committed\"` --- The container request is ready to be fulfilled.
  * `\"Final\"` --- The container request has been fulfilled or cancelled.

",
    "ContainerRequest.use_existing" =>
    "A boolean flag. If set, Arvados may choose to satisfy this container
request with an eligible container that already exists. Otherwise, Arvados will
satisfy this container request with a newer container, which will usually result
in the container running again.",

    "Group.group_class" =>
    "A string representing which type of group this is. One of:

  * `\"filter\"` --- A virtual project whose contents are selected dynamically by filters.
  * `\"project\"` --- An Arvados project that can contain collections,
    container records, workflows, and subprojects.
  * `\"role\"` --- A group of users that can be granted permissions in Arvados.

",
    "Group.frozen_by_uuid" =>
    "The UUID of the user that has frozen this group, if any. Frozen projects
cannot have their contents or metadata changed, even by admins.",

    "KeepService.service_host" => "The DNS hostname of this %s.",
    "KeepService.service_port" => "The TCP port where this %s listens.",
    "KeepService.service_ssl_flag" =>
    "A boolean flag that indicates whether or not this %s uses TLS/SSL.",
    "KeepService.service_type" =>
    "A string that describes which type of %s this is. One of:

  * `\"disk\"` --- A service that stores blocks on a local filesystem.
  * `\"blob\"` --- A service that stores blocks in a cloud object store.
  * `\"proxy\"` --- A keepproxy service.

",
    "KeepService.read_only" =>
    "A boolean flag. If set, this %s does not accept requests to write data
blocks; it only serves blocks it already has.",

    "Link.head_uuid" =>
    "The UUID of the Arvados object that is the originator or actor in this
relationship. May be null.",
    "Link.link_class" =>
    "A string that defines which kind of link this is. One of:

  * `\"permission\"` --- This link grants a permission to the user or group
    referenced by `head_uuid` to the object referenced by `tail_uuid`. The
    access level is set by `name`.
  * `\"star\"` --- This link represents a \"favorite.\" The user referenced
    by `head_uuid` wants quick access to the object referenced by `tail_uuid`.
  * `\"tag\"` --- This link represents an unstructured metadata tag. The object
    referenced by `tail_uuid` has the tag defined by `name`.

",
    "Link.name" =>
    "The primary value of this link. For `\"permission\"` links, this is one of
`\"can_read\"`, `\"can_write\"`, or `\"can_manage\"`.",
    "Link.tail_uuid" =>
    "The UUID of the Arvados object that is the target of this relationship.",

    "Log.id" =>
    "The serial number of this log. You can use this in filters to query logs
that were created before/after another.",
    "Log.event_type" =>
    "An arbitrary short string that classifies what type of log this is.",
    "Log.object_owner_uuid" =>
    "The `owner_uuid` of the object referenced by `object_uuid` at the time
this log was created.",
    "Log.object_uuid" =>
    "The UUID of the Arvados object that this log pertains to, such as a user
or container.",
    "Log.summary" =>
    "A text string that describes the logged event. This is the primary
attribute for simple logs.",

    "User.email" => "This user's email address.",
    "User.first_name" => "This user's first name.",
    "User.identity_url" =>
    "A URL that represents this user with the cluster's identity provider.",
    "User.is_active" =>
    "A boolean flag. If unset, this user is not permitted to make any Arvados
API requests.",
    "User.is_admin" =>
    "A boolean flag. If set, this user is an administrator of the Arvados
cluster, and automatically passes most permissions checks.",
    "User.last_name" => "This user's last name.",
    "User.prefs" => "A hash that stores cluster-wide user preferences.",
    "User.username" => "This user's Unix username on virtual machines.",

    "VirtualMachine.hostname" =>
    "The DNS hostname where users should access this %s.",

    "Workflow.definition" => "A string with the CWL source of this %s.",
  }

  def discovery_doc
    Rails.application.eager_load!
    remoteHosts = {}
    Rails.configuration.RemoteClusters.each {|k,v| if k != :"*" then remoteHosts[k] = v["Host"] end }
    discovery = {
      kind: "discovery#restDescription",
      discoveryVersion: "v1",
      id: "arvados:v1",
      name: "arvados",
      version: "v1",
      # format is YYYYMMDD, must be fixed width (needs to be lexically
      # sortable), updated manually, may be used by clients to
      # determine availability of API server features.
      revision: "20240627",
      source_version: AppVersion.hash,
      sourceVersion: AppVersion.hash, # source_version should be deprecated in the future
      packageVersion: AppVersion.package_version,
      generatedAt: db_current_time.iso8601,
      title: "Arvados API",
      description: "The API to interact with Arvados.",
      documentationLink: "http://doc.arvados.org/api/index.html",
      defaultCollectionReplication: Rails.configuration.Collections.DefaultReplication,
      protocol: "rest",
      baseUrl: root_url + "arvados/v1/",
      basePath: "/arvados/v1/",
      rootUrl: root_url,
      servicePath: "arvados/v1/",
      batchPath: "batch",
      uuidPrefix: Rails.configuration.ClusterID,
      defaultTrashLifetime: Rails.configuration.Collections.DefaultTrashLifetime,
      blobSignatureTtl: Rails.configuration.Collections.BlobSigningTTL,
      maxRequestSize: Rails.configuration.API.MaxRequestSize,
      maxItemsPerResponse: Rails.configuration.API.MaxItemsPerResponse,
      dockerImageFormats: Rails.configuration.Containers.SupportedDockerImageFormats.keys,
      crunchLogUpdatePeriod: Rails.configuration.Containers.Logging.LogUpdatePeriod,
      crunchLogUpdateSize: Rails.configuration.Containers.Logging.LogUpdateSize,
      remoteHosts: remoteHosts,
      remoteHostsViaDNS: Rails.configuration.RemoteClusters["*"].Proxy,
      websocketUrl: Rails.configuration.Services.Websocket.ExternalURL.to_s,
      workbenchUrl: Rails.configuration.Services.Workbench1.ExternalURL.to_s,
      workbench2Url: Rails.configuration.Services.Workbench2.ExternalURL.to_s,
      keepWebServiceUrl: Rails.configuration.Services.WebDAV.ExternalURL.to_s,
      parameters: {
        alt: {
          type: "string",
          description: "Data format for the response.",
          default: "json",
          enum: [
            "json"
          ],
          enumDescriptions: [
            "Responses with Content-Type of application/json"
          ],
          location: "query"
        },
        fields: {
          type: "string",
          description: "Selector specifying which fields to include in a partial response.",
          location: "query"
        },
        key: {
          type: "string",
          description: "API key. Your API key identifies your project and provides you with API access, quota, and reports. Required unless you provide an OAuth 2.0 token.",
          location: "query"
        },
        oauth_token: {
          type: "string",
          description: "OAuth 2.0 token for the current user.",
          location: "query"
        }
      },
      auth: {
        oauth2: {
          scopes: {
            "https://api.arvados.org/auth/arvados" => {
              description: "View and manage objects"
            },
            "https://api.arvados.org/auth/arvados.readonly" => {
              description: "View objects"
            }
          }
        }
      },
      schemas: {},
      resources: {}
    }

    ActiveRecord::Base.descendants.reject(&:abstract_class?).sort_by(&:to_s).each do |k|
      begin
        ctl_class = "Arvados::V1::#{k.to_s.pluralize}Controller".constantize
      rescue
        # No controller -> no discovery.
        next
      end
      human_name = ModelHumanNameMap[k.to_s] || k.to_s.underscore.humanize.downcase
      object_properties = {}
      k.columns.
        select { |col| k.selectable_attributes.include? col.name }.
        collect do |col|
        if k.serialized_attributes.has_key? col.name
          col_type = k.serialized_attributes[col.name].object_class.to_s
        elsif k.attribute_types[col.name].is_a? JsonbType::Hash
          col_type = Hash.to_s
        elsif k.attribute_types[col.name].is_a? JsonbType::Array
          col_type = Array.to_s
        else
          col_type = col.type
        end
        desc_fmt =
          SchemaDescriptionMap["#{k}.#{col.name}"] ||
          SchemaDescriptionMap[col.name] ||
          ""
        if k.attribute_types[col.name].type == :datetime
          desc_fmt += " The string encodes a UTC date and time in ISO 8601 format."
        end
        object_properties[col.name] = {
          description: desc_fmt % human_name,
          type: col_type,
        }
      end
      discovery[:schemas][k.to_s + 'List'] = {
        id: k.to_s + 'List',
        description: "A list of #{k} objects.",
        type: "object",
        properties: {
          kind: {
            type: "string",
            description: "Object type. Always arvados##{k.to_s.camelcase(:lower)}List.",
            default: "arvados##{k.to_s.camelcase(:lower)}List"
          },
          etag: {
            type: "string",
            description: "List cache version."
          },
          items: {
            type: "array",
            description: "An array of matching #{k} objects.",
            items: {
              "$ref" => k.to_s
            }
          },
        }
      }
      discovery[:schemas][k.to_s] = {
        id: k.to_s,
        description: SchemaDescriptionMap[k.to_s] || "Arvados #{human_name}.",
        type: "object",
        uuidPrefix: nil,
        properties: {
          etag: {
            type: "string",
            description: "Object cache version."
          }
        }.merge(object_properties)
      }
      if k.respond_to? :uuid_prefix
        discovery[:schemas][k.to_s][:uuidPrefix] ||= k.uuid_prefix
        discovery[:schemas][k.to_s][:properties][:uuid] ||= {
          type: "string",
          description: "This #{human_name}'s Arvados UUID, like `zzzzz-#{k.uuid_prefix}-12345abcde67890`."
        }
      end
      discovery[:resources][k.to_s.underscore.pluralize] = {
        methods: {
          get: {
            id: "arvados.#{k.to_s.underscore.pluralize}.get",
            path: "#{k.to_s.underscore.pluralize}/{uuid}",
            httpMethod: "GET",
            description: "Get a #{k.to_s} record by UUID.",
            parameters: {
              uuid: {
                type: "string",
                description: "The UUID of the #{k.to_s} to return.",
                required: true,
                location: "path"
              }
            },
            parameterOrder: [
              "uuid"
            ],
            response: {
              "$ref" => k.to_s
            },
            scopes: [
              "https://api.arvados.org/auth/arvados",
              "https://api.arvados.org/auth/arvados.readonly"
            ]
          },
          list: {
            id: "arvados.#{k.to_s.underscore.pluralize}.list",
            path: k.to_s.underscore.pluralize,
            httpMethod: "GET",
            description: "Retrieve a #{k.to_s}List.",
            parameters: {
            },
            response: {
              "$ref" => "#{k.to_s}List"
            },
            scopes: [
              "https://api.arvados.org/auth/arvados",
              "https://api.arvados.org/auth/arvados.readonly"
            ]
          },
          create: {
            id: "arvados.#{k.to_s.underscore.pluralize}.create",
            path: "#{k.to_s.underscore.pluralize}",
            httpMethod: "POST",
            description: "Create a new #{k.to_s}.",
            parameters: {},
            request: {
              required: true,
              properties: {
                k.to_s.underscore => {
                  "$ref" => k.to_s
                }
              }
            },
            response: {
              "$ref" => k.to_s
            },
            scopes: [
              "https://api.arvados.org/auth/arvados"
            ]
          },
          update: {
            id: "arvados.#{k.to_s.underscore.pluralize}.update",
            path: "#{k.to_s.underscore.pluralize}/{uuid}",
            httpMethod: "PUT",
            description: "Update attributes of an existing #{k.to_s}.",
            parameters: {
              uuid: {
                type: "string",
                description: "The UUID of the #{k.to_s} to update.",
                required: true,
                location: "path"
              }
            },
            request: {
              required: true,
              properties: {
                k.to_s.underscore => {
                  "$ref" => k.to_s
                }
              }
            },
            response: {
              "$ref" => k.to_s
            },
            scopes: [
              "https://api.arvados.org/auth/arvados"
            ]
          },
          delete: {
            id: "arvados.#{k.to_s.underscore.pluralize}.delete",
            path: "#{k.to_s.underscore.pluralize}/{uuid}",
            httpMethod: "DELETE",
            description: "Delete an existing #{k.to_s}.",
            parameters: {
              uuid: {
                type: "string",
                description: "The UUID of the #{k.to_s} to delete.",
                required: true,
                location: "path"
              }
            },
            response: {
              "$ref" => k.to_s
            },
            scopes: [
              "https://api.arvados.org/auth/arvados"
            ]
          }
        }
      }
      # Check for Rails routes that don't match the usual actions
      # listed above
      d_methods = discovery[:resources][k.to_s.underscore.pluralize][:methods]
      Rails.application.routes.routes.each do |route|
        action = route.defaults[:action]
        httpMethod = ['GET', 'POST', 'PUT', 'DELETE'].map { |method|
          method if route.verb.match(method)
        }.compact.first
        if httpMethod &&
          route.defaults[:controller] == 'arvados/v1/' + k.to_s.underscore.pluralize &&
          ctl_class.action_methods.include?(action)
          method_name = ActionNameMap[action] || action
          method_key = method_name.to_sym
          if !d_methods[method_key]
            method = {
              id: "arvados.#{k.to_s.underscore.pluralize}.#{method_name}",
              path: route.path.spec.to_s.sub('/arvados/v1/','').sub('(.:format)','').sub(/:(uu)?id/,'{uuid}'),
              httpMethod: httpMethod,
              description: ctl_class.send("_#{method_name}_method_description".to_sym),
              parameters: {},
              response: {
                "$ref" => (method_name == 'list' ? "#{k.to_s}List" : k.to_s)
              },
              scopes: [
                "https://api.arvados.org/auth/arvados"
              ]
            }
            route.segment_keys.each do |key|
              case key
              when :format
                next
              when :id, :uuid
                key = :uuid
                description = "The UUID of the #{k} to #{HttpMethodDescriptionMap[httpMethod]}."
              else
                description = ""
              end
              method[:parameters][key] = {
                type: "string",
                description: description,
                required: true,
                location: "path",
              }
            end
          else
            # We already built a generic method description, but we
            # might find some more required parameters through
            # introspection.
            method = d_methods[method_key]
          end
          if ctl_class.respond_to? "_#{action}_requires_parameters".to_sym
            ctl_class.send("_#{action}_requires_parameters".to_sym).each do |l, v|
              if v.is_a? Hash
                method[:parameters][l] = v
              else
                method[:parameters][l] = {}
              end
              if !method[:parameters][l][:default].nil?
                # The JAVA SDK is sensitive to all values being strings
                method[:parameters][l][:default] = method[:parameters][l][:default].to_s
              end
              method[:parameters][l][:type] ||= 'string'
              method[:parameters][l][:description] ||= ''
              method[:parameters][l][:location] = (route.segment_keys.include?(l) ? 'path' : 'query')
              if method[:parameters][l][:required].nil?
                method[:parameters][l][:required] = v != false
              end
            end
          end
          d_methods[method_key] = method
        end
      end
    end

    # The computed_permissions controller does not offer all of the
    # usual methods and attributes.  Modify discovery doc accordingly.
    discovery[:resources]['computed_permissions'][:methods].select! do |method|
      method == :list
    end
    discovery[:resources]['computed_permissions'][:methods][:list][:parameters].reject! do |param|
      [:cluster_id, :bypass_federation, :offset].include?(param)
    end
    discovery[:schemas]['ComputedPermission'].delete(:uuidPrefix)
    discovery[:schemas]['ComputedPermission'][:properties].reject! do |prop|
      [:uuid, :etag].include?(prop)
    end
    discovery[:schemas]['ComputedPermission'][:properties]['perm_level'][:type] = 'string'

    # The 'replace_files' and 'replace_segments' options are
    # implemented in lib/controller, not Rails -- we just need to add
    # them here so discovery-aware clients know how to validate them.
    [:create, :update].each do |action|
      discovery[:resources]['collections'][:methods][action][:parameters]['replace_files'] = {
        type: 'object',
        description:
          "Add, delete, and replace files and directories with new content
and/or content from other collections. Refer to the
[replace_files reference][] for details.

[replace_files reference]: https://doc.arvados.org/api/methods/collections.html#replace_files

",
        required: false,
        location: 'query',
        properties: {},
        additionalProperties: {type: 'string'},
      }
      discovery[:resources]['collections'][:methods][action][:parameters]['replace_segments'] = {
        type: 'object',
        description:
          "Replace existing block segments in the collection with new segments.
Refer to the [replace_segments reference][] for details.

[replace_segments reference]: https://doc.arvados.org/api/methods/collections.html#replace_segments

",
        required: false,
        location: 'query',
        properties: {},
        additionalProperties: {type: 'string'},
      }
    end

    discovery[:resources]['configs'] = {
      methods: {
        get: {
          id: "arvados.configs.get",
          path: "config",
          httpMethod: "GET",
          description: "Get this cluster's public configuration settings.",
          parameters: {
          },
          parameterOrder: [
          ],
          response: {
          },
          scopes: [
            "https://api.arvados.org/auth/arvados",
            "https://api.arvados.org/auth/arvados.readonly"
          ]
        },
      }
    }

    discovery[:resources]['vocabularies'] = {
      methods: {
        get: {
          id: "arvados.vocabularies.get",
          path: "vocabulary",
          httpMethod: "GET",
          description: "Get this cluster's configured vocabulary definition.

Refer to [metadata vocabulary documentation][] for details.

[metadata vocabulary documentation]: https://doc.aravdos.org/admin/metadata-vocabulary.html

",
          parameters: {
          },
          parameterOrder: [
          ],
          response: {
          },
          scopes: [
            "https://api.arvados.org/auth/arvados",
            "https://api.arvados.org/auth/arvados.readonly"
          ]
        },
      }
    }

    discovery[:resources]['sys'] = {
      methods: {
        get: {
          id: "arvados.sys.trash_sweep",
          path: "sys/trash_sweep",
          httpMethod: "POST",
          description:
            "Run scheduled data trash and sweep operations across this cluster's Keep services.",
          parameters: {
          },
          parameterOrder: [
          ],
          response: {
          },
          scopes: [
            "https://api.arvados.org/auth/arvados",
            "https://api.arvados.org/auth/arvados.readonly"
          ]
        },
      }
    }

    Rails.configuration.API.DisabledAPIs.each do |method, _|
      ctrl, action = method.to_s.split('.', 2)
      next if ctrl.in?(['api_clients', 'job_tasks', 'jobs', 'keep_disks', 'nodes', 'pipeline_instances', 'pipeline_templates', 'repositories'])
      discovery[:resources][ctrl][:methods].delete(action.to_sym)
    end
    discovery
  end
end
