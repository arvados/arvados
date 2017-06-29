# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'has_uuid'

class AddOutputAndLogUuidToContainerRequest < ActiveRecord::Migration
  extend HasUuid::ClassMethods

  def up
    add_column :container_requests, :output_uuid, :string
    add_column :container_requests, :log_uuid, :string

    no_such_out_coll = Server::Application.config.uuid_prefix + '-' + '4zz18' + '-xxxxxxxxxxxxxxx'
    no_such_log_coll = Server::Application.config.uuid_prefix + '-' + '4zz18' + '-yyyyyyyyyyyyyyy'

    update_sql <<-EOS
update container_requests set output_uuid = ('#{no_such_out_coll}'), log_uuid = ('#{no_such_log_coll}');
EOS
  end

  def down
    remove_column :container_requests, :log_uuid
    remove_column :container_requests, :output_uuid
  end
end
