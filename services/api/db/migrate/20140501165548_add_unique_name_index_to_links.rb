# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class AddUniqueNameIndexToLinks < ActiveRecord::Migration
  def change
    # Make sure PgPower is here. Otherwise the "where" will be ignored
    # and we'll end up with a far too restrictive unique
    # constraint. (Rails4 should work without PgPower, but that isn't
    # tested.)
    if not PgPower then raise "No partial column support" end

    add_index(:links, [:tail_uuid, :name], unique: true,
              where: "link_class='name'",
              name: 'links_tail_name_unique_if_link_class_name')
  end
end
