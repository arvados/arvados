# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class SetCurrentVersionUuidOnCollections < ActiveRecord::Migration
  def up
    # Set the current version uuid as itself
    Collection.where(current_version_uuid: nil).update_all("current_version_uuid=uuid")
  end

  def down
  end
end
