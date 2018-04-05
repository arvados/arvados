# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'current_api_client'

module SweepTrashedObjects
  extend CurrentApiClient

  def self.delete_project_and_contents(p_uuid)
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

  def self.sweep_now
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
    end
  end

  def self.sweep_if_stale
    return if Rails.configuration.trash_sweep_interval <= 0
    exp = Rails.configuration.trash_sweep_interval.seconds
    need = false
    Rails.cache.fetch('SweepTrashedObjects', expires_in: exp) do
      need = true
    end
    if need
      Thread.new do
        Thread.current.abort_on_exception = false
        begin
          sweep_now
        rescue => e
          Rails.logger.error "#{e.class}: #{e}\n#{e.backtrace.join("\n\t")}"
        ensure
          ActiveRecord::Base.connection.close
        end
      end
    end
  end
end
