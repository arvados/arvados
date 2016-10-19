class AddUseExistingToContainerRequests < ActiveRecord::Migration
  def up
    add_column :container_requests, :use_existing, :boolean, :default => true
  end

  def down
    remove_column :container_requests, :use_existing
  end
end
