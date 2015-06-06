require 'test_helper'

class RecordFiltersTest < ActiveSupport::TestCase
  include RecordFilters

  ArvadosModel.descendants.each do |model|
    # Postgres skips indexes when querying small tables, so these
    # tests fail inappropriately. We include it to remind ourselves
    # how to run the appropriate tests when we get around that snag.
    break

    ft_idx = model.connection.indexes(model.table_name).select do |idx|
      /_full_text_/ =~ idx.name
    end.first
    next unless ft_idx
    test "@@ query on #{model.to_s} uses full text index" do
      act_as_user users(:active) do
        (1..100).each do |i|
          m = model.new
          if m.respond_to? :name=
              m.name = "test number #{i}"
          end
          if m.respond_to? :manifest_text=
              m.manifest_text = ". d41d8cd98f00b204e9800998ecf8427e+0 0:0:foo\n"
          end
          m.save!
        end
      end
      query = record_filters [['any', '@@', 'foo:*']], model
      plan = model.
        where(query[:conditions].join(' AND '), *query[:params]).
        explain
      assert_match /#{ft_idx.name}/, plan
    end
  end
end
