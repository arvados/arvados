class OwnerUuidIndex < ActiveRecord::Migration
  def tables_with_owner_uuid
    ActiveRecord::Base.connection.tables.select do |table|
      columns = ActiveRecord::Base.connection.columns(table)
      columns.collect(&:name).include? 'owner_uuid'
    end
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
