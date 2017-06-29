# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class ActivateAllAdmins < ActiveRecord::Migration
  def up
    User.update_all({is_active: true}, ['is_admin=?', true])
  end

  def down
  end
end
