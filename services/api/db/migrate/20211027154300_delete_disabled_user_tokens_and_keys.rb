# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class DeleteDisabledUserTokensAndKeys < ActiveRecord::Migration[5.2]
  def up
    execute "delete from api_client_authorizations where user_id in (select id from users where is_active ='false' and uuid not like '%-tpzed-anonymouspublic' and uuid not like '%-tpzed-000000000000000')"
    execute "delete from authorized_keys where owner_uuid in (select uuid from users where is_active ='false' and uuid not like '%-tpzed-anonymouspublic' and uuid not like '%-tpzed-000000000000000')"
    execute "delete from authorized_keys where authorized_user_uuid in (select uuid from users where is_active ='false' and uuid not like '%-tpzed-anonymouspublic' and uuid not like '%-tpzed-000000000000000')"
  end

  def down
    # This migration is not reversible.
  end
end
