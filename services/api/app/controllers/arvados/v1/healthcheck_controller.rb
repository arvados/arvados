# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class Arvados::V1::HealthcheckController < ApplicationController
  skip_before_action :catch_redirect_hint
  skip_before_action :find_objects_for_index
  skip_before_action :find_object_by_uuid
  skip_before_action :load_filters_param
  skip_before_action :load_limit_offset_order_params
  skip_before_action :load_select_param
  skip_before_action :load_read_auths
  skip_before_action :load_where_param
  skip_before_action :render_404_if_no_object
  skip_before_action :require_auth_scope

  before_action :check_auth_header

  def check_auth_header
    mgmt_token = Rails.configuration.ManagementToken
    auth_header = request.headers['Authorization']

    if mgmt_token == ""
      send_json ({"errors" => "disabled"}), status: 404
    elsif !auth_header
      send_json ({"errors" => "authorization required"}), status: 401
    elsif auth_header != 'Bearer '+mgmt_token
      send_json ({"errors" => "authorization error"}), status: 403
    end
  end

  def ping
    resp = {"health" => "OK"}
    send_json resp
  end
end
