# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

#
# Load Arvados configuration from /etc/arvados/config.yml, using defaults
# from config.default.yml
#
# Existing application.yml is migrated into the new config structure.
# Keys in the legacy application.yml take precedence.
#
# Use "bundle exec config:dump" to get the complete active configuration
#
# Use "bundle exec config:migrate" to migrate application.yml and
# database.yml to config.yml.  After adding the output of
# config:migrate to /etc/arvados/config.yml, you will be able to
# delete application.yml and database.yml.

require "cgi"
require 'config_loader'
require 'open3'

begin
  # If secret_token.rb exists here, we need to load it first.
  require_relative 'secret_token.rb'
rescue LoadError
  # Normally secret_token.rb is missing and the secret token is
  # configured by application.yml (i.e., here!) instead.
end

# Load the defaults, used by config:migrate and fallback loading
# legacy application.yml
defaultYAML, stderr, status = Open3.capture3("arvados-server", "config-dump", "-config=-", "-skip-legacy", stdin_data: "Clusters: {xxxxx: {}}")
if !status.success?
  puts stderr
  raise "error loading config: #{status}"
end
confs = YAML.load(defaultYAML, deserialize_symbols: false)
clusterID, clusterConfig = confs["Clusters"].first
$arvados_config_defaults = clusterConfig
$arvados_config_defaults["ClusterID"] = clusterID

if ENV["ARVADOS_CONFIG"] == "none"
  # Don't load config. This magic value is set by packaging scripts so
  # they can run "rake assets:precompile" without a real config.
  $arvados_config_global = $arvados_config_defaults.deep_dup
else
  # Load the global config file
  Open3.popen2("arvados-server", "config-dump", "-skip-legacy") do |stdin, stdout, status_thread|
    confs = YAML.load(stdout, deserialize_symbols: false)
    if confs && !confs.empty?
      # config-dump merges defaults with user configuration, so every
      # key should be set.
      clusterID, clusterConfig = confs["Clusters"].first
      $arvados_config_global = clusterConfig
      $arvados_config_global["ClusterID"] = clusterID
    else
      # config-dump failed, assume we will be loading from legacy
      # application.yml, initialize with defaults.
      $arvados_config_global = $arvados_config_defaults.deep_dup
    end
  end
end

# Now make a copy
$arvados_config = $arvados_config_global.deep_dup

def arrayToHash cfg, k, v
  val = {}
  v.each do |entry|
    val[entry.to_s] = {}
  end
  ConfigLoader.set_cfg cfg, k, val
end

# Declare all our configuration items.
arvcfg = ConfigLoader.new
arvcfg.declare_config "ClusterID", NonemptyString, :uuid_prefix
arvcfg.declare_config "ManagementToken", String, :ManagementToken
arvcfg.declare_config "SystemRootToken", String
arvcfg.declare_config "Git.Repositories", String, :git_repositories_dir
arvcfg.declare_config "API.DisabledAPIs", Hash, :disable_api_methods, ->(cfg, k, v) { arrayToHash cfg, "API.DisabledAPIs", v }
arvcfg.declare_config "API.MaxRequestSize", Integer, :max_request_size
arvcfg.declare_config "API.MaxIndexDatabaseRead", Integer, :max_index_database_read
arvcfg.declare_config "API.MaxItemsPerResponse", Integer, :max_items_per_response
arvcfg.declare_config "API.MaxTokenLifetime", ActiveSupport::Duration
arvcfg.declare_config "API.RequestTimeout", ActiveSupport::Duration
arvcfg.declare_config "API.AsyncPermissionsUpdateInterval", ActiveSupport::Duration, :async_permissions_update_interval
arvcfg.declare_config "Users.AutoSetupNewUsers", Boolean, :auto_setup_new_users
arvcfg.declare_config "Users.AutoSetupNewUsersWithVmUUID", String, :auto_setup_new_users_with_vm_uuid
arvcfg.declare_config "Users.AutoSetupNewUsersWithRepository", Boolean, :auto_setup_new_users_with_repository
arvcfg.declare_config "Users.AutoSetupUsernameBlacklist", Hash, :auto_setup_name_blacklist, ->(cfg, k, v) { arrayToHash cfg, "Users.AutoSetupUsernameBlacklist", v }
arvcfg.declare_config "Users.NewUsersAreActive", Boolean, :new_users_are_active
arvcfg.declare_config "Users.AutoAdminUserWithEmail", String, :auto_admin_user
arvcfg.declare_config "Users.AutoAdminFirstUser", Boolean, :auto_admin_first_user
arvcfg.declare_config "Users.UserProfileNotificationAddress", String, :user_profile_notification_address
arvcfg.declare_config "Users.AdminNotifierEmailFrom", String, :admin_notifier_email_from
arvcfg.declare_config "Users.EmailSubjectPrefix", String, :email_subject_prefix
arvcfg.declare_config "Users.UserNotifierEmailFrom", String, :user_notifier_email_from
arvcfg.declare_config "Users.UserNotifierEmailBcc", Hash
arvcfg.declare_config "Users.NewUserNotificationRecipients", Hash, :new_user_notification_recipients, ->(cfg, k, v) { arrayToHash cfg, "Users.NewUserNotificationRecipients", v }
arvcfg.declare_config "Users.NewInactiveUserNotificationRecipients", Hash, :new_inactive_user_notification_recipients, method(:arrayToHash)
arvcfg.declare_config "Users.RoleGroupsVisibleToAll", Boolean
arvcfg.declare_config "Login.LoginCluster", String
arvcfg.declare_config "Login.TrustedClients", Hash
arvcfg.declare_config "Login.RemoteTokenRefresh", ActiveSupport::Duration
arvcfg.declare_config "Login.TokenLifetime", ActiveSupport::Duration
arvcfg.declare_config "TLS.Insecure", Boolean, :sso_insecure
arvcfg.declare_config "AuditLogs.MaxAge", ActiveSupport::Duration, :max_audit_log_age
arvcfg.declare_config "AuditLogs.MaxDeleteBatch", Integer, :max_audit_log_delete_batch
arvcfg.declare_config "AuditLogs.UnloggedAttributes", Hash, :unlogged_attributes, ->(cfg, k, v) { arrayToHash cfg, "AuditLogs.UnloggedAttributes", v }
arvcfg.declare_config "SystemLogs.MaxRequestLogParamsSize", Integer, :max_request_log_params_size
arvcfg.declare_config "Collections.DefaultReplication", Integer, :default_collection_replication
arvcfg.declare_config "Collections.DefaultTrashLifetime", ActiveSupport::Duration, :default_trash_lifetime
arvcfg.declare_config "Collections.CollectionVersioning", Boolean, :collection_versioning
arvcfg.declare_config "Collections.PreserveVersionIfIdle", ActiveSupport::Duration, :preserve_version_if_idle
arvcfg.declare_config "Collections.TrashSweepInterval", ActiveSupport::Duration, :trash_sweep_interval
arvcfg.declare_config "Collections.BlobSigningKey", String, :blob_signing_key
arvcfg.declare_config "Collections.BlobSigningTTL", ActiveSupport::Duration, :blob_signature_ttl
arvcfg.declare_config "Collections.BlobSigning", Boolean, :permit_create_collection_with_unsigned_manifest, ->(cfg, k, v) { ConfigLoader.set_cfg cfg, "Collections.BlobSigning", !v }
arvcfg.declare_config "Collections.ForwardSlashNameSubstitution", String
arvcfg.declare_config "Containers.SupportedDockerImageFormats", Hash, :docker_image_formats, ->(cfg, k, v) { arrayToHash cfg, "Containers.SupportedDockerImageFormats", v }
arvcfg.declare_config "Containers.LogReuseDecisions", Boolean, :log_reuse_decisions
arvcfg.declare_config "Containers.DefaultKeepCacheRAM", Integer, :container_default_keep_cache_ram
arvcfg.declare_config "Containers.MaxDispatchAttempts", Integer, :max_container_dispatch_attempts
arvcfg.declare_config "Containers.MaxRetryAttempts", Integer, :container_count_max
arvcfg.declare_config "Containers.UsePreemptibleInstances", Boolean, :preemptible_instances
arvcfg.declare_config "Containers.MaxComputeVMs", Integer, :max_compute_nodes
arvcfg.declare_config "Containers.Logging.LogBytesPerEvent", Integer, :crunch_log_bytes_per_event
arvcfg.declare_config "Containers.Logging.LogSecondsBetweenEvents", ActiveSupport::Duration, :crunch_log_seconds_between_events
arvcfg.declare_config "Containers.Logging.LogThrottlePeriod", ActiveSupport::Duration, :crunch_log_throttle_period
arvcfg.declare_config "Containers.Logging.LogThrottleBytes", Integer, :crunch_log_throttle_bytes
arvcfg.declare_config "Containers.Logging.LogThrottleLines", Integer, :crunch_log_throttle_lines
arvcfg.declare_config "Containers.Logging.LimitLogBytesPerJob", Integer, :crunch_limit_log_bytes_per_job
arvcfg.declare_config "Containers.Logging.LogPartialLineThrottlePeriod", ActiveSupport::Duration, :crunch_log_partial_line_throttle_period
arvcfg.declare_config "Containers.Logging.LogUpdatePeriod", ActiveSupport::Duration, :crunch_log_update_period
arvcfg.declare_config "Containers.Logging.LogUpdateSize", Integer, :crunch_log_update_size
arvcfg.declare_config "Containers.Logging.MaxAge", ActiveSupport::Duration, :clean_container_log_rows_after
arvcfg.declare_config "Containers.SLURM.Managed.DNSServerConfDir", Pathname, :dns_server_conf_dir
arvcfg.declare_config "Containers.SLURM.Managed.DNSServerConfTemplate", Pathname, :dns_server_conf_template
arvcfg.declare_config "Containers.SLURM.Managed.DNSServerReloadCommand", String, :dns_server_reload_command
arvcfg.declare_config "Containers.SLURM.Managed.DNSServerUpdateCommand", String, :dns_server_update_command
arvcfg.declare_config "Containers.SLURM.Managed.ComputeNodeDomain", String, :compute_node_domain
arvcfg.declare_config "Containers.SLURM.Managed.ComputeNodeNameservers", Hash, :compute_node_nameservers, ->(cfg, k, v) { arrayToHash cfg, "Containers.SLURM.Managed.ComputeNodeNameservers", v }
arvcfg.declare_config "Containers.SLURM.Managed.AssignNodeHostname", String, :assign_node_hostname
arvcfg.declare_config "Containers.JobsAPI.Enable", String, :enable_legacy_jobs_api, ->(cfg, k, v) { ConfigLoader.set_cfg cfg, "Containers.JobsAPI.Enable", v.to_s }
arvcfg.declare_config "Containers.JobsAPI.GitInternalDir", String, :git_internal_dir
arvcfg.declare_config "Mail.MailchimpAPIKey", String, :mailchimp_api_key
arvcfg.declare_config "Mail.MailchimpListID", String, :mailchimp_list_id
arvcfg.declare_config "Services.Controller.ExternalURL", URI
arvcfg.declare_config "Services.Workbench1.ExternalURL", URI, :workbench_address
arvcfg.declare_config "Services.Websocket.ExternalURL", URI, :websocket_address
arvcfg.declare_config "Services.WebDAV.ExternalURL", URI, :keep_web_service_url
arvcfg.declare_config "Services.GitHTTP.ExternalURL", URI, :git_repo_https_base
arvcfg.declare_config "Services.GitSSH.ExternalURL", URI, :git_repo_ssh_base, ->(cfg, k, v) { ConfigLoader.set_cfg cfg, "Services.GitSSH.ExternalURL", "ssh://#{v}" }
arvcfg.declare_config "RemoteClusters", Hash, :remote_hosts, ->(cfg, k, v) {
  h = if cfg["RemoteClusters"] then
        cfg["RemoteClusters"].deep_dup
      else
        {}
      end
  v.each do |clusterid, host|
    if h[clusterid].nil?
      h[clusterid] = {
        "Host" => host,
        "Proxy" => true,
        "Scheme" => "https",
        "Insecure" => false,
        "ActivateUsers" => false
      }
    end
  end
  ConfigLoader.set_cfg cfg, "RemoteClusters", h
}
arvcfg.declare_config "RemoteClusters.*.Proxy", Boolean, :remote_hosts_via_dns
arvcfg.declare_config "StorageClasses", Hash

dbcfg = ConfigLoader.new

dbcfg.declare_config "PostgreSQL.ConnectionPool", Integer, :pool
dbcfg.declare_config "PostgreSQL.Connection.host", String, :host
dbcfg.declare_config "PostgreSQL.Connection.port", String, :port
dbcfg.declare_config "PostgreSQL.Connection.user", String, :username
dbcfg.declare_config "PostgreSQL.Connection.password", String, :password
dbcfg.declare_config "PostgreSQL.Connection.dbname", String, :database
dbcfg.declare_config "PostgreSQL.Connection.template", String, :template
dbcfg.declare_config "PostgreSQL.Connection.encoding", String, :encoding
dbcfg.declare_config "PostgreSQL.Connection.collation", String, :collation

application_config = {}
%w(application.default application).each do |cfgfile|
  path = "#{::Rails.root.to_s}/config/#{cfgfile}.yml"
  confs = ConfigLoader.load(path, erb: true)
  # Ignore empty YAML file:
  next if confs == false
  application_config.deep_merge!(confs['common'] || {})
  application_config.deep_merge!(confs[::Rails.env.to_s] || {})
end

db_config = {}
path = "#{::Rails.root.to_s}/config/database.yml"
if !ENV['ARVADOS_CONFIG_NOLEGACY'] && File.exist?(path)
  db_config = ConfigLoader.load(path, erb: true)
end

$remaining_config = arvcfg.migrate_config(application_config, $arvados_config)
dbcfg.migrate_config(db_config[::Rails.env.to_s] || {}, $arvados_config)

if application_config[:auto_activate_users_from]
  application_config[:auto_activate_users_from].each do |cluster|
    if $arvados_config.RemoteClusters[cluster]
      $arvados_config.RemoteClusters[cluster]["ActivateUsers"] = true
    end
  end
end

if application_config[:host] || application_config[:port] || application_config[:scheme]
  if !application_config[:host] || application_config[:host].empty?
    raise "Must set 'host' when setting 'port' or 'scheme'"
  end
  $arvados_config.Services["Controller"]["ExternalURL"] = URI((application_config[:scheme] || "https")+"://"+application_config[:host]+
                                                              (if application_config[:port] then ":#{application_config[:port]}" else "" end))
end

# Checks for wrongly typed configuration items, coerces properties
# into correct types (such as Duration), and optionally raise error
# for essential configuration that can't be empty.
arvcfg.coercion_and_check $arvados_config_defaults, check_nonempty: false
arvcfg.coercion_and_check $arvados_config_global, check_nonempty: false
arvcfg.coercion_and_check $arvados_config, check_nonempty: true
dbcfg.coercion_and_check $arvados_config, check_nonempty: true

# * $arvados_config_defaults is the defaults
# * $arvados_config_global is $arvados_config_defaults merged with the contents of /etc/arvados/config.yml
# These are used by the rake config: tasks
#
# * $arvados_config is $arvados_config_global merged with the migrated contents of application.yml
# This is what actually gets copied into the Rails configuration object.

if $arvados_config["Collections"]["DefaultTrashLifetime"] < 86400.seconds then
  raise "default_trash_lifetime is %d, must be at least 86400" % Rails.configuration.Collections.DefaultTrashLifetime
end

default_storage_classes = []
$arvados_config["StorageClasses"].each do |cls, cfg|
  if cfg["Default"]
    default_storage_classes << cls
  end
end
if default_storage_classes.length == 0
  default_storage_classes = ["default"]
end
$arvados_config["DefaultStorageClasses"] = default_storage_classes.sort

#
# Special case for test database where there's no database.yml,
# because the Arvados config.yml doesn't have a concept of multiple
# rails environments.
#
if ::Rails.env.to_s == "test" && db_config["test"].nil?
  $arvados_config["PostgreSQL"]["Connection"]["dbname"] = "arvados_test"
end
if ::Rails.env.to_s == "test"
  # Use template0 when creating a new database. Avoids
  # character-encoding/collation problems.
  $arvados_config["PostgreSQL"]["Connection"]["template"] = "template0"
  # Some test cases depend on en_US.UTF-8 collation.
  $arvados_config["PostgreSQL"]["Connection"]["collation"] = "en_US.UTF-8"
end

if ENV["ARVADOS_CONFIG"] == "none"
  # We need the postgresql connection URI to be valid, even if we
  # don't use it.
  $arvados_config["PostgreSQL"]["Connection"]["host"] = "localhost"
  $arvados_config["PostgreSQL"]["Connection"]["user"] = "x"
  $arvados_config["PostgreSQL"]["Connection"]["password"] = "x"
  $arvados_config["PostgreSQL"]["Connection"]["dbname"] = "x"
end

if $arvados_config["PostgreSQL"]["Connection"]["password"].empty?
  raise "Database password is empty, PostgreSQL section is: #{$arvados_config["PostgreSQL"]}"
end

dbhost = $arvados_config["PostgreSQL"]["Connection"]["host"]
if $arvados_config["PostgreSQL"]["Connection"]["port"] != 0
  dbhost += ":#{$arvados_config["PostgreSQL"]["Connection"]["port"]}"
end

#
# If DATABASE_URL is set, then ActiveRecord won't error out if database.yml doesn't exist.
#
# For config migration, we've previously populated the PostgreSQL
# section of the config from database.yml
#
database_url = "postgresql://#{CGI.escape $arvados_config["PostgreSQL"]["Connection"]["user"]}:"+
                      "#{CGI.escape $arvados_config["PostgreSQL"]["Connection"]["password"]}@"+
                      "#{dbhost}/#{CGI.escape $arvados_config["PostgreSQL"]["Connection"]["dbname"]}?"+
                      "template=#{$arvados_config["PostgreSQL"]["Connection"]["template"]}&"+
                      "encoding=#{$arvados_config["PostgreSQL"]["Connection"]["client_encoding"]}&"+
                      "collation=#{$arvados_config["PostgreSQL"]["Connection"]["collation"]}&"+
                      "pool=#{$arvados_config["PostgreSQL"]["ConnectionPool"]}"

ENV["DATABASE_URL"] = database_url

Server::Application.configure do
  # Copy into the Rails config object.  This also turns Hash into
  # OrderedOptions so that application code can use
  # Rails.configuration.API.Blah instead of
  # Rails.configuration.API["Blah"]
  ConfigLoader.copy_into_config $arvados_config, config
  ConfigLoader.copy_into_config $remaining_config, config

  # We don't rely on cookies for authentication, so instead of
  # requiring a signing key in config, we assign a new random one at
  # startup.
  secrets.secret_key_base = rand(1<<255).to_s(36)
end
