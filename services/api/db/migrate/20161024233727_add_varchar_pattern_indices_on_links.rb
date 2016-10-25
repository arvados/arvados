class AddVarcharPatternIndicesOnLinks < ActiveRecord::Migration
  def up
    execute "CREATE INDEX links_varchar_index_on_head_uuid ON links (head_uuid varchar_pattern_ops);"
    execute "CREATE INDEX links_varchar_index_on_tail_uuid ON links (tail_uuid varchar_pattern_ops);"
    execute "CREATE INDEX links_varchar_index_on_owner_uuid ON links (owner_uuid varchar_pattern_ops);"
  end

  def down
    execute "DROP INDEX links_varchar_index_on_owner_uuid;"
    execute "DROP INDEX links_varchar_index_on_tail_uuid;"
    execute "DROP INDEX links_varchar_index_on_head_uuid;"
  end
end
