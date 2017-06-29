# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class RenameJobCommandToScript < ActiveRecord::Migration
  def up
    rename_column :jobs, :command, :script
    rename_column :jobs, :command_parameters, :script_parameters
    rename_column :jobs, :command_version, :script_version
    rename_index :jobs, :index_jobs_on_command, :index_jobs_on_script
  end

  def down
    rename_index :jobs, :index_jobs_on_script, :index_jobs_on_command
    rename_column :jobs, :script_version, :command_version
    rename_column :jobs, :script_parameters, :command_parameters
    rename_column :jobs, :script, :command
  end
end
