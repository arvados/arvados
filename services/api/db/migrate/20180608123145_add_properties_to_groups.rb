class AddPropertiesToGroups < ActiveRecord::Migration
  def up
    add_column :groups, :properties, :jsonb, default: {}
    ActiveRecord::Base.connection.execute("CREATE INDEX group_index_on_properties ON groups USING gin (properties);")
  end

  def down
    ActiveRecord::Base.connection.execute("DROP INDEX IF EXISTS group_index_on_properties")
    remove_column :groups, :properties
  end
end
