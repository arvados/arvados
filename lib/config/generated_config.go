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
    # Token used internally by Arvados components to authenticate to
    # one another. Use a string of at least 50 random alphanumerics.
    SystemRootToken: ""

    # Token to be included in all healthcheck requests. Disabled by default.
    # Server expects request header of the format "Authorization: Bearer xxx"
    ManagementToken: ""

    Services:

      # In each of the service sections below, the keys under
      # InternalURLs are the endpoints where the service should be
      # listening, and reachable from other hosts in the cluster.
      SAMPLE:
        InternalURLs:
          "http://host1.example:12345": {}
          "http://host2.example:12345":
            # Rendezvous is normally empty/omitted. When changing the
            # URL of a Keepstore service, Rendezvous should be set to
            # the old URL (with trailing slash omitted) to preserve
            # rendezvous ordering.
            Rendezvous: ""
          SAMPLE:
            Rendezvous: ""
        ExternalURL: "-"

      RailsAPI:
        InternalURLs: {}
        ExternalURL: "-"
      Controller:
        InternalURLs: {}
        ExternalURL: ""
      Websocket:
        InternalURLs: {}
        ExternalURL: ""
      Keepbalance:
        InternalURLs: {}
        ExternalURL: "-"
      GitHTTP:
        InternalURLs: {}
        ExternalURL: ""
      GitSSH:
        InternalURLs: {}
        ExternalURL: ""
      DispatchCloud:
        InternalURLs: {}
        ExternalURL: "-"
      SSO:
        InternalURLs: {}
        ExternalURL: ""
      Keepproxy:
        InternalURLs: {}
        ExternalURL: ""
      WebDAV:
        InternalURLs: {}
        # Base URL for Workbench inline preview.  If blank, use
        # WebDAVDownload instead, and disable inline preview.
        # If both are empty, downloading collections from workbench
        # will be impossible.
        #
        # It is important to properly configure the download service
        # to migitate cross-site-scripting (XSS) attacks.  A HTML page
        # can be stored in collection.  If an attacker causes a victim
        # to visit that page through Workbench, it will be rendered by
        # the browser.  If all collections are served at the same
        # domain, the browser will consider collections as coming from
        # the same origin and having access to the same browsing data,
        # enabling malicious Javascript on that page to access Arvados
        # on behalf of the victim.
        #
        # This is mitigating by having separate domains for each
        # collection, or limiting preview to circumstances where the
        # collection is not accessed with the user's regular
        # full-access token.
        #
        # Serve preview links using uuid or pdh in subdomain
        # (requires wildcard DNS and TLS certificate)
        #   https://*.collections.uuid_prefix.arvadosapi.com
        #
        # Serve preview links using uuid or pdh in main domain
        # (requires wildcard DNS and TLS certificate)
        #   https://*--collections.uuid_prefix.arvadosapi.com
        #
        # Serve preview links by setting uuid or pdh in the path.
        # This configuration only allows previews of public data or
        # collection-sharing links, because these use the anonymous
        # user token or the token is already embedded in the URL.
        # Other data must be handled as downloads via WebDAVDownload:
        #   https://collections.uuid_prefix.arvadosapi.com
        #
        ExternalURL: ""

      WebDAVDownload:
        InternalURLs: {}
        # Base URL for download links. If blank, serve links to WebDAV
        # with disposition=attachment query param.  Unlike preview links,
        # browsers do not render attachments, so there is no risk of XSS.
        #
        # If WebDAVDownload is blank, and WebDAV uses a
        # single-origin form, then Workbench will show an error page
        #
        # Serve download links by setting uuid or pdh in the path:
        #   https://download.uuid_prefix.arvadosapi.com
        #
        ExternalURL: ""

      Keepstore:
        InternalURLs: {}
        ExternalURL: "-"
      Composer:
        InternalURLs: {}
        ExternalURL: ""
      WebShell:
        InternalURLs: {}
        # ShellInABox service endpoint URL for a given VM.  If empty, do not
        # offer web shell logins.
        #
        # E.g., using a path-based proxy server to forward connections to shell hosts:
        # https://webshell.uuid_prefix.arvadosapi.com
        #
        # E.g., using a name-based proxy server to forward connections to shell hosts:
        # https://*.webshell.uuid_prefix.arvadosapi.com
        ExternalURL: ""
      Workbench1:
        InternalURLs: {}
        ExternalURL: ""
      Workbench2:
        InternalURLs: {}
        ExternalURL: ""
      Health:
        InternalURLs: {}
        ExternalURL: "-"

    PostgreSQL:
      # max concurrent connections per arvados server daemon
      ConnectionPool: 32
      Connection:
        # All parameters here are passed to the PG client library in a connection string;
        # see https://www.postgresql.org/docs/current/static/libpq-connect.html#LIBPQ-PARAMKEYWORDS
        host: ""
        port: ""
        user: ""
        password: ""
        dbname: ""
        SAMPLE: ""
    API:
      # Limits for how long a client token created by regular users can be valid,
      # and also is used as a default expiration policy when no expiration date is
      # specified.
      # Default value zero means token expirations don't get clamped and no
      # default expiration is set.
      MaxTokenLifetime: 0s

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

      # Maximum number of concurrent requests to accept in a single
      # service process, or 0 for no limit.
      MaxConcurrentRequests: 0

      # Maximum number of 64MiB memory buffers per Keepstore server process, or
      # 0 for no limit. When this limit is reached, up to
      # (MaxConcurrentRequests - MaxKeepBlobBuffers) HTTP requests requiring
      # buffers (like GET and PUT) will wait for buffer space to be released.
      # Any HTTP requests beyond MaxConcurrentRequests will receive an
      # immediate 503 response.
      #
      # MaxKeepBlobBuffers should be set such that (MaxKeepBlobBuffers * 64MiB
      # * 1.1) fits comfortably in memory. On a host dedicated to running
      # Keepstore, divide total memory by 88MiB to suggest a suitable value.
      # For example, if grep MemTotal /proc/meminfo reports MemTotal: 7125440
      # kB, compute 7125440 / (88 * 1024)=79 and set MaxKeepBlobBuffers: 79
      MaxKeepBlobBuffers: 128

      # API methods to disable. Disabled methods are not listed in the
      # discovery document, and respond 404 to all requests.
      # Example: {"jobs.create":{}, "pipeline_instances.create": {}}
      DisabledAPIs: {}

      # Interval (seconds) between asynchronous permission view updates. Any
      # permission-updating API called with the 'async' parameter schedules a an
      # update on the permission view in the future, if not already scheduled.
      AsyncPermissionsUpdateInterval: 20s

      # Maximum number of concurrent outgoing requests to make while
      # serving a single incoming multi-cluster (federated) request.
      MaxRequestAmplification: 4

      # Maximum wall clock time to spend handling an incoming request.
      RequestTimeout: 5m

      # Websocket will send a periodic empty event after 'SendTimeout'
      # if there is no other activity to maintain the connection /
      # detect dropped connections.
      SendTimeout: 60s

      WebsocketClientEventQueue: 64
      WebsocketServerEventQueue: 4

      # Timeout on requests to internal Keep services.
      KeepServiceRequestTimeout: 15s

    Users:
      # Config parameters to automatically setup new users.  If enabled,
      # this users will be able to self-activate.  Enable this if you want
      # to run an open instance where anyone can create an account and use
      # the system without requiring manual approval.
      #
      # The params AutoSetupNewUsersWith* are meaningful only when AutoSetupNewUsers is turned on.
      # AutoSetupUsernameBlacklist is a list of usernames to be blacklisted for auto setup.
      AutoSetupNewUsers: false
      AutoSetupNewUsersWithVmUUID: ""
      AutoSetupNewUsersWithRepository: false
      AutoSetupUsernameBlacklist:
        arvados: {}
        git: {}
        gitolite: {}
        gitolite-admin: {}
        root: {}
        syslog: {}
        SAMPLE: {}

      # When NewUsersAreActive is set to true, new users will be active
      # immediately.  This skips the "self-activate" step which enforces
      # user agreements.  Should only be enabled for development.
      NewUsersAreActive: false

      # The e-mail address of the user you would like to become marked as an admin
      # user on their first login.
      AutoAdminUserWithEmail: ""

      # If AutoAdminFirstUser is set to true, the first user to log in when no
      # other admin users exist will automatically become an admin user.
      AutoAdminFirstUser: false

      # Email address to notify whenever a user creates a profile for the
      # first time
      UserProfileNotificationAddress: ""
      AdminNotifierEmailFrom: arvados@example.com
      EmailSubjectPrefix: "[ARVADOS] "
      UserNotifierEmailFrom: arvados@example.com
      NewUserNotificationRecipients: {}
      NewInactiveUserNotificationRecipients: {}

      # Set AnonymousUserToken to enable anonymous user access. Populate this
      # field with a long random string. Then run "bundle exec
      # ./script/get_anonymous_user_token.rb" in the directory where your API
      # server is running to record the token in the database.
      AnonymousUserToken: ""

      # If a new user has an alternate email address (local@domain)
      # with the domain given here, its local part becomes the new
      # user's default username. Otherwise, the user's primary email
      # address is used.
      PreferDomainForUsername: ""

      UserSetupMailText: |
        <% if not @user.full_name.empty? -%>
        <%= @user.full_name %>,
        <% else -%>
        Hi there,
        <% end -%>

        Your Arvados account has been set up.  You can log in at

        <%= Rails.configuration.Services.Workbench1.ExternalURL %>

        Thanks,
        Your Arvados administrator.

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
      # If MaxDeleteBatch is 0, log entries will never be
      # deleted by Arvados. Cleanup can be done by an external process
      # without affecting any Arvados system processes, as long as very
      # recent (<5 minutes old) logs are not deleted.
      #
      # 100000 is a reasonable batch size for most sites.
      MaxDeleteBatch: 0

      # Attributes to suppress in events and audit logs.  Notably,
      # specifying {"manifest_text": {}} here typically makes the database
      # smaller and faster.
      #
      # Warning: Using any non-empty value here can have undesirable side
      # effects for any client or component that relies on event logs.
      # Use at your own risk.
      UnloggedAttributes: {}

    SystemLogs:

      # Logging threshold: panic, fatal, error, warn, info, debug, or
      # trace
      LogLevel: info

      # Logging format: json or text
      Format: json

      # Maximum characters of (JSON-encoded) query parameters to include
      # in each request log entry. When params exceed this size, they will
      # be JSON-encoded, truncated to this size, and logged as
      # params_truncated.
      MaxRequestLogParamsSize: 2000

    Collections:

      # Enable access controls for data stored in Keep. This should
      # always be set to true on a production cluster.
      BlobSigning: true

      # BlobSigningKey is a string of alphanumeric characters used to
      # generate permission signatures for Keep locators. It must be
      # identical to the permission key given to Keep. IMPORTANT: This
      # is a site secret. It should be at least 50 characters.
      #
      # Modifying BlobSigningKey will invalidate all existing
      # signatures, which can cause programs to fail (e.g., arv-put,
      # arv-get, and Crunch jobs).  To avoid errors, rotate keys only
      # when no such processes are running.
      BlobSigningKey: ""

      # Enable garbage collection of unreferenced blobs in Keep.
      BlobTrash: true

      # Time to leave unreferenced blobs in "trashed" state before
      # deleting them, or 0 to skip the "trashed" state entirely and
      # delete unreferenced blobs.
      #
      # If you use any Amazon S3 buckets as storage volumes, this
      # must be at least 24h to avoid occasional data loss.
      BlobTrashLifetime: 336h

      # How often to check for (and delete) trashed blocks whose
      # BlobTrashLifetime has expired.
      BlobTrashCheckInterval: 24h

      # Maximum number of concurrent "trash blob" and "delete trashed
      # blob" operations conducted by a single keepstore process. Each
      # of these can be set to 0 to disable the respective operation.
      #
      # If BlobTrashLifetime is zero, "trash" and "delete trash"
      # happen at once, so only the lower of these two values is used.
      BlobTrashConcurrency: 4
      BlobDeleteConcurrency: 4

      # Maximum number of concurrent "create additional replica of
      # existing blob" operations conducted by a single keepstore
      # process.
      BlobReplicateConcurrency: 4

      # Default replication level for collections. This is used when a
      # collection's replication_desired attribute is nil.
      DefaultReplication: 2

      # BlobSigningTTL determines the minimum lifetime of transient
      # data, i.e., blocks that are not referenced by
      # collections. Unreferenced blocks exist for two reasons:
      #
      # 1) A data block must be written to a disk/cloud backend device
      # before a collection can be created/updated with a reference to
      # it.
      #
      # 2) Deleting or updating a collection can remove the last
      # remaining reference to a data block.
      #
      # If BlobSigningTTL is too short, long-running
      # processes/containers will fail when they take too long (a)
      # between writing blocks and writing collections that reference
      # them, or (b) between reading collections and reading the
      # referenced blocks.
      #
      # If BlobSigningTTL is too long, data will still be stored long
      # after the referring collections are deleted, and you will
      # needlessly fill up disks or waste money on cloud storage.
      #
      # Modifying BlobSigningTTL invalidates existing signatures; see
      # BlobSigningKey note above.
      #
      # The default is 2 weeks.
      BlobSigningTTL: 336h

      # When running keep-balance, this is the destination filename for
      # the list of lost block hashes if there are any, one per line.
      # Updated automically during each successful run.
      BlobMissingReport: ""

      # keep-balance operates periodically, i.e.: do a
      # scan/balance operation, sleep, repeat.
      #
      # BalancePeriod determines the interval between start times of
      # successive scan/balance operations. If a scan/balance operation
      # takes longer than RunPeriod, the next one will follow it
      # immediately.
      #
      # If SIGUSR1 is received during an idle period between operations,
      # the next operation will start immediately.
      BalancePeriod: 10m

      # Limits the number of collections retrieved by keep-balance per
      # API transaction. If this is zero, page size is
      # determined by the API server's own page size limits (see
      # API.MaxItemsPerResponse and API.MaxIndexDatabaseRead).
      BalanceCollectionBatch: 0

      # The size of keep-balance's internal queue of
      # collections. Higher values use more memory and improve throughput
      # by allowing keep-balance to fetch the next page of collections
      # while the current page is still being processed. If this is zero
      # or omitted, pages are processed serially.
      BalanceCollectionBuffers: 1000

      # Maximum time for a rebalancing run. This ensures keep-balance
      # eventually gives up and retries if, for example, a network
      # error causes a hung connection that is never closed by the
      # OS. It should be long enough that it doesn't interrupt a
      # long-running balancing operation.
      BalanceTimeout: 6h

      # Default lifetime for ephemeral collections: 2 weeks. This must not
      # be less than BlobSigningTTL.
      DefaultTrashLifetime: 336h

      # Interval (seconds) between trash sweeps. During a trash sweep,
      # collections are marked as trash if their trash_at time has
      # arrived, and deleted if their delete_at time has arrived.
      TrashSweepInterval: 60s

      # If true, enable collection versioning.
      # When a collection's preserve_version field is true or the current version
      # is older than the amount of seconds defined on PreserveVersionIfIdle,
      # a snapshot of the collection's previous state is created and linked to
      # the current collection.
      CollectionVersioning: false

      #   0s = auto-create a new version on every update.
      #  -1s = never auto-create new versions.
      # > 0s = auto-create a new version when older than the specified number of seconds.
      PreserveVersionIfIdle: -1s

      # If non-empty, allow project and collection names to contain
      # the "/" character (slash/stroke/solidus), and replace "/" with
      # the given string in the filesystem hierarchy presented by
      # WebDAV. Example values are "%2f" and "{slash}". Names that
      # contain the substitution string itself may result in confusing
      # behavior, so a value like "_" is not recommended.
      #
      # If the default empty value is used, the server will reject
      # requests to create or rename a collection when the new name
      # contains "/".
      #
      # If the value "/" is used, project and collection names
      # containing "/" will be allowed, but they will not be
      # accessible via WebDAV.
      #
      # Use of this feature is not recommended, if it can be avoided.
      ForwardSlashNameSubstitution: ""

      # Include "folder objects" in S3 ListObjects responses.
      S3FolderObjects: true

      # Managed collection properties. At creation time, if the client didn't
      # provide the listed keys, they will be automatically populated following
      # one of the following behaviors:
      #
      # * UUID of the user who owns the containing project.
      #   responsible_person_uuid: {Function: original_owner, Protected: true}
      #
      # * Default concrete value.
      #   foo_bar: {Value: baz, Protected: false}
      #
      # If Protected is true, only an admin user can modify its value.
      ManagedProperties:
        SAMPLE: {Function: original_owner, Protected: true}

      # In "trust all content" mode, Workbench will redirect download
      # requests to WebDAV preview link, even in the cases when
      # WebDAV would have to expose XSS vulnerabilities in order to
      # handle the redirect (see discussion on Services.WebDAV).
      #
      # This setting has no effect in the recommended configuration,
      # where the WebDAV is configured to have a separate domain for
      # every collection; in this case XSS protection is provided by
      # browsers' same-origin policy.
      #
      # The default setting (false) is appropriate for a multi-user site.
      TrustAllContent: false

      # Cache parameters for WebDAV content serving:
      WebDAVCache:
        # Time to cache manifests, permission checks, and sessions.
        TTL: 300s

        # Time to cache collection state.
        UUIDTTL: 5s

        # Block cache entries. Each block consumes up to 64 MiB RAM.
        MaxBlockEntries: 4

        # Collection cache entries.
        MaxCollectionEntries: 1000

        # Approximate memory limit (in bytes) for collection cache.
        MaxCollectionBytes: 100000000

        # Permission cache entries.
        MaxPermissionEntries: 1000

        # UUID cache entries.
        MaxUUIDEntries: 1000

        # Persistent sessions.
        MaxSessions: 100

    Login:
      # One of the following mechanisms (SSO, Google, PAM, LDAP, or
      # LoginCluster) should be enabled; see
      # https://doc.arvados.org/install/setup-login.html

      Google:
        # Authenticate with Google.
        Enable: false

        # Use the Google Cloud console to enable the People API (APIs
        # and Services > Enable APIs and services > Google People API
        # > Enable), generate a Client ID and secret (APIs and
        # Services > Credentials > Create credentials > OAuth client
        # ID > Web application) and add your controller's /login URL
        # (e.g., "https://zzzzz.example.com/login") as an authorized
        # redirect URL.
        #
        # Incompatible with ForceLegacyAPI14. ProviderAppID must be
        # blank.
        ClientID: ""
        ClientSecret: ""

        # Allow users to log in to existing accounts using any verified
        # email address listed by their Google account. If true, the
        # Google People API must be enabled in order for Google login to
        # work. If false, only the primary email address will be used.
        AlternateEmailAddresses: true

        # Send additional parameters with authentication requests. See
        # https://developers.google.com/identity/protocols/oauth2/openid-connect#authenticationuriparameters
        # for a list of supported parameters.
        AuthenticationRequestParameters:
          # Show the "choose which Google account" page, even if the
          # client is currently logged in to exactly one Google
          # account.
          prompt: select_account

          SAMPLE: ""

      OpenIDConnect:
        # Authenticate with an OpenID Connect provider.
        Enable: false

        # Issuer URL, e.g., "https://login.example.com".
        #
        # This must be exactly equal to the URL returned by the issuer
        # itself in its config response ("isser" key). If the
        # configured value is "https://example" and the provider
        # returns "https://example:443" or "https://example/" then
        # login will fail, even though those URLs are equivalent
        # (RFC3986).
        Issuer: ""

        # Your client ID and client secret (supplied by the provider).
        ClientID: ""
        ClientSecret: ""

        # OpenID claim field containing the user's email
        # address. Normally "email"; see
        # https://openid.net/specs/openid-connect-core-1_0.html#StandardClaims
        EmailClaim: "email"

        # OpenID claim field containing the email verification
        # flag. Normally "email_verified".  To accept every returned
        # email address without checking a "verified" field at all,
        # use the empty string "".
        EmailVerifiedClaim: "email_verified"

        # OpenID claim field containing the user's preferred
        # username. If empty, use the mailbox part of the user's email
        # address.
        UsernameClaim: ""

        # Send additional parameters with authentication requests,
        # like {display: page, prompt: consent}. See
        # https://openid.net/specs/openid-connect-core-1_0.html#AuthRequest
        # and refer to your provider's documentation for supported
        # parameters.
        AuthenticationRequestParameters:
          SAMPLE: ""

      PAM:
        # (Experimental) Use PAM to authenticate users.
        Enable: false

        # PAM service name. PAM will apply the policy in the
        # corresponding config file (e.g., /etc/pam.d/arvados) or, if
        # there is none, the default "other" config.
        Service: arvados

        # Domain name (e.g., "example.com") to use to construct the
        # user's email address if PAM authentication returns a
        # username with no "@". If empty, use the PAM username as the
        # user's email address, whether or not it contains "@".
        #
        # Note that the email address is used as the primary key for
        # user records when logging in. Therefore, if you change
        # PAMDefaultEmailDomain after the initial installation, you
        # should also update existing user records to reflect the new
        # domain. Otherwise, next time those users log in, they will
        # be given new accounts instead of accessing their existing
        # accounts.
        DefaultEmailDomain: ""

      LDAP:
        # Use an LDAP service to authenticate users.
        Enable: false

        # Server URL, like "ldap://ldapserver.example.com:389" or
        # "ldaps://ldapserver.example.com:636".
        URL: "ldap://ldap:389"

        # Use StartTLS upon connecting to the server.
        StartTLS: true

        # Skip TLS certificate name verification.
        InsecureTLS: false

        # Strip the @domain part if a user supplies an email-style
        # username with this domain. If "*", strip any user-provided
        # domain. If "", never strip the domain part. Example:
        # "example.com"
        StripDomain: ""

        # If, after applying StripDomain, the username contains no "@"
        # character, append this domain to form an email-style
        # username. Example: "example.com"
        AppendDomain: ""

        # The LDAP attribute to filter on when looking up a username
        # (after applying StripDomain and AppendDomain).
        SearchAttribute: uid

        # Bind with this username (DN or UPN) and password when
        # looking up the user record.
        #
        # Example user: "cn=admin,dc=example,dc=com"
        SearchBindUser: ""
        SearchBindPassword: ""

        # Directory base for username lookup. Example:
        # "ou=Users,dc=example,dc=com"
        SearchBase: ""

        # Additional filters to apply when looking up users' LDAP
        # entries. This can be used to restrict access to a subset of
        # LDAP users, or to disambiguate users from other directory
        # entries that have the SearchAttribute present.
        #
        # Special characters in assertion values must be escaped (see
        # RFC4515).
        #
        # Example: "(objectClass=person)"
        SearchFilters: ""

        # LDAP attribute to use as the user's email address.
        #
        # Important: This must not be an attribute whose value can be
        # edited in the directory by the users themselves. Otherwise,
        # users can take over other users' Arvados accounts trivially
        # (email address is the primary key for Arvados accounts.)
        EmailAttribute: mail

        # LDAP attribute to use as the preferred Arvados username. If
        # no value is found (or this config is empty) the username
        # originally supplied by the user will be used.
        UsernameAttribute: uid

      SSO:
        # Authenticate with a separate SSO server. (Deprecated)
        Enable: false

        # ProviderAppID and ProviderAppSecret are generated during SSO
        # setup; see
        # https://doc.arvados.org/v2.0/install/install-sso.html#update-config
        ProviderAppID: ""
        ProviderAppSecret: ""

      Test:
        # Authenticate users listed here in the config file. This
        # feature is intended to be used in test environments, and
        # should not be used in production.
        Enable: false
        Users:
          SAMPLE:
            Email: alice@example.com
            Password: xyzzy

      # The cluster ID to delegate the user database.  When set,
      # logins on this cluster will be redirected to the login cluster
      # (login cluster must appear in RemoteClusters with Proxy: true)
      LoginCluster: ""

      # How long a cached token belonging to a remote cluster will
      # remain valid before it needs to be revalidated.
      RemoteTokenRefresh: 5m

      # How long a client token created from a login flow will be valid without
      # asking the user to re-login. Example values: 60m, 8h.
      # Default value zero means tokens don't have expiration.
      TokenLifetime: 0s

      # When the token is returned to a client, the token itself may
      # be restricted from manipulating other tokens based on whether
      # the client is "trusted" or not.  The local Workbench1 and
      # Workbench2 are trusted by default, but if this is a
      # LoginCluster, you probably want to include the other Workbench
      # instances in the federation in this list.
      TrustedClients:
        SAMPLE:
          "https://workbench.federate1.example": {}
          "https://workbench.federate2.example": {}

    Git:
      # Path to git or gitolite-shell executable. Each authenticated
      # request will execute this program with the single argument "http-backend"
      GitCommand: /usr/bin/git

      # Path to Gitolite's home directory. If a non-empty path is given,
      # the CGI environment will be set up to support the use of
      # gitolite-shell as a GitCommand: for example, if GitoliteHome is
      # "/gh", then the CGI environment will have GITOLITE_HTTP_HOME=/gh,
      # PATH=$PATH:/gh/bin, and GL_BYPASS_ACCESS_CHECKS=1.
      GitoliteHome: ""

      # Git repositories must be readable by api server, or you won't be
      # able to submit crunch jobs. To pass the test suites, put a clone
      # of the arvados tree in {git_repositories_dir}/arvados.git or
      # {git_repositories_dir}/arvados/.git
      Repositories: /var/lib/arvados/git/repositories

    TLS:
      Certificate: ""
      Key: ""
      Insecure: false

    Containers:
      # List of supported Docker Registry image formats that compute nodes
      # are able to use. ` + "`" + `arv keep docker` + "`" + ` will error out if a user tries
      # to store an image with an unsupported format. Use an empty array
      # to skip the compatibility check (and display a warning message to
      # that effect).
      #
      # Example for sites running docker < 1.10: {"v1": {}}
      # Example for sites running docker >= 1.10: {"v2": {}}
      # Example for disabling check: {}
      SupportedDockerImageFormats:
        "v2": {}
        SAMPLE: {}

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

      # PEM encoded SSH key (RSA, DSA, or ECDSA) used by the
      # (experimental) cloud dispatcher for executing containers on
      # worker VMs. Begins with "-----BEGIN RSA PRIVATE KEY-----\n"
      # and ends with "\n-----END RSA PRIVATE KEY-----\n".
      DispatchPrivateKey: ""

      # Maximum time to wait for workers to come up before abandoning
      # stale locks from a previous dispatch process.
      StaleLockTimeout: 1m

      # The crunch-run command used to start a container on a worker node.
      #
      # When dispatching to cloud VMs, this is used only if
      # DeployRunnerBinary in the CloudVMs section is set to the empty
      # string.
      CrunchRunCommand: "crunch-run"

      # Extra arguments to add to crunch-run invocation
      # Example: ["--cgroup-parent-subsystem=memory"]
      CrunchRunArgumentsList: []

      # Extra RAM to reserve on the node, in addition to
      # the amount specified in the container's RuntimeConstraints
      ReserveExtraRAM: 256MiB

      # Minimum time between two attempts to run the same container
      MinRetryPeriod: 0s

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
        LogSecondsBetweenEvents: 5s

        # The sample period for throttling logs.
        LogThrottlePeriod: 60s

        # Maximum number of bytes that job can log over crunch_log_throttle_period
        # before being silenced until the end of the period.
        LogThrottleBytes: 65536

        # Maximum number of lines that job can log over crunch_log_throttle_period
        # before being silenced until the end of the period.
        LogThrottleLines: 1024

        # Maximum bytes that may be logged by a single job.  Log bytes that are
        # silenced by throttling are not counted against this total.
        LimitLogBytesPerJob: 67108864

        LogPartialLineThrottlePeriod: 5s

        # Container logs are written to Keep and saved in a
        # collection, which is updated periodically while the
        # container runs.  This value sets the interval between
        # collection updates.
        LogUpdatePeriod: 30m

        # The log collection is also updated when the specified amount of
        # log data (given in bytes) is produced in less than one update
        # period.
        LogUpdateSize: 32MiB

      ShellAccess:
        # An admin user can use "arvados-client shell" to start an
        # interactive shell (with any user ID) in any running
        # container.
        Admin: false

        # Any user can use "arvados-client shell" to start an
        # interactive shell (with any user ID) in any running
        # container that they started, provided it isn't also
        # associated with a different user's container request.
        #
        # Interactive sessions make it easy to alter the container's
        # runtime environment in ways that aren't recorded or
        # reproducible. Consider the implications for automatic
        # container reuse before enabling and using this feature. In
        # particular, note that starting an interactive session does
        # not disqualify a container from being reused by a different
        # user/workflow in the future.
        User: false

      SLURM:
        PrioritySpread: 0
        SbatchArgumentsList: []
        SbatchEnvironmentVariables:
          SAMPLE: ""
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
            "192.168.1.1": {}
            SAMPLE: {}

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
        # Enable the legacy 'jobs' API (crunch v1).  This value must be a string.
        #
        # Note: this only enables read-only access, creating new
        # legacy jobs and pipelines is not supported.
        #
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

      CloudVMs:
        # Enable the cloud scheduler (experimental).
        Enable: false

        # Name/number of port where workers' SSH services listen.
        SSHPort: "22"

        # Interval between queue polls.
        PollInterval: 10s

        # Shell command to execute on each worker to determine whether
        # the worker is booted and ready to run containers. It should
        # exit zero if the worker is ready.
        BootProbeCommand: "docker ps -q"

        # Minimum interval between consecutive probes to a single
        # worker.
        ProbeInterval: 10s

        # Maximum probes per second, across all workers in a pool.
        MaxProbesPerSecond: 10

        # Time before repeating SIGTERM when killing a container.
        TimeoutSignal: 5s

        # Time to give up on a process (most likely arv-mount) that
        # still holds a container lockfile after its main supervisor
        # process has exited, and declare the instance broken.
        TimeoutStaleRunLock: 5s

        # Time to give up on SIGTERM and write off the worker.
        TimeoutTERM: 2m

        # Maximum create/destroy-instance operations per second (0 =
        # unlimited).
        MaxCloudOpsPerSecond: 0

        # Maximum concurrent node creation operations (0 = unlimited). This is
        # recommended by Azure in certain scenarios (see
        # https://docs.microsoft.com/en-us/azure/virtual-machines/linux/capture-image)
        # and can be used with other cloud providers too, if desired.
        MaxConcurrentInstanceCreateOps: 0

        # Interval between cloud provider syncs/updates ("list all
        # instances").
        SyncInterval: 1m

        # Time to leave an idle worker running (in case new containers
        # appear in the queue that it can run) before shutting it
        # down.
        TimeoutIdle: 1m

        # Time to wait for a new worker to boot (i.e., pass
        # BootProbeCommand) before giving up and shutting it down.
        TimeoutBooting: 10m

        # Maximum time a worker can stay alive with no successful
        # probes before being automatically shut down.
        TimeoutProbe: 10m

        # Time after shutting down a worker to retry the
        # shutdown/destroy operation.
        TimeoutShutdown: 10s

        # Worker VM image ID.
        # (aws) AMI identifier
        # (azure) managed disks: the name of the managed disk image
        # (azure) shared image gallery: the name of the image definition. Also
        # see the SharedImageGalleryName and SharedImageGalleryImageVersion fields.
        # (azure) unmanaged disks (deprecated): the complete URI of the VHD, e.g.
        # https://xxxxx.blob.core.windows.net/system/Microsoft.Compute/Images/images/xxxxx.vhd
        ImageID: ""

        # An executable file (located on the dispatcher host) to be
        # copied to cloud instances at runtime and used as the
        # container runner/supervisor. The default value is the
        # dispatcher program itself.
        #
        # Use the empty string to disable this step: nothing will be
        # copied, and cloud instances are assumed to have a suitable
        # version of crunch-run installed; see CrunchRunCommand above.
        DeployRunnerBinary: "/proc/self/exe"

        # Tags to add on all resources (VMs, NICs, disks) created by
        # the container dispatcher. (Arvados's own tags --
        # InstanceType, IdleBehavior, and InstanceSecret -- will also
        # be added.)
        ResourceTags:
          SAMPLE: "tag value"

        # Prefix for predefined tags used by Arvados (InstanceSetID,
        # InstanceType, InstanceSecret, IdleBehavior). With the
        # default value "Arvados", tags are "ArvadosInstanceSetID",
        # "ArvadosInstanceSecret", etc.
        #
        # This should only be changed while no cloud resources are in
        # use and the cloud dispatcher is not running. Otherwise,
        # VMs/resources that were added using the old tag prefix will
        # need to be detected and cleaned up manually.
        TagKeyPrefix: Arvados

        # Cloud driver: "azure" (Microsoft Azure) or "ec2" (Amazon AWS).
        Driver: ec2

        # Cloud-specific driver parameters.
        DriverParameters:

          # (ec2) Credentials. Omit or leave blank if using IAM role.
          AccessKeyID: ""
          SecretAccessKey: ""

          # (ec2) Instance configuration.
          SecurityGroupIDs:
            "SAMPLE": {}
          SubnetID: ""
          Region: ""
          EBSVolumeType: gp2
          AdminUsername: debian

          # (azure) Credentials.
          SubscriptionID: ""
          ClientID: ""
          ClientSecret: ""
          TenantID: ""

          # (azure) Instance configuration.
          CloudEnvironment: AzurePublicCloud
          Location: centralus

          # (azure) The resource group where the VM and virtual NIC will be
          # created.
          ResourceGroup: ""

          # (azure) The resource group of the Network to use for the virtual
          # NIC (if different from ResourceGroup)
          NetworkResourceGroup: ""
          Network: ""
          Subnet: ""

          # (azure) managed disks: The resource group where the managed disk
          # image can be found (if different from ResourceGroup).
          ImageResourceGroup: ""

          # (azure) shared image gallery: the name of the gallery
          SharedImageGalleryName: ""
          # (azure) shared image gallery: the version of the image definition
          SharedImageGalleryImageVersion: ""

          # (azure) unmanaged disks (deprecated): Where to store the VM VHD blobs
          StorageAccount: ""
          BlobContainer: ""

          # (azure) How long to wait before deleting VHD and NIC
          # objects that are no longer being used.
          DeleteDanglingResourcesAfter: 20s

          # Account (that already exists in the VM image) that will be
          # set up with an ssh authorized key to allow the compute
          # dispatcher to connect.
          AdminUsername: arvados

    InstanceTypes:

      # Use the instance type name as the key (in place of "SAMPLE" in
      # this sample entry).
      SAMPLE:
        # Cloud provider's instance type. Defaults to the configured type name.
        ProviderType: ""
        VCPUs: 1
        RAM: 128MiB
        IncludedScratch: 16GB
        AddedScratch: 0
        Price: 0.1
        Preemptible: false

    Volumes:
      SAMPLE:
        # AccessViaHosts specifies which keepstore processes can read
        # and write data on the volume.
        #
        # For a local filesystem, AccessViaHosts has one entry,
        # indicating which server the filesystem is located on.
        #
        # For a network-attached backend accessible by all keepstore
        # servers, like a cloud storage bucket or an NFS mount,
        # AccessViaHosts can be empty/omitted.
        #
        # Further info/examples:
        # https://doc.arvados.org/install/configure-fs-storage.html
        # https://doc.arvados.org/install/configure-s3-object-storage.html
        # https://doc.arvados.org/install/configure-azure-blob-storage.html
        AccessViaHosts:
          SAMPLE:
            ReadOnly: false
          "http://host1.example:25107": {}
        ReadOnly: false
        Replication: 1
        StorageClasses:
          default: true
          SAMPLE: true
        Driver: s3
        DriverParameters:
          # for s3 driver -- see
          # https://doc.arvados.org/install/configure-s3-object-storage.html
          IAMRole: aaaaa
          AccessKey: aaaaa
          SecretKey: aaaaa
          Endpoint: ""
          Region: us-east-1a
          Bucket: aaaaa
          LocationConstraint: false
          V2Signature: false
          IndexPageSize: 1000
          ConnectTimeout: 1m
          ReadTimeout: 10m
          RaceWindow: 24h
          # Use aws-s3-go (v2) instead of goamz
          UseAWSS3v2Driver: false

          # For S3 driver, potentially unsafe tuning parameter,
          # intentionally excluded from main documentation.
          #
          # Enable deletion (garbage collection) even when the
          # configured BlobTrashLifetime is zero.  WARNING: eventual
          # consistency may result in race conditions that can cause
          # data loss.  Do not enable this unless you understand and
          # accept the risk.
          UnsafeDelete: false

          # for azure driver -- see
          # https://doc.arvados.org/install/configure-azure-blob-storage.html
          StorageAccountName: aaaaa
          StorageAccountKey: aaaaa
          StorageBaseURL: core.windows.net
          ContainerName: aaaaa
          RequestTimeout: 30s
          ListBlobsRetryDelay: 10s
          ListBlobsMaxAttempts: 10
          MaxGetBytes: 0
          WriteRaceInterval: 15s
          WriteRacePollTime: 1s

          # for local directory driver -- see
          # https://doc.arvados.org/install/configure-fs-storage.html
          Root: /var/lib/arvados/keep-data

          # For local directory driver, potentially confusing tuning
          # parameter, intentionally excluded from main documentation.
          #
          # When true, read and write operations (for whole 64MiB
          # blocks) on an individual volume will queued and issued
          # serially.  When false, read and write operations will be
          # issued concurrently.
          #
          # May possibly improve throughput if you have physical spinning disks
          # and experience contention when there are multiple requests
          # to the same volume.
          #
          # Otherwise, when using SSDs, RAID, or a shared network filesystem, you
          # should leave this alone.
          Serialize: false

    Mail:
      MailchimpAPIKey: ""
      MailchimpListID: ""
      SendUserSetupNotificationEmail: true

      # Bug/issue report notification to and from addresses
      IssueReporterEmailFrom: "arvados@example.com"
      IssueReporterEmailTo: "arvados@example.com"
      SupportEmailAddress: "arvados@example.com"

      # Generic issue email from
      EmailFrom: "arvados@example.com"
    RemoteClusters:
      "*":
        Host: ""
        Proxy: false
        Scheme: https
        Insecure: false
        ActivateUsers: false
      SAMPLE:
        # API endpoint host or host:port; default is {id}.arvadosapi.com
        Host: sample.arvadosapi.com

        # Perform a proxy request when a local client requests an
        # object belonging to this remote.
        Proxy: false

        # Default "https". Can be set to "http" for testing.
        Scheme: https

        # Disable TLS verify. Can be set to true for testing.
        Insecure: false

        # When users present tokens issued by this remote cluster, and
        # their accounts are active on the remote cluster, activate
        # them on this cluster too.
        ActivateUsers: false

    Workbench:
      # Workbench1 configs
      Theme: default
      ActivationContactLink: mailto:info@arvados.org
      ArvadosDocsite: https://doc.arvados.org
      ArvadosPublicDataDocURL: https://playground.arvados.org/projects/public
      ShowUserAgreementInline: false
      SecretKeyBase: ""

      # Scratch directory used by the remote repository browsing
      # feature. If it doesn't exist, it (and any missing parents) will be
      # created using mkdir_p.
      RepositoryCache: /var/www/arvados-workbench/current/tmp/git

      # Below is a sample setting of user_profile_form_fields config parameter.
      # This configuration parameter should be set to either false (to disable) or
      # to a map as shown below.
      # Configure the map of input fields to be displayed in the profile page
      # using the attribute "key" for each of the input fields.
      # This sample shows configuration with one required and one optional form fields.
      # For each of these input fields:
      #   You can specify "Type" as "text" or "select".
      #   List the "Options" to be displayed for each of the "select" menu.
      #   Set "Required" as "true" for any of these fields to make them required.
      # If any of the required fields are missing in the user's profile, the user will be
      # redirected to the profile page before they can access any Workbench features.
      UserProfileFormFields:
        SAMPLE:
          Type: select
          FormFieldTitle: Best color
          FormFieldDescription: your favorite color
          Required: false
          Position: 1
          Options:
            red: {}
            blue: {}
            green: {}
            SAMPLE: {}

        # exampleTextValue:  # key that will be set in properties
        #   Type: text  #
        #   FormFieldTitle: ""
        #   FormFieldDescription: ""
        #   Required: true
        #   Position: 1
        # exampleOptionsValue:
        #   Type: select
        #   FormFieldTitle: ""
        #   FormFieldDescription: ""
        #   Required: true
        #   Position: 1
        #   Options:
        #     red: {}
        #     blue: {}
        #     yellow: {}

      # Use "UserProfileFormMessage to configure the message you want
      # to display on the profile page.
      UserProfileFormMessage: 'Welcome to Arvados. All <span style="color:red">required fields</span> must be completed before you can proceed.'

      # Mimetypes of applications for which the view icon
      # would be enabled in a collection's show page.
      # It is sufficient to list only applications here.
      # No need to list text and image types.
      ApplicationMimetypesWithViewIcon:
        cwl: {}
        fasta: {}
        go: {}
        javascript: {}
        json: {}
        pdf: {}
        python: {}
        x-python: {}
        r: {}
        rtf: {}
        sam: {}
        x-sh: {}
        vnd.realvnc.bed: {}
        xml: {}
        xsl: {}
        SAMPLE: {}

      # The maximum number of bytes to load in the log viewer
      LogViewerMaxBytes: 1M

      # When anonymous_user_token is configured, show public projects page
      EnablePublicProjectsPage: true

      # By default, disable the "Getting Started" popup which is specific to Arvados playground
      EnableGettingStartedPopup: false

      # Ask Arvados API server to compress its response payloads.
      APIResponseCompression: true

      # Timeouts for API requests.
      APIClientConnectTimeout: 2m
      APIClientReceiveTimeout: 5m

      # Maximum number of historic log records of a running job to fetch
      # and display in the Log tab, while subscribing to web sockets.
      RunningJobLogRecordsToFetch: 2000

      # In systems with many shared projects, loading of dashboard and topnav
      # can be slow due to collections indexing; use the following parameters
      # to suppress these properties
      ShowRecentCollectionsOnDashboard: true
      ShowUserNotifications: true

      # Enable/disable "multi-site search" in top nav ("true"/"false"), or
      # a link to the multi-site search page on a "home" Workbench site.
      #
      # Example:
      #   https://workbench.zzzzz.arvadosapi.com/collections/multisite
      MultiSiteSearch: ""

      # Should workbench allow management of local git repositories? Set to false if
      # the jobs api is disabled and there are no local git repositories.
      Repositories: true

      SiteName: Arvados Workbench
      ProfilingEnabled: false

      # This is related to obsolete Google OpenID 1.0 login
      # but some workbench stuff still expects it to be set.
      DefaultOpenIdPrefix: "https://www.google.com/accounts/o8/id"

      # Workbench2 configs
      VocabularyURL: ""
      FileViewersConfigURL: ""

      # Idle time after which the user's session will be auto closed.
      # This feature is disabled when set to zero.
      IdleTimeout: 0s

      # Workbench welcome screen, this is HTML text that will be
      # incorporated directly onto the page.
      WelcomePageHTML: |
        <img src="/arvados-logo-big.png" style="width: 20%; float: right; padding: 1em;" />
        <h2>Please log in.</h2>

        <p>The "Log in" button below will show you a sign-in
        page. After you log in, you will be redirected back to
        Arvados Workbench.</p>

        <p>If you have never used Arvados Workbench before, logging in
        for the first time will automatically create a new
        account.</p>

        <i>Arvados Workbench uses your name and email address only for
        identification, and does not retrieve any other personal
        information.</i>

      # Workbench screen displayed to inactive users.  This is HTML
      # text that will be incorporated directly onto the page.
      InactivePageHTML: |
        <img src="/arvados-logo-big.png" style="width: 20%; float: right; padding: 1em;" />
        <h3>Hi! You're logged in, but...</h3>
        <p>Your account is inactive.</p>
        <p>An administrator must activate your account before you can get
        any further.</p>

      # Connecting to Arvados shell VMs tends to be site-specific.
      # Put any special instructions here. This is HTML text that will
      # be incorporated directly onto the Workbench page.
      SSHHelpPageHTML: |
        <a href="https://doc.arvados.org/user/getting_started/ssh-access-unix.html">Accessing an Arvados VM with SSH</a> (generic instructions).
        Site configurations vary.  Contact your local cluster administrator if you have difficulty accessing an Arvados shell node.

      # Sample text if you are using a "switchyard" ssh proxy.
      # Replace "zzzzz" with your Cluster ID.
      #SSHHelpPageHTML: |
      # <p>Add a section like this to your SSH configuration file ( <i>~/.ssh/config</i>):</p>
      # <pre>Host *.zzzzz
      #  TCPKeepAlive yes
      #  ServerAliveInterval 60
      #  ProxyCommand ssh -p2222 turnout@switchyard.zzzzz.arvadosapi.com -x -a $SSH_PROXY_FLAGS %h
      # </pre>

      # If you are using a switchyard ssh proxy, shell node hostnames
      # may require a special hostname suffix.  In the sample ssh
      # configuration above, this would be ".zzzzz"
      # This is added to the hostname in the "command line" column
      # the Workbench "shell VMs" page.
      #
      # If your shell nodes are directly accessible by users without a
      # proxy and have fully qualified host names, you should leave
      # this blank.
      SSHHelpHostSuffix: ""

    # Bypass new (Arvados 1.5) API implementations, and hand off
    # requests directly to Rails instead. This can provide a temporary
    # workaround for clients that are incompatible with the new API
    # implementation. Note that it also disables some new federation
    # features and will be removed in a future release.
    ForceLegacyAPI14: false

# (Experimental) Restart services automatically when config file
# changes are detected. Only supported by ` + "`" + `arvados-server boot` + "`" + ` in
# dev/test mode.
AutoReloadConfig: false
`)
