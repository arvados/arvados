require 'test_helper'

class Arvados::V1::GroupsControllerTest < ActionController::TestCase

  test "attempt to delete group without write access" do
    authorize_with :active
    post :destroy, id: groups(:public).uuid
    assert_response 403
  end

end
