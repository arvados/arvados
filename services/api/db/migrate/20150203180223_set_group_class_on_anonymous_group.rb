# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class SetGroupClassOnAnonymousGroup < ActiveRecord::Migration[4.2]
  include CurrentApiClient
  def up
    act_as_system_user do
      anonymous_group.update group_class: 'role', name: 'Anonymous users', description: 'Anonymous users'
    end
  end

  def down
    act_as_system_user do
      anonymous_group.update group_class: nil, name: 'Anonymous group', description: 'Anonymous group'
    end
  end
end
