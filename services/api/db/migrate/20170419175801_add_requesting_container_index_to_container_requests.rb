class AddRequestingContainerIndexToContainerRequests < ActiveRecord::Migration
  def change
    add_index :container_requests, :requesting_container_uuid
  end
end
