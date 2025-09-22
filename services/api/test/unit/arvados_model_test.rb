# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'test_helper'

class ArvadosModelTest < ActiveSupport::TestCase
  fixtures :all

  def create_with_attrs attrs
    a = Collection.create({properties: {'foo' => 'bar'}}.merge(attrs))
    a if a.valid?
  end

  test 'non-admin cannot assign uuid' do
    set_user_from_auth :active_trustedclient
    want_uuid = Collection.generate_uuid
    a = create_with_attrs(uuid: want_uuid)
    assert_nil a, "Non-admin should not assign uuid."
  end

  test 'admin can assign valid uuid' do
    set_user_from_auth :admin_trustedclient
    want_uuid = Collection.generate_uuid
    a = create_with_attrs(uuid: want_uuid)
    assert_equal want_uuid, a.uuid, "Admin should assign valid uuid."
    assert a.uuid.length==27, "Auto assigned uuid length is wrong."
  end

  test 'admin cannot assign uuid with wrong object type' do
    set_user_from_auth :admin_trustedclient
    want_uuid = Group.generate_uuid
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
  ].each do |x|
    test "prevent symbol keys in serialized db columns: #{x.inspect}" do
      set_user_from_auth :active
      link = Link.create!(link_class: 'test',
                          properties: x)
      raw = ActiveRecord::Base.connection.
          select_value("select properties from links where uuid='#{link.uuid}'")
      refute_match(/:[fb]/, raw)
    end
  end

  [ {['foo'] => 'bar'},
    {'a' => {['foo', :foo] => 'bar'}},
    {'a' => {{'foo' => 'bar'} => 'bar'}},
    {'a' => {['foo', :foo] => ['bar', 'baz']}},
  ].each do |x|
    test "refuse non-string keys in serialized db columns: #{x.inspect}" do
      set_user_from_auth :active
      assert_raises(ArgumentError) do
        Link.create!(link_class: 'test',
                     properties: x)
      end
    end
  end

  test "No HashWithIndifferentAccess in database" do
    set_user_from_auth :admin_trustedclient
    link = Link.create!(link_class: 'test',
                        properties: {'foo' => 'bar'}.with_indifferent_access)
    raw = ActiveRecord::Base.connection.
      select_value("select properties from links where uuid='#{link.uuid}'")
    assert_equal '{"foo": "bar"}', raw
  end

  test "store long string" do
    set_user_from_auth :active
    longstring = "a"
    while longstring.length < 2**16
      longstring = longstring + longstring
    end
    g = Group.create! name: 'Has a long description', description: longstring, group_class: "project"
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
    ActiveRecord::Base.descendants.each do |model_class|
      next if model_class.abstract_class?
      next if !model_class.respond_to?('searchable_columns')

      search_index_columns = model_class.searchable_columns('ilike')
      # Disappointing, but text columns aren't indexed yet.
      search_index_columns -= model_class.columns.select { |c|
        c.type == :text or c.name == 'description' or c.name == 'file_names'
      }.collect(&:name)
      next if search_index_columns.empty?

      indexes = ActiveRecord::Base.connection.indexes(model_class.table_name)
      search_index_by_columns = indexes.select do |index|
        # After rails 5.0 upgrade, AR::Base.connection.indexes() started to include
        # GIN indexes, with its 'columns' attribute being a String like
        # 'to_tsvector(...)'
        index.columns.is_a?(Array) ? index.columns.sort == search_index_columns.sort : false
      end
      search_index_by_name = indexes.select do |index|
        index.name == "#{model_class.table_name}_search_index"
      end
      assert !search_index_by_columns.empty?, "#{model_class.table_name} (#{model_class.to_s}) has no search index with columns #{search_index_columns}. Instead found search index with columns #{search_index_by_name.first.andand.columns}"
    end
  end

  [Collection, ContainerRequest, Group, Workflow].each do |model|
    test "trigram index exists on #{model} model" do
      expect = model.full_text_searchable_columns
      conn = ActiveRecord::Base.connection
      index_name = "#{model.table_name}_trgm_text_search_idx"
      indexes = conn.exec_query("SELECT indexdef FROM pg_indexes WHERE tablename = '#{model.table_name}' AND indexname = '#{index_name}'")
      assert_not_equal(indexes.length, 0)
      indexes.each do |res|
        searchable = res['indexdef'].scan(/COALESCE\(+([A-Za-z_]+)/).flatten
        assert_equal(
          searchable, expect,
          "Invalid or no trigram index for #{model} named #{index_name}\nexpect: #{expect.inspect}\nfound: #{searchable}",
        )
      end
    end

    test "UUID and hash columns are excluded from #{model} full text index" do
      assert_equal(
        model.full_text_searchable_columns & full_text_excluded_columns, [],
        "UUID/hash columns returned by #{model}.full_text_searchable_columns",
      )
    end
  end

  test "selectable_attributes includes database attributes" do
    assert_includes(Collection.selectable_attributes, "name")
  end

  test "selectable_attributes includes non-database attributes" do
    assert_includes(Collection.selectable_attributes, "unsigned_manifest_text")
  end

  test "selectable_attributes includes common attributes in extensions" do
    assert_includes(Collection.selectable_attributes, "uuid")
  end

  test "selectable_attributes does not include unexposed attributes" do
    refute_includes(Collection.selectable_attributes, "id")
  end

  test "selectable_attributes on a non-default template" do
    attr_a = Collection.selectable_attributes(:common)
    assert_includes(attr_a, "uuid")
    refute_includes(attr_a, "name")
  end

  test 'create and retrieve using created_at time' do
    set_user_from_auth :active
    group = Group.create! name: 'test create and retrieve group', group_class: "project"
    assert group.valid?, "group is not valid"

    results = Group.where(created_at: group.created_at)
    assert_includes results.map(&:uuid), group.uuid,
      "Expected new group uuid in results when searched with its created_at timestamp"
  end

  test 'create and update twice and expect different update times' do
    set_user_from_auth :active
    group = Group.create! name: 'test create and retrieve group', group_class: "project"
    assert group.valid?, "group is not valid"

    # update 1
    group.update!(name: "test create and update name 1")
    results = Group.where(uuid: group.uuid)
    assert_equal "test create and update name 1", results.first.name, "Expected name to be updated to 1"
    modified_at_1 = results.first.modified_at.to_f

    # update 2
    group.update!(name: "test create and update name 2")
    results = Group.where(uuid: group.uuid)
    assert_equal "test create and update name 2", results.first.name, "Expected name to be updated to 2"
    modified_at_2 = results.first.modified_at.to_f

    assert_equal true, (modified_at_2 > modified_at_1), "Expected modified time 2 to be newer than 1"
  end

  test 'jsonb column' do
    set_user_from_auth :active

    c = Collection.create!(properties: {})
    assert_equal({}, c.properties)

    c.update(properties: {'foo' => 'foo'})
    c.reload
    assert_equal({'foo' => 'foo'}, c.properties)

    c.update(properties: nil)
    c.reload
    assert_equal({}, c.properties)

    c.update(properties: {foo: 'bar'})
    assert_equal({'foo' => 'bar'}, c.properties)
    c.reload
    assert_equal({'foo' => 'bar'}, c.properties)
  end

  {
    Collection => ["description", "manifest_text"],
    Container => [
      "command",
      "environment",
      "output_properties",
      "runtime_constraints",
      "secret_mounts",
    ],
    ContainerRequest => [
      "command",
      "environment",
      "mounts",
      "output_glob",
      "output_properties",
      "properties",
      "runtime_constraints",
      "secret_mounts",
    ],
    Group => ["description", "properties"],
    Log => ["properties", "summary"],
  }.each_pair do |model, expect|
    test "#{model.name} limits expected columns on index" do
      assert_equal(
        (model.limit_index_columns_read & expect).sort,
        expect.sort,
      )
    end
  end

  {
    Collection => ["delete_at", "preserve_version", "trash_at", "version"],
    Container => ["cost", "progress", "state", "subrequests_cost"],
    ContainerRequest => ["container_uuid", "cwd", "requesting_container_uuid"],
    Group => ["group_class", "is_trashed", "trashed_at"],
    Log => ["event_at", "event_type"],
  }.each_pair do |model, colnames|
    test "#{model.name} does not limit expected columns on index" do
      assert_equal(model.limit_index_columns_read & colnames, [])
    end
  end

  test 'serialized attributes dirty tracking with audit log settings' do
    Rails.configuration.AuditLogs.MaxDeleteBatch = 1000
    set_user_from_auth :admin
    [false, true].each do |auditlogs_enabled|
      if auditlogs_enabled
        Rails.configuration.AuditLogs.MaxAge = 3600
      else
        Rails.configuration.AuditLogs.MaxAge = 0
      end
      tested_serialized = false
      [
        User.find_by_uuid(users(:active).uuid),
        ContainerRequest.find_by_uuid(container_requests(:queued).uuid),
        Container.find_by_uuid(containers(:queued).uuid),
        Group.find_by_uuid(groups(:afiltergroup).uuid),
        Collection.find_by_uuid(collections(:collection_with_one_property).uuid),
      ].each do |obj|
        if !obj.class.serialized_attributes.empty?
          tested_serialized = true
        end
        # obj shouldn't have changed since it's just retrieved from the database
        assert_not(obj.changed?, "#{obj.class} model's attribute(s) appear as changed: '#{obj.changes.keys.join(',')}' with audit logs #{auditlogs_enabled ? '': 'not '}enabled.")
      end
      assert(tested_serialized, "did not test any models with serialized attributes")
    end
  end

  [
    # prefs column uses `serialize [...], Hash`
    ['users(:active)', 'prefs', '"baddata"'],
    ['users(:active)', 'prefs', '[]'],
    # output_properties column uses `attribute ..., :jsonbHash`
    ['container_requests(:running)', 'output_properties', '"baddata"'],
    ['container_requests(:running)', 'output_properties', '["baddata"]'],
    # output_storage_classes column uses `attribute ..., :jsonbArray`
    ['container_requests(:running)', 'output_storage_classes', '"baddata"'],
    ['container_requests(:running)', 'output_storage_classes', '{}'],
    # environment column uses `serialize [...], Hash`
    ['container_requests(:running)', 'environment', '"baddata"'],
    ['container_requests(:running)', 'environment', '[]'],
    # output_glob column uses `serialize [...], Array`
    ['container_requests(:running)', 'output_glob', '"baddata"'],
    ['container_requests(:running)', 'output_glob', '{}'],
  ].each do |get_fixture, attr, bad_data|
    test "refuse to load #{get_fixture} from database with wrong type of serialized attribute #{attr}, #{bad_data}" do
      object = eval(get_fixture)
      initial_value = object.attributes[attr].dup
      ActiveRecord::Base.connection.exec_query(
        "UPDATE #{object.class.table_name} SET #{attr}=$2 WHERE uuid=$1",
        "",
        [object.uuid, bad_data])
      begin
        e = assert_raises(RuntimeError) do
          object.reload
        end
        assert_match /invalid serialized data for #{object.class.to_s} #{attr}/, e.message
      ensure
        ActiveRecord::Base.connection.exec_query(
          "UPDATE #{object.class.table_name} SET #{attr}=$2 WHERE uuid=$1",
          "",
          [object.uuid, initial_value.to_json])
      end
    end
  end
end
