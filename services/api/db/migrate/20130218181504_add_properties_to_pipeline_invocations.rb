# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class AddPropertiesToPipelineInvocations < ActiveRecord::Migration
  def change
    add_column :pipeline_invocations, :properties, :text
  end
end
