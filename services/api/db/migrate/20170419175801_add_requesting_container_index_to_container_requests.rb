# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class AddRequestingContainerIndexToContainerRequests < ActiveRecord::Migration
  def change
    add_index :container_requests, :requesting_container_uuid
  end
end
