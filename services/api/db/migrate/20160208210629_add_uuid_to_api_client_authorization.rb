# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'has_uuid'

class AddUuidToApiClientAuthorization < ActiveRecord::Migration
  extend HasUuid::ClassMethods

  def up
    add_column :api_client_authorizations, :uuid, :string
    add_index :api_client_authorizations, :uuid, :unique => true

    prefix = Server::Application.config.uuid_prefix + '-' +
             Digest::MD5.hexdigest('ApiClientAuthorization'.to_s).to_i(16).to_s(36)[-5..-1] + '-'

    update_sql <<-EOS
update api_client_authorizations set uuid = (select concat('#{prefix}',
array_to_string(ARRAY (SELECT substring(api_token FROM (ceil(random()*36))::int FOR 1) FROM generate_series(1, 15)), '')
));
EOS

    change_column_null :api_client_authorizations, :uuid, false
  end

  def down
    if column_exists?(:api_client_authorizations, :uuid)
      remove_index :api_client_authorizations, :uuid
      remove_column :api_client_authorizations, :uuid
    end
  end
end
