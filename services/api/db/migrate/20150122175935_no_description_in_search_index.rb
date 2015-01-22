require "./db/migrate/20141208185217_search_index.rb"

class NoDescriptionInSearchIndex < ActiveRecord::Migration
  def change
    all_tables = %w{collections groups jobs pipeline_instances pipeline_templates}
    all_tables.each do |table|
      indexes = ActiveRecord::Base.connection.indexes(table)
      search_index_by_name = indexes.select do |index|
        index.name == "#{table}_search_index"
      end

      index_columns = search_index_by_name.first.andand.columns
      has_description = index_columns.select.each do |column|
        column == 'description'
      end

      if !has_description.empty?
        SearchIndex.new.migrate(:down)
        SearchIndex.new.migrate(:up)
        break
      end
    end
  end
end
