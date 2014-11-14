class AddArvadosSdkVersionToJobs < ActiveRecord::Migration
  def up
    change_table :jobs do |t|
      t.column :arvados_sdk_version, :string
    end
  end

  def down
    change_table :jobs do |t|
      t.remove :arvados_sdk_version
    end
  end
end
