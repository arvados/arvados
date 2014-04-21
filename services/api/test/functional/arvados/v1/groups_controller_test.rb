require 'test_helper'

class Arvados::V1::GroupsControllerTest < ActionController::TestCase

  test "attempt to delete group without read or write access" do
    authorize_with :active
    post :destroy, id: groups(:empty_lonely_group).uuid
    assert_response 404
  end

  test "attempt to delete group without write access" do
    authorize_with :active
    post :destroy, id: groups(:all_users).uuid
    assert_response 403
  end

  test "get list of folders" do
    authorize_with :active
    get :index, filters: [['group_class', '=', 'folder']], format: :json
    assert_response :success
    group_uuids = []
    jresponse['items'].each do |group|
      assert_equal 'folder', group['group_class']
      group_uuids << group['uuid']
    end
    assert_not_nil group_uuids.index groups(:afolder).uuid
    assert_not_nil group_uuids.index groups(:asubfolder).uuid
  end

end
