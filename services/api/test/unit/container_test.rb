require 'test_helper'

class ContainerTest < ActiveSupport::TestCase
  include DbCurrentTime

  DEFAULT_ATTRS = {
    command: ['echo', 'foo'],
    container_image: 'img',
    output_path: '/tmp',
    priority: 1,
    runtime_constraints: {"vcpus" => 1, "ram" => 1},
  }

  def minimal_new attrs={}
    cr = ContainerRequest.new DEFAULT_ATTRS.merge(attrs)
    act_as_user users(:active) do
      cr.save!
    end
    c = Container.new DEFAULT_ATTRS.merge(attrs)
    act_as_system_user do
      c.save!
      assert cr.update_attributes(container_uuid: c.uuid,
                                  state: ContainerRequest::Committed,
                                  ), show_errors(cr)
    end
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
                              {container_image: "img2"},
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

  test "Container running" do
    c, _ = minimal_new priority: 1

    set_user_from_auth :dispatch1
    check_illegal_updates c, [{state: Container::Running},
                              {state: Container::Complete}]

    c.update_attributes! state: Container::Locked
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

    refute c.update_attributes(state: Container::Locked), "no priority"
    c.reload
    assert cr.update_attributes priority: 1

    refute c.update_attributes(state: Container::Running), "not locked"
    c.reload
    refute c.update_attributes(state: Container::Complete), "not locked"
    c.reload

    assert c.update_attributes(state: Container::Locked), show_errors(c)
    assert c.locked_by_uuid
    assert c.auth_uuid

    assert c.update_attributes(state: Container::Queued), show_errors(c)
    refute c.locked_by_uuid
    refute c.auth_uuid

    refute c.update_attributes(state: Container::Running), "not locked"
    c.reload
    refute c.locked_by_uuid
    refute c.auth_uuid

    assert c.update_attributes(state: Container::Locked), show_errors(c)
    assert c.update_attributes(state: Container::Running), show_errors(c)
    assert c.locked_by_uuid
    assert c.auth_uuid

    auth_uuid_was = c.auth_uuid

    refute c.update_attributes(state: Container::Locked), "already running"
    c.reload
    refute c.update_attributes(state: Container::Queued), "already running"
    c.reload

    assert c.update_attributes(state: Container::Complete), show_errors(c)
    refute c.locked_by_uuid
    refute c.auth_uuid

    auth_exp = ApiClientAuthorization.find_by_uuid(auth_uuid_was).expires_at
    assert_operator auth_exp, :<, db_current_time
  end

  test "Container queued cancel" do
    c, _ = minimal_new
    set_user_from_auth :dispatch1
    assert c.update_attributes(state: Container::Cancelled), show_errors(c)
    check_no_change_from_cancelled c
  end

  test "Container locked cancel" do
    c, _ = minimal_new
    set_user_from_auth :dispatch1
    assert c.update_attributes(state: Container::Locked), show_errors(c)
    assert c.update_attributes(state: Container::Cancelled), show_errors(c)
    check_no_change_from_cancelled c
  end

  test "Container running cancel" do
    c, _ = minimal_new
    set_user_from_auth :dispatch1
    c.update_attributes! state: Container::Queued
    c.update_attributes! state: Container::Locked
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
    c.update_attributes! state: Container::Locked
    c.update_attributes! state: Container::Running

    check_illegal_updates c, [{exit_code: 1},
                              {exit_code: 1, state: Container::Cancelled}]

    assert c.update_attributes(exit_code: 1, state: Container::Complete)
  end
end
