# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

# Protect referential integrity of owner_uuid columns in other tables
# that can refer to the uuid column in this table.

module CanBeAnOwner

  def self.included(base)
    base.extend(ClassMethods)

    # Rails' "has_many" can prevent us from destroying the owner
    # record when other objects refer to it.
    ActiveRecord::Base.connection.tables.each do |t|
      next if t == base.table_name
      next if t.in?([
                      'schema_migrations',
                      'permission_refresh_lock',
                      'ar_internal_metadata',
                      'commit_ancestors',
                      'commits',
                      'humans',
                      'jobs',
                      'job_tasks',
                      'keep_disks',
                      'materialized_permissions',
                      'nodes',
                      'pipeline_instances',
                      'pipeline_templates',
                      'repositories',
                      'specimens',
                      'traits',
                      'uuid_locks',
                    ])
      klass = t.classify.constantize
      next unless klass and 'owner_uuid'.in?(klass.columns.collect(&:name))
      base.has_many(t.to_sym,
                    foreign_key: 'owner_uuid',
                    primary_key: 'uuid',
                    dependent: :restrict_with_exception)
    end
    # We need custom protection for changing an owner's primary
    # key. (Apart from this restriction, admins are allowed to change
    # UUIDs.)
    base.validate :restrict_uuid_change_breaking_associations
  end

  module ClassMethods
    def install_view(type)
      conn = ActiveRecord::Base.connection
      transaction do
        # Check whether the temporary view has already been created
        # during this connection. If not, create it.
        conn.exec_query "SAVEPOINT check_#{type}_view"
        begin
          conn.exec_query("SELECT 1 FROM #{type}_view LIMIT 0")
        rescue
          conn.exec_query "ROLLBACK TO SAVEPOINT check_#{type}_view"
          sql = File.read(Rails.root.join("lib", "create_#{type}_view.sql"))
          conn.exec_query(sql)
        ensure
          conn.exec_query "RELEASE SAVEPOINT check_#{type}_view"
        end
      end
    end
  end

  def descendant_project_uuids
    self.class.install_view('ancestor')
    ActiveRecord::Base.connection.
      exec_query('SELECT ancestor_view.uuid
                  FROM ancestor_view
                  LEFT JOIN groups ON groups.uuid=ancestor_view.uuid
                  WHERE ancestor_uuid = $1 AND groups.group_class = $2',
                  # "name" arg is a query label that appears in logs:
                  "descendant_project_uuids for #{self.uuid}",
                  # "binds" arg is an array of [col_id, value] for '$1' vars:
                  [self.uuid, 'project'],
                  ).rows.map do |project_uuid,|
      project_uuid
    end
  end

  protected

  def restrict_uuid_change_breaking_associations
    return true if new_record? or not uuid_changed?

    # Check for objects that have my old uuid listed as their owner.
    self.class.reflect_on_all_associations(:has_many).each do |assoc|
      next unless assoc.foreign_key == 'owner_uuid'
      if assoc.klass.where(owner_uuid: uuid_was).any?
        errors.add(:uuid,
                   "cannot be changed on a #{self.class} that owns objects")
        return false
      end
    end

    # if I owned myself before, I'll just continue to own myself with
    # my new uuid.
    if owner_uuid == uuid_was
      self.owner_uuid = uuid
    end
  end
end
