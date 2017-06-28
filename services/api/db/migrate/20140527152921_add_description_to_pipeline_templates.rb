# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class AddDescriptionToPipelineTemplates < ActiveRecord::Migration
  def change
    add_column :pipeline_templates, :description, :text
  end
end
