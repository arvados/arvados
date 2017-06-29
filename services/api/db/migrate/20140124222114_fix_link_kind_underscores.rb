# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class FixLinkKindUnderscores < ActiveRecord::Migration
  def up
    update_sql <<-EOS
UPDATE links
 SET head_kind = 'arvados#virtualMachine'
 WHERE head_kind = 'arvados#virtual_machine'
EOS
  end

  def down
    update_sql <<-EOS
UPDATE links
 SET head_kind = 'arvados#virtual_machine'
 WHERE head_kind = 'arvados#virtualMachine'
EOS
  end
end
