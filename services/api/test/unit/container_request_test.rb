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
    cr = ContainerRequest.new
    cr.command = ["echo", "foo"]
    cr.container_image = "img"
    cr.cwd = "/tmp"
    cr.environment = {}
    cr.mounts = {"BAR" => "FOO"}
    cr.output_path = "/tmpout"
    cr.runtime_constraints = {}
    cr.name = "foo"
    cr.description = "bar"
    cr.save!

    assert_nil cr.container_uuid
    assert_nil cr.priority

    check_bogus_states cr

    cr.reload
    cr.command = ["echo", "foo3"]
    cr.container_image = "img3"
    cr.cwd = "/tmp3"
    cr.environment = {"BUP" => "BOP"}
    cr.mounts = {"BAR" => "BAZ"}
    cr.output_path = "/tmp4"
    cr.priority = 2
    cr.runtime_constraints = {"X" => "Y"}
    cr.name = "foo3"
    cr.description = "bar3"
    cr.save!

    assert_nil cr.container_uuid
  end

  test "Container request priority must be non-nil" do
    set_user_from_auth :active_trustedclient
    cr = ContainerRequest.new
    cr.command = ["echo", "foo"]
    cr.container_image = "img"
    cr.cwd = "/tmp"
    cr.environment = {}
    cr.mounts = {"BAR" => "FOO"}
    cr.output_path = "/tmpout"
    cr.runtime_constraints = {}
    cr.name = "foo"
    cr.description = "bar"
    cr.save!

    cr.reload
    cr.state = "Committed"
    assert_raises(ActiveRecord::RecordInvalid) do
      cr.save!
    end
  end

  test "Container request commit" do
    set_user_from_auth :active_trustedclient
    cr = ContainerRequest.new
    cr.command = ["echo", "foo"]
    cr.container_image = "img"
    cr.cwd = "/tmp"
    cr.environment = {}
    cr.mounts = {"BAR" => "FOO"}
    cr.output_path = "/tmpout"
    cr.priority = 1
    cr.runtime_constraints = {}
    cr.name = "foo"
    cr.description = "bar"
    cr.save!

    cr.reload
    assert_nil cr.container_uuid

    cr.reload
    cr.state = "Committed"
    cr.save!

    cr.reload

    c = Container.find_by_uuid cr.container_uuid
    assert_equal ["echo", "foo"], c.command
    assert_equal "img", c.container_image
    assert_equal "/tmp", c.cwd
    assert_equal({}, c.environment)
    assert_equal({"BAR" => "FOO"}, c.mounts)
    assert_equal "/tmpout", c.output_path
    assert_equal({}, c.runtime_constraints)
    assert_equal 1, c.priority

    assert_raises(ActiveRecord::RecordInvalid) do
      cr.priority = nil
      cr.save!
    end

    cr.priority = 0
    cr.save!

    cr.reload
    c.reload
    assert_equal 0, cr.priority
    assert_equal 0, c.priority

  end


  test "Container request max priority" do
    set_user_from_auth :active_trustedclient
    cr = ContainerRequest.new
    cr.state = "Committed"
    cr.container_image = "img"
    cr.command = ["foo", "bar"]
    cr.output_path = "/tmp"
    cr.cwd = "/tmp"
    cr.priority = 5
    cr.save!

    c = Container.find_by_uuid cr.container_uuid
    assert_equal 5, c.priority

    cr2 = ContainerRequest.new
    cr2.container_image = "img"
    cr2.command = ["foo", "bar"]
    cr2.output_path = "/tmp"
    cr2.cwd = "/tmp"
    cr2.priority = 10
    cr2.save!

    act_as_system_user do
      cr2.state = "Committed"
      cr2.container_uuid = cr.container_uuid
      cr2.save!
    end

    c.reload
    assert_equal 10, c.priority

    cr2.reload
    cr2.priority = 0
    cr2.save!

    c.reload
    assert_equal 5, c.priority

    cr.reload
    cr.priority = 0
    cr.save!

    c.reload
    assert_equal 0, c.priority

  end


  test "Independent container requests" do
    set_user_from_auth :active_trustedclient
    cr = ContainerRequest.new
    cr.state = "Committed"
    cr.container_image = "img"
    cr.command = ["foo", "bar"]
    cr.output_path = "/tmp"
    cr.cwd = "/tmp"
    cr.priority = 5
    cr.save!

    cr2 = ContainerRequest.new
    cr2.state = "Committed"
    cr2.container_image = "img"
    cr2.command = ["foo", "bar"]
    cr2.output_path = "/tmp"
    cr2.cwd = "/tmp"
    cr2.priority = 10
    cr2.save!

    c = Container.find_by_uuid cr.container_uuid
    assert_equal 5, c.priority

    c2 = Container.find_by_uuid cr2.container_uuid
    assert_equal 10, c2.priority

    cr.priority = 0
    cr.save!

    c.reload
    assert_equal 0, c.priority

    c2.reload
    assert_equal 10, c2.priority
  end


  test "Container cancelled finalizes request" do
    set_user_from_auth :active_trustedclient
    cr = ContainerRequest.new
    cr.state = "Committed"
    cr.container_image = "img"
    cr.command = ["foo", "bar"]
    cr.output_path = "/tmp"
    cr.cwd = "/tmp"
    cr.priority = 5
    cr.save!

    cr.reload
    assert_equal "Committed", cr.state

    c = Container.find_by_uuid cr.container_uuid
    assert_equal "Queued", c.state

    act_as_system_user do
      c.state = "Cancelled"
      c.save!
    end

    cr.reload
    assert_equal "Final", cr.state

  end


  test "Container complete finalizes request" do
    set_user_from_auth :active_trustedclient
    cr = ContainerRequest.new
    cr.state = "Committed"
    cr.container_image = "img"
    cr.command = ["foo", "bar"]
    cr.output_path = "/tmp"
    cr.cwd = "/tmp"
    cr.priority = 5
    cr.save!

    cr.reload
    assert_equal "Committed", cr.state

    c = Container.find_by_uuid cr.container_uuid
    assert_equal "Queued", c.state

    act_as_system_user do
      c.state = "Running"
      c.save!
    end

    cr.reload
    assert_equal "Committed", cr.state

    act_as_system_user do
      c.state = "Complete"
      c.save!
    end

    cr.reload
    assert_equal "Final", cr.state

  end

  test "Container makes container request, then is cancelled" do
    set_user_from_auth :active_trustedclient
    cr = ContainerRequest.new
    cr.state = "Committed"
    cr.container_image = "img"
    cr.command = ["foo", "bar"]
    cr.output_path = "/tmp"
    cr.cwd = "/tmp"
    cr.priority = 5
    cr.save!

    c = Container.find_by_uuid cr.container_uuid
    assert_equal 5, c.priority

    cr2 = ContainerRequest.new
    cr2.state = "Committed"
    cr2.container_image = "img"
    cr2.command = ["foo", "bar"]
    cr2.output_path = "/tmp"
    cr2.cwd = "/tmp"
    cr2.priority = 10
    cr2.requesting_container_uuid = c.uuid
    cr2.save!

    c2 = Container.find_by_uuid cr2.container_uuid
    assert_equal 10, c2.priority

    act_as_system_user do
      c.state = "Cancelled"
      c.save!
    end

    cr.reload
    assert_equal "Final", cr.state

    cr2.reload
    assert_equal 0, cr2.priority

    c2.reload
    assert_equal 0, c2.priority
  end

end
