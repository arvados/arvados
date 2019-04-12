# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class AddTasksSummaryToJobs < ActiveRecord::Migration[4.2]
  def change
    add_column :jobs, :tasks_summary, :text
  end
end
