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

  test "unique uuid index exists on all models with the column uuid" do 
    tables = ActiveRecord::Base.connection.tables
    tables.each do |table|
      columns = ActiveRecord::Base.connection.columns(table)

      uuid_column = columns.select do |column|
        column.name == 'uuid'
      end

      if !uuid_column.empty?
        indexes = ActiveRecord::Base.connection.indexes(table)
        uuid_index = indexes.select do |index|
          index.columns == ['uuid'] and index.unique == true
        end

        assert !uuid_index.empty?, "#{table} does not have unique uuid index"
      end
    end
  end

  test "owner uuid index exists on all models with the owner_uuid column" do
    all_tables = ActiveRecord::Base.connection.tables

    all_tables.each do |table|
      columns = ActiveRecord::Base.connection.columns(table)

      uuid_column = columns.select do |column|
        column.name == 'owner_uuid'
      end

      if !uuid_column.empty?
        indexes = ActiveRecord::Base.connection.indexes(table)
        owner_uuid_index = indexes.select do |index|
          index.columns == ['owner_uuid']
        end
        assert !owner_uuid_index.empty?, "#{table} does not have owner_uuid index"
      end
    end
  end
end
