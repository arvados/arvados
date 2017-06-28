# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class NormalizeCollectionUuid < ActiveRecord::Migration
  def count_orphans
    %w(head tail).each do |ht|
      results = ActiveRecord::Base.connection.execute(<<-EOS)
SELECT COUNT(links.*)
 FROM links
 LEFT JOIN collections c
   ON links.#{ht}_uuid = c.uuid
 WHERE (#{ht}_kind='arvados#collection' or #{ht}_uuid ~ '^[0-9a-f]{32,}')
   AND #{ht}_uuid IS NOT NULL
   AND #{ht}_uuid NOT IN (SELECT uuid FROM collections)
EOS
      puts "#{results.first['count'].to_i} links with #{ht}_uuid pointing nowhere."
    end
  end

  def up
    # Normalize uuids in the collections table to
    # {hash}+{size}. Existing uuids might be {hash},
    # {hash}+{size}+K@{instance-name}, {hash}+K@{instance-name}, etc.

    count_orphans
    puts "Normalizing collection UUIDs."

    update_sql <<-EOS
UPDATE collections
 SET uuid = regexp_replace(uuid,'\\+.*','') || '+' || length(manifest_text)
 WHERE uuid !~ '^[0-9a-f]{32,}\\+[0-9]+$'
   AND (regexp_replace(uuid,'\\+.*','') || '+' || length(manifest_text))
     NOT IN (SELECT uuid FROM collections)
EOS

    count_orphans
    puts "Updating links by stripping +K@.* from *_uuid attributes."

    update_sql <<-EOS
UPDATE links
 SET head_uuid = regexp_replace(head_uuid,'\\+K@.*','')
 WHERE head_uuid like '%+K@%'
EOS
    update_sql <<-EOS
UPDATE links
 SET tail_uuid = regexp_replace(tail_uuid,'\\+K@.*','')
 WHERE tail_uuid like '%+K@%'
EOS

    count_orphans
    puts "Updating links by searching bare collection hashes using regexp."

    # Next, update {hash} (and any other non-normalized forms) to
    # {hash}+{size}. This can only work where the corresponding
    # collection is found in the collections table (otherwise we can't
    # know the size).
    %w(head tail).each do |ht|
      update_sql <<-EOS
UPDATE links
 SET #{ht}_uuid = c.uuid
 FROM collections c
 WHERE #{ht}_uuid IS NOT NULL
   AND (#{ht}_kind='arvados#collection' or #{ht}_uuid ~ '^[0-9a-f]{32,}')
   AND #{ht}_uuid NOT IN (SELECT uuid FROM collections)
   AND regexp_replace(#{ht}_uuid,'\\+.*','') = regexp_replace(c.uuid,'\\+.*','')
   AND c.uuid ~ '^[0-9a-f]{32,}\\+[0-9]+$'
EOS
    end

    count_orphans
    puts "Stripping \"+K@.*\" from jobs.output, jobs.log, job_tasks.output."

    update_sql <<-EOS
UPDATE jobs
 SET output = regexp_replace(output,'\\+K@.*','')
 WHERE output ~ '^[0-9a-f]{32,}\\+[0-9]+\\+K@\\w+$'
EOS
    update_sql <<-EOS
UPDATE jobs
 SET log = regexp_replace(log,'\\+K@.*','')
 WHERE log ~ '^[0-9a-f]{32,}\\+[0-9]+\\+K@\\w+$'
EOS
    update_sql <<-EOS
UPDATE job_tasks
 SET output = regexp_replace(output,'\\+K@.*','')
 WHERE output ~ '^[0-9a-f]{32,}\\+[0-9]+\\+K@\\w+$'
EOS

    puts "Done."
  end

  def down
  end
end
