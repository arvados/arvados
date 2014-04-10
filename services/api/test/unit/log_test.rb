require 'test_helper'

class LogTest < ActiveSupport::TestCase
  include CurrentApiClient

  EVENT_TEST_METHODS = {
    :create => [:created_at, :assert_nil, :assert_not_nil],
    :update => [:modified_at, :assert_not_nil, :assert_not_nil],
    :destroy => [nil, :assert_not_nil, :assert_nil],
  }

  def setup
    @start_time = Time.now
    @log_count = 1
  end

  def assert_properties(test_method, event, props, *keys)
    verb = (test_method == :assert_nil) ? 'not include' : 'include'
    keys.each do |prop_name|
      self.send(test_method, props[prop_name],
                "#{event.to_s} log should #{verb} #{prop_name}")
    end
  end

  def get_logs_about(thing)
    Log.where(object_uuid: thing.uuid).order("created_at ASC").all
  end

  def assert_logged(thing, event_type)
    logs = get_logs_about(thing)
    assert_equal(@log_count, logs.size, "log count mismatch")
    @log_count += 1
    log = logs.last
    props = log.properties
    assert_equal(system_user_uuid, log.owner_uuid,
                 "log is not owned by system user")
    assert_equal(current_user.andand.uuid, log.modified_by_user_uuid,
                 "log is not 'modified by' current user")
    assert_equal(current_api_client.andand.uuid, log.modified_by_client_uuid,
                 "log is not 'modified by' current client")
    assert_equal(thing.kind, log.object_kind, "log kind mismatch")
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
    yield props if block_given?
  end

  def set_user_from_auth(auth_name)
    client_auth = api_client_authorizations(auth_name)
    Thread.current[:api_client_authorization] = client_auth
    Thread.current[:api_client] = client_auth.api_client
    Thread.current[:user] = client_auth.user
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

  test "destroying an authorization makes a log" do
    set_user_from_auth :admin_trustedclient
    auth = api_client_authorizations(:spectator)
    orig_etag = auth.etag
    orig_attrs = auth.attributes
    auth.destroy
    assert_logged(auth, :destroy) do |props|
      assert_equal(orig_etag, props['old_etag'], "destroyed auth etag mismatch")
      assert_equal(orig_attrs, props['old_attributes'],
                   "destroyed auth attributes mismatch")
    end
  end

  test "saving an unchanged client still makes a log" do
    set_user_from_auth :admin_trustedclient
    client = api_clients(:untrusted)
    client.is_trusted = client.is_trusted
    client.save!
    assert_logged(client, :update) do |props|
      ['old', 'new'].each do |age|
        assert_equal(client.etag, props["#{age}_etag"],
                     "unchanged client #{age} etag mismatch")
        assert_equal(client.attributes, props["#{age}_attributes"],
                     "unchanged client #{age} attributes mismatch")
      end
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
end
