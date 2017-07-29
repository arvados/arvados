# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class Arvados::V1::HealthcheckController < ApplicationController
  skip_before_filter :catch_redirect_hint
  skip_before_filter :find_objects_for_index
  skip_before_filter :find_object_by_uuid
  skip_before_filter :load_filters_param
  skip_before_filter :load_limit_offset_order_params
  skip_before_filter :load_read_auths
  skip_before_filter :load_where_param
  skip_before_filter :render_404_if_no_object
  skip_before_filter :require_auth_scope

  before_filter :check_auth_header

  def check_auth_header
    mgmt_token = Rails.configuration.ManagementToken
    auth_header = request.headers['Authorization']

    if !mgmt_token
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
