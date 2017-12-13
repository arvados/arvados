class AddGinIndexToCollectionProperties < ActiveRecord::Migration
  def up
    ActiveRecord::Base.connection.execute("CREATE INDEX collection_index_on_properties ON collections USING gin (properties);")
  end
  def down
    ActiveRecord::Base.connection.execute("DROP INDEX collection_index_on_properties")
  end
end
