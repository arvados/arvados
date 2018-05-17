# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class LinkAccountController < ApplicationController
  skip_before_filter :find_objects_for_index

  def index
  end

  def merge
    User.merge params[:new_user_token]
    redirect_to "/"
  end

  def model_class
    "User"
  end
end
