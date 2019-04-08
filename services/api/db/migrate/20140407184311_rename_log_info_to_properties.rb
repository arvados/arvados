# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class RenameLogInfoToProperties < ActiveRecord::Migration[4.2]
  def change
    rename_column :logs, :info, :properties
  end
end
