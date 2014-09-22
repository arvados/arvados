class AddJobIdToNodes < ActiveRecord::Migration
  def change
    change_table :nodes do |t|
      t.references :job
    end
  end
end
