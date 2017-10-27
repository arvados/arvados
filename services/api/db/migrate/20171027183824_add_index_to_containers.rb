class AddIndexToContainers < ActiveRecord::Migration
  def up
    ActiveRecord::Base.connection.execute("CREATE INDEX index_containers_on_modified_at_uuid ON containers USING btree (modified_at desc, uuid asc)")
    ActiveRecord::Base.connection.execute("CREATE INDEX index_container_requests_on_container_uuid on container_requests (container_uuid)")
  end

  def down
    ActiveRecord::Base.connection.execute("DROP INDEX IF EXISTS index_containers_on_modified_at_uuid")
    ActiveRecord::Base.connection.execute("DROP INDEX IF EXISTS index_container_requests_on_container_uuid")
  end
end
