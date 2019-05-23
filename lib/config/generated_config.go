// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

package config

var DefaultYAML = []byte(`# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

# Do not use this file for site configuration. Create
# /etc/arvados/config.yml instead.
#
# The order of precedence (highest to lowest):
# 1. Legacy component-specific config files (deprecated)
# 2. /etc/arvados/config.yml
# 3. config.default.yml

Clusters:
  xxxxx:
    SystemRootToken: ""

    # Token to be included in all healthcheck requests. Disabled by default.
    # Server expects request header of the format "Authorization: Bearer xxx"
    ManagementToken: ""

    Services:
      RailsAPI:
        InternalURLs: {}
      GitHTTP:
        InternalURLs: {}
        ExternalURL: ""
      Keepstore:
        InternalURLs: {}
      Controller:
        InternalURLs: {}
        ExternalURL: ""
      Websocket:
        InternalURLs: {}
        ExternalURL: ""
      Keepbalance:
        InternalURLs: {}
      GitHTTP:
        InternalURLs: {}
        ExternalURL: ""
      GitSSH:
        ExternalURL: ""
      DispatchCloud:
        InternalURLs: {}
      SSO:
        ExternalURL: ""
      Keepproxy:
        InternalURLs: {}
        ExternalURL: ""
      WebDAV:
        InternalURLs: {}
        ExternalURL: ""
      WebDAVDownload:
        InternalURLs: {}
        ExternalURL: ""
      Keepstore:
        InternalURLs: {}
      Composer:
        ExternalURL: ""
      WebShell:
        ExternalURL: ""
      Workbench1:
        InternalURLs: {}
        ExternalURL: ""
      Workbench2:
        ExternalURL: ""
    PostgreSQL:
      # max concurrent connections per arvados server daemon
      ConnectionPool: 32
      Connection:
        # All parameters here are passed to the PG client library in a connection string;
        # see https://www.postgresql.org/docs/current/static/libpq-connect.html#LIBPQ-PARAMKEYWORDS
        Host: ""
        Port: ""
        User: ""
        Password: ""
        DBName: ""
    API:
      # Maximum size (in bytes) allowed for a single API request.  This
      # limit is published in the discovery document for use by clients.
      # Note: You must separately configure the upstream web server or
      # proxy to actually enforce the desired maximum request size on the
      # server side.
      MaxRequestSize: 134217728

      # Limit the number of bytes read from the database during an index
      # request (by retrieving and returning fewer rows than would
      # normally be returned in a single response).
      # Note 1: This setting never reduces the number of returned rows to
      # zero, no matter how big the first data row is.
      # Note 2: Currently, this is only checked against a specific set of
      # columns that tend to get large (collections.manifest_text,
      # containers.mounts, workflows.definition). Other fields (e.g.,
      # "properties" hashes) are not counted against this limit.
      MaxIndexDatabaseRead: 134217728

      # Maximum number of items to return when responding to a APIs that
      # can return partial result sets using limit and offset parameters
      # (e.g., *.index, groups.contents). If a request specifies a "limit"
      # parameter higher than this value, this value is used instead.
      MaxItemsPerResponse: 1000

      # API methods to disable. Disabled methods are not listed in the
      # discovery document, and respond 404 to all requests.
      # Example: ["jobs.create", "pipeline_instances.create"]
      DisabledAPIs: []

      # Interval (seconds) between asynchronous permission view updates. Any
      # permission-updating API called with the 'async' parameter schedules a an
      # update on the permission view in the future, if not already scheduled.
      AsyncPermissionsUpdateInterval: 20

      # Maximum number of concurrent outgoing requests to make while
      # serving a single incoming multi-cluster (federated) request.
      MaxRequestAmplification: 4

      # RailsSessionSecretToken is a string of alphanumeric characters
      # used by Rails to sign session tokens. IMPORTANT: This is a
      # site secret. It should be at least 50 characters.
      RailsSessionSecretToken: ""

    Users:
      # Config parameters to automatically setup new users.  If enabled,
      # this users will be able to self-activate.  Enable this if you want
      # to run an open instance where anyone can create an account and use
      # the system without requiring manual approval.
      #
      # The params auto_setup_new_users_with_* are meaningful only when auto_setup_new_users is turned on.
      # auto_setup_name_blacklist is a list of usernames to be blacklisted for auto setup.
      AutoSetupNewUsers: false
      AutoSetupNewUsersWithVmUUID: ""
      AutoSetupNewUsersWithRepository: false
      AutoSetupUsernameBlacklist: [arvados, git, gitolite, gitolite-admin, root, syslog]

      # When new_users_are_active is set to true, new users will be active
      # immediately.  This skips the "self-activate" step which enforces
      # user agreements.  Should only be enabled for development.
      NewUsersAreActive: false

      # The e-mail address of the user you would like to become marked as an admin
      # user on their first login.
      # In the default configuration, authentication happens through the Arvados SSO
      # server, which uses OAuth2 against Google's servers, so in that case this
      # should be an address associated with a Google account.
      AutoAdminUserWithEmail: ""

      # If auto_admin_first_user is set to true, the first user to log in when no
      # other admin users exist will automatically become an admin user.
      AutoAdminFirstUser: false

      # Email address to notify whenever a user creates a profile for the
      # first time
      UserProfileNotificationAddress: ""
      AdminNotifierEmailFrom: arvados@example.com
      EmailSubjectPrefix: "[ARVADOS] "
      UserNotifierEmailFrom: arvados@example.com
      NewUserNotificationRecipients: []
      NewInactiveUserNotificationRecipients: []

    AuditLogs:
      # Time to keep audit logs, in seconds. (An audit log is a row added
      # to the "logs" table in the PostgreSQL database each time an
      # Arvados object is created, modified, or deleted.)
      #
      # Currently, websocket event notifications rely on audit logs, so
      # this should not be set lower than 300 (5 minutes).
      MaxAge: 336h

      # Maximum number of log rows to delete in a single SQL transaction.
      #
      # If max_audit_log_delete_batch is 0, log entries will never be
      # deleted by Arvados. Cleanup can be done by an external process
      # without affecting any Arvados system processes, as long as very
      # recent (<5 minutes old) logs are not deleted.
      #
      # 100000 is a reasonable batch size for most sites.
      MaxDeleteBatch: 0

      # Attributes to suppress in events and audit logs.  Notably,
      # specifying ["manifest_text"] here typically makes the database
      # smaller and faster.
      #
      # Warning: Using any non-empty value here can have undesirable side
      # effects for any client or component that relies on event logs.
      # Use at your own risk.
      UnloggedAttributes: []

    SystemLogs:
      # Maximum characters of (JSON-encoded) query parameters to include
      # in each request log entry. When params exceed this size, they will
      # be JSON-encoded, truncated to this size, and logged as
      # params_truncated.
      MaxRequestLogParamsSize: 2000

    Collections:
      # Allow clients to create collections by providing a manifest with
      # unsigned data blob locators. IMPORTANT: This effectively disables
      # access controls for data stored in Keep: a client who knows a hash
      # can write a manifest that references the hash, pass it to
      # collections.create (which will create a permission link), use
      # collections.get to obtain a signature for that data locator, and
      # use that signed locator to retrieve the data from Keep. Therefore,
      # do not turn this on if your users expect to keep data private from
      # one another!
      BlobSigning: true

      # blob_signing_key is a string of alphanumeric characters used to
      # generate permission signatures for Keep locators. It must be
      # identical to the permission key given to Keep. IMPORTANT: This is
      # a site secret. It should be at least 50 characters.
      #
      # Modifying blob_signing_key will invalidate all existing
      # signatures, which can cause programs to fail (e.g., arv-put,
      # arv-get, and Crunch jobs).  To avoid errors, rotate keys only when
      # no such processes are running.
      BlobSigningKey: ""

      # Default replication level for collections. This is used when a
      # collection's replication_desired attribute is nil.
      DefaultReplication: 2

      # Lifetime (in seconds) of blob permission signatures generated by
      # the API server. This determines how long a client can take (after
      # retrieving a collection record) to retrieve the collection data
      # from Keep. If the client needs more time than that (assuming the
      # collection still has the same content and the relevant user/token
      # still has permission) the client can retrieve the collection again
      # to get fresh signatures.
      #
      # This must be exactly equal to the -blob-signature-ttl flag used by
      # keepstore servers.  Otherwise, reading data blocks and saving
      # collections will fail with HTTP 403 permission errors.
      #
      # Modifying blob_signature_ttl invalidates existing signatures; see
      # blob_signing_key note above.
      #
      # The default is 2 weeks.
      BlobSigningTTL: 336h

      # Default lifetime for ephemeral collections: 2 weeks. This must not
      # be less than blob_signature_ttl.
      DefaultTrashLifetime: 336h

      # Interval (seconds) between trash sweeps. During a trash sweep,
      # collections are marked as trash if their trash_at time has
      # arrived, and deleted if their delete_at time has arrived.
      TrashSweepInterval: 60

      # If true, enable collection versioning.
      # When a collection's preserve_version field is true or the current version
      # is older than the amount of seconds defined on preserve_version_if_idle,
      # a snapshot of the collection's previous state is created and linked to
      # the current collection.
      CollectionVersioning: false

      #   0 = auto-create a new version on every update.
      #  -1 = never auto-create new versions.
      # > 0 = auto-create a new version when older than the specified number of seconds.
      PreserveVersionIfIdle: -1

    Login:
      # These settings are provided by your OAuth2 provider (e.g.,
      # sso-provider).
      ProviderAppSecret: ""
      ProviderAppID: ""

    Git:
      # Git repositories must be readable by api server, or you won't be
      # able to submit crunch jobs. To pass the test suites, put a clone
      # of the arvados tree in {git_repositories_dir}/arvados.git or
      # {git_repositories_dir}/arvados/.git
      Repositories: /var/lib/arvados/git/repositories

    TLS:
      Insecure: false

    Containers:
      # List of supported Docker Registry image formats that compute nodes
      # are able to use. ` + "`" + `arv keep docker` + "`" + ` will error out if a user tries
      # to store an image with an unsupported format. Use an empty array
      # to skip the compatibility check (and display a warning message to
      # that effect).
      #
      # Example for sites running docker < 1.10: ["v1"]
      # Example for sites running docker >= 1.10: ["v2"]
      # Example for disabling check: []
      SupportedDockerImageFormats: ["v2"]

      # Include details about job reuse decisions in the server log. This
      # causes additional database queries to run, so it should not be
      # enabled unless you expect to examine the resulting logs for
      # troubleshooting purposes.
      LogReuseDecisions: false

      # Default value for keep_cache_ram of a container's runtime_constraints.
      DefaultKeepCacheRAM: 268435456

      # Number of times a container can be unlocked before being
      # automatically cancelled.
      MaxDispatchAttempts: 5

      # Default value for container_count_max for container requests.  This is the
      # number of times Arvados will create a new container to satisfy a container
      # request.  If a container is cancelled it will retry a new container if
      # container_count < container_count_max on any container requests associated
      # with the cancelled container.
      MaxRetryAttempts: 3

      # The maximum number of compute nodes that can be in use simultaneously
      # If this limit is reduced, any existing nodes with slot number >= new limit
      # will not be counted against the new limit. In other words, the new limit
      # won't be strictly enforced until those nodes with higher slot numbers
      # go down.
      MaxComputeVMs: 64

      # Preemptible instance support (e.g. AWS Spot Instances)
      # When true, child containers will get created with the preemptible
      # scheduling parameter parameter set.
      UsePreemptibleInstances: false

      # Include details about job reuse decisions in the server log. This
      # causes additional database queries to run, so it should not be
      # enabled unless you expect to examine the resulting logs for
      # troubleshooting purposes.
      LogReuseDecisions: false

      Logging:
        # When you run the db:delete_old_container_logs task, it will find
        # containers that have been finished for at least this many seconds,
        # and delete their stdout, stderr, arv-mount, crunch-run, and
        # crunchstat logs from the logs table.
        MaxAge: 720h

        # These two settings control how frequently log events are flushed to the
        # database.  Log lines are buffered until either crunch_log_bytes_per_event
        # has been reached or crunch_log_seconds_between_events has elapsed since
        # the last flush.
        LogBytesPerEvent: 4096
        LogSecondsBetweenEvents: 1

        # The sample period for throttling logs, in seconds.
        LogThrottlePeriod: 60

        # Maximum number of bytes that job can log over crunch_log_throttle_period
        # before being silenced until the end of the period.
        LogThrottleBytes: 65536

        # Maximum number of lines that job can log over crunch_log_throttle_period
        # before being silenced until the end of the period.
        LogThrottleLines: 1024

        # Maximum bytes that may be logged by a single job.  Log bytes that are
        # silenced by throttling are not counted against this total.
        LimitLogBytesPerJob: 67108864

        LogPartialLineThrottlePeriod: 5

        # Container logs are written to Keep and saved in a collection,
        # which is updated periodically while the container runs.  This
        # value sets the interval (given in seconds) between collection
        # updates.
        LogUpdatePeriod: 1800

        # The log collection is also updated when the specified amount of
        # log data (given in bytes) is produced in less than one update
        # period.
        LogUpdateSize: 33554432

      SLURM:
        Managed:
          # Path to dns server configuration directory
          # (e.g. /etc/unbound.d/conf.d). If false, do not write any config
          # files or touch restart.txt (see below).
          DNSServerConfDir: ""

          # Template file for the dns server host snippets. See
          # unbound.template in this directory for an example. If false, do
          # not write any config files.
          DNSServerConfTemplate: ""

          # String to write to {dns_server_conf_dir}/restart.txt (with a
          # trailing newline) after updating local data. If false, do not
          # open or write the restart.txt file.
          DNSServerReloadCommand: ""

          # Command to run after each DNS update. Template variables will be
          # substituted; see the "unbound" example below. If false, do not run
          # a command.
          DNSServerUpdateCommand: ""

          ComputeNodeDomain: ""
          ComputeNodeNameservers:
            - 192.168.1.1

          # Hostname to assign to a compute node when it sends a "ping" and the
          # hostname in its Node record is nil.
          # During bootstrapping, the "ping" script is expected to notice the
          # hostname given in the ping response, and update its unix hostname
          # accordingly.
          # If false, leave the hostname alone (this is appropriate if your compute
          # nodes' hostnames are already assigned by some other mechanism).
          #
          # One way or another, the hostnames of your node records should agree
          # with your DNS records and your /etc/slurm-llnl/slurm.conf files.
          #
          # Example for compute0000, compute0001, ....:
          # assign_node_hostname: compute%<slot_number>04d
          # (See http://ruby-doc.org/core-2.2.2/Kernel.html#method-i-format for more.)
          AssignNodeHostname: "compute%<slot_number>d"

      JobsAPI:
        # Enable the legacy Jobs API.  This value must be a string.
        # 'auto' -- (default) enable the Jobs API only if it has been used before
        #         (i.e., there are job records in the database)
        # 'true' -- enable the Jobs API despite lack of existing records.
        # 'false' -- disable the Jobs API despite presence of existing records.
        Enable: 'auto'

        # Git repositories must be readable by api server, or you won't be
        # able to submit crunch jobs. To pass the test suites, put a clone
        # of the arvados tree in {git_repositories_dir}/arvados.git or
        # {git_repositories_dir}/arvados/.git
        GitInternalDir: /var/lib/arvados/internal.git

        # Docker image to be used when none found in runtime_constraints of a job
        DefaultDockerImage: ""

        # none or slurm_immediate
        CrunchJobWrapper: none

        # username, or false = do not set uid when running jobs.
        CrunchJobUser: crunch

        # The web service must be able to create/write this file, and
        # crunch-job must be able to stat() it.
        CrunchRefreshTrigger: /tmp/crunch_refresh_trigger

        # Control job reuse behavior when two completed jobs match the
        # search criteria and have different outputs.
        #
        # If true, in case of a conflict, reuse the earliest job (this is
        # similar to container reuse behavior).
        #
        # If false, in case of a conflict, do not reuse any completed job,
        # but do reuse an already-running job if available (this is the
        # original job reuse behavior, and is still the default).
        ReuseJobIfOutputsDiffer: false

    Mail:
      MailchimpAPIKey: ""
      MailchimpListID: ""
      SendUserSetupNotificationEmail: ""
      IssueReporterEmailFrom: ""
      IssueReporterEmailTo: ""
      SupportEmailAddress: ""
      EmailFrom: ""
    RemoteClusters:
      "*":
        Proxy: false
        ActivateUsers: false
      SAMPLE:
        Host: sample.arvadosapi.com
        Proxy: false
        Scheme: https
        Insecure: false
        ActivateUsers: false
`)
