# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class RenameFolderToProject < ActiveRecord::Migration
  def up
    Group.update_all("group_class = 'project'", "group_class = 'folder'")
  end

  def down
    Group.update_all("group_class = 'folder'", "group_class = 'project'")
  end
end
