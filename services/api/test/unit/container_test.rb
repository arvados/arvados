# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'test_helper'
require 'helpers/container_test_helper'

class ContainerTest < ActiveSupport::TestCase
  include DbCurrentTime
  include ContainerTestHelper

  DEFAULT_ATTRS = {
    command: ['echo', 'foo'],
    container_image: 'fa3c1a9cb6783f85f2ecda037e07b8c3+167',
    output_path: '/tmp',
    priority: 1,
    runtime_constraints: {"vcpus" => 1, "ram" => 1},
  }

  REUSABLE_COMMON_ATTRS = {
    container_image: "9ae44d5792468c58bcf85ce7353c7027+124",
    cwd: "test",
    command: ["echo", "hello"],
    output_path: "test",
    runtime_constraints: {
      "ram" => 12000000000,
      "vcpus" => 4,
    },
    mounts: {
      "test" => {"kind" => "json"},
    },
    environment: {
      "var" => "val",
    },
    secret_mounts: {},
  }

  def minimal_new attrs={}
    cr = ContainerRequest.new DEFAULT_ATTRS.merge(attrs)
    cr.state = ContainerRequest::Committed
    act_as_user users(:active) do
      cr.save!
    end
    c = Container.find_by_uuid cr.container_uuid
    assert_not_nil c
    return c, cr
  end

  def check_illegal_updates c, bad_updates
    bad_updates.each do |u|
      refute c.update_attributes(u), u.inspect
      refute c.valid?, u.inspect
      c.reload
    end
  end

  def check_illegal_modify c
    check_illegal_updates c, [{command: ["echo", "bar"]},
                              {container_image: "arvados/apitestfixture:june10"},
                              {cwd: "/tmp2"},
                              {environment: {"FOO" => "BAR"}},
                              {mounts: {"FOO" => "BAR"}},
                              {output_path: "/tmp3"},
                              {locked_by_uuid: "zzzzz-gj3su-027z32aux8dg2s1"},
                              {auth_uuid: "zzzzz-gj3su-017z32aux8dg2s1"},
                              {runtime_constraints: {"FOO" => "BAR"}}]
  end

  def check_bogus_states c
    check_illegal_updates c, [{state: nil},
                              {state: "Flubber"}]
  end

  def check_no_change_from_cancelled c
    check_illegal_modify c
    check_bogus_states c
    check_illegal_updates c, [{ priority: 3 },
                              { state: Container::Queued },
                              { state: Container::Locked },
                              { state: Container::Running },
                              { state: Container::Complete }]
  end

  test "Container create" do
    act_as_system_user do
      c, _ = minimal_new(environment: {},
                      mounts: {"BAR" => "FOO"},
                      output_path: "/tmp",
                      priority: 1,
                      runtime_constraints: {"vcpus" => 1, "ram" => 1})

      check_illegal_modify c
      check_bogus_states c

      c.reload
      c.priority = 2
      c.save!
    end
  end

  test "Container valid priority" do
    act_as_system_user do
      c, _ = minimal_new(environment: {},
                      mounts: {"BAR" => "FOO"},
                      output_path: "/tmp",
                      priority: 1,
                      runtime_constraints: {"vcpus" => 1, "ram" => 1})

      assert_raises(ActiveRecord::RecordInvalid) do
        c.priority = -1
        c.save!
      end

      c.priority = 0
      c.save!

      c.priority = 1
      c.save!

      c.priority = 500
      c.save!

      c.priority = 999
      c.save!

      c.priority = 1000
      c.save!

      c.priority = 1000 << 50
      c.save!
    end
  end


  test "Container serialized hash attributes sorted before save" do
    env = {"C" => 3, "B" => 2, "A" => 1}
    m = {"F" => {"kind" => 3}, "E" => {"kind" => 2}, "D" => {"kind" => 1}}
    rc = {"vcpus" => 1, "ram" => 1, "keep_cache_ram" => 1}
    c, _ = minimal_new(environment: env, mounts: m, runtime_constraints: rc)
    assert_equal c.environment.to_json, Container.deep_sort_hash(env).to_json
    assert_equal c.mounts.to_json, Container.deep_sort_hash(m).to_json
    assert_equal c.runtime_constraints.to_json, Container.deep_sort_hash(rc).to_json
  end

  test 'deep_sort_hash on array of hashes' do
    a = {'z' => [[{'a' => 'a', 'b' => 'b'}]]}
    b = {'z' => [[{'b' => 'b', 'a' => 'a'}]]}
    assert_equal Container.deep_sort_hash(a).to_json, Container.deep_sort_hash(b).to_json
  end

  test "find_reusable method should select higher priority queued container" do
    set_user_from_auth :active
    common_attrs = REUSABLE_COMMON_ATTRS.merge({environment:{"var" => "queued"}})
    c_low_priority, _ = minimal_new(common_attrs.merge({use_existing:false, priority:1}))
    c_high_priority, _ = minimal_new(common_attrs.merge({use_existing:false, priority:2}))
    assert_not_equal c_low_priority.uuid, c_high_priority.uuid
    assert_equal Container::Queued, c_low_priority.state
    assert_equal Container::Queued, c_high_priority.state
    reused = Container.find_reusable(common_attrs)
    assert_not_nil reused
    assert_equal reused.uuid, c_high_priority.uuid
  end

  test "find_reusable method should select latest completed container" do
    set_user_from_auth :active
    common_attrs = REUSABLE_COMMON_ATTRS.merge({environment: {"var" => "complete"}})
    completed_attrs = {
      state: Container::Complete,
      exit_code: 0,
      log: 'ea10d51bcf88862dbcc36eb292017dfd+45',
      output: '1f4b0bc7583c2a7f9102c395f4ffc5e3+45'
    }

    c_older, _ = minimal_new(common_attrs.merge({use_existing: false}))
    c_recent, _ = minimal_new(common_attrs.merge({use_existing: false}))
    assert_not_equal c_older.uuid, c_recent.uuid

    set_user_from_auth :dispatch1
    c_older.update_attributes!({state: Container::Locked})
    c_older.update_attributes!({state: Container::Running})
    c_older.update_attributes!(completed_attrs)

    c_recent.update_attributes!({state: Container::Locked})
    c_recent.update_attributes!({state: Container::Running})
    c_recent.update_attributes!(completed_attrs)

    reused = Container.find_reusable(common_attrs)
    assert_not_nil reused
    assert_equal reused.uuid, c_older.uuid
  end

  test "find_reusable method should select oldest completed container when inconsistent outputs exist" do
    set_user_from_auth :active
    common_attrs = REUSABLE_COMMON_ATTRS.merge({environment: {"var" => "complete"}, priority: 1})
    completed_attrs = {
      state: Container::Complete,
      exit_code: 0,
      log: 'ea10d51bcf88862dbcc36eb292017dfd+45',
    }

    cr = ContainerRequest.new common_attrs
    cr.use_existing = false
    cr.state = ContainerRequest::Committed
    cr.save!
    c_output1 = Container.where(uuid: cr.container_uuid).first

    cr = ContainerRequest.new common_attrs
    cr.use_existing = false
    cr.state = ContainerRequest::Committed
    cr.save!
    c_output2 = Container.where(uuid: cr.container_uuid).first

    assert_not_equal c_output1.uuid, c_output2.uuid

    set_user_from_auth :dispatch1

    out1 = '1f4b0bc7583c2a7f9102c395f4ffc5e3+45'
    log1 = collections(:real_log_collection).portable_data_hash
    c_output1.update_attributes!({state: Container::Locked})
    c_output1.update_attributes!({state: Container::Running})
    c_output1.update_attributes!(completed_attrs.merge({log: log1, output: out1}))

    out2 = 'fa7aeb5140e2848d39b416daeef4ffc5+45'
    c_output2.update_attributes!({state: Container::Locked})
    c_output2.update_attributes!({state: Container::Running})
    c_output2.update_attributes!(completed_attrs.merge({log: log1, output: out2}))

    reused = Container.resolve(ContainerRequest.new(common_attrs))
    assert_equal c_output1.uuid, reused.uuid
  end

  test "find_reusable method should select running container by start date" do
    set_user_from_auth :active
    common_attrs = REUSABLE_COMMON_ATTRS.merge({environment: {"var" => "running"}})
    c_slower, _ = minimal_new(common_attrs.merge({use_existing: false}))
    c_faster_started_first, _ = minimal_new(common_attrs.merge({use_existing: false}))
    c_faster_started_second, _ = minimal_new(common_attrs.merge({use_existing: false}))
    # Confirm the 3 container UUIDs are different.
    assert_equal 3, [c_slower.uuid, c_faster_started_first.uuid, c_faster_started_second.uuid].uniq.length
    set_user_from_auth :dispatch1
    c_slower.update_attributes!({state: Container::Locked})
    c_slower.update_attributes!({state: Container::Running,
                                 progress: 0.1})
    c_faster_started_first.update_attributes!({state: Container::Locked})
    c_faster_started_first.update_attributes!({state: Container::Running,
                                               progress: 0.15})
    c_faster_started_second.update_attributes!({state: Container::Locked})
    c_faster_started_second.update_attributes!({state: Container::Running,
                                                progress: 0.15})
    reused = Container.find_reusable(common_attrs)
    assert_not_nil reused
    # Selected container is the one that started first
    assert_equal reused.uuid, c_faster_started_first.uuid
  end

  test "find_reusable method should select running container by progress" do
    set_user_from_auth :active
    common_attrs = REUSABLE_COMMON_ATTRS.merge({environment: {"var" => "running2"}})
    c_slower, _ = minimal_new(common_attrs.merge({use_existing: false}))
    c_faster_started_first, _ = minimal_new(common_attrs.merge({use_existing: false}))
    c_faster_started_second, _ = minimal_new(common_attrs.merge({use_existing: false}))
    # Confirm the 3 container UUIDs are different.
    assert_equal 3, [c_slower.uuid, c_faster_started_first.uuid, c_faster_started_second.uuid].uniq.length
    set_user_from_auth :dispatch1
    c_slower.update_attributes!({state: Container::Locked})
    c_slower.update_attributes!({state: Container::Running,
                                 progress: 0.1})
    c_faster_started_first.update_attributes!({state: Container::Locked})
    c_faster_started_first.update_attributes!({state: Container::Running,
                                               progress: 0.15})
    c_faster_started_second.update_attributes!({state: Container::Locked})
    c_faster_started_second.update_attributes!({state: Container::Running,
                                                progress: 0.2})
    reused = Container.find_reusable(common_attrs)
    assert_not_nil reused
    # Selected container is the one with most progress done
    assert_equal reused.uuid, c_faster_started_second.uuid
  end

  test "find_reusable method should select locked container most likely to start sooner" do
    set_user_from_auth :active
    common_attrs = REUSABLE_COMMON_ATTRS.merge({environment: {"var" => "locked"}})
    c_low_priority, _ = minimal_new(common_attrs.merge({use_existing: false}))
    c_high_priority_older, _ = minimal_new(common_attrs.merge({use_existing: false}))
    c_high_priority_newer, _ = minimal_new(common_attrs.merge({use_existing: false}))
    # Confirm the 3 container UUIDs are different.
    assert_equal 3, [c_low_priority.uuid, c_high_priority_older.uuid, c_high_priority_newer.uuid].uniq.length
    set_user_from_auth :dispatch1
    c_low_priority.update_attributes!({state: Container::Locked,
                                       priority: 1})
    c_high_priority_older.update_attributes!({state: Container::Locked,
                                              priority: 2})
    c_high_priority_newer.update_attributes!({state: Container::Locked,
                                              priority: 2})
    reused = Container.find_reusable(common_attrs)
    assert_not_nil reused
    assert_equal reused.uuid, c_high_priority_older.uuid
  end

  test "find_reusable method should select running over failed container" do
    set_user_from_auth :active
    common_attrs = REUSABLE_COMMON_ATTRS.merge({environment: {"var" => "failed_vs_running"}})
    c_failed, _ = minimal_new(common_attrs.merge({use_existing: false}))
    c_running, _ = minimal_new(common_attrs.merge({use_existing: false}))
    assert_not_equal c_failed.uuid, c_running.uuid
    set_user_from_auth :dispatch1
    c_failed.update_attributes!({state: Container::Locked})
    c_failed.update_attributes!({state: Container::Running})
    c_failed.update_attributes!({state: Container::Complete,
                                 exit_code: 42,
                                 log: 'ea10d51bcf88862dbcc36eb292017dfd+45',
                                 output: 'ea10d51bcf88862dbcc36eb292017dfd+45'})
    c_running.update_attributes!({state: Container::Locked})
    c_running.update_attributes!({state: Container::Running,
                                  progress: 0.15})
    reused = Container.find_reusable(common_attrs)
    assert_not_nil reused
    assert_equal reused.uuid, c_running.uuid
  end

  test "find_reusable method should select complete over running container" do
    set_user_from_auth :active
    common_attrs = REUSABLE_COMMON_ATTRS.merge({environment: {"var" => "completed_vs_running"}})
    c_completed, _ = minimal_new(common_attrs.merge({use_existing: false}))
    c_running, _ = minimal_new(common_attrs.merge({use_existing: false}))
    assert_not_equal c_completed.uuid, c_running.uuid
    set_user_from_auth :dispatch1
    c_completed.update_attributes!({state: Container::Locked})
    c_completed.update_attributes!({state: Container::Running})
    c_completed.update_attributes!({state: Container::Complete,
                                    exit_code: 0,
                                    log: 'ea10d51bcf88862dbcc36eb292017dfd+45',
                                    output: '1f4b0bc7583c2a7f9102c395f4ffc5e3+45'})
    c_running.update_attributes!({state: Container::Locked})
    c_running.update_attributes!({state: Container::Running,
                                  progress: 0.15})
    reused = Container.find_reusable(common_attrs)
    assert_not_nil reused
    assert_equal c_completed.uuid, reused.uuid
  end

  test "find_reusable method should select running over locked container" do
    set_user_from_auth :active
    common_attrs = REUSABLE_COMMON_ATTRS.merge({environment: {"var" => "running_vs_locked"}})
    c_locked, _ = minimal_new(common_attrs.merge({use_existing: false}))
    c_running, _ = minimal_new(common_attrs.merge({use_existing: false}))
    assert_not_equal c_running.uuid, c_locked.uuid
    set_user_from_auth :dispatch1
    c_locked.update_attributes!({state: Container::Locked})
    c_running.update_attributes!({state: Container::Locked})
    c_running.update_attributes!({state: Container::Running,
                                  progress: 0.15})
    reused = Container.find_reusable(common_attrs)
    assert_not_nil reused
    assert_equal reused.uuid, c_running.uuid
  end

  test "find_reusable method should select locked over queued container" do
    set_user_from_auth :active
    common_attrs = REUSABLE_COMMON_ATTRS.merge({environment: {"var" => "running_vs_locked"}})
    c_locked, _ = minimal_new(common_attrs.merge({use_existing: false}))
    c_queued, _ = minimal_new(common_attrs.merge({use_existing: false}))
    assert_not_equal c_queued.uuid, c_locked.uuid
    set_user_from_auth :dispatch1
    c_locked.update_attributes!({state: Container::Locked})
    reused = Container.find_reusable(common_attrs)
    assert_not_nil reused
    assert_equal reused.uuid, c_locked.uuid
  end

  test "find_reusable method should not select failed container" do
    set_user_from_auth :active
    attrs = REUSABLE_COMMON_ATTRS.merge({environment: {"var" => "failed"}})
    c, _ = minimal_new(attrs)
    set_user_from_auth :dispatch1
    c.update_attributes!({state: Container::Locked})
    c.update_attributes!({state: Container::Running})
    c.update_attributes!({state: Container::Complete,
                          exit_code: 33})
    reused = Container.find_reusable(attrs)
    assert_nil reused
  end

  test "find_reusable with logging disabled" do
    set_user_from_auth :active
    Rails.logger.expects(:info).never
    Container.find_reusable(REUSABLE_COMMON_ATTRS)
  end

  test "find_reusable with logging enabled" do
    set_user_from_auth :active
    Rails.configuration.log_reuse_decisions = true
    Rails.logger.expects(:info).at_least(3)
    Container.find_reusable(REUSABLE_COMMON_ATTRS)
  end

  test "Container running" do
    c, _ = minimal_new priority: 1

    set_user_from_auth :dispatch1
    check_illegal_updates c, [{state: Container::Running},
                              {state: Container::Complete}]

    c.lock
    c.update_attributes! state: Container::Running

    check_illegal_modify c
    check_bogus_states c

    check_illegal_updates c, [{state: Container::Queued}]
    c.reload

    c.update_attributes! priority: 3
  end

  test "Lock and unlock" do
    c, cr = minimal_new priority: 0

    set_user_from_auth :dispatch1
    assert_equal Container::Queued, c.state

    assert_raise(ArvadosModel::LockFailedError) do
      # "no priority"
      c.lock
    end
    c.reload
    assert cr.update_attributes priority: 1

    refute c.update_attributes(state: Container::Running), "not locked"
    c.reload
    refute c.update_attributes(state: Container::Complete), "not locked"
    c.reload

    assert c.lock, show_errors(c)
    assert c.locked_by_uuid
    assert c.auth_uuid

    assert_raise(ArvadosModel::LockFailedError) {c.lock}
    c.reload

    assert c.unlock, show_errors(c)
    refute c.locked_by_uuid
    refute c.auth_uuid

    refute c.update_attributes(state: Container::Running), "not locked"
    c.reload
    refute c.locked_by_uuid
    refute c.auth_uuid

    assert c.lock, show_errors(c)
    assert c.update_attributes(state: Container::Running), show_errors(c)
    assert c.locked_by_uuid
    assert c.auth_uuid

    auth_uuid_was = c.auth_uuid

    assert_raise(ArvadosModel::LockFailedError) do
      # Running to Locked is not allowed
      c.lock
    end
    c.reload
    assert_raise(ArvadosModel::InvalidStateTransitionError) do
      # Running to Queued is not allowed
      c.unlock
    end
    c.reload

    assert c.update_attributes(state: Container::Complete), show_errors(c)
    refute c.locked_by_uuid
    refute c.auth_uuid

    auth_exp = ApiClientAuthorization.find_by_uuid(auth_uuid_was).expires_at
    assert_operator auth_exp, :<, db_current_time
  end

  test "Container queued cancel" do
    c, cr = minimal_new({container_count_max: 1})
    set_user_from_auth :dispatch1
    assert c.update_attributes(state: Container::Cancelled), show_errors(c)
    check_no_change_from_cancelled c
    cr.reload
    assert_equal ContainerRequest::Final, cr.state
  end

  test "Container queued count" do
    assert_equal 1, Container.readable_by(users(:active)).where(state: "Queued").count
  end

  test "Container locked cancel" do
    c, _ = minimal_new
    set_user_from_auth :dispatch1
    assert c.lock, show_errors(c)
    assert c.update_attributes(state: Container::Cancelled), show_errors(c)
    check_no_change_from_cancelled c
  end

  test "Container locked cancel with log" do
    c, _ = minimal_new
    set_user_from_auth :dispatch1
    assert c.lock, show_errors(c)
    assert c.update_attributes(
             state: Container::Cancelled,
             log: collections(:real_log_collection).portable_data_hash,
           ), show_errors(c)
    check_no_change_from_cancelled c
  end

  test "Container running cancel" do
    c, _ = minimal_new
    set_user_from_auth :dispatch1
    c.lock
    c.update_attributes! state: Container::Running
    c.update_attributes! state: Container::Cancelled
    check_no_change_from_cancelled c
  end

  test "Container create forbidden for non-admin" do
    set_user_from_auth :active_trustedclient
    c = Container.new DEFAULT_ATTRS
    c.environment = {}
    c.mounts = {"BAR" => "FOO"}
    c.output_path = "/tmp"
    c.priority = 1
    c.runtime_constraints = {}
    assert_raises(ArvadosModel::PermissionDeniedError) do
      c.save!
    end
  end

  test "Container only set exit code on complete" do
    c, _ = minimal_new
    set_user_from_auth :dispatch1
    c.lock
    c.update_attributes! state: Container::Running

    check_illegal_updates c, [{exit_code: 1},
                              {exit_code: 1, state: Container::Cancelled}]

    assert c.update_attributes(exit_code: 1, state: Container::Complete)
  end

  test "locked_by_uuid can set output on running container" do
    c, _ = minimal_new
    set_user_from_auth :dispatch1
    c.lock
    c.update_attributes! state: Container::Running

    assert_equal c.locked_by_uuid, Thread.current[:api_client_authorization].uuid

    assert c.update_attributes output: collections(:collection_owned_by_active).portable_data_hash
    assert c.update_attributes! state: Container::Complete
  end

  test "auth_uuid can set output on running container, but not change container state" do
    c, _ = minimal_new
    set_user_from_auth :dispatch1
    c.lock
    c.update_attributes! state: Container::Running

    Thread.current[:api_client_authorization] = ApiClientAuthorization.find_by_uuid(c.auth_uuid)
    Thread.current[:user] = User.find_by_id(Thread.current[:api_client_authorization].user_id)
    assert c.update_attributes output: collections(:collection_owned_by_active).portable_data_hash

    assert_raises ArvadosModel::PermissionDeniedError do
      # auth_uuid cannot set container state
      c.update_attributes state: Container::Complete
    end
  end

  test "not allowed to set output that is not readable by current user" do
    c, _ = minimal_new
    set_user_from_auth :dispatch1
    c.lock
    c.update_attributes! state: Container::Running

    Thread.current[:api_client_authorization] = ApiClientAuthorization.find_by_uuid(c.auth_uuid)
    Thread.current[:user] = User.find_by_id(Thread.current[:api_client_authorization].user_id)

    assert_raises ActiveRecord::RecordInvalid do
      c.update_attributes! output: collections(:collection_not_readable_by_active).portable_data_hash
    end
  end

  test "other token cannot set output on running container" do
    c, _ = minimal_new
    set_user_from_auth :dispatch1
    c.lock
    c.update_attributes! state: Container::Running

    set_user_from_auth :running_to_be_deleted_container_auth
    assert_raises ArvadosModel::PermissionDeniedError do
      c.update_attributes! output: collections(:foo_file).portable_data_hash
    end
  end

  test "can set trashed output on running container" do
    c, _ = minimal_new
    set_user_from_auth :dispatch1
    c.lock
    c.update_attributes! state: Container::Running

    output = Collection.find_by_uuid('zzzzz-4zz18-mto52zx1s7sn3jk')

    assert output.is_trashed
    assert c.update_attributes output: output.portable_data_hash
    assert c.update_attributes! state: Container::Complete
  end

  test "not allowed to set trashed output that is not readable by current user" do
    c, _ = minimal_new
    set_user_from_auth :dispatch1
    c.lock
    c.update_attributes! state: Container::Running

    output = Collection.find_by_uuid('zzzzz-4zz18-mto52zx1s7sn3jr')

    Thread.current[:api_client_authorization] = ApiClientAuthorization.find_by_uuid(c.auth_uuid)
    Thread.current[:user] = User.find_by_id(Thread.current[:api_client_authorization].user_id)

    assert_raises ActiveRecord::RecordInvalid do
      c.update_attributes! output: output.portable_data_hash
    end
  end

  [
    {state: Container::Complete, exit_code: 0, output: '1f4b0bc7583c2a7f9102c395f4ffc5e3+45'},
    {state: Container::Cancelled},
  ].each do |final_attrs|
    test "secret_mounts is null after container is #{final_attrs[:state]}" do
      c, cr = minimal_new(secret_mounts: {'/secret' => {'kind' => 'text', 'content' => 'foo'}},
                          container_count_max: 1)
      set_user_from_auth :dispatch1
      c.lock
      c.update_attributes!(state: Container::Running)
      c.reload
      assert c.secret_mounts.has_key?('/secret')

      c.update_attributes!(final_attrs)
      c.reload
      assert_equal({}, c.secret_mounts)
      cr.reload
      assert_equal({}, cr.secret_mounts)
      assert_no_secrets_logged
    end
  end
end
