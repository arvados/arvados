class AddRepositoryColumnToJob < ActiveRecord::Migration
  def up
    add_column :jobs, :repository, :string
  end

  def down
    remove_column :jobs, :repository
  end
end
