# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class CreateKeepServices < ActiveRecord::Migration
  include CurrentApiClient

  def change
    act_as_system_user do
      create_table :keep_services do |t|
        t.string :uuid, :null => false
        t.string :owner_uuid, :null => false
        t.string :modified_by_client_uuid
        t.string :modified_by_user_uuid
        t.datetime :modified_at
        t.string   :service_host
        t.integer  :service_port
        t.boolean  :service_ssl_flag
        t.string   :service_type

        t.timestamps
      end
      add_index :keep_services, :uuid, :unique => true

      add_column :keep_disks, :keep_service_uuid, :string

      KeepDisk.reset_column_information

      services = {}

      KeepDisk.find_each do |k|
        services["#{k[:service_host]}_#{k[:service_port]}_#{k[:service_ssl_flag]}"] = {
          service_host: k[:service_host],
          service_port: k[:service_port],
          service_ssl_flag: k[:service_ssl_flag],
          service_type: 'disk',
          owner_uuid: k[:owner_uuid]
        }
      end

      services.each do |k, v|
        v['uuid'] = KeepService.create(v).uuid
      end

      KeepDisk.find_each do |k|
        k.keep_service_uuid = services["#{k[:service_host]}_#{k[:service_port]}_#{k[:service_ssl_flag]}"]['uuid']
        k.save
      end

      remove_column :keep_disks, :service_host
      remove_column :keep_disks, :service_port
      remove_column :keep_disks, :service_ssl_flag
    end
  end
end
