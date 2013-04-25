class AddServiceHostAndServicePortAndServiceSslFlagToKeepDisks < ActiveRecord::Migration
  def change
    add_column :keep_disks, :service_host, :string
    add_column :keep_disks, :service_port, :integer
    add_column :keep_disks, :service_ssl_flag, :boolean
    add_index :keep_disks, [:service_host, :service_port, :last_ping_at],
      name: 'keep_disks_service_host_port_ping_at_index'
  end
end
