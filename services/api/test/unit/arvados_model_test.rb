require 'test_helper'

class ArvadosModelTest < ActiveSupport::TestCase
  fixtures :all

  def create_with_attrs attrs
    a = Specimen.create({material: 'caloric'}.merge(attrs))
    a if a.valid?
  end

  test 'non-admin cannot assign uuid' do
    set_user_from_auth :active_trustedclient
    want_uuid = Specimen.generate_uuid
    a = create_with_attrs(uuid: want_uuid)
    assert_nil a, "Non-admin should not assign uuid."
  end

  test 'admin can assign valid uuid' do
    set_user_from_auth :admin_trustedclient
    want_uuid = Specimen.generate_uuid
    a = create_with_attrs(uuid: want_uuid)
    assert_equal want_uuid, a.uuid, "Admin should assign valid uuid."
    assert a.uuid.length==27, "Auto assigned uuid length is wrong."
  end

  test 'admin cannot assign uuid with wrong object type' do
    set_user_from_auth :admin_trustedclient
    want_uuid = Human.generate_uuid
    a = create_with_attrs(uuid: want_uuid)
    assert_nil a, "Admin should not be able to assign invalid uuid."
  end

  test 'admin cannot assign badly formed uuid' do
    set_user_from_auth :admin_trustedclient
    a = create_with_attrs(uuid: "ntoheunthaoesunhasoeuhtnsaoeunhtsth")
    assert_nil a, "Admin should not be able to assign invalid uuid."
  end

  test 'admin cannot assign empty uuid' do
    set_user_from_auth :admin_trustedclient
    a = create_with_attrs(uuid: "")
    assert_nil a, "Admin cannot assign empty uuid."
  end

  [ {:a => 'foo'},
    {'a' => :foo},
    {:a => ['foo', 'bar']},
    {'a' => [:foo, 'bar']},
    {'a' => ['foo', :bar]},
    {:a => [:foo, :bar]},
    {:a => {'foo' => {'bar' => 'baz'}}},
    {'a' => {:foo => {'bar' => 'baz'}}},
    {'a' => {'foo' => {:bar => 'baz'}}},
    {'a' => {'foo' => {'bar' => :baz}}},
    {'a' => {'foo' => ['bar', :baz]}},
    {'a' => {['foo', :foo] => ['bar', 'baz']}},
  ].each do |x|
    test "refuse symbol keys in serialized attribute: #{x.inspect}" do
      set_user_from_auth :admin_trustedclient
      assert_nothing_raised do
        Link.create!(link_class: 'test',
                     properties: {})
      end
      assert_raises ActiveRecord::RecordInvalid do
        Link.create!(link_class: 'test',
                     properties: x)
      end
    end
  end

  test "Stringify symbols coming from serialized attribute in database" do
    set_user_from_auth :admin_trustedclient
    fixed = Link.find_by_uuid(links(:has_symbol_keys_in_database_somehow).uuid)
    assert_equal(["baz", "foo"], fixed.properties.keys.sort,
                 "Hash symbol keys from DB did not get stringified.")
    assert_equal(['waz', 'waz', 'waz', 1, nil, false, true],
                 fixed.properties['baz'],
                 "Array symbol values from DB did not get stringified.")
    assert_equal true, fixed.save, "Failed to save fixed model back to db."
  end

  test "No HashWithIndifferentAccess in database" do
    set_user_from_auth :admin_trustedclient
    assert_raises ActiveRecord::RecordInvalid do
      Link.create!(link_class: 'test',
                   properties: {'foo' => 'bar'}.with_indifferent_access)
    end
  end

  test "store long string" do
    set_user_from_auth :active
    longstring = "a"
    while longstring.length < 2**28
      longstring = longstring + longstring
    end
    g = Group.create! name: 'Has a long description', description: longstring
    g = Group.find_by_uuid g.uuid
    assert_equal g.description, longstring
  end

  [['uuid', {unique: true}],
   ['owner_uuid', {}]].each do |the_column, requires|
    test "unique index on all models with #{the_column}" do
      checked = 0
      ActiveRecord::Base.connection.tables.each do |table|
        columns = ActiveRecord::Base.connection.columns(table)

        next unless columns.collect(&:name).include? the_column

        indexes = ActiveRecord::Base.connection.indexes(table).reject do |index|
          requires.map do |key, val|
            index.send(key) == val
          end.include? false
        end
        assert_includes indexes.collect(&:columns), [the_column], 'no index'
        checked += 1
      end
      # Sanity check: make sure we didn't just systematically miss everything.
      assert_operator(10, :<, checked,
                      "Only #{checked} tables have a #{the_column}?!")
    end
  end

  test "search index exists on models that go into projects" do
    all_tables =  ActiveRecord::Base.connection.tables
    all_tables.delete 'schema_migrations'

    all_tables.each do |table|
      table_class = table.classify.constantize
      if table_class.respond_to?('searchable_columns')
        search_index_columns = table_class.searchable_columns('ilike')

        indexes = ActiveRecord::Base.connection.indexes(table)
        search_index_by_columns = indexes.select do |index|
          index.columns == search_index_columns
        end
        search_index_by_name = indexes.select do |index|
          index.name == "#{table}_search_index"
        end
        assert !search_index_by_columns.empty?, "#{table} has no search index with columns #{search_index_columns}. Instead found search index with columns #{search_index_by_name.first.andand.columns}"
      end
    end
  end
end
