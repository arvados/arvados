# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class EnforceRequiredCredentialFields < ActiveRecord::Migration[7.0]
  def up
    execute <<~SQL
      UPDATE credentials SET name = 'Unnamed' WHERE name IS NULL;
      UPDATE credentials SET credential_class = 'unknown' WHERE credential_class IS NULL;
      UPDATE credentials SET external_id = 'unknown' WHERE external_id IS NULL;
      UPDATE credentials SET secret = '' WHERE secret IS NULL;
      UPDATE credentials SET expires_at = NOW() WHERE expires_at IS NULL;
    SQL

    execute <<~SQL
      ALTER TABLE credentials
      ADD CONSTRAINT credentials_name_not_null CHECK (name IS NOT NULL) NOT VALID,
      ADD CONSTRAINT credentials_credential_class_not_null CHECK (credential_class IS NOT NULL) NOT VALID,
      ADD CONSTRAINT credentials_external_id_not_null CHECK (external_id IS NOT NULL) NOT VALID,
      ADD CONSTRAINT credentials_secret_not_null CHECK (secret IS NOT NULL) NOT VALID,
      ADD CONSTRAINT credentials_expires_at_not_null CHECK (expires_at IS NOT NULL) NOT VALID;
    SQL

    execute <<~SQL
      ALTER TABLE credentials VALIDATE CONSTRAINT credentials_name_not_null;
      ALTER TABLE credentials VALIDATE CONSTRAINT credentials_credential_class_not_null;
      ALTER TABLE credentials VALIDATE CONSTRAINT credentials_external_id_not_null;
      ALTER TABLE credentials VALIDATE CONSTRAINT credentials_secret_not_null;
      ALTER TABLE credentials VALIDATE CONSTRAINT credentials_expires_at_not_null;
    SQL
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

