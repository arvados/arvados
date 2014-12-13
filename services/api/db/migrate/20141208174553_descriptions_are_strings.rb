class DescriptionsAreStrings < ActiveRecord::Migration
  def tables_with_description_column
    %w{collections groups jobs pipeline_instances pipeline_templates}
  end

  def up
    tables_with_description_column.each do |table|
      change_column table.to_sym, :description, :string, :limit => 10000
    end
  end

  def down
    tables_with_description_column.each do |table|
      if table != 'collections'
        change_column table.to_sym, :description, :text
      end
    end
  end
end
