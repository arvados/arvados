# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class StatusController < ApplicationController
  skip_around_filter :require_thread_api_token
  skip_before_filter :find_object_by_uuid
  def status
    # Allow non-credentialed cross-origin requests
    headers['Access-Control-Allow-Origin'] = '*'
    resp = {
      apiBaseURL: arvados_api_client.arvados_v1_base.sub(%r{/arvados/v\d+.*}, '/'),
      version: AppVersion.hash,
    }
    render json: resp
  end
end
