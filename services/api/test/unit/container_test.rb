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
    runtime_constraints: {"vcpus" => 1, "ram" => 1, "cuda" => {"device_count":0, "driver_version": "", "hardware_capability": ""}},
  }

  REUSABLE_COMMON_ATTRS = {
    container_image: "9ae44d5792468c58bcf85ce7353c7027+124",
    cwd: "test",
    command: ["echo", "hello"],
    output_path: "test",
    runtime_constraints: {
      "API" => false,
      "keep_cache_disk" => 0,
      "keep_cache_ram" => 0,
      "ram" => 12000000000,
      "vcpus" => 4
    },
    mounts: {
      "test" => {"kind" => "json"},
    },
    environment: {
      "var" => "val",
    },
    secret_mounts: {},
    runtime_user_uuid: "zzzzz-tpzed-xurymjxw79nv3jz",
    runtime_auth_scopes: ["all"],
    scheduling_parameters: {},
  }

  REUSABLE_ATTRS_SLIM = {
    command: ["echo", "slim"],
    container_image: "9ae44d5792468c58bcf85ce7353c7027+124",
    cwd: "test",
    environment: {},
    mounts: {},
    output_path: "test",
    runtime_auth_scopes: ["all"],
    runtime_constraints: {
      "API" => false,
      "keep_cache_disk" => 0,
      "keep_cache_ram" => 0,
      "ram" => 8 << 30,
      "vcpus" => 4
    },
    runtime_user_uuid: "zzzzz-tpzed-xurymjxw79nv3jz",
    secret_mounts: {},
    scheduling_parameters: {},
  }

  def request_only attrs
    attrs.reject {|k| [:runtime_user_uuid, :runtime_auth_scopes].include? k}
  end

  def minimal_new attrs={}
    cr = ContainerRequest.new request_only(DEFAULT_ATTRS.merge(attrs))
    cr.state = ContainerRequest::Committed
    cr.save!
    c = Container.find_by_uuid cr.container_uuid
    assert_not_nil c
    return c, cr
  end

  def check_illegal_updates c, bad_updates
    bad_updates.each do |u|
      refute c.update(u), u.inspect
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
                      mounts: {"BAR" => {"kind" => "FOO"}},
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
                      mounts: {"BAR" => {"kind" => "FOO"}},
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

  test "Container runtime_status data types" do
    set_user_from_auth :active
    attrs = {
      environment: {},
      mounts: {"BAR" => {"kind" => "FOO"}},
      output_path: "/tmp",
      priority: 1,
      runtime_constraints: {"vcpus" => 1, "ram" => 1}
    }
    c, _ = minimal_new(attrs)
    assert_equal c.runtime_status, {}
    assert_equal Container::Queued, c.state

    set_user_from_auth :dispatch1
    c.update! state: Container::Locked
    c.update! state: Container::Running

    [
      'error', 'errorDetail', 'warning', 'warningDetail', 'activity'
    ].each do |k|
      # String type is allowed
      string_val = 'A string is accepted'
      c.update! runtime_status: {k => string_val}
      assert_equal string_val, c.runtime_status[k]

      # Other types aren't allowed
      [
        42, false, [], {}, nil
      ].each do |unallowed_val|
        assert_raises ActiveRecord::RecordInvalid do
          c.update! runtime_status: {k => unallowed_val}
        end
      end
    end
  end

  test "Container runtime_status updates" do
    set_user_from_auth :active
    attrs = {
      environment: {},
      mounts: {"BAR" => {"kind" => "FOO"}},
      output_path: "/tmp",
      priority: 1,
      runtime_constraints: {"vcpus" => 1, "ram" => 1}
    }
    c1, _ = minimal_new(attrs)
    assert_equal c1.runtime_status, {}

    assert_equal Container::Queued, c1.state
    assert_raises ArvadosModel::PermissionDeniedError do
      c1.update! runtime_status: {'error' => 'Oops!'}
    end

    set_user_from_auth :dispatch1

    # Allow updates when state = Locked
    c1.update! state: Container::Locked
    c1.update! runtime_status: {'error' => 'Oops!'}
    assert c1.runtime_status.key? 'error'

    # Reset when transitioning from Locked to Queued
    c1.update! state: Container::Queued
    assert_equal c1.runtime_status, {}

    # Allow updates when state = Running
    c1.update! state: Container::Locked
    c1.update! state: Container::Running
    c1.update! runtime_status: {'error' => 'Oops!'}
    assert c1.runtime_status.key? 'error'

    # Don't allow updates on other states
    c1.update! state: Container::Complete
    assert_raises ActiveRecord::RecordInvalid do
      c1.update! runtime_status: {'error' => 'Some other error'}
    end

    set_user_from_auth :active
    c2, _ = minimal_new(attrs)
    assert_equal c2.runtime_status, {}
    set_user_from_auth :dispatch1
    c2.update! state: Container::Locked
    c2.update! state: Container::Running
    c2.update! state: Container::Cancelled
    assert_raises ActiveRecord::RecordInvalid do
      c2.update! runtime_status: {'error' => 'Oops!'}
    end
  end

  test "Container serialized hash attributes sorted before save" do
    set_user_from_auth :active
    env = {"C" => "3", "B" => "2", "A" => "1"}
    m = {"F" => {"kind" => "3"}, "E" => {"kind" => "2"}, "D" => {"kind" => "1"}}
    rc = {"vcpus" => 1, "ram" => 1, "keep_cache_ram" => 1, "keep_cache_disk" => 0, "API" => true, "cuda" => {"device_count":0, "driver_version": "", "hardware_capability": ""}}
    c, _ = minimal_new(environment: env, mounts: m, runtime_constraints: rc)
    c.reload
    assert_equal Container.deep_sort_hash(env).to_json, c.environment.to_json
    assert_equal Container.deep_sort_hash(m).to_json, c.mounts.to_json
    assert_equal Container.deep_sort_hash(rc).to_json, c.runtime_constraints.to_json
  end

  test 'deep_sort_hash on array of hashes' do
    a = {'z' => [[{'a' => 'a', 'b' => 'b'}]]}
    b = {'z' => [[{'b' => 'b', 'a' => 'a'}]]}
    assert_equal Container.deep_sort_hash(a).to_json, Container.deep_sort_hash(b).to_json
  end

  test "find_reusable method should select higher priority queued container" do
        Rails.configuration.Containers.LogReuseDecisions = true
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
    c_older.update!({state: Container::Locked})
    c_older.update!({state: Container::Running})
    c_older.update!(completed_attrs)

    c_recent.update!({state: Container::Locked})
    c_recent.update!({state: Container::Running})
    c_recent.update!(completed_attrs)

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

    cr = ContainerRequest.new request_only(common_attrs)
    cr.use_existing = false
    cr.state = ContainerRequest::Committed
    cr.save!
    c_output1 = Container.where(uuid: cr.container_uuid).first

    cr = ContainerRequest.new request_only(common_attrs)
    cr.use_existing = false
    cr.state = ContainerRequest::Committed
    cr.save!
    c_output2 = Container.where(uuid: cr.container_uuid).first

    assert_not_equal c_output1.uuid, c_output2.uuid

    set_user_from_auth :dispatch1

    out1 = '1f4b0bc7583c2a7f9102c395f4ffc5e3+45'
    log1 = collections(:real_log_collection).portable_data_hash
    c_output1.update!({state: Container::Locked})
    c_output1.update!({state: Container::Running})
    c_output1.update!(completed_attrs.merge({log: log1, output: out1}))

    out2 = 'fa7aeb5140e2848d39b416daeef4ffc5+45'
    c_output2.update!({state: Container::Locked})
    c_output2.update!({state: Container::Running})
    c_output2.update!(completed_attrs.merge({log: log1, output: out2}))

    set_user_from_auth :active
    reused = Container.resolve(ContainerRequest.new(request_only(common_attrs)))
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
    c_slower.update!({state: Container::Locked})
    c_slower.update!({state: Container::Running,
                                 progress: 0.1})
    c_faster_started_first.update!({state: Container::Locked})
    c_faster_started_first.update!({state: Container::Running,
                                               progress: 0.15})
    c_faster_started_second.update!({state: Container::Locked})
    c_faster_started_second.update!({state: Container::Running,
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
    c_slower.update!({state: Container::Locked})
    c_slower.update!({state: Container::Running,
                                 progress: 0.1})
    c_faster_started_first.update!({state: Container::Locked})
    c_faster_started_first.update!({state: Container::Running,
                                               progress: 0.15})
    c_faster_started_second.update!({state: Container::Locked})
    c_faster_started_second.update!({state: Container::Running,
                                                progress: 0.2})
    reused = Container.find_reusable(common_attrs)
    assert_not_nil reused
    # Selected container is the one with most progress done
    assert_equal reused.uuid, c_faster_started_second.uuid
  end

  test "find_reusable method should select non-failing running container" do
    set_user_from_auth :active
    common_attrs = REUSABLE_COMMON_ATTRS.merge({environment: {"var" => "running2"}})
    c_slower, _ = minimal_new(common_attrs.merge({use_existing: false}))
    c_faster_started_first, _ = minimal_new(common_attrs.merge({use_existing: false}))
    c_faster_started_second, _ = minimal_new(common_attrs.merge({use_existing: false}))
    # Confirm the 3 container UUIDs are different.
    assert_equal 3, [c_slower.uuid, c_faster_started_first.uuid, c_faster_started_second.uuid].uniq.length
    set_user_from_auth :dispatch1
    c_slower.update!({state: Container::Locked})
    c_slower.update!({state: Container::Running,
                                 progress: 0.1})
    c_faster_started_first.update!({state: Container::Locked})
    c_faster_started_first.update!({state: Container::Running,
                                               runtime_status: {'warning' => 'This is not an error'},
                                               progress: 0.15})
    c_faster_started_second.update!({state: Container::Locked})
    assert_equal 0, Container.where("runtime_status->'error' is not null").count
    c_faster_started_second.update!({state: Container::Running,
                                                runtime_status: {'error' => 'Something bad happened'},
                                                progress: 0.2})
    assert_equal 1, Container.where("runtime_status->'error' is not null").count
    reused = Container.find_reusable(common_attrs)
    assert_not_nil reused
    # Selected the non-failing container even if it's the one with less progress done
    assert_equal reused.uuid, c_faster_started_first.uuid
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
    c_low_priority.update!({state: Container::Locked,
                                       priority: 1})
    c_high_priority_older.update!({state: Container::Locked,
                                              priority: 2})
    c_high_priority_newer.update!({state: Container::Locked,
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
    c_failed.update!({state: Container::Locked})
    c_failed.update!({state: Container::Running})
    c_failed.update!({state: Container::Complete,
                                 exit_code: 42,
                                 log: 'ea10d51bcf88862dbcc36eb292017dfd+45',
                                 output: 'ea10d51bcf88862dbcc36eb292017dfd+45'})
    c_running.update!({state: Container::Locked})
    c_running.update!({state: Container::Running,
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
    c_completed.update!({state: Container::Locked})
    c_completed.update!({state: Container::Running})
    c_completed.update!({state: Container::Complete,
                                    exit_code: 0,
                                    log: 'ea10d51bcf88862dbcc36eb292017dfd+45',
                                    output: '1f4b0bc7583c2a7f9102c395f4ffc5e3+45'})
    c_running.update!({state: Container::Locked})
    c_running.update!({state: Container::Running,
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
    c_locked.update!({state: Container::Locked})
    c_running.update!({state: Container::Locked})
    c_running.update!({state: Container::Running,
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
    c_locked.update!({state: Container::Locked})
    reused = Container.find_reusable(common_attrs)
    assert_not_nil reused
    assert_equal reused.uuid, c_locked.uuid
  end

  test "find_reusable method should not select failed container" do
    set_user_from_auth :active
    attrs = REUSABLE_COMMON_ATTRS.merge({environment: {"var" => "failed"}})
    c, _ = minimal_new(attrs)
    set_user_from_auth :dispatch1
    c.update!({state: Container::Locked})
    c.update!({state: Container::Running})
    c.update!({state: Container::Complete,
                          exit_code: 33})
    reused = Container.find_reusable(attrs)
    assert_nil reused
  end

  [[false, false, true],
   [false, true, true],
   [true, false, false],
   [true, true, true]
  ].each do |c1_preemptible, c2_preemptible, should_reuse|
    [[Container::Queued, 1],
     [Container::Locked, 1],
     [Container::Running, 0],   # not cancelled yet, but obviously will be soon
    ].each do |c1_state, c1_priority|
      test "find_reusable for #{c2_preemptible ? '' : 'non-'}preemptible req should #{should_reuse ? '' : 'not'} reuse a #{c1_state} #{c1_preemptible ? '' : 'non-'}preemptible container with priority #{c1_priority}" do
        configure_preemptible_instance_type
        set_user_from_auth :active
        c1_attrs = REUSABLE_COMMON_ATTRS.merge({environment: {"test" => name, "state" => c1_state}, scheduling_parameters: {"preemptible" => c1_preemptible}})
        c1, _ = minimal_new(c1_attrs)
        set_user_from_auth :dispatch1
        c1.update!({state: Container::Locked}) if c1_state != Container::Queued
        c1.update!({state: Container::Running, priority: c1_priority}) if c1_state == Container::Running
        c2_attrs = c1_attrs.merge({scheduling_parameters: {"preemptible" => c2_preemptible}})
        reused = Container.find_reusable(c2_attrs)
        if should_reuse && c1_priority > 0
          assert_not_nil reused
        else
          assert_nil reused
        end
      end
    end
  end

  test "find_reusable with logging disabled" do
    set_user_from_auth :active
    Rails.logger.expects(:info).never
    Container.find_reusable(REUSABLE_COMMON_ATTRS)
  end

  test "find_reusable with logging enabled" do
    set_user_from_auth :active
    Rails.configuration.Containers.LogReuseDecisions = true
    Rails.logger.expects(:info).at_least(3)
    Container.find_reusable(REUSABLE_COMMON_ATTRS)
  end

  def runtime_token_attr tok
    auth = api_client_authorizations(tok)
    {runtime_user_uuid: User.find_by_id(auth.user_id).uuid,
     runtime_auth_scopes: auth.scopes,
     runtime_token: auth.token}
  end

  test "find_reusable method with same runtime_token" do
    set_user_from_auth :active
    common_attrs = REUSABLE_COMMON_ATTRS.merge({use_existing:false, priority:1, environment:{"var" => "queued"}})
    c1, _ = minimal_new(common_attrs.merge({runtime_token: api_client_authorizations(:container_runtime_token).token}))
    assert_equal Container::Queued, c1.state
    reused = Container.find_reusable(common_attrs.merge(runtime_token_attr(:container_runtime_token)))
    assert_not_nil reused
    assert_equal reused.uuid, c1.uuid
  end

  test "find_reusable method with different runtime_token, same user" do
    set_user_from_auth :active
    common_attrs = REUSABLE_COMMON_ATTRS.merge({use_existing:false, priority:1, environment:{"var" => "queued"}})
    c1, _ = minimal_new(common_attrs.merge({runtime_token: api_client_authorizations(:crt_user).token}))
    assert_equal Container::Queued, c1.state
    reused = Container.find_reusable(common_attrs.merge(runtime_token_attr(:container_runtime_token)))
    assert_not_nil reused
    assert_equal reused.uuid, c1.uuid
  end

  test "find_reusable method with nil runtime_token, then runtime_token with same user" do
    set_user_from_auth :crt_user
    common_attrs = REUSABLE_COMMON_ATTRS.merge({use_existing:false, priority:1, environment:{"var" => "queued"}})
    c1, _ = minimal_new(common_attrs)
    assert_equal Container::Queued, c1.state
    assert_equal users(:container_runtime_token_user).uuid, c1.runtime_user_uuid
    reused = Container.find_reusable(common_attrs.merge(runtime_token_attr(:container_runtime_token)))
    assert_not_nil reused
    assert_equal reused.uuid, c1.uuid
  end

  test "find_reusable method with different runtime_token, different user" do
    set_user_from_auth :crt_user
    common_attrs = REUSABLE_COMMON_ATTRS.merge({use_existing:false, priority:1, environment:{"var" => "queued"}})
    c1, _ = minimal_new(common_attrs.merge({runtime_token: api_client_authorizations(:active).token}))
    assert_equal Container::Queued, c1.state
    reused = Container.find_reusable(common_attrs.merge(runtime_token_attr(:container_runtime_token)))
    # See #14584
    assert_not_nil reused
    assert_equal c1.uuid, reused.uuid
  end

  test "find_reusable method with nil runtime_token, then runtime_token with different user" do
    set_user_from_auth :active
    common_attrs = REUSABLE_COMMON_ATTRS.merge({use_existing:false, priority:1, environment:{"var" => "queued"}})
    c1, _ = minimal_new(common_attrs.merge({runtime_token: nil}))
    assert_equal Container::Queued, c1.state
    reused = Container.find_reusable(common_attrs.merge(runtime_token_attr(:container_runtime_token)))
    # See #14584
    assert_not_nil reused
    assert_equal c1.uuid, reused.uuid
  end

  test "find_reusable method with different runtime_token, different scope, same user" do
    set_user_from_auth :active
    common_attrs = REUSABLE_COMMON_ATTRS.merge({use_existing:false, priority:1, environment:{"var" => "queued"}})
    c1, _ = minimal_new(common_attrs.merge({runtime_token: api_client_authorizations(:runtime_token_limited_scope).token}))
    assert_equal Container::Queued, c1.state
    reused = Container.find_reusable(common_attrs.merge(runtime_token_attr(:container_runtime_token)))
    # See #14584
    assert_not_nil reused
    assert_equal c1.uuid, reused.uuid
  end

  test "find_reusable method with cuda" do
    set_user_from_auth :active
    # No cuda
    no_cuda_attrs = REUSABLE_COMMON_ATTRS.merge({use_existing:false, priority:1, environment:{"var" => "queued"},
                                                runtime_constraints: {"vcpus" => 1, "ram" => 1, "keep_cache_disk"=>0, "keep_cache_ram"=>268435456, "API" => false,
                                                                      "cuda" => {"device_count":0, "driver_version": "", "hardware_capability": ""}},})
    c1, _ = minimal_new(no_cuda_attrs)
    assert_equal Container::Queued, c1.state

    # has cuda
    cuda_attrs = REUSABLE_COMMON_ATTRS.merge({use_existing:false, priority:1, environment:{"var" => "queued"},
                                                runtime_constraints: {"vcpus" => 1, "ram" => 1, "keep_cache_disk"=>0, "keep_cache_ram"=>268435456, "API" => false,
                                                                      "cuda" => {"device_count":1, "driver_version": "11.0", "hardware_capability": "9.0"}},})
    c2, _ = minimal_new(cuda_attrs)
    assert_equal Container::Queued, c2.state

    # should find the no cuda one
    reused = Container.find_reusable(no_cuda_attrs)
    assert_not_nil reused
    assert_equal reused.uuid, c1.uuid

    # should find the cuda one
    reused = Container.find_reusable(cuda_attrs)
    assert_not_nil reused
    assert_equal reused.uuid, c2.uuid
  end

  test "Container running" do
    set_user_from_auth :active
    c, _ = minimal_new priority: 1

    set_user_from_auth :dispatch1
    check_illegal_updates c, [{state: Container::Running},
                              {state: Container::Complete}]

    c.lock
    c.update! state: Container::Running

    check_illegal_modify c
    check_bogus_states c

    check_illegal_updates c, [{state: Container::Queued}]
    c.reload

    c.update! priority: 3
  end

  test "Lock and unlock" do
    set_user_from_auth :active
    c, cr = minimal_new priority: 0

    set_user_from_auth :dispatch1
    assert_equal Container::Queued, c.state

    assert_raise(ArvadosModel::LockFailedError) do
      # "no priority"
      c.lock
    end
    c.reload
    assert cr.update priority: 1

    refute c.update(state: Container::Running), "not locked"
    c.reload
    refute c.update(state: Container::Complete), "not locked"
    c.reload

    assert c.lock, show_errors(c)
    assert c.locked_by_uuid
    assert c.auth_uuid

    assert_raise(ArvadosModel::LockFailedError) {c.lock}
    c.reload

    assert c.unlock, show_errors(c)
    refute c.locked_by_uuid
    refute c.auth_uuid

    refute c.update(state: Container::Running), "not locked"
    c.reload
    refute c.locked_by_uuid
    refute c.auth_uuid

    assert c.lock, show_errors(c)
    assert c.update(state: Container::Running), show_errors(c)
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

    assert c.update(state: Container::Complete), show_errors(c)
    refute c.locked_by_uuid
    refute c.auth_uuid

    auth_exp = ApiClientAuthorization.find_by_uuid(auth_uuid_was).expires_at
    assert_operator auth_exp, :<, db_current_time

    assert_nil ApiClientAuthorization.validate(token: ApiClientAuthorization.find_by_uuid(auth_uuid_was).token)
  end

  test "Exceed maximum lock-unlock cycles" do
    Rails.configuration.Containers.MaxDispatchAttempts = 3

    set_user_from_auth :active
    c, cr = minimal_new

    set_user_from_auth :dispatch1
    assert_equal Container::Queued, c.state
    assert_equal 0, c.lock_count

    c.lock
    c.reload
    assert_equal 1, c.lock_count
    assert_equal Container::Locked, c.state

    c.unlock
    c.reload
    assert_equal 1, c.lock_count
    assert_equal Container::Queued, c.state

    c.lock
    c.reload
    assert_equal 2, c.lock_count
    assert_equal Container::Locked, c.state

    c.unlock
    c.reload
    assert_equal 2, c.lock_count
    assert_equal Container::Queued, c.state

    c.lock
    c.reload
    assert_equal 3, c.lock_count
    assert_equal Container::Locked, c.state

    c.unlock
    c.reload
    assert_equal 3, c.lock_count
    assert_equal Container::Cancelled, c.state

    assert_raise(ArvadosModel::LockFailedError) do
      # Cancelled to Locked is not allowed
      c.lock
    end
  end

  test "Container queued cancel" do
    set_user_from_auth :active
    c, cr = minimal_new({container_count_max: 1})
    set_user_from_auth :dispatch1
    assert c.update(state: Container::Cancelled), show_errors(c)
    check_no_change_from_cancelled c
    cr.reload
    assert_equal ContainerRequest::Final, cr.state
  end

  test "Container queued count" do
    assert_equal 1, Container.readable_by(users(:active)).where(state: "Queued").count
  end

  test "Containers with no matching request are readable by admin" do
    uuids = Container.includes('container_requests').where(container_requests: {uuid: nil}).collect(&:uuid)
    assert_not_empty uuids
    assert_empty Container.readable_by(users(:active)).where(uuid: uuids)
    assert_not_empty Container.readable_by(users(:admin)).where(uuid: uuids)
    assert_equal uuids.count, Container.readable_by(users(:admin)).where(uuid: uuids).count
  end

  test "Container locked cancel" do
    set_user_from_auth :active
    c, _ = minimal_new
    set_user_from_auth :dispatch1
    assert c.lock, show_errors(c)
    assert c.update(state: Container::Cancelled), show_errors(c)
    check_no_change_from_cancelled c
  end

  test "Container locked with non-expiring token" do
    Rails.configuration.API.TokenMaxLifetime = 1.hour
    set_user_from_auth :active
    c, _ = minimal_new
    set_user_from_auth :dispatch1
    assert c.lock, show_errors(c)
    refute c.auth.nil?
    assert c.auth.expires_at.nil?
    assert c.auth.user_id == User.find_by_uuid(users(:active).uuid).id
  end

  test "Container locked cancel with log" do
    set_user_from_auth :active
    c, _ = minimal_new
    set_user_from_auth :dispatch1
    assert c.lock, show_errors(c)
    assert c.update(
             state: Container::Cancelled,
             log: collections(:real_log_collection).portable_data_hash,
           ), show_errors(c)
    check_no_change_from_cancelled c
  end

  test "Container running cancel" do
    set_user_from_auth :active
    c, _ = minimal_new
    set_user_from_auth :dispatch1
    c.lock
    c.update! state: Container::Running
    c.update! state: Container::Cancelled
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

  [
    [Container::Queued, {state: Container::Locked}],
    [Container::Queued, {state: Container::Running}],
    [Container::Queued, {state: Container::Complete}],
    [Container::Queued, {state: Container::Cancelled}],
    [Container::Queued, {priority: 123456789}],
    [Container::Queued, {runtime_status: {'error' => 'oops'}}],
    [Container::Queued, {cwd: '/'}],
    [Container::Locked, {state: Container::Running}],
    [Container::Locked, {state: Container::Queued}],
    [Container::Locked, {priority: 123456789}],
    [Container::Locked, {runtime_status: {'error' => 'oops'}}],
    [Container::Locked, {cwd: '/'}],
    [Container::Running, {state: Container::Complete}],
    [Container::Running, {state: Container::Cancelled}],
    [Container::Running, {priority: 123456789}],
    [Container::Running, {runtime_status: {'error' => 'oops'}}],
    [Container::Running, {cwd: '/'}],
    [Container::Running, {gateway_address: "172.16.0.1:12345"}],
    [Container::Running, {interactive_session_started: true}],
    [Container::Complete, {state: Container::Cancelled}],
    [Container::Complete, {priority: 123456789}],
    [Container::Complete, {runtime_status: {'error' => 'oops'}}],
    [Container::Complete, {cwd: '/'}],
    [Container::Cancelled, {cwd: '/'}],
  ].each do |start_state, updates|
    test "Container update #{updates.inspect} when #{start_state} forbidden for non-admin" do
      set_user_from_auth :active
      c, _ = minimal_new
      if start_state != Container::Queued
        set_user_from_auth :dispatch1
        c.lock
        if start_state != Container::Locked
          c.update! state: Container::Running
          if start_state != Container::Running
            c.update! state: start_state
          end
        end
      end
      assert_equal c.state, start_state
      set_user_from_auth :active
      assert_raises(ArvadosModel::PermissionDeniedError) do
        c.update! updates
      end
    end
  end

  test "can only change exit code while running and at completion" do
    set_user_from_auth :active
    c, _ = minimal_new
    set_user_from_auth :dispatch1
    c.lock
    check_illegal_updates c, [{exit_code: 1}]
    c.update! state: Container::Running
    assert c.update(exit_code: 1)
    assert c.update(exit_code: 1, state: Container::Complete)
  end

  test "locked_by_uuid can update log when locked/running, and output when running" do
    set_user_from_auth :active
    logcoll = collections(:real_log_collection)
    c, cr1 = minimal_new
    cr2 = ContainerRequest.new(DEFAULT_ATTRS)
    cr2.state = ContainerRequest::Committed
    act_as_user users(:active) do
      cr2.save!
    end
    assert_equal cr1.container_uuid, cr2.container_uuid

    logpdh_time1 = logcoll.portable_data_hash

    set_user_from_auth :dispatch1
    c.lock
    assert_equal c.locked_by_uuid, Thread.current[:api_client_authorization].uuid
    c.update!(log: logpdh_time1)
    c.update!(state: Container::Running)
    cr1.reload
    cr2.reload
    cr1log_uuid = cr1.log_uuid
    cr2log_uuid = cr2.log_uuid
    assert_not_nil cr1log_uuid
    assert_not_nil cr2log_uuid
    assert_not_equal logcoll.uuid, cr1log_uuid
    assert_not_equal logcoll.uuid, cr2log_uuid
    assert_not_equal cr1log_uuid, cr2log_uuid

    logcoll.update!(manifest_text: logcoll.manifest_text + ". acbd18db4cc2f85cedef654fccc4a4d8+3 0:3:foo.txt\n")
    logpdh_time2 = logcoll.portable_data_hash

    assert c.update(output: collections(:collection_owned_by_active).portable_data_hash)
    assert c.update(log: logpdh_time2)
    assert c.update(state: Container::Complete, log: logcoll.portable_data_hash)
    c.reload
    assert_equal collections(:collection_owned_by_active).portable_data_hash, c.output
    assert_equal logpdh_time2, c.log
    refute c.update(output: nil)
    refute c.update(log: nil)
    cr1.reload
    cr2.reload
    assert_equal cr1log_uuid, cr1.log_uuid
    assert_equal cr2log_uuid, cr2.log_uuid
    assert_equal 1, Collection.where(uuid: [cr1log_uuid, cr2log_uuid]).to_a.collect(&:portable_data_hash).uniq.length
    assert_equal ". acbd18db4cc2f85cedef654fccc4a4d8+3 cdd549ae79fe6640fa3d5c6261d8303c+195 0:3:foo.txt 3:195:zzzzz-8i9sb-0vsrcqi7whchuil.log.txt
./log\\040for\\040container\\040#{cr1.container_uuid} acbd18db4cc2f85cedef654fccc4a4d8+3 cdd549ae79fe6640fa3d5c6261d8303c+195 0:3:foo.txt 3:195:zzzzz-8i9sb-0vsrcqi7whchuil.log.txt
", Collection.find_by_uuid(cr1log_uuid).manifest_text
  end

  ["auth_uuid", "runtime_token"].each do |tok|
    test "#{tok} can set output, progress, runtime_status, state, exit_code on running container -- but not log" do
      if tok == "runtime_token"
        set_user_from_auth :spectator
        c, _ = minimal_new(container_image: "9ae44d5792468c58bcf85ce7353c7027+124",
                           runtime_token: api_client_authorizations(:active).token)
      else
        set_user_from_auth :active
        c, _ = minimal_new
      end
      set_user_from_auth :dispatch1
      c.lock
      c.update! state: Container::Running

      if tok == "runtime_token"
        auth = ApiClientAuthorization.validate(token: c.runtime_token)
        Thread.current[:api_client_authorization] = auth
        Thread.current[:api_client] = auth.api_client
        Thread.current[:token] = auth.token
        Thread.current[:user] = auth.user
      else
        auth = ApiClientAuthorization.find_by_uuid(c.auth_uuid)
        Thread.current[:api_client_authorization] = auth
        Thread.current[:api_client] = auth.api_client
        Thread.current[:token] = auth.token
        Thread.current[:user] = auth.user
      end

      assert c.update(gateway_address: "127.0.0.1:9")
      assert c.update(output: collections(:collection_owned_by_active).portable_data_hash)
      assert c.update(runtime_status: {'warning' => 'something happened'})
      assert c.update(progress: 0.5)
      assert c.update(exit_code: 0)
      refute c.update(log: collections(:real_log_collection).portable_data_hash)
      c.reload
      assert c.update(state: Container::Complete, exit_code: 0)
    end
  end

  test "not allowed to set output that is not readable by current user" do
    set_user_from_auth :active
    c, _ = minimal_new
    set_user_from_auth :dispatch1
    c.lock
    c.update! state: Container::Running

    Thread.current[:api_client_authorization] = ApiClientAuthorization.find_by_uuid(c.auth_uuid)
    Thread.current[:user] = User.find_by_id(Thread.current[:api_client_authorization].user_id)

    assert_raises ActiveRecord::RecordInvalid do
      c.update! output: collections(:collection_not_readable_by_active).portable_data_hash
    end
  end

  test "other token cannot set output on running container" do
    set_user_from_auth :active
    c, _ = minimal_new
    set_user_from_auth :dispatch1
    c.lock
    c.update! state: Container::Running

    set_user_from_auth :running_to_be_deleted_container_auth
    assert_raises(ArvadosModel::PermissionDeniedError) do
      c.update(output: collections(:foo_file).portable_data_hash)
    end
  end

  test "can set trashed output on running container" do
    set_user_from_auth :active
    c, _ = minimal_new
    set_user_from_auth :dispatch1
    c.lock
    c.update! state: Container::Running

    output = Collection.find_by_uuid('zzzzz-4zz18-mto52zx1s7sn3jk')

    assert output.is_trashed
    assert c.update output: output.portable_data_hash
    assert c.update! state: Container::Complete
  end

  test "not allowed to set trashed output that is not readable by current user" do
    set_user_from_auth :active
    c, _ = minimal_new
    set_user_from_auth :dispatch1
    c.lock
    c.update! state: Container::Running

    output = Collection.find_by_uuid('zzzzz-4zz18-mto52zx1s7sn3jr')

    Thread.current[:api_client_authorization] = ApiClientAuthorization.find_by_uuid(c.auth_uuid)
    Thread.current[:user] = User.find_by_id(Thread.current[:api_client_authorization].user_id)

    assert_raises ActiveRecord::RecordInvalid do
      c.update! output: output.portable_data_hash
    end
  end

  test "user cannot delete" do
    set_user_from_auth :active
    c, _ = minimal_new
    assert_raises ArvadosModel::PermissionDeniedError do
      c.destroy
    end
    assert Container.find_by_uuid(c.uuid)
  end

  [
    {state: Container::Complete, exit_code: 0, output: '1f4b0bc7583c2a7f9102c395f4ffc5e3+45'},
    {state: Container::Cancelled},
  ].each do |final_attrs|
    test "secret_mounts and runtime_token are null after container is #{final_attrs[:state]}" do
      set_user_from_auth :active
      c, cr = minimal_new(secret_mounts: {'/secret' => {'kind' => 'text', 'content' => 'foo'}},
                          container_count_max: 1, runtime_token: api_client_authorizations(:active).token)
      set_user_from_auth :dispatch1
      c.lock
      c.update!(state: Container::Running)
      c.reload
      assert c.secret_mounts.has_key?('/secret')
      assert_equal api_client_authorizations(:active).token, c.runtime_token

      c.update!(final_attrs)
      c.reload
      assert_equal({}, c.secret_mounts)
      assert_nil c.runtime_token
      cr.reload
      assert_equal({}, cr.secret_mounts)
      assert_nil cr.runtime_token
      assert_no_secrets_logged
    end
  end

  def configure_preemptible_instance_type
    Rails.configuration.InstanceTypes = ConfigLoader.to_OrderedOptions({
      "a1.small.pre" => {
        "Preemptible" => true,
        "Price" => 0.1,
        "ProviderType" => "a1.small",
        "VCPUs" => 1,
        "RAM" => 1000000000,
      },
    })
  end

  def vary_parameters(**kwargs)
    # kwargs is a hash that maps parameters to an array of values.
    # This function enumerates every possible hash where each key has one of
    # the values from its array.
    # The output keys are strings since that's what container hash attributes
    # want.
    # A nil value yields a hash without that key.
    [[:_, nil]].product(
      *kwargs.map { |(key, values)| [key.to_s].product(values) },
    ).map { |param_pairs| Hash[param_pairs].compact }
  end

  def retry_with_scheduling_parameters(param_hashes)
    set_user_from_auth :admin
    containers = {}
    requests = []
    param_hashes.each do |scheduling_parameters|
      container, request = minimal_new(scheduling_parameters: scheduling_parameters)
      containers[container.uuid] = container
      requests << request
    end
    refute(containers.empty?, "buggy test: no scheduling parameters enumerated")
    assert_equal(1, containers.length)
    _, container1 = containers.shift
    container1.lock
    container1.update!(state: Container::Cancelled)
    container1.reload
    request1 = requests.shift
    request1.reload
    assert_not_equal(container1.uuid, request1.container_uuid)
    requests.each do |request|
      request.reload
      assert_equal(request1.container_uuid, request.container_uuid)
    end
    container2 = Container.find_by_uuid(request1.container_uuid)
    assert_not_nil(container2)
    return container2
  end

  preemptible_values = [true, false, nil]
  preemptible_values.permutation(1).chain(
    preemptible_values.product(preemptible_values),
    preemptible_values.product(preemptible_values, preemptible_values),
  ).each do |preemptible_a|
    # If the first req has preemptible=true but a subsequent req
    # doesn't, we want to avoid reusing the first container, so this
    # test isn't appropriate.
    next if preemptible_a[0] &&
            ((preemptible_a.length > 1 && !preemptible_a[1]) ||
             (preemptible_a.length > 2 && !preemptible_a[2]))
    test "retry requests scheduled with preemptible=#{preemptible_a}" do
      configure_preemptible_instance_type
      param_hashes = vary_parameters(preemptible: preemptible_a)
      container = retry_with_scheduling_parameters(param_hashes)
      assert_equal(preemptible_a.all?,
                   container.scheduling_parameters["preemptible"] || false)
    end
  end

  partition_values = [nil, [], ["alpha"], ["alpha", "bravo"], ["bravo", "charlie"]]
  partition_values.permutation(1).chain(
    partition_values.permutation(2),
  ).each do |partitions_a|
    test "retry requests scheduled with partitions=#{partitions_a}" do
      param_hashes = vary_parameters(partitions: partitions_a)
      container = retry_with_scheduling_parameters(param_hashes)
      expected = if partitions_a.any? { |value| value.nil? or value.empty? }
                   []
                 else
                   partitions_a.flatten.uniq
                 end
      actual = container.scheduling_parameters["partitions"] || []
      assert_equal(expected.sort, actual.sort)
    end
  end

  runtime_values = [nil, 0, 1, 2, 3]
  runtime_values.permutation(1).chain(
    runtime_values.permutation(2),
    runtime_values.permutation(3),
  ).each do |max_run_time_a|
    test "retry requests scheduled with max_run_time=#{max_run_time_a}" do
      param_hashes = vary_parameters(max_run_time: max_run_time_a)
      container = retry_with_scheduling_parameters(param_hashes)
      expected = if max_run_time_a.any? { |value| value.nil? or value == 0 }
                   0
                 else
                   max_run_time_a.max
                 end
      actual = container.scheduling_parameters["max_run_time"] || 0
      assert_equal(expected, actual)
    end
  end

  test "retry requests with multi-varied scheduling parameters" do
    configure_preemptible_instance_type
    param_hashes = [{
                     "partitions": ["alpha", "bravo"],
                     "preemptible": false,
                     "max_run_time": 10,
                    }, {
                     "partitions": ["alpha", "charlie"],
                     "max_run_time": 20,
                    }, {
                     "partitions": ["bravo", "charlie"],
                     "preemptible": true,
                     "max_run_time": 30,
                    }]
    container = retry_with_scheduling_parameters(param_hashes)
    actual = container.scheduling_parameters
    assert_equal(["alpha", "bravo", "charlie"], actual["partitions"]&.sort)
    assert_equal(false, actual["preemptible"] || false)
    assert_equal(30, actual["max_run_time"])
  end

  test "retry requests with unset scheduling parameters" do
    configure_preemptible_instance_type
    param_hashes = vary_parameters(
      preemptible: [nil, true],
      partitions: [nil, ["alpha"]],
      max_run_time: [nil, 5],
    )
    container = retry_with_scheduling_parameters(param_hashes)
    actual = container.scheduling_parameters
    assert_equal([], actual["partitions"] || [])
    assert_equal(false, actual["preemptible"] || false)
    assert_equal(0, actual["max_run_time"] || 0)
  end

  test "retry requests with default scheduling parameters" do
    configure_preemptible_instance_type
    param_hashes = vary_parameters(
      preemptible: [false, true],
      partitions: [[], ["bravo"]],
      max_run_time: [0, 1],
    )
    container = retry_with_scheduling_parameters(param_hashes)
    actual = container.scheduling_parameters
    assert_equal([], actual["partitions"] || [])
    assert_equal(false, actual["preemptible"] || false)
    assert_equal(0, actual["max_run_time"] || 0)
  end

  def run_container(request_params, final_attrs)
    final_attrs[:state] ||= Container::Complete
    if final_attrs[:state] == Container::Complete
      final_attrs[:exit_code] ||= 0
      final_attrs[:log] ||= collections(:log_collection).portable_data_hash
      final_attrs[:output] ||= collections(:multilevel_collection_1).portable_data_hash
    end
    container, request = minimal_new(request_params)
    container.lock
    container.update!(state: Container::Running)
    container.update!(final_attrs)
    return container, request
  end

  def check_reuse_with_variations(default_keep_cache_ram, vary_attr, start_value, variations)
    container_params = REUSABLE_ATTRS_SLIM.merge(vary_attr => start_value)
    orig_default = Rails.configuration.Containers.DefaultKeepCacheRAM
    begin
      Rails.configuration.Containers.DefaultKeepCacheRAM = default_keep_cache_ram
      set_user_from_auth :admin
      expected, _ = run_container(container_params, {})
      variations.each do |variation|
        full_variation = REUSABLE_ATTRS_SLIM[vary_attr].merge(variation)
        parameters = REUSABLE_ATTRS_SLIM.merge(vary_attr => full_variation)
        actual = Container.find_reusable(parameters)
        assert_equal(expected.uuid, actual&.uuid,
                     "request with #{vary_attr}=#{variation} did not reuse container")
      end
    ensure
      Rails.configuration.Containers.DefaultKeepCacheRAM = orig_default
    end
  end

  # Test that we can reuse a container with a known keep_cache_ram constraint,
  # no matter what keep_cache_* constraints the new request uses.
  [0, 2 << 30, 4 << 30].product(
    [0, 1],
    [true, false],
  ).each do |(default_keep_cache_ram, multiplier, keep_disk_constraint)|
    test "reuse request with DefaultKeepCacheRAM=#{default_keep_cache_ram}, keep_cache_ram*=#{multiplier}, keep_cache_disk=#{keep_disk_constraint}" do
      runtime_constraints = REUSABLE_ATTRS_SLIM[:runtime_constraints].merge(
        "keep_cache_ram" => default_keep_cache_ram * multiplier,
      )
      if not keep_disk_constraint
        # Simulate a container that predates keep_cache_disk by deleting
        # the constraint entirely.
        runtime_constraints.delete("keep_cache_disk")
      end
      # Important values are:
      # * 0
      # * 2GiB, the minimum default keep_cache_disk
      # * 8GiB, the default keep_cache_disk based on container ram
      # * 32GiB, the maximum default keep_cache_disk
      # Check these values and values in between.
      vary_values = [0, 1, 2, 6, 8, 10, 32, 33].map { |v| v << 30 }.to_a
      variations = vary_parameters(keep_cache_ram: vary_values)
                     .chain(vary_parameters(keep_cache_disk: vary_values))
      check_reuse_with_variations(
        default_keep_cache_ram,
        :runtime_constraints,
        runtime_constraints,
        variations,
      )
    end
  end

  # Test that we can reuse a container with a known keep_cache_disk constraint,
  # no matter what keep_cache_* constraints the new request uses.
  # keep_cache_disk values are the important values discussed in the test above.
  [0, 2 << 30, 4 << 30]
    .product([0, 2 << 30, 8 << 30, 32 << 30])
    .each do |(default_keep_cache_ram, keep_cache_disk)|
    test "reuse request with DefaultKeepCacheRAM=#{default_keep_cache_ram} and keep_cache_disk=#{keep_cache_disk}" do
      runtime_constraints = REUSABLE_ATTRS_SLIM[:runtime_constraints].merge(
        "keep_cache_disk" => keep_cache_disk,
      )
      vary_values = [0, 1, 2, 6, 8, 10, 32, 33].map { |v| v << 30 }.to_a
      variations = vary_parameters(keep_cache_ram: vary_values)
                     .chain(vary_parameters(keep_cache_disk: vary_values))
      check_reuse_with_variations(
        default_keep_cache_ram,
        :runtime_constraints,
        runtime_constraints,
        variations,
      )
    end
  end

  # Test that a container request can reuse a container with an exactly
  # matching keep_cache_* constraint, no matter what the defaults.
  [0, 2 << 30, 4 << 30].product(
    ["keep_cache_disk", "keep_cache_ram"],
    [135790, 13 << 30, 135 << 30],
  ).each do |(default_keep_cache_ram, constraint_key, constraint_value)|
    test "reuse request with #{constraint_key}=#{constraint_value} and DefaultKeepCacheRAM=#{default_keep_cache_ram}" do
      runtime_constraints = REUSABLE_ATTRS_SLIM[:runtime_constraints].merge(
        constraint_key => constraint_value,
      )
      check_reuse_with_variations(
        default_keep_cache_ram,
        :runtime_constraints,
        runtime_constraints,
        [runtime_constraints],
      )
    end
  end
end
