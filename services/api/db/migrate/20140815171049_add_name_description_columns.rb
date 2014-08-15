class AddNameDescriptionColumns < ActiveRecord::Migration
  def up
    add_column :jobs, :name, :string
    add_column :jobs, :description, :text
    add_column :pipeline_instances, :description, :text
  end

  def down
    remove_column :jobs, :name
    remove_column :jobs, :description
    remove_column :pipeline_instances, :description
  end
end
