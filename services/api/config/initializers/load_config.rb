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
  Rails.configuration.sso_app_id = 'xxx'
  Rails.configuration.sso_app_secret = 'xxx'
  Rails.configuration.sso_provider_url = '//xxx'
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

config_key_map =
  {
    "git_repositories_dir":             "Git.Repositories",
   "disable_api_methods":              "API.DisabledAPIs",
   "max_request_size":                 "API.MaxRequestSize",
   "max_index_database_read":          "API.MaxIndexDatabaseRead",
   "max_items_per_response":           "API.MaxItemsPerResponse",
   "async_permissions_update_interval":         "API.AsyncPermissionsUpdateInterval",
   "auto_setup_new_users":                      "Users.AutoSetupNewUsers",
   "auto_setup_new_users_with_vm_uuid":         "Users.AutoSetupNewUsersWithVmUUID",
   "auto_setup_new_users_with_repository":      "Users.AutoSetupNewUsersWithRepository",
   "auto_setup_name_blacklist":                 "Users.AutoSetupUsernameBlacklist",
   "new_users_are_active":                      "Users.NewUsersAreActive",
   "auto_admin_user":                           "Users.AutoAdminUserWithEmail",
   "auto_admin_first_user":                     "Users.AutoAdminFirstUser",
   "user_profile_notification_address":         "Users.UserProfileNotificationAddress",
   "admin_notifier_email_from":                 "Users.AdminNotifierEmailFrom",
   "email_subject_prefix":                      "Users.EmailSubjectPrefix",
   "user_notifier_email_from":                  "Users.UserNotifierEmailFrom",
   "new_user_notification_recipients":          "Users.NewUserNotificationRecipients",
   "new_inactive_user_notification_recipients": "Users.NewInactiveUserNotificationRecipients",
   "sso_app_secret":                            "Login.ProviderAppSecret",
   "sso_app_id":                                "Login.ProviderAppID",
   "max_audit_log_age":                         "AuditLogs.MaxAge",
   "max_audit_log_delete_batch":                "AuditLogs.MaxDeleteBatch",
   "unlogged_attributes":                       "AuditLogs.UnloggedAttributes",
   "max_request_log_params_size":               "SystemLogs.MaxRequestLogParamsSize",
   "default_collection_replication":            "Collections.DefaultReplication",
   "default_trash_lifetime":                    "Collections.DefaultTrashLifetime",
   "collection_versioning":                     "Collections.CollectionVersioning",
   "preserve_version_if_idle":                  "Collections.PreserveVersionIfIdle",
   "trash_sweep_interval":                      "Collections.TrashSweepInterval",
   "blob_signing_key":                          "Collections.BlobSigningKey",
   "blob_signature_ttl":                        "Collections.BlobSigningTTL",
   "permit_create_collection_with_unsigned_manifest": "Collections.BlobSigning", # XXX
   "docker_image_formats":             "Containers.SupportedDockerImageFormats",
   "log_reuse_decisions":              "Containers.LogReuseDecisions",
   "container_default_keep_cache_ram": "Containers.DefaultKeepCacheRAM",
   "max_container_dispatch_attempts":  "Containers.MaxDispatchAttempts",
   "container_count_max":              "Containers.MaxRetryAttempts",
   "preemptible_instances":            "Containers.UsePreemptibleInstances",
   "max_compute_nodes":                "Containers.MaxComputeVMs",
   "crunch_log_bytes_per_event":       "Containers.Logging.LogBytesPerEvent",
   "crunch_log_seconds_between_events": "Containers.Logging.LogSecondsBetweenEvents",
   "crunch_log_throttle_period":        "Containers.Logging.LogThrottlePeriod",
   "crunch_log_throttle_bytes":         "Containers.Logging.LogThrottleBytes",
   "crunch_log_throttle_lines":         "Containers.Logging.LogThrottleLines",
   "crunch_limit_log_bytes_per_job":    "Containers.Logging.LimitLogBytesPerJob",
   "crunch_log_partial_line_throttle_period": "Containers.Logging.LogPartialLineThrottlePeriod",
   "crunch_log_update_period":                "Containers.Logging.LogUpdatePeriod",
   "crunch_log_update_size":                  "Containers.Logging.LogUpdateSize",
   "clean_container_log_rows_after":          "Containers.Logging.MaxAge",
   "dns_server_conf_dir":                     "Containers.SLURM.Managed.DNSServerConfDir",
   "dns_server_conf_template":                "Containers.SLURM.Managed.DNSServerConfTemplate",
   "dns_server_reload_command":               "Containers.SLURM.Managed.DNSServerReloadCommand",
   "dns_server_update_command":               "Containers.SLURM.Managed.DNSServerUpdateCommand",
   "compute_node_domain":                     "Containers.SLURM.Managed.ComputeNodeDomain",
   "compute_node_nameservers":                "Containers.SLURM.Managed.ComputeNodeNameservers",
   "assign_node_hostname":                    "Containers.SLURM.Managed.AssignNodeHostname",
   "enable_legacy_jobs_api":                  "Containers.JobsAPI.Enable",
   "crunch_job_wrapper":                      "Containers.JobsAPI.CrunchJobWrapper",
   "crunch_job_user":                         "Containers.JobsAPI.CrunchJobUser",
   "crunch_refresh_trigger":                  "Containers.JobsAPI.CrunchRefreshTrigger",
   "git_internal_dir":                        "Containers.JobsAPI.GitInternalDir",
   "reuse_job_if_outputs_differ":             "Containers.JobsAPI.ReuseJobIfOutputsDiffer",
   "default_docker_image_for_jobs":           "Containers.JobsAPI.DefaultDockerImage",
   "mailchimp_api_key":                       "Mail.MailchimpAPIKey",
   "mailchimp_list_id":                       "Mail.MailchimpListID",
}

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
  cfg = $arvados_config

  if config_key_map[k.to_sym]
     k = config_key_map[k.to_sym]
  end

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

puts $arvados_config.to_yaml

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
