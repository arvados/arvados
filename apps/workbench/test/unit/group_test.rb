require 'test_helper'

class GroupTest < ActiveSupport::TestCase
  test "get contents with names" do
    use_token :active
    oi = Group.
      find(api_fixture('groups')['asubproject']['uuid']).
      contents()
    assert_operator(0, :<, oi.count,
                    "Expected to find some items belonging to :active user")
    assert_operator(0, :<, oi.items_available,
                    "Expected contents response to have items_available > 0")
    oi_uuids = oi.collect { |i| i['uuid'] }

    expect_uuid = api_fixture('specimens')['in_asubproject']['uuid']
    assert_includes(oi_uuids, expect_uuid,
                    "Expected '#{expect_uuid}' in asubproject's contents")
  end

  test "can select specific group columns" do
    use_token :admin
    Group.select(["uuid", "name"]).limit(5).each do |user|
      assert_not_nil user.uuid
      assert_not_nil user.name
      assert_nil user.owner_uuid
    end
  end

  test "project editable by its admin" do
    use_token :subproject_admin
    project = Group.find(api_fixture("groups")["asubproject"]["uuid"])
    assert(project.editable?, "project not editable by admin")
  end

  test "project not editable by reader" do
    use_token :project_viewer
    project = Group.find(api_fixture("groups")["aproject"]["uuid"])
    refute(project.editable?, "project editable by reader")
  end
end
