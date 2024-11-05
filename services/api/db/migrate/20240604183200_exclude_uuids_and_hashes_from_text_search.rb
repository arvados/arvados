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
    old_value = query_value('SHOW statement_timeout')
    execute "SET statement_timeout TO '0'"
    trgm_indexes.each do |model, indx, _|
      execute "DROP INDEX IF EXISTS #{indx}"
      execute "CREATE INDEX #{indx} ON #{model} USING gin((#{model.classify.constantize.full_text_trgm}) gin_trgm_ops)"
    end
    execute "SET statement_timeout TO #{quote(old_value)}"
  end

  def down
    old_value = query_value('SHOW statement_timeout')
    execute "SET statement_timeout TO '0'"
    trgm_indexes.each do |model, indx, full_text_trgm|
      execute "DROP INDEX IF EXISTS #{indx}"
      execute "CREATE INDEX #{indx} ON #{model} USING gin((#{full_text_trgm}) gin_trgm_ops)"
    end
    execute "SET statement_timeout TO #{quote(old_value)}"
  end
end
