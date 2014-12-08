class SearchIndex < ActiveRecord::Migration
  def tables_with_searchable_columns
    all_tables =  ActiveRecord::Base.connection.tables
    all_tables.delete 'schema_migrations'

    my_tables = []
    all_tables.each do |table|
      table_class = table.classify.constantize
      if table_class.respond_to?('searchable_columns')
        my_tables << table
      end
    end

    my_tables
  end

  def up
    tables_with_searchable_columns.each do |table|
      indices = table.classify.constantize.searchable_columns('ilike')
      add_index(table.to_sym, indices, name: "#{table}_search_index")
    end
  end

  def down
    tables_with_searchable_columns.each do |table, indices|
      remove_index(table.to_sym, name: "#{table}_search_index")
    end
  end
end
