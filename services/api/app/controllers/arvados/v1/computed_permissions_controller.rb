# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class Arvados::V1::ComputedPermissionsController < ApplicationController
  before_action :admin_required

  def object_list(**args)
    if !['none', '', nil].include?(params[:count])
      raise ArgumentError.new("count parameter must be 'none'")
    end
    params[:count] = 'none'

    if !['0', 0, nil].include?(params[:offset])
      raise ArgumentError.new("non-zero offset parameter #{params[:offset].inspect} is not supported")
    end

    super
  end
end
