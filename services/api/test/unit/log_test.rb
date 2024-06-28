# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'test_helper'
require 'audit_logs'

class LogTest < ActiveSupport::TestCase
  include CurrentApiClient

  EVENT_TEST_METHODS = {
    :create => [:created_at, :assert_nil, :assert_not_nil],
    :update => [:modified_at, :assert_not_nil, :assert_not_nil],
    :delete => [nil, :assert_not_nil, :assert_nil],
  }

  setup do
    @start_time = Time.now
    @log_count = 1
  end

  def assert_properties(test_method, event, props, *keys)
    verb = (test_method == :assert_nil) ? 'have nil' : 'define'
    keys.each do |prop_name|
      assert_includes(props, prop_name, "log properties missing #{prop_name}")
      self.send(test_method, props[prop_name],
                "#{event.to_s} log should #{verb} #{prop_name}")
    end
  end

  def get_logs_about(thing)
    Log.where(object_uuid: thing.uuid).order("created_at ASC").all
  end

  def clear_logs_about(thing)
    Log.where(object_uuid: thing.uuid).delete_all
  end

  def assert_logged(thing, event_type)
    logs = get_logs_about(thing)
    assert_equal(@log_count, logs.size, "log count mismatch")
    @log_count += 1
    log = logs.last
    props = log.properties
    assert_equal(current_user.andand.uuid, log.owner_uuid,
                 "log is not owned by current user")
    assert_equal(current_user.andand.uuid, log.modified_by_user_uuid,
                 "log is not 'modified by' current user")
    assert_equal(thing.uuid, log.object_uuid, "log UUID mismatch")
    assert_equal(event_type.to_s, log.event_type, "log event type mismatch")
    time_method, old_props_test, new_props_test = EVENT_TEST_METHODS[event_type]
    if time_method.nil? or (timestamp = thing.send(time_method)).nil?
      assert(log.event_at >= @start_time, "log timestamp too old")
    else
      assert_in_delta(timestamp, log.event_at, 1, "log timestamp mismatch")
    end
    assert_properties(old_props_test, event_type, props,
                      'old_etag', 'old_attributes')
    assert_properties(new_props_test, event_type, props,
                      'new_etag', 'new_attributes')
    ['old_attributes', 'new_attributes'].each do |logattr|
      next if !props[logattr]
      assert_match /"created_at":"\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{9}Z"/, Oj.dump(props, mode: :compat)
    end
    yield props if block_given?
  end

  def assert_logged_with_clean_properties(obj, event_type, excluded_attr)
    assert_logged(obj, event_type) do |props|
      ['old_attributes', 'new_attributes'].map do |logattr|
        attributes = props[logattr]
        next if attributes.nil?
        refute_includes(attributes, excluded_attr,
                        "log #{logattr} includes #{excluded_attr}")
      end
      yield props if block_given?
    end
  end

  test "creating a user makes a log" do
    set_user_from_auth :admin_trustedclient
    u = User.new(first_name: "Log", last_name: "Test")
    u.save!
    assert_logged(u, :create) do |props|
      assert_equal(u.etag, props['new_etag'], "new user etag mismatch")
      assert_equal(u.first_name, props['new_attributes']['first_name'],
                   "new user first name mismatch")
      assert_equal(u.last_name, props['new_attributes']['last_name'],
                   "new user first name mismatch")
    end
  end

  test "updating a virtual machine makes a log" do
    set_user_from_auth :admin_trustedclient
    vm = virtual_machines(:testvm)
    orig_etag = vm.etag
    vm.hostname = 'testvm.testshell'
    vm.save!
    assert_logged(vm, :update) do |props|
      assert_equal(orig_etag, props['old_etag'], "updated VM old etag mismatch")
      assert_equal(vm.etag, props['new_etag'], "updated VM new etag mismatch")
      assert_equal('testvm.shell', props['old_attributes']['hostname'],
                   "updated VM old name mismatch")
      assert_equal('testvm.testshell', props['new_attributes']['hostname'],
                   "updated VM new name mismatch")
    end
  end

  test "old_attributes preserves values deep inside a hash" do
    set_user_from_auth :active
    it = collections(:collection_owned_by_active)
    clear_logs_about it
    it.properties = {'foo' => {'bar' => ['baz', 'qux', {'quux' => 'bleat'}]}}
    it.save!
    assert_logged it, :update
    it.properties['foo']['bar'][2]['quux'] = 'blert'
    it.save!
    assert_logged it, :update do |props|
      assert_equal 'bleat', props['old_attributes']['properties']['foo']['bar'][2]['quux']
      assert_equal 'blert', props['new_attributes']['properties']['foo']['bar'][2]['quux']
    end
  end

  test "destroying an authorization makes a log" do
    set_user_from_auth :admin_trustedclient
    auth = api_client_authorizations(:spectator)
    orig_etag = auth.etag
    orig_attrs = auth.attributes
    orig_attrs.delete 'api_token'
    auth.destroy
    assert_logged(auth, :delete) do |props|
      assert_equal(orig_etag, props['old_etag'], "destroyed auth etag mismatch")
      assert_equal(orig_attrs, props['old_attributes'],
                   "destroyed auth attributes mismatch")
    end
  end

  test "updating a group twice makes two logs" do
    set_user_from_auth :admin_trustedclient
    group = groups(:empty_lonely_group)
    name1 = group.name
    name2 = "#{name1} under test"
    group.name = name2
    group.save!
    assert_logged(group, :update) do |props|
      assert_equal(name1, props['old_attributes']['name'],
                   "group start name mismatch")
      assert_equal(name2, props['new_attributes']['name'],
                   "group updated name mismatch")
    end
    group.name = name1
    group.save!
    assert_logged(group, :update) do |props|
      assert_equal(name2, props['old_attributes']['name'],
                   "group pre-revert name mismatch")
      assert_equal(name1, props['new_attributes']['name'],
                   "group final name mismatch")
    end
  end

  test "making a log doesn't get logged" do
    set_user_from_auth :active_trustedclient
    log = Log.new
    log.save!
    assert_equal(0, get_logs_about(log).size, "made a Log about a Log")
  end

  test "non-admins can't modify or delete logs" do
    set_user_from_auth :active_trustedclient
    log = Log.new(summary: "immutable log test")
    assert_nothing_raised { log.save! }
    log.summary = "log mutation test should fail"
    assert_raise(ArvadosModel::PermissionDeniedError) { log.save! }
    assert_raise(ArvadosModel::PermissionDeniedError) { log.destroy }
  end

  test "admins can modify and delete logs" do
    set_user_from_auth :admin_trustedclient
    log = Log.new(summary: "admin log mutation test")
    assert_nothing_raised { log.save! }
    log.summary = "admin mutated log test"
    assert_nothing_raised { log.save! }
    assert_nothing_raised { log.destroy }
  end

  test "failure saving log causes failure saving object" do
    Log.class_eval do
      alias_method :_orig_validations, :perform_validations
      def perform_validations(options)
        false
      end
    end
    begin
      set_user_from_auth :active_trustedclient
      user = users(:active)
      user.first_name = 'Test'
      assert_raise(ActiveRecord::RecordInvalid) { user.save! }
    ensure
      Log.class_eval do
        alias_method :perform_validations, :_orig_validations
      end
    end
  end

  test "don't log changes only to ApiClientAuthorization.last_used_*" do
    set_user_from_auth :admin_trustedclient
    auth = api_client_authorizations(:spectator)
    start_log_count = get_logs_about(auth).size
    auth.last_used_at = Time.now
    auth.last_used_by_ip_address = '::1'
    auth.save!
    assert_equal(start_log_count, get_logs_about(auth).size,
                 "log count changed after 'using' ApiClientAuthorization")
    auth.created_by_ip_address = '::1'
    auth.save!
    assert_logged(auth, :update)
  end

  test "don't log changes only to Collection.preserve_version" do
    set_user_from_auth :admin_trustedclient
    col = collections(:collection_owned_by_active)
    clear_logs_about col
    start_log_count = get_logs_about(col).size
    assert_equal false, col.preserve_version
    col.preserve_version = true
    col.save!
    assert_equal(start_log_count, get_logs_about(col).size,
                 "log count changed after updating Collection.preserve_version")
    col.name = 'updated by admin'
    col.save!
    assert_logged(col, :update)
  end

  test "token isn't included in ApiClientAuthorization logs" do
    set_user_from_auth :admin_trustedclient
    auth = ApiClientAuthorization.new
    auth.user = users(:spectator)
    auth.save!
    assert_logged_with_clean_properties(auth, :create, 'api_token')
    auth.expires_at = Time.now
    auth.save!
    assert_logged_with_clean_properties(auth, :update, 'api_token')
    auth.destroy
    assert_logged_with_clean_properties(auth, :delete, 'api_token')
  end

  test "use ownership and permission links to determine which logs a user can see" do
    known_logs = [:noop,
                  :admin_changes_collection_owned_by_active,
                  :admin_changes_collection_owned_by_foo,
                  :system_adds_foo_file,
                  :system_adds_baz,
                  :log_owned_by_active,
                  :crunchstat_for_running_container]

    c = Log.readable_by(users(:admin)).order("id asc").each.to_a
    assert_log_result c, known_logs, known_logs

    c = Log.readable_by(users(:active)).order("id asc").each.to_a
    assert_log_result c, known_logs, [:admin_changes_collection_owned_by_active,
                                      :system_adds_foo_file,             # readable via link
                                      :system_adds_baz,                  # readable via 'all users' group
                                      :log_owned_by_active,              # log owned by active
                                      :crunchstat_for_running_container] # log & job owned by active

    c = Log.readable_by(users(:spectator)).order("id asc").each.to_a
    assert_log_result c, known_logs, [:noop,                             # object_uuid is spectator
                                      :system_adds_baz]                  # readable via 'all users' group

    c = Log.readable_by(users(:user_foo_in_sharing_group)).order("id asc").each.to_a
    assert_log_result c, known_logs, [:admin_changes_collection_owned_by_foo] # collection's parent is readable via role group
  end

  def assert_log_result result, known_logs, expected_logs
    # All of expected_logs must appear in result. Additional logs can
    # appear too, but only if they are _not_ listed in known_logs
    # (i.e., we do not make any assertions about logs not mentioned in
    # either "known" or "expected".)
    result_ids = result.collect(&:id)
    expected_logs.each do |want|
      assert_includes result_ids, logs(want).id
    end
    (known_logs - expected_logs).each do |notwant|
      refute_includes result_ids, logs(notwant).id
    end
  end

  test "non-empty configuration.unlogged_attributes" do
    Rails.configuration.AuditLogs.UnloggedAttributes = ConfigLoader.to_OrderedOptions({"manifest_text"=>{}})
    txt = ". acbd18db4cc2f85cedef654fccc4a4d8+3 0:3:foo\n"

    act_as_system_user do
      coll = Collection.create(manifest_text: txt)
      assert_logged_with_clean_properties(coll, :create, 'manifest_text')
      coll.name = "testing"
      coll.save!
      assert_logged_with_clean_properties(coll, :update, 'manifest_text')
      coll.destroy
      assert_logged_with_clean_properties(coll, :delete, 'manifest_text')
    end
  end

  test "empty configuration.unlogged_attributes" do
    Rails.configuration.AuditLogs.UnloggedAttributes = ConfigLoader.to_OrderedOptions({})
    txt = ". acbd18db4cc2f85cedef654fccc4a4d8+3 0:3:foo\n"

    act_as_system_user do
      coll = Collection.create(manifest_text: txt)
      assert_logged(coll, :create) do |props|
        assert_equal(txt, props['new_attributes']['manifest_text'])
      end
      coll.update!(name: "testing")
      assert_logged(coll, :update) do |props|
        assert_equal(txt, props['old_attributes']['manifest_text'])
        assert_equal(txt, props['new_attributes']['manifest_text'])
      end
      coll.destroy
      assert_logged(coll, :delete) do |props|
        assert_equal(txt, props['old_attributes']['manifest_text'])
      end
    end
  end

  def assert_no_logs_deleted
    logs_before = Log.unscoped.all.count
    assert logs_before > 0
    yield
    assert_equal logs_before, Log.unscoped.all.count
  end

  def remaining_audit_logs
    Log.unscoped.where('event_type in (?)', %w(create update destroy delete))
  end

  # Default settings should not delete anything -- some sites rely on
  # the original "keep everything forever" behavior.
  test 'retain old audit logs with default settings' do
    assert_no_logs_deleted do
      AuditLogs.delete_old(
        max_age: Rails.configuration.AuditLogs.MaxAge,
        max_batch: Rails.configuration.AuditLogs.MaxDeleteBatch)
    end
  end

  # Batch size 0 should retain all logs -- even if max_age is very
  # short, and even if the default settings (and associated test) have
  # changed.
  test 'retain old audit logs with max_audit_log_delete_batch=0' do
    assert_no_logs_deleted do
      AuditLogs.delete_old(max_age: 1, max_batch: 0)
    end
  end

  # We recommend a more conservative age of 5 minutes for production,
  # but 3 minutes suits our test data better (and is test-worthy in
  # that it's expected to work correctly in production).
  test 'delete old audit logs with production settings' do
    initial_log_count = remaining_audit_logs.count
    assert initial_log_count > 0
    AuditLogs.delete_old(max_age: 180, max_batch: 100000)
    assert_operator remaining_audit_logs.count, :<, initial_log_count
  end

  test 'delete all audit logs in multiple batches' do
    assert remaining_audit_logs.count > 2
    AuditLogs.delete_old(max_age: 0.00001, max_batch: 2)
    assert_equal [], remaining_audit_logs.collect(&:uuid)
  end

  test 'delete old audit logs in thread' do
    Rails.configuration.AuditLogs.MaxAge = 20
    Rails.configuration.AuditLogs.MaxDeleteBatch = 100000
    Rails.cache.delete 'AuditLogs'
    initial_audit_log_count = remaining_audit_logs.count
    assert initial_audit_log_count > 0
    act_as_system_user do
      Log.create!()
    end
    deadline = Time.now + 10
    while remaining_audit_logs.count == initial_audit_log_count
      if Time.now > deadline
        raise "timed out"
      end
      sleep 0.1
    end
    assert_operator remaining_audit_logs.count, :<, initial_audit_log_count
  end
end
