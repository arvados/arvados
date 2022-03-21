# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require '20200501150153_permission_table_constants'

class FrozenGroups < ActiveRecord::Migration[5.0]
  def up
    create_table :frozen_groups, :id => false do |t|
      t.string :uuid
    end
    add_index :frozen_groups, :uuid, :unique => true

    ActiveRecord::Base.connection.execute %{
create or replace function project_subtree_with_is_frozen (starting_uuid varchar(27), starting_is_frozen boolean)
returns table (uuid varchar(27), is_frozen boolean)
STABLE
language SQL
as $$
WITH RECURSIVE
  project_subtree(uuid, is_frozen) as (
    values (starting_uuid, starting_is_frozen)
    union
    select groups.uuid, project_subtree.is_frozen or groups.frozen_by_uuid is not null
      from groups join project_subtree on project_subtree.uuid = groups.owner_uuid
  )
  select uuid, is_frozen from project_subtree;
$$;
}

    # Initialize the table. After this, it is updated incrementally.
    # See app/models/group.rb#update_frozen_groups
    refresh_frozen
  end

  def down
    drop_table :frozen_groups
  end
end
