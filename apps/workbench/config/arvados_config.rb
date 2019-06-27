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

begin
  # If secret_token.rb exists here, we need to load it first.
  require_relative 'secret_token.rb'
rescue LoadError
  # Normally secret_token.rb is missing and the secret token is
  # configured by application.yml (i.e., here!) instead.
end

# Load the defaults
$arvados_config_defaults = ConfigLoader.load "#{::Rails.root.to_s}/config/config.default.yml"
if $arvados_config_defaults.empty?
  raise "Missing #{::Rails.root.to_s}/config/config.default.yml"
end

def remove_sample_entries(h)
  return unless h.is_a? Hash
  h.delete("SAMPLE")
  h.each { |k, v| remove_sample_entries(v) }
end
remove_sample_entries($arvados_config_defaults)

clusterID, clusterConfig = $arvados_config_defaults["Clusters"].first
$arvados_config_defaults = clusterConfig
$arvados_config_defaults["ClusterID"] = clusterID

# Initialize the global config with the defaults
$arvados_config_global = $arvados_config_defaults.deep_dup

# Load the global config file
confs = ConfigLoader.load "/etc/arvados/config.yml"
if !confs.empty?
  clusterID, clusterConfig = confs["Clusters"].first
  $arvados_config_global["ClusterID"] = clusterID

  # Copy the cluster config over the defaults
  $arvados_config_global.deep_merge!(clusterConfig)
end

# Now make a copy
$arvados_config = $arvados_config_global.deep_dup

# Declare all our configuration items.
arvcfg = ConfigLoader.new

arvcfg.declare_config "ManagementToken", String, :ManagementToken
arvcfg.declare_config "TLS.Insecure", Boolean, :arvados_insecure_https

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

arvcfg.declare_config "Services.WebDAV.ExternalURL", URI, :keep_web_service_url, ->(cfg, k, v) {
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

arvcfg.declare_config "Workbench.ApplicationMimetypesWithViewIcon", Hash, :application_mimetypes_with_view_icon, ->(cfg, k, v) {
  mimetypes = {}
  v.each do |m|
    mimetypes[m] = {}
  end
  ConfigLoader.set_cfg cfg, "Workbench.ApplicationMimetypesWithViewIcon", mimetypes
}

arvcfg.declare_config "Users.AnonymousUserToken", String, :anonymous_user_token


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
  secrets.secret_key_base = $arvados_config["Workbench"]["SecretToken"]
end
