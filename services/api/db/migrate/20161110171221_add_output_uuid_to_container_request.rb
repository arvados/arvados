require 'has_uuid'

class AddOutputUuidToContainerRequest < ActiveRecord::Migration
  extend HasUuid::ClassMethods

  def up
    add_column :container_requests, :output_uuid, :string

    no_such_coll = Server::Application.config.uuid_prefix + '-' + '4zz18' + '-xxxxxxxxxxxxxxx'
    update_sql <<-EOS
update container_requests set output_uuid = ('#{no_such_coll}');
EOS
  end

  def down
    remove_column :container_requests, :output_uuid
  end
end
