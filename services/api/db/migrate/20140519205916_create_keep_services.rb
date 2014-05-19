class CreateKeepServices < ActiveRecord::Migration
  def change
    create_table :keep_services do |t|
      t.string :uuid, :null => false
      t.string :owner_uuid, :null => false
      t.string :modified_by_client_uuid
      t.string :modified_by_user_uuid
      t.datetime :modified_at
      t.string   :service_host
      t.integer  :service_port
      t.boolean  :service_ssl_flag
      t.string   :service_type

      t.timestamps
    end
    add_index :keep_services, :uuid, :unique => true
  end
end
