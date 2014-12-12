class OwnerUuidIndex < ActiveRecord::Migration
  def tables_with_owner_uuid
    %w{api_clients authorized_keys collections groups humans
       job_tasks jobs keep_disks keep_services links logs
       nodes pipeline_instances pipeline_templates repositories
       specimens traits users virtual_machines}
  end

  def up
    tables_with_owner_uuid.each do |table|
      add_index table.to_sym, :owner_uuid
    end
  end

  def down
    tables_with_owner_uuid.each do |table|
      indexes = ActiveRecord::Base.connection.indexes(table)
      owner_uuid_index = indexes.select do |index|
        index.columns == ['owner_uuid']
      end
      if !owner_uuid_index.empty?
        remove_index table.to_sym, :owner_uuid
      end
    end
  end
end
