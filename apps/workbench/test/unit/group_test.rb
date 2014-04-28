require 'test_helper'

class GroupTest < ActiveSupport::TestCase
  test "get owned_items with names" do
    use_token :active
    oi = Group.
      find(api_fixture('groups')['asubfolder']['uuid']).
      owned_items(include_linked: true)
    assert_operator(0, :<, oi.count,
                    "Expected to find some items belonging to :active user")
    assert_operator(0, :<, oi.items_available,
                    "Expected owned_items response to have items_available > 0")
    assert_operator(0, :<, oi.result_links.count,
                    "Expected to receive name links with owned_items response")
    oi_uuids = oi.collect { |i| i['uuid'] }

    expect_uuid = api_fixture('specimens')['in_asubfolder']['uuid']
    assert_includes(oi_uuids, expect_uuid,
                    "Expected '#{expect_uuid}' in asubfolder's owned_items")

    expect_uuid = api_fixture('specimens')['in_afolder_linked_from_asubfolder']['uuid']
    expect_name = api_fixture('links')['specimen_is_in_two_folders']['name']
    assert_includes(oi_uuids, expect_uuid,
                    "Expected '#{expect_uuid}' in asubfolder's owned_items")
    assert_equal(expect_name, oi.name_for(expect_uuid),
                 "Expected name_for '#{expect_uuid}' to be '#{expect_name}'")
  end
end
