# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class ApiClientAuthorizationsController < ApplicationController

  def index_pane_list
    %w(Recent Help)
  end

end
