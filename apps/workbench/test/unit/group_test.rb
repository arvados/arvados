require 'test_helper'

class GroupTest < ActiveSupport::TestCase
  test "get contents with names" do
    use_token :active
    oi = Group.
      find(api_fixture('groups')['asubproject']['uuid']).
      contents(include_linked: true)
    assert_operator(0, :<, oi.count,
                    "Expected to find some items belonging to :active user")
    assert_operator(0, :<, oi.items_available,
                    "Expected contents response to have items_available > 0")
    assert_operator(0, :<, oi.result_links.count,
                    "Expected to receive name links with contents response")
    oi_uuids = oi.collect { |i| i['uuid'] }

    expect_uuid = api_fixture('specimens')['in_asubproject']['uuid']
    assert_includes(oi_uuids, expect_uuid,
                    "Expected '#{expect_uuid}' in asubproject's contents")

    expect_uuid = api_fixture('specimens')['in_aproject_linked_from_asubproject']['uuid']
    expect_name = api_fixture('links')['specimen_is_in_two_projects']['name']
    assert_includes(oi_uuids, expect_uuid,
                    "Expected '#{expect_uuid}' in asubproject's contents")
    assert_equal(expect_name, oi.name_for(expect_uuid),
                 "Expected name_for '#{expect_uuid}' to be '#{expect_name}'")
  end

  test "can select specific group columns" do
    use_token :admin
    Group.select(["uuid", "name"]).limit(5).each do |user|
      assert_not_nil user.uuid
      assert_not_nil user.name
      assert_nil user.owner_uuid
    end
  end
end
