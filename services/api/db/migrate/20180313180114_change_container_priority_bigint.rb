class ChangeContainerPriorityBigint < ActiveRecord::Migration
  def change
    change_column :containers, :priority, :integer, limit: 8
  end
end
