# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'migrate_yaml_to_json'

class YamlToJson < ActiveRecord::Migration
  def up
    [
      ['collections', 'properties'],
      ['containers', 'environment'],
      ['containers', 'mounts'],
      ['containers', 'runtime_constraints'],
      ['containers', 'command'],
      ['containers', 'scheduling_parameters'],
      ['container_requests', 'properties'],
      ['container_requests', 'environment'],
      ['container_requests', 'mounts'],
      ['container_requests', 'runtime_constraints'],
      ['container_requests', 'command'],
      ['container_requests', 'scheduling_parameters'],
      ['humans', 'properties'],
      ['job_tasks', 'parameters'],
      ['links', 'properties'],
      ['nodes', 'info'],
      ['nodes', 'properties'],
      ['pipeline_instances', 'components'],
      ['pipeline_instances', 'properties'],
      ['pipeline_instances', 'components_summary'],
      ['pipeline_templates', 'components'],
      ['specimens', 'properties'],
      ['traits', 'properties'],
      ['users', 'prefs'],
    ].each do |table, column|
      MigrateYAMLToJSON.migrate(table, column)
    end
  end

  def down
  end
end
