# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class AddExitCodeToContainers < ActiveRecord::Migration[4.2]
  def change
    add_column :containers, :exit_code, :integer
  end
end
