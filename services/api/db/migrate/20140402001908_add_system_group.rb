# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class AddSystemGroup < ActiveRecord::Migration
  include CurrentApiClient

  def up
    # Make sure the system group exists.
    system_group
  end

  def down
    act_as_system_user do
      system_group.destroy

      # Destroy the automatically generated links giving system_group
      # permission on all users.
      Link.destroy_all(tail_uuid: system_group_uuid, head_kind: 'arvados#user')
    end
  end
end
