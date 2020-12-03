# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'set'

include CurrentApiClient
include ArvadosModelUpdates

def fix_collection_versions_timestamps
  act_as_system_user do
    uuids = [].to_set
    # Get UUIDs from collections with more than 1 version
    Collection.where(version: 2).find_each(batch_size: 100) do |c|
      uuids.add(c.current_version_uuid)
    end
    uuids.each do |uuid|
      first_pair = true
      # All versions of a particular collection get fixed together.
      ActiveRecord::Base.transaction do
        Collection.where(current_version_uuid: uuid).order(version: :desc).each_cons(2) do |c1, c2|
          # Skip if the 2 newest versions' modified_at values are separate enough;
          # this means that this collection doesn't require fixing, allowing for
          # migration re-runs in case of transient problems.
          break if first_pair && (c2.modified_at.to_f - c1.modified_at.to_f) > 1
          first_pair = false
          # Fix modified_at timestamps by assigning to N-1's value to N.
          # Special case: the first version's modified_at will be == to created_at
          leave_modified_by_user_alone do
            leave_modified_at_alone do
              c1.modified_at = c2.modified_at
              c1.save!(validate: false)
              if c2.version == 1
                c2.modified_at = c2.created_at
                c2.save!(validate: false)
              end
            end
          end
        end
      end
    end
  end
end