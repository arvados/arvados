class AddSchedulingParametersToContainer < ActiveRecord::Migration
  def change
    add_column :containers, :scheduling_parameters, :text
    add_column :container_requests, :scheduling_parameters, :text
  end
end
