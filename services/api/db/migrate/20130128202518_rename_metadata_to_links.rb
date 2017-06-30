# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class RenameMetadataToLinks < ActiveRecord::Migration
  def up
    rename_table :metadata, :links
    rename_column :links, :tail, :tail_uuid
    rename_column :links, :head, :head_uuid
    rename_column :links, :info, :properties
    rename_column :links, :metadata_class, :link_class
    rename_index :links, :index_metadata_on_head_kind, :index_links_on_head_kind
    rename_index :links, :index_metadata_on_head, :index_links_on_head_uuid
    rename_index :links, :index_metadata_on_tail_kind, :index_links_on_tail_kind
    rename_index :links, :index_metadata_on_tail, :index_links_on_tail_uuid
    rename_index :links, :index_metadata_on_uuid, :index_links_on_uuid
  end

  def down
    rename_index :links, :index_links_on_uuid, :index_metadata_on_uuid
    rename_index :links, :index_links_on_head_kind, :index_metadata_on_head_kind
    rename_index :links, :index_links_on_head_uuid, :index_metadata_on_head
    rename_index :links, :index_links_on_tail_kind, :index_metadata_on_tail_kind
    rename_index :links, :index_links_on_tail_uuid, :index_metadata_on_tail
    rename_column :links, :link_class, :metadata_class
    rename_column :links, :properties, :info
    rename_column :links, :head_uuid, :head
    rename_column :links, :tail_uuid, :tail
    rename_table :links, :metadata
  end
end
