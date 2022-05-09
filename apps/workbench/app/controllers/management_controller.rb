# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'app_version'

class ManagementController < ApplicationController
  skip_around_action :thread_clear
  skip_around_action :set_thread_api_token
  skip_around_action :require_thread_api_token
  skip_before_action :ensure_arvados_api_exists
  skip_before_action :accept_uuid_as_id_param
  skip_before_action :check_user_agreements
  skip_before_action :check_user_profile
  skip_before_action :load_filters_and_paging_params
  skip_before_action :find_object_by_uuid

  before_action :check_auth_header

  def check_auth_header
    mgmt_token = Rails.configuration.ManagementToken
    auth_header = request.headers['Authorization']

    if mgmt_token.empty?
      render :json => {:errors => "disabled"}, :status => 404
    elsif !auth_header
      render :json => {:errors => "authorization required"}, :status => 401
    elsif auth_header != 'Bearer '+mgmt_token
      render :json => {:errors => "authorization error"}, :status => 403
    end
  end

  def metrics
    render content_type: 'text/plain', plain: <<~EOF
# HELP arvados_config_load_timestamp_seconds Time when config file was loaded.
# TYPE arvados_config_load_timestamp_seconds gauge
arvados_config_load_timestamp_seconds{sha256="#{Rails.configuration.SourceSHA256}"} #{Rails.configuration.LoadTimestamp.to_f}
# HELP arvados_config_source_timestamp_seconds Timestamp of config file when it was loaded.
# TYPE arvados_config_source_timestamp_seconds gauge
arvados_config_source_timestamp_seconds{sha256="#{Rails.configuration.SourceSHA256}"} #{Rails.configuration.SourceTimestamp.to_f}
# HELP arvados_version_running Indicated version is running.
# TYPE arvados_version_running gauge
arvados_version_running{version="#{AppVersion.package_version}"} 1
EOF
  end

  def health
    resp = {"health" => "OK"}
    render json: resp
  end
end
