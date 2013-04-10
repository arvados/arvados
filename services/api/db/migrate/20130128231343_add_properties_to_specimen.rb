class AddPropertiesToSpecimen < ActiveRecord::Migration
  def change
    add_column :specimens, :properties, :text
  end
end
