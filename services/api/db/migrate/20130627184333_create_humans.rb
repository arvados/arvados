class CreateHumans < ActiveRecord::Migration
  def change
    create_table :humans do |t|
      t.string :uuid, :null => false
      t.string :owner, :null => false
      t.string :modified_by_client
      t.string :modified_by_user
      t.datetime :modified_at
      t.text :properties

      t.timestamps
    end
    add_index :humans, :uuid, :unique => true
  end
end
