# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class AddInteractiveSessionStartedToContainers < ActiveRecord::Migration[5.2]
  def change
    add_column :containers, :interactive_session_started, :boolean, null: false, default: false
  end
end
