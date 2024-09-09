# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class ExcludeUuidsAndHashesFromTextSearch < ActiveRecord::Migration[7.0]
  def trgm_indexes
    [
      # Table name, index name, pre-migration full_text_trgm
      ["collections", "collections_trgm_text_search_idx", "(coalesce(owner_uuid,'') || ' ' || coalesce(modified_by_client_uuid,'') || ' ' || coalesce(modified_by_user_uuid,'') || ' ' || coalesce(portable_data_hash,'') || ' ' || coalesce(uuid,'') || ' ' || coalesce(name,'') || ' ' || coalesce(description,'') || ' ' || coalesce(properties::text,'') || ' ' || coalesce(file_names,''))"],
      # container_requests handled by 20240820202230_exclude_container_image_from_text_search.rb
      ["groups", "groups_trgm_text_search_idx", "(coalesce(uuid,'') || ' ' || coalesce(owner_uuid,'') || ' ' || coalesce(modified_by_client_uuid,'') || ' ' || coalesce(modified_by_user_uuid,'') || ' ' || coalesce(name,'') || ' ' || coalesce(description,'') || ' ' || coalesce(group_class,'') || ' ' || coalesce(properties::text,''))"],
      ["workflows", "workflows_trgm_text_search_idx", "(coalesce(uuid,'') || ' ' || coalesce(owner_uuid,'') || ' ' || coalesce(modified_by_client_uuid,'') || ' ' || coalesce(modified_by_user_uuid,'') || ' ' || coalesce(name,'') || ' ' || coalesce(description,''))"],
    ]
  end

  def up
    trgm_indexes.each do |model, indx, _|
      execute "DROP INDEX IF EXISTS #{indx}"
      execute "CREATE INDEX #{indx} ON #{model} USING gin((#{model.classify.constantize.full_text_trgm}) gin_trgm_ops)"
    end
  end

  def down
    trgm_indexes.each do |model, indx, full_text_trgm|
      execute "DROP INDEX IF EXISTS #{indx}"
      execute "CREATE INDEX #{indx} ON #{model} USING gin((#{full_text_trgm}) gin_trgm_ops)"
    end
  end
end
