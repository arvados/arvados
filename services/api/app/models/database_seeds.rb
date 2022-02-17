# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'update_permissions'

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
      anonymous_user_token_api_client
      system_root_token_api_client
      public_project_group
      public_project_read_permission
      empty_collection
    end
    refresh_trashed
  end
end
