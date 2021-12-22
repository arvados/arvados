# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

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
    all_tables =  ActiveRecord::Base.connection.tables
    all_tables.delete 'schema_migrations'
    all_tables.delete 'permission_refresh_lock'
    all_tables.delete 'ar_internal_metadata'

    all_tables.each do |table|
      table_class = table.classify.constantize
      if table_class.respond_to?('searchable_columns')
        search_index_columns = table_class.searchable_columns('ilike')
        # Disappointing, but text columns aren't indexed yet.
        search_index_columns -= table_class.columns.select { |c|
          c.type == :text or c.name == 'description' or c.name == 'file_names'
        }.collect(&:name)

        indexes = ActiveRecord::Base.connection.indexes(table)
        search_index_by_columns = indexes.select do |index|
          # After rails 5.0 upgrade, AR::Base.connection.indexes() started to include
          # GIN indexes, with its 'columns' attribute being a String like
          # 'to_tsvector(...)'
          index.columns.is_a?(Array) ? index.columns.sort == search_index_columns.sort : false
        end
        search_index_by_name = indexes.select do |index|
          index.name == "#{table}_search_index"
        end
        assert !search_index_by_columns.empty?, "#{table} has no search index with columns #{search_index_columns}. Instead found search index with columns #{search_index_by_name.first.andand.columns}"
      end
    end
  end

  [
    %w[collections collections_trgm_text_search_idx],
    %w[container_requests container_requests_trgm_text_search_idx],
    %w[groups groups_trgm_text_search_idx],
    %w[jobs jobs_trgm_text_search_idx],
    %w[pipeline_instances pipeline_instances_trgm_text_search_idx],
    %w[pipeline_templates pipeline_templates_trgm_text_search_idx],
    %w[workflows workflows_trgm_text_search_idx]
  ].each do |model|
    table = model[0]
    indexname = model[1]
    test "trigram index exists on #{table} model" do
      table_class = table.classify.constantize
      expect = table_class.full_text_searchable_columns
      ok = false
      conn = ActiveRecord::Base.connection
      conn.exec_query("SELECT indexdef FROM pg_indexes WHERE tablename = '#{table}' AND indexname = '#{indexname}'").each do |res|
        searchable = res['indexdef'].scan(/COALESCE\(+([A-Za-z_]+)/).flatten
        ok = (expect == searchable)
        assert ok, "Invalid or no trigram index on #{table} named #{indexname}\nexpect: #{expect.inspect}\nfound: #{searchable}"
      end
    end
  end

  test "selectable_attributes includes database attributes" do
    assert_includes(Job.selectable_attributes, "success")
  end

  test "selectable_attributes includes non-database attributes" do
    assert_includes(Job.selectable_attributes, "node_uuids")
  end

  test "selectable_attributes includes common attributes in extensions" do
    assert_includes(Job.selectable_attributes, "uuid")
  end

  test "selectable_attributes does not include unexposed attributes" do
    refute_includes(Job.selectable_attributes, "nodes")
  end

  test "selectable_attributes on a non-default template" do
    attr_a = Job.selectable_attributes(:common)
    assert_includes(attr_a, "uuid")
    refute_includes(attr_a, "success")
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
    group.update_attributes!(name: "test create and update name 1")
    results = Group.where(uuid: group.uuid)
    assert_equal "test create and update name 1", results.first.name, "Expected name to be updated to 1"
    updated_at_1 = results.first.updated_at.to_f

    # update 2
    group.update_attributes!(name: "test create and update name 2")
    results = Group.where(uuid: group.uuid)
    assert_equal "test create and update name 2", results.first.name, "Expected name to be updated to 2"
    updated_at_2 = results.first.updated_at.to_f

    assert_equal true, (updated_at_2 > updated_at_1), "Expected updated time 2 to be newer than 1"
  end

  test 'jsonb column' do
    set_user_from_auth :active

    c = Collection.create!(properties: {})
    assert_equal({}, c.properties)

    c.update_attributes(properties: {'foo' => 'foo'})
    c.reload
    assert_equal({'foo' => 'foo'}, c.properties)

    c.update_attributes(properties: nil)
    c.reload
    assert_equal({}, c.properties)

    c.update_attributes(properties: {foo: 'bar'})
    assert_equal({'foo' => 'bar'}, c.properties)
    c.reload
    assert_equal({'foo' => 'bar'}, c.properties)
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
      [
        User.find_by_uuid(users(:active).uuid),
        ContainerRequest.find_by_uuid(container_requests(:queued).uuid),
        Container.find_by_uuid(containers(:queued).uuid),
        PipelineInstance.find_by_uuid(pipeline_instances(:has_component_with_completed_jobs).uuid),
        PipelineTemplate.find_by_uuid(pipeline_templates(:two_part).uuid),
        Job.find_by_uuid(jobs(:running).uuid)
      ].each do |obj|
        assert_not(obj.class.serialized_attributes.empty?,
          "#{obj.class} model doesn't have serialized attributes")
        # obj shouldn't have changed since it's just retrieved from the database
        assert_not(obj.changed?, "#{obj.class} model's attribute(s) appear as changed: '#{obj.changes.keys.join(',')}' with audit logs #{auditlogs_enabled ? '': 'not '}enabled.")
      end
    end
  end
end
