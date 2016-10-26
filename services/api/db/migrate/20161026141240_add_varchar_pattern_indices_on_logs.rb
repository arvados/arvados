class AddVarcharPatternIndicesOnLogs < ActiveRecord::Migration
  def up
    execute "CREATE INDEX logs_varchar_index_on_object_uuid ON logs (object_uuid varchar_pattern_ops);"
    execute "CREATE INDEX logs_varchar_index_on_owner_uuid ON logs (owner_uuid varchar_pattern_ops);"
  end

  def down
    execute "DROP INDEX logs_varchar_index_on_object_uuid;"
    execute "DROP INDEX logs_varchar_index_on_owner_uuid;"
  end
end
