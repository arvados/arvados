class AddOutputTtlToContainerRequests < ActiveRecord::Migration
  def change
    add_column :container_requests, :output_ttl, :integer, default: 0, null: false
  end
end
