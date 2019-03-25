# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'config_loader'

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

declare_config "ClusterID", NonemptyString, :uuid_prefix
declare_config "ManagementToken", String, :ManagementToken
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
declare_config "Login.ProviderAppSecret", NonemptyString, :sso_app_secret
declare_config "Login.ProviderAppID", NonemptyString, :sso_app_id
declare_config "TLS.Insecure", Boolean, :sso_insecure
declare_config "Services.SSO.ExternalURL", NonemptyString, :sso_provider_url
declare_config "AuditLogs.MaxAge", ActiveSupport::Duration, :max_audit_log_age
declare_config "AuditLogs.MaxDeleteBatch", Integer, :max_audit_log_delete_batch
declare_config "AuditLogs.UnloggedAttributes", Array, :unlogged_attributes
declare_config "SystemLogs.MaxRequestLogParamsSize", Integer, :max_request_log_params_size
declare_config "Collections.DefaultReplication", Integer, :default_collection_replication
declare_config "Collections.DefaultTrashLifetime", ActiveSupport::Duration, :default_trash_lifetime
declare_config "Collections.CollectionVersioning", Boolean, :collection_versioning
declare_config "Collections.PreserveVersionIfIdle", ActiveSupport::Duration, :preserve_version_if_idle
declare_config "Collections.TrashSweepInterval", ActiveSupport::Duration, :trash_sweep_interval
declare_config "Collections.BlobSigningKey", NonemptyString, :blob_signing_key
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
declare_config "Containers.JobsAPI.Enable", String, :enable_legacy_jobs_api, ->(cfg, k, v) { set_cfg cfg, "Containers.JobsAPI.Enable", v.to_s }
declare_config "Containers.JobsAPI.CrunchJobWrapper", String, :crunch_job_wrapper
declare_config "Containers.JobsAPI.CrunchJobUser", String, :crunch_job_user
declare_config "Containers.JobsAPI.CrunchRefreshTrigger", String, :crunch_refresh_trigger
declare_config "Containers.JobsAPI.GitInternalDir", String, :git_internal_dir
declare_config "Containers.JobsAPI.ReuseJobIfOutputsDiffer", Boolean, :reuse_job_if_outputs_differ
declare_config "Containers.JobsAPI.DefaultDockerImage", String, :default_docker_image_for_jobs
declare_config "Mail.MailchimpAPIKey", String, :mailchimp_api_key
declare_config "Mail.MailchimpListID", String, :mailchimp_list_id
declare_config "Services.Workbench1.ExternalURL", URI, :workbench_address
declare_config "Services.Websocket.ExternalURL", URI, :websocket_address
declare_config "Services.WebDAV.ExternalURL", URI, :keep_web_service_url
declare_config "Services.GitHTTP.ExternalURL", URI, :git_repo_https_base
declare_config "Services.GitSSH.ExternalURL", URI, :git_repo_ssh_base, ->(cfg, k, v) { set_cfg cfg, "Services.GitSSH.ExternalURL", "ssh://#{v}" }
declare_config "RemoteClusters", Hash, :remote_hosts, ->(cfg, k, v) {
  h = {}
  v.each do |clusterid, host|
    h[clusterid] = {
      "Host" => host,
      "Proxy" => true,
      "Scheme" => "https",
      "Insecure" => false,
      "ActivateUsers" => false
    }
  end
  set_cfg cfg, "RemoteClusters", h
}
declare_config "RemoteClusters.*.Proxy", Boolean, :remote_hosts_via_dns

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

$remaining_config = migrate_config application_config, $arvados_config

if application_config[:auto_activate_users_from]
  application_config[:auto_activate_users_from].each do |cluster|
    if $arvados_config.RemoteClusters[cluster]
      $arvados_config.RemoteClusters[cluster]["ActivateUsers"] = true
    end
  end
end

# Checks for wrongly typed configuration items, and essential items
# that can't be empty
coercion_and_check $arvados_config

Server::Application.configure do
  copy_into_config $arvados_config, config
  copy_into_config $remaining_config, config
  config.secret_key_base = config.secret_token
end
