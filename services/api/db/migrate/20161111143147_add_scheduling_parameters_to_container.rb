# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class AddSchedulingParametersToContainer < ActiveRecord::Migration
  def change
    add_column :containers, :scheduling_parameters, :text
    add_column :container_requests, :scheduling_parameters, :text
  end
end
