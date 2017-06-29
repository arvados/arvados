# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class DescriptionsAreStrings < ActiveRecord::Migration
  def tables_with_description_column
    %w{collections groups jobs pipeline_instances pipeline_templates}
  end

  def up
    tables_with_description_column.each do |table|
      change_column table.to_sym, :description, :string, :limit => 2**19
    end
  end

  def down
    tables_with_description_column.each do |table|
      if table == 'collections'
        change_column table.to_sym, :description, :string # implicit limit 255
      else
        change_column table.to_sym, :description, :text
      end
    end
  end
end
