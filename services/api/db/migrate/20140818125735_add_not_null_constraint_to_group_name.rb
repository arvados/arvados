class AddNotNullConstraintToGroupName < ActiveRecord::Migration
  def change
    ActiveRecord::Base.connection.execute("update groups set name=uuid where name is null or name=''")
    change_column_null :groups, :name, false
  end
end
