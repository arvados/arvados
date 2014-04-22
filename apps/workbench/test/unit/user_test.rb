require 'test_helper'

class UserTest < ActiveSupport::TestCase
  test "get owned_items" do
    use_token :active
    oi = User.find(api_fixture('users')['active']['uuid']).owned_items
    assert_operator 0, :<, oi.count
    assert_operator 0, :<, oi.items_available
    oi_uuids = oi.collect { |i| i['uuid'] }
    expect = api_fixture('specimens')['owned_by_active_user']['uuid']
    assert_includes(oi_uuids, expect,
                    "Expected active user's owned_items to include #{expect}")
  end
end
