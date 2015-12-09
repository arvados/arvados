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
    c.output_path = "/tmpout"
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
    c.output_path = "/tmpout"
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
    assert_equal ["echo", "foo"], t.command
    assert_equal "img", t.container_image
    assert_equal "/tmp", t.cwd
    assert_equal({}, t.environment)
    assert_equal({"BAR" => "FOO"}, t.mounts)
    assert_equal "/tmpout", t.output_path
    assert_equal({}, t.runtime_constraints)
    assert_equal 1, t.priority

    c.priority = 0
    c.save!

    c.reload
    t.reload
    assert_equal 0, c.priority
    assert_equal 0, t.priority

  end


  test "Container request max priority" do
    set_user_from_auth :active_trustedclient
    c = ContainerRequest.new
    c.state = "Committed"
    c.container_image = "img"
    c.command = ["foo", "bar"]
    c.output_path = "/tmp"
    c.cwd = "/tmp"
    c.priority = 5
    c.save!

    t = Container.find_by_uuid c.container_uuid
    assert_equal 5, t.priority

    c2 = ContainerRequest.new
    c2.container_image = "img"
    c2.command = ["foo", "bar"]
    c2.output_path = "/tmp"
    c2.cwd = "/tmp"
    c2.priority = 10
    c2.save!

    act_as_system_user do
      c2.state = "Committed"
      c2.container_uuid = c.container_uuid
      c2.save!
    end

    t.reload
    assert_equal 10, t.priority

    c2.reload
    c2.priority = 0
    c2.save!

    t.reload
    assert_equal 5, t.priority

    c.reload
    c.priority = 0
    c.save!

    t.reload
    assert_equal 0, t.priority

  end

  test "Container request finalize" do
    set_user_from_auth :active_trustedclient
    c = ContainerRequest.new
    c.state = "Committed"
    c.container_image = "img"
    c.command = ["foo", "bar"]
    c.output_path = "/tmp"
    c.cwd = "/tmp"
    c.priority = 5
    c.save!

    t = Container.find_by_uuid c.container_uuid
    assert_equal 5, t.priority

    c.state = "Final"
    c.save!

    t.reload
    assert_equal 0, t.priority

  end


  test "Independent containers" do
    set_user_from_auth :active_trustedclient
    c = ContainerRequest.new
    c.state = "Committed"
    c.container_image = "img"
    c.command = ["foo", "bar"]
    c.output_path = "/tmp"
    c.cwd = "/tmp"
    c.priority = 5
    c.save!

    c2 = ContainerRequest.new
    c2.state = "Committed"
    c2.container_image = "img"
    c2.command = ["foo", "bar"]
    c2.output_path = "/tmp"
    c2.cwd = "/tmp"
    c2.priority = 10
    c2.save!

    t = Container.find_by_uuid c.container_uuid
    assert_equal 5, t.priority

    t2 = Container.find_by_uuid c2.container_uuid
    assert_equal 10, t2.priority

    c.priority = 0
    c.save!

    t.reload
    assert_equal 0, t.priority

    t2.reload
    assert_equal 10, t2.priority
  end

  test "Container container cancel" do
    set_user_from_auth :active_trustedclient
    c = ContainerRequest.new
    c.state = "Committed"
    c.container_image = "img"
    c.command = ["foo", "bar"]
    c.output_path = "/tmp"
    c.cwd = "/tmp"
    c.priority = 5
    c.save!

    c.reload
    assert_equal "Committed", c.state

    t = Container.find_by_uuid c.container_uuid
    assert_equal "Queued", t.state

    act_as_system_user do
      t.state = "Cancelled"
      t.save!
    end

    c.reload
    assert_equal "Final", c.state

  end


  test "Container container complete" do
    set_user_from_auth :active_trustedclient
    c = ContainerRequest.new
    c.state = "Committed"
    c.container_image = "img"
    c.command = ["foo", "bar"]
    c.output_path = "/tmp"
    c.cwd = "/tmp"
    c.priority = 5
    c.save!

    c.reload
    assert_equal "Committed", c.state

    t = Container.find_by_uuid c.container_uuid
    assert_equal "Queued", t.state

    act_as_system_user do
      t.state = "Running"
      t.save!
    end

    c.reload
    assert_equal "Committed", c.state

    act_as_system_user do
      t.state = "Complete"
      t.save!
    end

    c.reload
    assert_equal "Final", c.state

  end

end
