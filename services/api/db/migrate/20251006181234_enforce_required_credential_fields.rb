# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class EnforceRequiredCredentialFields < ActiveRecord::Migration[7.0]
  def up
    execute <<~SQL
      UPDATE credentials SET name = '' WHERE name IS NULL;
      UPDATE credentials SET credential_class = '' WHERE credential_class IS NULL;
      UPDATE credentials SET external_id = '' WHERE external_id IS NULL;
      UPDATE credentials SET secret = '' WHERE secret IS NULL;
      UPDATE credentials SET expires_at = NOW() WHERE expires_at IS NULL;
    SQL

    change_column_null :credentials, :name, false
    change_column_null :credentials, :credential_class, false
    change_column_null :credentials, :external_id, false
    change_column_null :credentials, :secret, false
    change_column_null :credentials, :expires_at, false
  end

  def down
    execute <<~SQL
      ALTER TABLE credentials
      DROP CONSTRAINT IF EXISTS credentials_name_not_null,
      DROP CONSTRAINT IF EXISTS credentials_credential_class_not_null,
      DROP CONSTRAINT IF EXISTS credentials_external_id_not_null,
      DROP CONSTRAINT IF EXISTS credentials_secret_not_null,
      DROP CONSTRAINT IF EXISTS credentials_expires_at_not_null;
    SQL
  end
end

