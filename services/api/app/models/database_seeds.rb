# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'update_permissions'

# Seed database with default/initial data if needed.
#
# This runs before db:migrate in
# build/rails-package-scripts/postinst.sh so it must only do things
# that are safe in an in-use/production database.
class DatabaseSeeds
  extend CurrentApiClient
  def self.install
    batch_update_permissions do
      system_user
      system_group
      all_users_group
      anonymous_group
      anonymous_group_read_permission
      anonymous_user
      public_project_group
      public_project_read_permission
      empty_collection
    end
    refresh_trashed
  end
end
