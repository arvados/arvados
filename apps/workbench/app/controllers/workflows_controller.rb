# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class WorkflowsController < ApplicationController
  skip_around_action :require_thread_api_token, if: proc { |ctrl|
    !Rails.configuration.Users.AnonymousUserToken.empty? and
    'show' == ctrl.action_name
  }

  def show_pane_list
    %w(Definition Advanced)
  end
end
