# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class DropCommitAncestors < ActiveRecord::Migration[5.0]
  def change
    drop_table :commit_ancestors
  end
end
