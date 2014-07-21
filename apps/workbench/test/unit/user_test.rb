require 'test_helper'

class UserTest < ActiveSupport::TestCase
  test "can select specific user columns" do
    use_token :admin
    User.select(["uuid", "is_active"]).limit(5).each do |user|
      assert_not_nil user.uuid
      assert_not_nil user.is_active
      assert_nil user.first_name
    end
  end
end
