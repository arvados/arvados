class AddRuntimeTokenToContainer < ActiveRecord::Migration
  def change
    add_column :containers, :runtime_token, :text, :null => true
  end
end
