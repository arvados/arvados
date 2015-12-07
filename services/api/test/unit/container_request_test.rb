require 'test_helper'

class ContainerRequestTest < ActiveSupport::TestCase
  def check_illegal_modify c
      assert_raises(ActiveRecord::RecordInvalid) do
        c.reload
        c.command = ["echo", "bar"]
        c.save!
      end

      assert_raises(ActiveRecord::RecordInvalid) do
        c.reload
        c.container_image = "img2"
        c.save!
      end

      assert_raises(ActiveRecord::RecordInvalid) do
        c.reload
        c.cwd = "/tmp2"
        c.save!
      end

      assert_raises(ActiveRecord::RecordInvalid) do
        c.reload
        c.environment = {"FOO" => "BAR"}
        c.save!
      end

      assert_raises(ActiveRecord::RecordInvalid) do
        c.reload
        c.mounts = {"FOO" => "BAR"}
        c.save!
      end

      assert_raises(ActiveRecord::RecordInvalid) do
        c.reload
        c.output_path = "/tmp3"
        c.save!
      end

      assert_raises(ActiveRecord::RecordInvalid) do
        c.reload
        c.runtime_constraints = {"FOO" => "BAR"}
        c.save!
      end

      assert_raises(ActiveRecord::RecordInvalid) do
        c.reload
        c.name = "baz"
        c.save!
      end

      assert_raises(ActiveRecord::RecordInvalid) do
        c.reload
        c.description = "baz"
        c.save!
      end

  end

  def check_bogus_states c
      assert_raises(ActiveRecord::RecordInvalid) do
        c.reload
        c.state = nil
        c.save!
      end

      assert_raises(ActiveRecord::RecordInvalid) do
        c.reload
        c.state = "Flubber"
        c.save!
      end
  end

  test "Container request create" do
    set_user_from_auth :active_trustedclient
    c = ContainerRequest.new
    c.command = ["echo", "foo"]
    c.container_image = "img"
    c.cwd = "/tmp"
    c.environment = {}
    c.mounts = {"BAR" => "FOO"}
    c.output_path = "/tmp"
    c.priority = 1
    c.runtime_constraints = {}
    c.name = "foo"
    c.description = "bar"
    c.save!

    assert_nil c.container_uuid

    check_bogus_states c

    c.reload
    c.command = ["echo", "foo3"]
    c.container_image = "img3"
    c.cwd = "/tmp3"
    c.environment = {"BUP" => "BOP"}
    c.mounts = {"BAR" => "BAZ"}
    c.output_path = "/tmp4"
    c.priority = 2
    c.runtime_constraints = {"X" => "Y"}
    c.name = "foo3"
    c.description = "bar3"
    c.save!

    assert_nil c.container_uuid
  end

  test "Container request commit" do
    set_user_from_auth :active_trustedclient
    c = ContainerRequest.new
    c.command = ["echo", "foo"]
    c.container_image = "img"
    c.cwd = "/tmp"
    c.environment = {}
    c.mounts = {"BAR" => "FOO"}
    c.output_path = "/tmp"
    c.priority = 1
    c.runtime_constraints = {}
    c.name = "foo"
    c.description = "bar"
    c.save!

    c.reload
    assert_nil c.container_uuid

    c.reload
    c.state = "Committed"
    c.save!

    c.reload

    t = Container.find_by_uuid c.container_uuid
    assert_equal c.command, t.command
    assert_equal c.container_image, t.container_image
    assert_equal c.cwd, t.cwd
  end


end
