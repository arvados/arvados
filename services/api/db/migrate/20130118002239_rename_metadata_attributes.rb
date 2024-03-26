# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class RenameMetadataAttributes < ActiveRecord::Migration[4.2]
  def up
    rename_column :metadata, :target_kind, :tail_kind
    rename_column :metadata, :target_uuid, :tail
    rename_column :metadata, :value, :head
    rename_column :metadata, :key, :name
    add_column :metadata, :head_kind, :string
    add_index :metadata, :head
    add_index :metadata, :head_kind
    add_index :metadata, :tail
    add_index :metadata, :tail_kind
    begin
      Metadatum.where('head like ?', 'orvos#%').each do |m|
        kind_uuid = m.head.match /^(orvos\#.*)\#([-0-9a-z]+)$/
        if kind_uuid
          m.update(head_kind: kind_uuid[1],
                              head: kind_uuid[2])
        end
      end
    rescue
    end
  end

  def down
    begin
      Metadatum.where('head_kind is not null and head_kind <> ? and head is not null', '').each do |m|
        m.update(head: m.head_kind + '#' + m.head)
      end
    rescue
    end
    remove_index :metadata, :tail_kind
    remove_index :metadata, :tail
    remove_index :metadata, :head_kind
    remove_index :metadata, :head
    rename_column :metadata, :name, :key
    remove_column :metadata, :head_kind
    rename_column :metadata, :head, :value
    rename_column :metadata, :tail, :target_uuid
    rename_column :metadata, :tail_kind, :target_kind
  end
end
