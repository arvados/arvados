require 'test_helper'

class ContainerTest < ActiveSupport::TestCase
  def minimal_new
    c = Container.new
    c.command = ["echo", "foo"]
    c.container_image = "img"
    c.output_path = "/tmp"
    c
  end

  def show_errors c
    return lambda { c.errors.full_messages.inspect }
  end

  def check_illegal_updates c, bad_updates
    bad_updates.each do |u|
      refute c.update_attributes(u), u.inspect
      refute c.valid?
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
      c = minimal_new
      c.environment = {}
      c.mounts = {"BAR" => "FOO"}
      c.output_path = "/tmp"
      c.priority = 1
      c.runtime_constraints = {}
      c.save!

      check_illegal_modify c
      check_bogus_states c

      c.reload
      c.priority = 2
      c.save!
    end
  end

  test "Container running" do
    act_as_system_user do
      c = minimal_new
      c.save!

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
  end

  test "Lock and unlock" do
    act_as_system_user do
      c = minimal_new
      c.save!
      assert_equal Container::Queued, c.state

      refute c.update_attributes(state: Container::Running), "not locked"
      c.reload
      refute c.update_attributes(state: Container::Complete), "not locked"
      c.reload

      assert c.update_attributes(state: Container::Locked), show_errors(c)
      assert c.update_attributes(state: Container::Queued), show_errors(c)

      refute c.update_attributes(state: Container::Running), "not locked"
      c.reload

      assert c.update_attributes(state: Container::Locked), show_errors(c)
      assert c.update_attributes(state: Container::Running), show_errors(c)

      refute c.update_attributes(state: Container::Locked), "already running"
      c.reload
      refute c.update_attributes(state: Container::Queued), "already running"
      c.reload

      assert c.update_attributes(state: Container::Complete), show_errors(c)
    end
  end

  test "Container queued cancel" do
    act_as_system_user do
      c = minimal_new
      c.save!
      assert c.update_attributes(state: Container::Cancelled), show_errors(c)
      check_no_change_from_cancelled c
    end
  end

  test "Container locked cancel" do
    act_as_system_user do
      c = minimal_new
      c.save!
      assert c.update_attributes(state: Container::Locked), show_errors(c)
      assert c.update_attributes(state: Container::Cancelled), show_errors(c)
      check_no_change_from_cancelled c
    end
  end

  test "Container running cancel" do
    act_as_system_user do
      c = minimal_new
      c.save!
      c.update_attributes! state: Container::Queued
      c.update_attributes! state: Container::Locked
      c.update_attributes! state: Container::Running
      c.update_attributes! state: Container::Cancelled
      check_no_change_from_cancelled c
    end
  end

  test "Container create forbidden for non-admin" do
    set_user_from_auth :active_trustedclient
    c = minimal_new
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
    act_as_system_user do
      c = minimal_new
      c.save!
      c.update_attributes! state: Container::Locked
      c.update_attributes! state: Container::Running

      check_illegal_updates c, [{exit_code: 1},
                                {exit_code: 1, state: Container::Cancelled}]

      assert c.update_attributes(exit_code: 1, state: Container::Complete)
    end
  end
end
