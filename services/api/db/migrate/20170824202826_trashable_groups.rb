# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class TrashableGroups < ActiveRecord::Migration
  def up
    add_column :groups, :trash_at, :datetime
    add_index(:groups, :trash_at)

    add_column :groups, :is_trashed, :boolean, null: false, default: false
    add_index(:groups, :is_trashed)

    add_column :groups, :delete_at, :datetime
    add_index(:groups, :delete_at)

    Group.reset_column_information
    add_index(:groups, [:owner_uuid, :name],
              unique: true,
              where: 'is_trashed = false',
              name: 'index_groups_on_owner_uuid_and_name')
    remove_index(:groups,
                 name: 'groups_owner_uuid_name_unique')
  end

  def down
    Group.transaction do
      add_index(:groups, [:owner_uuid, :name], unique: true,
                name: 'groups_owner_uuid_name_unique')
      remove_index(:groups,
                   name: 'index_groups_on_owner_uuid_and_name')

      remove_index(:groups, :delete_at)
      remove_column(:groups, :delete_at)

      remove_index(:groups, :is_trashed)
      remove_column(:groups, :is_trashed)

      remove_index(:groups, :trash_at)
      remove_column(:groups, :trash_at)
    end
  end
end
