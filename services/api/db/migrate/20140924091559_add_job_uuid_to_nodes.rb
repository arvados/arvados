class AddJobUuidToNodes < ActiveRecord::Migration
  def up
    change_table :nodes do |t|
      t.column :job_uuid, :string
    end
  end

  def down
    change_table :nodes do |t|
      t.remove :job_uuid
    end
  end
end
