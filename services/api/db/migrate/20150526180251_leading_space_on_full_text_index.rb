require "./db/migrate/20150123142953_full_text_search.rb"

class LeadingSpaceOnFullTextIndex < ActiveRecord::Migration
  def up
    # Inspect one of the full-text indexes (chosen arbitrarily) to
    # determine whether this migration is needed.
    ft_index_name = 'jobs_full_text_search_idx'
    ActiveRecord::Base.connection.indexes('jobs').each do |idx|
      if idx.name == ft_index_name
        if idx.columns.first.index "((((' '"
          # Index is already correct. This happens if the source tree
          # already had the new version of full_text_tsvector by the
          # time the initial FullTextSearch migration ran.
          $stderr.puts "This migration is not needed."
        else
          # Index was created using the old full_text_tsvector. Drop
          # and re-create all full text indexes.
          FullTextSearch.new.migrate(:down)
          FullTextSearch.new.migrate(:up)
        end
        return
      end
    end
    raise "Did not find index '#{ft_index_name}'. Earlier migration missed??"
  end

  def down
    $stderr.puts <<EOS
Down-migration is not supported for this change, and might be unnecessary.

If you run a code base older than 20150526180251 against this
database, full text search will be slow even on collections where it
used to work well. If this is a concern, first check out the desired
older version of the code base, and then run
"rake db:migrate:down VERSION=20150123142953"
followed by
"rake db:migrate:up VERSION=20150123142953"
.
EOS
  end
end
