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
# Use "bundle exec config:migrate" to migrate application.yml to
# config.yml.  After adding the output of config:migrate to
# /etc/arvados/config.yml, you will be able to delete application.yml.

require 'config_loader'
require 'config_validators'
require 'open3'

# Load the defaults, used by config:migrate and fallback loading
# legacy application.yml
load_time = Time.now.utc
defaultYAML, stderr, status = Open3.capture3("arvados-server", "config-dump", "-config=-", "-skip-legacy", stdin_data: "Clusters: {xxxxx: {}}")
if !status.success?
  puts stderr
  raise "error loading config: #{status}"
end
confs = YAML.load(defaultYAML, deserialize_symbols: false)
clusterID, clusterConfig = confs["Clusters"].first
$arvados_config_defaults = clusterConfig
$arvados_config_defaults["ClusterID"] = clusterID
$arvados_config_defaults["SourceTimestamp"] = Time.rfc3339(confs["SourceTimestamp"])
$arvados_config_defaults["SourceSHA256"] = confs["SourceSHA256"]

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
      $arvados_config_global["SourceTimestamp"] = Time.rfc3339(confs["SourceTimestamp"])
      $arvados_config_global["SourceSHA256"] = confs["SourceSHA256"]
    else
      # config-dump failed, assume we will be loading from legacy
      # application.yml, initialize with defaults.
      $arvados_config_global = $arvados_config_defaults.deep_dup
    end
  end
end

# Now make a copy
$arvados_config = $arvados_config_global.deep_dup
$arvados_config["LoadTimestamp"] = load_time

# Declare all our configuration items.
arvcfg = ConfigLoader.new

arvcfg.declare_config "ManagementToken", String, :ManagementToken
arvcfg.declare_config "TLS.Insecure", Boolean, :arvados_insecure_https
arvcfg.declare_config "Collections.TrustAllContent", Boolean, :trust_all_content

arvcfg.declare_config "Services.Controller.ExternalURL", URI, :arvados_v1_base, ->(cfg, k, v) {
  u = URI(v)
  u.path = ""
  ConfigLoader.set_cfg cfg, "Services.Controller.ExternalURL", u
}

arvcfg.declare_config "Services.WebShell.ExternalURL", URI, :shell_in_a_box_url, ->(cfg, k, v) {
  v ||= ""
  u = URI(v.sub("%{hostname}", "*"))
  u.path = ""
  ConfigLoader.set_cfg cfg, "Services.WebShell.ExternalURL", u
}

arvcfg.declare_config "Services.WebDAV.ExternalURL", URI, :keep_web_url, ->(cfg, k, v) {
  v ||= ""
  u = URI(v.sub("%{uuid_or_pdh}", "*"))
  u.path = ""
  ConfigLoader.set_cfg cfg, "Services.WebDAV.ExternalURL", u
}

arvcfg.declare_config "Services.WebDAVDownload.ExternalURL", URI, :keep_web_download_url, ->(cfg, k, v) {
  v ||= ""
  u = URI(v.sub("%{uuid_or_pdh}", "*"))
  u.path = ""
  ConfigLoader.set_cfg cfg, "Services.WebDAVDownload.ExternalURL", u
}

arvcfg.declare_config "Services.Composer.ExternalURL", URI, :composer_url
arvcfg.declare_config "Services.Workbench2.ExternalURL", URI, :workbench2_url

arvcfg.declare_config "Users.AnonymousUserToken", String, :anonymous_user_token

arvcfg.declare_config "Workbench.SecretKeyBase", String, :secret_key_base

arvcfg.declare_config "Workbench.ApplicationMimetypesWithViewIcon", Hash, :application_mimetypes_with_view_icon, ->(cfg, k, v) {
  mimetypes = {}
  v.each do |m|
    mimetypes[m] = {}
  end
  ConfigLoader.set_cfg cfg, "Workbench.ApplicationMimetypesWithViewIcon", mimetypes
}

arvcfg.declare_config "Workbench.RunningJobLogRecordsToFetch", Integer, :running_job_log_records_to_fetch
arvcfg.declare_config "Workbench.LogViewerMaxBytes", Integer, :log_viewer_max_bytes
arvcfg.declare_config "Workbench.ProfilingEnabled", Boolean, :profiling_enabled
arvcfg.declare_config "Workbench.APIResponseCompression", Boolean, :api_response_compression
arvcfg.declare_config "Workbench.UserProfileFormFields", Hash, :user_profile_form_fields, ->(cfg, k, v) {
  if !v
    v = []
  end
  entries = {}
  v.each_with_index do |s,i|
    entries[s["key"]] = {
      "Type" => s["type"],
      "FormFieldTitle" => s["form_field_title"],
      "FormFieldDescription" => s["form_field_description"],
      "Required" => s["required"],
      "Position": i
    }
    if s["options"]
      entries[s["key"]]["Options"] = {}
      s["options"].each do |o|
        entries[s["key"]]["Options"][o] = {}
      end
    end
  end
  ConfigLoader.set_cfg cfg, "Workbench.UserProfileFormFields", entries
}
arvcfg.declare_config "Workbench.UserProfileFormMessage", String, :user_profile_form_message
arvcfg.declare_config "Workbench.Theme", String, :arvados_theme
arvcfg.declare_config "Workbench.ShowUserNotifications", Boolean, :show_user_notifications
arvcfg.declare_config "Workbench.ShowUserAgreementInline", Boolean, :show_user_agreement_inline
arvcfg.declare_config "Workbench.RepositoryCache", String, :repository_cache
arvcfg.declare_config "Workbench.Repositories", Boolean, :repositories
arvcfg.declare_config "Workbench.APIClientConnectTimeout", ActiveSupport::Duration, :api_client_connect_timeout
arvcfg.declare_config "Workbench.APIClientReceiveTimeout", ActiveSupport::Duration, :api_client_receive_timeout
arvcfg.declare_config "Workbench.APIResponseCompression", Boolean, :api_response_compression
arvcfg.declare_config "Workbench.SiteName", String, :site_name
arvcfg.declare_config "Workbench.MultiSiteSearch", String, :multi_site_search, ->(cfg, k, v) {
  if !v
    v = ""
  end
  ConfigLoader.set_cfg cfg, "Workbench.MultiSiteSearch", v.to_s
}
arvcfg.declare_config "Workbench.EnablePublicProjectsPage", Boolean, :enable_public_projects_page
arvcfg.declare_config "Workbench.EnableGettingStartedPopup", Boolean, :enable_getting_started_popup
arvcfg.declare_config "Workbench.ArvadosPublicDataDocURL", String, :arvados_public_data_doc_url
arvcfg.declare_config "Workbench.ArvadosDocsite", String, :arvados_docsite
arvcfg.declare_config "Workbench.ShowRecentCollectionsOnDashboard", Boolean, :show_recent_collections_on_dashboard
arvcfg.declare_config "Workbench.ActivationContactLink", String, :activation_contact_link
arvcfg.declare_config "Workbench.DefaultOpenIdPrefix", String, :default_openid_prefix

arvcfg.declare_config "Mail.SendUserSetupNotificationEmail", Boolean, :send_user_setup_notification_email
arvcfg.declare_config "Mail.IssueReporterEmailFrom", String, :issue_reporter_email_from
arvcfg.declare_config "Mail.IssueReporterEmailTo", String, :issue_reporter_email_to
arvcfg.declare_config "Mail.SupportEmailAddress", String, :support_email_address
arvcfg.declare_config "Mail.EmailFrom", String, :email_from

application_config = {}
%w(application.default application).each do |cfgfile|
  path = "#{::Rails.root.to_s}/config/#{cfgfile}.yml"
  confs = ConfigLoader.load(path, erb: true)
  # Ignore empty YAML file:
  next if confs == false
  application_config.deep_merge!(confs['common'] || {})
  application_config.deep_merge!(confs[::Rails.env.to_s] || {})
end

$remaining_config = arvcfg.migrate_config(application_config, $arvados_config)

# Checks for wrongly typed configuration items, coerces properties
# into correct types (such as Duration), and optionally raise error
# for essential configuration that can't be empty.
arvcfg.coercion_and_check $arvados_config_defaults, check_nonempty: false
arvcfg.coercion_and_check $arvados_config_global, check_nonempty: false
arvcfg.coercion_and_check $arvados_config, check_nonempty: true

# * $arvados_config_defaults is the defaults
# * $arvados_config_global is $arvados_config_defaults merged with the contents of /etc/arvados/config.yml
# These are used by the rake config: tasks
#
# * $arvados_config is $arvados_config_global merged with the migrated contents of application.yml
# This is what actually gets copied into the Rails configuration object.

ArvadosWorkbench::Application.configure do
  # Copy into the Rails config object.  This also turns Hash into
  # OrderedOptions so that application code can use
  # Rails.configuration.API.Blah instead of
  # Rails.configuration.API["Blah"]
  ConfigLoader.copy_into_config $arvados_config, config
  ConfigLoader.copy_into_config $remaining_config, config
  secrets.secret_key_base = $arvados_config["Workbench"]["SecretKeyBase"]
  if ENV["ARVADOS_CONFIG"] != "none"
    ConfigValidators.validate_wb2_url_config()
    ConfigValidators.validate_download_config()
  end
  if Rails.configuration.Users.AnonymousUserToken and
     !Rails.configuration.Users.AnonymousUserToken.starts_with?("v2/")
    Rails.configuration.Users.AnonymousUserToken = "v2/#{clusterID}-gj3su-anonymouspublic/#{Rails.configuration.Users.AnonymousUserToken}"
  end
end
