# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class KeepService < ArvadosModel
  include HasUuid
  include KindAndEtag
  include CommonApiTemplate
  extend DbCurrentTime

  SERVER_START_TIME = db_current_time

  api_accessible :user, extend: :common do |t|
    t.add  :service_host
    t.add  :service_port
    t.add  :service_ssl_flag
    t.add  :service_type
    t.add  :read_only
  end
  api_accessible :superuser, :extend => :user do |t|
  end

  # return the set of keep services from the database (if this is an
  # older installation or test system where entries have been added
  # manually) or, preferably, the cluster config file.
  def self.all *args
    if super.count == 0
      from_config
    else
      super
    end
  end

  def self.where *args
    all.where *args
  end

  protected

  def permission_to_create
    current_user.andand.is_admin
  end

  def permission_to_update
    current_user.andand.is_admin
  end

  def self.from_config
    config_time = connection.quote(SERVER_START_TIME)
    owner = connection.quote(system_user_uuid)
    values = []
    id = 1
    Rails.configuration.Services.Keepstore.InternalURLs.each do |url, info|
      values << "(#{id}, " + quoted_column_values_from_url(url: url.to_s, rendezvous: info.Rendezvous).join(", ") + ", 'disk', 'f'::bool, #{config_time}, #{config_time}, #{owner}, #{owner}, null)"
      id += 1
    end
    url = Rails.configuration.Services.Keepproxy.ExternalURL.to_s
    if !url.blank?
      values << "(#{id}, " + quoted_column_values_from_url(url: url, rendezvous: "").join(", ") + ", 'proxy', 'f'::bool, #{config_time}, #{config_time}, #{owner}, #{owner}, null)"
      id += 1
    end
    if values.length == 0
      # return empty set as AR relation
      return unscoped.where('1=0')
    else
      sql = "(values #{values.join(", ")}) as keep_services (id, uuid, service_host, service_port, service_ssl_flag, service_type, read_only, created_at, modified_at, owner_uuid, modified_by_user_uuid, modified_by_client_uuid)"
      return unscoped.from(sql)
    end
  end

  private

  def self.quoted_column_values_from_url(url:, rendezvous:)
    rvz = rendezvous
    rvz = url if rvz.blank?
    if /^[a-zA-Z0-9]{15}$/ !~ rvz
      # If rvz is an URL (either the real service URL, or an alternate
      # one specified in config in order to preserve rendezvous order
      # when changing hosts/ports), hash it to get 15 alphanums.
      rvz = Digest::MD5.hexdigest(rvz)[0..15]
    end
    uuid = Rails.configuration.ClusterID + "-bi6l4-" + rvz
    uri = URI::parse(url)
    [uuid, uri.host, uri.port].map { |x| connection.quote(x) } + [(uri.scheme == 'https' ? "'t'::bool" : "'f'::bool")]
  end

end
