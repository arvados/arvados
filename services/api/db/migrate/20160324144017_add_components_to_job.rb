# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class AddComponentsToJob < ActiveRecord::Migration
  def up
    add_column :jobs, :components, :text
  end

  def down
    if column_exists?(:jobs, :components)
      remove_column :jobs, :components
    end
  end
end
