class AddMd5IndexToContainers < ActiveRecord::Migration
  def up
    ActiveRecord::Base.connection.execute 'CREATE INDEX index_containers_on_reuse_columns on containers (md5(command), cwd, md5(environment), output_path, container_image, md5(mounts), secret_mounts_md5, md5(runtime_constraints))'
  end
  def down
    ActiveRecord::Base.connection.execute 'DROP INDEX index_containers_on_reuse_columns'
  end
end
