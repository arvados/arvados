# If the database reflects an obsolete version of the 20141208185217
# migration (i.e., before commit:5c1db683), revert it and reapply the
# current version. (The down-migration is the same in both versions.)

require "./db/migrate/20141208185217_search_index.rb"

class NoDescriptionInSearchIndex < ActiveRecord::Migration
  def up
    all_tables = %w{collections groups jobs pipeline_instances pipeline_templates}
    all_tables.each do |table|
      indexes = ActiveRecord::Base.connection.indexes(table)
      search_index_by_name = indexes.select do |index|
        index.name == "#{table}_search_index"
      end

      if !search_index_by_name.empty?
        index_columns = search_index_by_name.first.columns
        has_description = index_columns.include? 'description'
        if has_description
          SearchIndex.new.migrate(:down)
          SearchIndex.new.migrate(:up)
          break
        end
      end
    end
  end

  def down
  end
end
