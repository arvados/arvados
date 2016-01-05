class AddExitCodeToContainers < ActiveRecord::Migration
  def change
    add_column :containers, :exit_code, :integer
  end
end
