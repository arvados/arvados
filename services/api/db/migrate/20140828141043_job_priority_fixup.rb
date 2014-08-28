class JobPriorityFixup < ActiveRecord::Migration
  def up
    change_column :jobs, :priority, :string, null: false, default: "0"
  end

  def down
    change_column :jobs, :priority, :string, null: true, default: nil
  end
end
