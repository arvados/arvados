# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class StaticController < ApplicationController
  respond_to :json, :html

  skip_before_action :find_object_by_uuid
  skip_before_action :render_404_if_no_object
  skip_before_action :require_auth_scope, only: [:home, :empty, :login_failure]

  def home
    respond_to do |f|
      f.html do
        if !Rails.configuration.Services.Workbench1.ExternalURL.to_s.empty?
          redirect_to Rails.configuration.Services.Workbench1.ExternalURL.to_s, allow_other_host: true
        else
          render_not_found "Oops, this is an API endpoint. You probably want to point your browser to an Arvados Workbench site instead."
        end
      end
      f.json do
        render_not_found "Path not found."
      end
    end
  end

  def empty
    render plain: ""
  end

end
