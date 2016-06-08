class AddComponentsToJob < ActiveRecord::Migration
  def up
    add_column :jobs, :components, :text
  end

  def down
    if column_exists?(:jobs, :components)
      remove_column :jobs, :components
    end
  end
end
