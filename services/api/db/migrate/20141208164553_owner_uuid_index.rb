class OwnerUuidIndex < ActiveRecord::Migration
  def tables_with_owner_uuid
    all_tables = ActiveRecord::Base.connection.tables
    my_tables = []
    all_tables.each do |table|
      columns = ActiveRecord::Base.connection.columns(table)
      uuid_column = columns.select do |column|
        column.name == 'owner_uuid'
      end
      my_tables << table if !uuid_column.empty?
    end
    my_tables
  end

  def up
    tables_with_owner_uuid.each do |table|
      add_index table.to_sym, :owner_uuid
    end
  end

  def down
    tables_with_owner_uuid.each do |table|
      remove_index table.to_sym, :owner_uuid
    end
  end
end
