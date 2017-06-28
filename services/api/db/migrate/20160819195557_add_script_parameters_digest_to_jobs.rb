# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class AddScriptParametersDigestToJobs < ActiveRecord::Migration
  def change
    add_column :jobs, :script_parameters_digest, :string
    add_index :jobs, :script_parameters_digest
  end
end
