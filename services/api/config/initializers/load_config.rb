# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

begin
  # If secret_token.rb exists here, we need to load it first.
  require_relative 'secret_token.rb'
rescue LoadError
  # Normally secret_token.rb is missing and the secret token is
  # configured by application.yml (i.e., here!) instead.
end

if (File.exist?(File.expand_path '../omniauth.rb', __FILE__) and
    not defined? WARNED_OMNIAUTH_CONFIG)
  Rails.logger.warn <<-EOS
DEPRECATED CONFIGURATION:
 Please move your SSO provider config into config/application.yml
 and delete config/initializers/omniauth.rb.
EOS
  # Real values will be copied from globals by omniauth_init.rb. For
  # now, assign some strings so the generic *.yml config loader
  # doesn't overwrite them or complain that they're missing.
  Rails.configuration.Login["ProviderAppID"] = 'xxx'
  Rails.configuration.Login["ProviderAppSecret"] = 'xxx'
  Rails.configuration.Services["SSO"]["ExternalURL"] = '//xxx'
  WARNED_OMNIAUTH_CONFIG = true
end

$arvados_config = {}

["#{::Rails.root.to_s}/config/config.defaults.yml", "/etc/arvados/config.yml"].each do |path|
  if File.exist? path
    confs = YAML.load(IO.read(path), deserialize_symbols: false)
    if confs
      clusters = confs["Clusters"].first
      $arvados_config["ClusterID"] = clusters[0]
      $arvados_config.merge!(clusters[1])
    end
  end
end

def set_cfg cfg, k, v
  # "foo.bar: baz" --> { config.foo.bar = baz }
  ks = k.split '.'
  k = ks.pop
  ks.each do |kk|
    cfg = cfg[kk]
    if cfg.nil?
      break
    end
  end
  if !cfg.nil?
    cfg[k] = v
  end
end

$config_migrate_map = {}
$config_types = {}
def declare_config(assign_to, configtype, migrate_from=nil)
  if migrate_from
    $config_migrate_map[migrate_from] = ->(cfg, k, v) {
      set_cfg cfg, assign_to, v
    }
  end
  $config_types[assign_to] = configtype
end

module Boolean; end
class TrueClass; include Boolean; end
class FalseClass; include Boolean; end

declare_config "ClusterID", String, :uuid_prefix
declare_config "Git.Repositories", String, :git_repositories_dir
declare_config "API.DisabledAPIs", Array, :disable_api_methods
declare_config "API.MaxRequestSize", Integer, :max_request_size
declare_config "API.MaxIndexDatabaseRead", Integer, :max_index_database_read
declare_config "API.MaxItemsPerResponse", Integer, :max_items_per_response
declare_config "API.AsyncPermissionsUpdateInterval", ActiveSupport::Duration, :async_permissions_update_interval
declare_config "Users.AutoSetupNewUsers", Boolean, :auto_setup_new_users
declare_config "Users.AutoSetupNewUsersWithVmUUID", String, :auto_setup_new_users_with_vm_uuid
declare_config "Users.AutoSetupNewUsersWithRepository", Boolean, :auto_setup_new_users_with_repository
declare_config "Users.AutoSetupUsernameBlacklist", Array, :auto_setup_name_blacklist
declare_config "Users.NewUsersAreActive", Boolean, :new_users_are_active
declare_config "Users.AutoAdminUserWithEmail", String, :auto_admin_user
declare_config "Users.AutoAdminFirstUser", Boolean, :auto_admin_first_user
declare_config "Users.UserProfileNotificationAddress", String, :user_profile_notification_address
declare_config "Users.AdminNotifierEmailFrom", String, :admin_notifier_email_from
declare_config "Users.EmailSubjectPrefix", String, :email_subject_prefix
declare_config "Users.UserNotifierEmailFrom", String, :user_notifier_email_from
declare_config "Users.NewUserNotificationRecipients", Array, :new_user_notification_recipients
declare_config "Users.NewInactiveUserNotificationRecipients", Array, :new_inactive_user_notification_recipients
declare_config "Login.ProviderAppSecret", String, :sso_app_secret
declare_config "Login.ProviderAppID", String, :sso_app_id
declare_config "TLS.Insecure", Boolean, :sso_insecure
declare_config "Services.SSO.ExternalURL", String, :sso_provider_url
declare_config "AuditLogs.MaxAge", ActiveSupport::Duration, :max_audit_log_age
declare_config "AuditLogs.MaxDeleteBatch", Integer, :max_audit_log_delete_batch
declare_config "AuditLogs.UnloggedAttributes", Array, :unlogged_attributes
declare_config "SystemLogs.MaxRequestLogParamsSize", Integer, :max_request_log_params_size
declare_config "Collections.DefaultReplication", Integer, :default_collection_replication
declare_config "Collections.DefaultTrashLifetime", ActiveSupport::Duration, :default_trash_lifetime
declare_config "Collections.CollectionVersioning", Boolean, :collection_versioning
declare_config "Collections.PreserveVersionIfIdle", ActiveSupport::Duration, :preserve_version_if_idle
declare_config "Collections.TrashSweepInterval", ActiveSupport::Duration, :trash_sweep_interval
declare_config "Collections.BlobSigningKey", String, :blob_signing_key
declare_config "Collections.BlobSigningTTL", Integer, :blob_signature_ttl
declare_config "Collections.BlobSigning", Boolean, :permit_create_collection_with_unsigned_manifest
declare_config "Containers.SupportedDockerImageFormats", Array, :docker_image_formats
declare_config "Containers.LogReuseDecisions", Boolean, :log_reuse_decisions
declare_config "Containers.DefaultKeepCacheRAM", Integer, :container_default_keep_cache_ram
declare_config "Containers.MaxDispatchAttempts", Integer, :max_container_dispatch_attempts
declare_config "Containers.MaxRetryAttempts", Integer, :container_count_max
declare_config "Containers.UsePreemptibleInstances", Boolean, :preemptible_instances
declare_config "Containers.MaxComputeVMs", Integer, :max_compute_nodes
declare_config "Containers.Logging.LogBytesPerEvent", Integer, :crunch_log_bytes_per_event
declare_config "Containers.Logging.LogSecondsBetweenEvents", ActiveSupport::Duration, :crunch_log_seconds_between_events
declare_config "Containers.Logging.LogThrottlePeriod", ActiveSupport::Duration, :crunch_log_throttle_period
declare_config "Containers.Logging.LogThrottleBytes", Integer, :crunch_log_throttle_bytes
declare_config "Containers.Logging.LogThrottleLines", Integer, :crunch_log_throttle_lines
declare_config "Containers.Logging.LimitLogBytesPerJob", Integer, :crunch_limit_log_bytes_per_job
declare_config "Containers.Logging.LogPartialLineThrottlePeriod", ActiveSupport::Duration, :crunch_log_partial_line_throttle_period
declare_config "Containers.Logging.LogUpdatePeriod", ActiveSupport::Duration, :crunch_log_update_period
declare_config "Containers.Logging.LogUpdateSize", Integer, :crunch_log_update_size
declare_config "Containers.Logging.MaxAge", ActiveSupport::Duration, :clean_container_log_rows_after
declare_config "Containers.SLURM.Managed.DNSServerConfDir", String, :dns_server_conf_dir
declare_config "Containers.SLURM.Managed.DNSServerConfTemplate", String, :dns_server_conf_template
declare_config "Containers.SLURM.Managed.DNSServerReloadCommand", String, :dns_server_reload_command
declare_config "Containers.SLURM.Managed.DNSServerUpdateCommand", String, :dns_server_update_command
declare_config "Containers.SLURM.Managed.ComputeNodeDomain", String, :compute_node_domain
declare_config "Containers.SLURM.Managed.ComputeNodeNameservers", Array, :compute_node_nameservers
declare_config "Containers.SLURM.Managed.AssignNodeHostname", String, :assign_node_hostname
declare_config "Containers.JobsAPI.Enable", String, :enable_legacy_jobs_api
declare_config "Containers.JobsAPI.CrunchJobWrapper", String, :crunch_job_wrapper
declare_config "Containers.JobsAPI.CrunchJobUser", String, :crunch_job_user
declare_config "Containers.JobsAPI.CrunchRefreshTrigger", String, :crunch_refresh_trigger
declare_config "Containers.JobsAPI.GitInternalDir", String, :git_internal_dir
declare_config "Containers.JobsAPI.ReuseJobIfOutputsDiffer", Boolean, :reuse_job_if_outputs_differ
declare_config "Containers.JobsAPI.DefaultDockerImage", String, :default_docker_image_for_jobs
declare_config "Mail.MailchimpAPIKey", String, :mailchimp_api_key
declare_config "Mail.MailchimpListID", String, :mailchimp_list_id
declare_config "Services.Workbench1.ExternalURL", String, :workbench_address
declare_config "Services.Websocket.ExternalURL", String, :websocket_address
declare_config "Services.WebDAV.ExternalURL", String, :keep_web_service_url
declare_config "Services.GitHTTP.ExternalURL", String, :git_repo_https_base
declare_config "Services.GitSSH.ExternalURL", String, :git_repo_ssh_base

application_config = {}
%w(application.default application).each do |cfgfile|
  path = "#{::Rails.root.to_s}/config/#{cfgfile}.yml"
  if File.exist? path
    yaml = ERB.new(IO.read path).result(binding)
    confs = YAML.load(yaml, deserialize_symbols: true)
    # Ignore empty YAML file:
    next if confs == false
    application_config.merge!(confs['common'] || {})
    application_config.merge!(confs[::Rails.env.to_s] || {})
  end
end

application_config.each do |k, v|
  if $config_migrate_map[k.to_sym]
    $config_migrate_map[k.to_sym].call $arvados_config, k, v
  else
    set_cfg $arvados_config, k, v
  end
end

$config_types.each do |cfgkey, cfgtype|
  cfg = $arvados_config
  k = cfgkey
  ks = k.split '.'
  k = ks.pop
  ks.each do |kk|
    cfg = cfg[kk]
    if cfg.nil?
      break
    end
  end
  if cfgtype == String and !cfg[k]
    cfg[k] = ""
  end
  if cfgtype == ActiveSupport::Duration
    if cfg[k].is_a? Integer
      cfg[k] = cfg[k].seconds
    elsif cfg[k].is_a? String
      # TODO handle suffixes
    end
  end

  if cfg.nil?
    raise "missing #{cfgkey}"
  end

  if !cfg[k].is_a? cfgtype
    raise "#{cfgkey} expected #{cfgtype} but was #{cfg[k].class}"
  end
end

Server::Application.configure do
  nils = []
  $arvados_config.each do |k, v|
    cfg = config
    if cfg.respond_to?(k.to_sym) and !cfg.send(k).nil?
    # Config must have been set already in environments/*.rb.
    #
    # After config files have been migrated, this mechanism should
    # be deprecated, then removed.
    elsif v.nil?
      # Config variables are not allowed to be nil. Make a "naughty"
      # list, and present it below.
      nils << k
    else
      cfg.send "#{k}=", v
    end
  end
  if !nils.empty?
    raise <<EOS
Refusing to start in #{::Rails.env.to_s} mode with missing configuration.

The following configuration settings must be specified in
config/application.yml:
* #{nils.join "\n* "}

EOS
  end
  config.secret_key_base = config.secret_token
end
