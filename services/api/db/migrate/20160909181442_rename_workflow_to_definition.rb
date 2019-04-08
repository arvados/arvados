# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class RenameWorkflowToDefinition < ActiveRecord::Migration[4.2]
  def up
    rename_column :workflows, :workflow, :definition
  end

  def down
    rename_column :workflows, :definition, :workflow
  end
end

