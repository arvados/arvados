# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'fix_roles_projects'

class FixRolesProjects < ActiveRecord::Migration[5.0]
  def change
    # This migration is not reversible.  However, the results are
    # backwards compatible.
    fix_roles_projects
  end
end
