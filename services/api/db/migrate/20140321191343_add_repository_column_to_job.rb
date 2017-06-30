# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class AddRepositoryColumnToJob < ActiveRecord::Migration
  def up
    add_column :jobs, :repository, :string
  end

  def down
    remove_column :jobs, :repository
  end
end
