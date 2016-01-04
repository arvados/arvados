require 'test_helper'

class ContainerTest < ActiveSupport::TestCase
  def check_illegal_modify c
    c.reload
    c.command = ["echo", "bar"]
    assert_raises(ActiveRecord::RecordInvalid) do
      c.save!
    end

    c.reload
    c.container_image = "img2"
    assert_raises(ActiveRecord::RecordInvalid) do
      c.save!
    end

    c.reload
    c.cwd = "/tmp2"
    assert_raises(ActiveRecord::RecordInvalid) do
      c.save!
    end

    c.reload
    c.environment = {"FOO" => "BAR"}
    assert_raises(ActiveRecord::RecordInvalid) do
      c.save!
    end

    c.reload
    c.mounts = {"FOO" => "BAR"}
    assert_raises(ActiveRecord::RecordInvalid) do
      c.save!
    end

    c.reload
    c.output_path = "/tmp3"
    assert_raises(ActiveRecord::RecordInvalid) do
      c.save!
    end

    c.reload
    c.runtime_constraints = {"FOO" => "BAR"}
    assert_raises(ActiveRecord::RecordInvalid) do
      c.save!
    end
  end

  def check_bogus_states c
    c.reload
    c.state = nil
    assert_raises(ActiveRecord::RecordInvalid) do
      c.save!
    end

    c.reload
    c.state = "Flubber"
    assert_raises(ActiveRecord::RecordInvalid) do
      c.save!
    end
  end

  def check_no_change_from_complete c
    check_illegal_modify c
    check_bogus_states c

    c.reload
    c.priority = 3
    assert_raises(ActiveRecord::RecordInvalid) do
      c.save!
    end

    c.reload
    c.state = "Queued"
    assert_raises(ActiveRecord::RecordInvalid) do
      c.save!
    end

    c.reload
    c.state = "Running"
    assert_raises(ActiveRecord::RecordInvalid) do
      c.save!
    end

    c.reload
    c.state = "Complete"
    assert_raises(ActiveRecord::RecordInvalid) do
      c.save!
    end
  end

  test "Container create" do
    act_as_system_user do
      c = Container.new
      c.command = ["echo", "foo"]
      c.container_image = "img"
      c.cwd = "/tmp"
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
      c = Container.new
      c.command = ["echo", "foo"]
      c.container_image = "img"
      c.output_path = "/tmp"
      c.save!

      c.reload
      c.state = "Complete"
      assert_raises(ActiveRecord::RecordInvalid) do
        c.save!
      end

      c.reload
      c.state = "Running"
      c.save!

      check_illegal_modify c
      check_bogus_states c

      c.reload
      c.state = "Queued"
      assert_raises(ActiveRecord::RecordInvalid) do
        c.save!
      end

      c.reload
      c.priority = 3
      c.save!
    end
  end

  test "Container queued cancel" do
    act_as_system_user do
      c = Container.new
      c.command = ["echo", "foo"]
      c.container_image = "img"
      c.output_path = "/tmp"
      c.save!

      c.reload
      c.state = "Cancelled"
      c.save!

      check_no_change_from_complete c
    end
  end

  test "Container running cancel" do
    act_as_system_user do
      c = Container.new
      c.command = ["echo", "foo"]
      c.container_image = "img"
      c.output_path = "/tmp"
      c.save!

      c.reload
      c.state = "Running"
      c.save!

      c.reload
      c.state = "Cancelled"
      c.save!

      check_no_change_from_complete c
    end
  end

  test "Container create forbidden for non-admin" do
    set_user_from_auth :active_trustedclient
    c = Container.new
    c.command = ["echo", "foo"]
    c.container_image = "img"
    c.cwd = "/tmp"
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
      c = Container.new
      c.command = ["echo", "foo"]
      c.container_image = "img"
      c.output_path = "/tmp"
      c.save!

      c.reload
      c.state = "Running"
      c.save!

      c.reload
      c.exit_code = 1
      assert_raises(ActiveRecord::RecordInvalid) do
        c.save!
      end

      c.reload
      c.exit_code = 1
      c.state = "Cancelled"
      assert_raises(ActiveRecord::RecordInvalid) do
        c.save!
      end

      c.reload
      c.exit_code = 1
      c.state = "Complete"
      c.save!
    end
  end
end
