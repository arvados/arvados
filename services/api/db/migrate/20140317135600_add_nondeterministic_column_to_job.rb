class AddNondeterministicColumnToJob < ActiveRecord::Migration
  def up
    add_column :jobs, :nondeterministic, :boolean
  end

  def down
    remove_column :jobs, :nondeterministic
  end
end
