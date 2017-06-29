# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class AnonymousGroup < ActiveRecord::Migration
  include CurrentApiClient

  def up
    # create the anonymous group and user
    anonymous_group
    anonymous_user
  end

  def down
    act_as_system_user do
      anonymous_user.destroy
      anonymous_group.destroy
    end
  end

end
