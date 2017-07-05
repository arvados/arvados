# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'migrate_yaml_to_json'

class JobsYamlToJson < ActiveRecord::Migration
  def up
    [
      'components',
      'script_parameters',
      'runtime_constraints',
      'tasks_summary',
    ].each do |column|
      MigrateYAMLToJSON.migrate("jobs", column)
    end
  end

  def down
  end
end
