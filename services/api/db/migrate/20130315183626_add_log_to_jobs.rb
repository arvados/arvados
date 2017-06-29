# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class AddLogToJobs < ActiveRecord::Migration
  def change
    add_column :jobs, :log, :string
  end
end
