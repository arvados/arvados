# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class AddModifiedAtToLogs < ActiveRecord::Migration
  def change
    add_column :logs, :modified_at, :datetime
  end
end
