# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class SysController < ApplicationController
  skip_before_action :find_object_by_uuid
  skip_before_action :render_404_if_no_object
  before_action :admin_required

  def sweep_trash
    act_as_system_user do
      # Sweep trashed collections
      Collection.
        where('delete_at is not null and delete_at < statement_timestamp()').
        destroy_all
      Collection.
        where('is_trashed = false and trash_at < statement_timestamp()').
        update_all('is_trashed = true')

      # Sweep trashed projects and their contents
      Group.
        where({group_class: 'project'}).
        where('delete_at is not null and delete_at < statement_timestamp()').each do |project|
          delete_project_and_contents(project.uuid)
      end
      Group.
        where({group_class: 'project'}).
        where('is_trashed = false and trash_at < statement_timestamp()').
        update_all('is_trashed = true')

      # Sweep expired tokens
      ActiveRecord::Base.connection.execute("DELETE from api_client_authorizations where expires_at <= statement_timestamp()")
    end
  end

  protected

  def delete_project_and_contents(p_uuid)
    p = Group.find_by_uuid(p_uuid)
    if !p || p.group_class != 'project'
      raise "can't sweep group '#{p_uuid}', it may not exist or not be a project"
    end
    # First delete sub projects
    Group.where({group_class: 'project', owner_uuid: p_uuid}).each do |sub_project|
      delete_project_and_contents(sub_project.uuid)
    end
    # Next, iterate over all tables which have owner_uuid fields, with some
    # exceptions, and delete records owned by this project
    skipped_classes = ['Group', 'User']
    ActiveRecord::Base.descendants.reject(&:abstract_class?).each do |klass|
      if !skipped_classes.include?(klass.name) && klass.columns.collect(&:name).include?('owner_uuid')
        klass.where({owner_uuid: p_uuid}).destroy_all
      end
    end
    # Finally delete the project itself
    p.destroy
  end
end
