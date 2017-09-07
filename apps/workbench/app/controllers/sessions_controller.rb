# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class SessionsController < ApplicationController
  skip_around_filter :require_thread_api_token, :only => [:destroy, :logged_out]
  skip_around_filter :set_thread_api_token, :only => [:destroy, :logged_out]
  skip_before_filter :find_object_by_uuid
  skip_before_filter :find_objects_for_index
  skip_before_filter :ensure_arvados_api_exists

  def destroy
    session.clear
    redirect_to arvados_api_client.arvados_logout_url(return_to: root_url)
  end

  def logged_out
    redirect_to root_url if session[:arvados_api_token]
    render_index
  end

  def index
  end
end
