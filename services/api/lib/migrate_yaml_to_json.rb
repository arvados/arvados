# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

module MigrateYAMLToJSON
  def self.migrate(table, column)
    conn = ActiveRecord::Base.connection
    n = conn.update(
      "UPDATE #{table} SET #{column}=$1 WHERE #{column}=$2",
      "#{table}.#{column} convert YAML to JSON",
      ["{}", "--- {}\n"])
    Rails.logger.info("#{table}.#{column}: #{n} rows updated using empty hash")
    finished = false
    while !finished
      n = 0
      conn.exec_query(
        "SELECT id, #{column} FROM #{table} WHERE #{column} LIKE $1 LIMIT 100",
        "#{table}.#{column} check for YAML",
        ['---%'],
      ).rows.map do |id, yaml|
        n += 1
        json = SafeJSON.dump(YAML.safe_load(yaml))
        conn.exec_query(
          "UPDATE #{table} SET #{column}=$1 WHERE id=$2 AND #{column}=$3",
          "#{table}.#{column} convert YAML to JSON",
          [json, id, yaml])
      end
      Rails.logger.info("#{table}.#{column}: #{n} rows updated")
      finished = (n == 0)
    end
  end
end
