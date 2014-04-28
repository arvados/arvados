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
    json_response['items'].each do |group|
      assert_equal 'folder', group['group_class']
      group_uuids << group['uuid']
    end
    assert_includes group_uuids, groups(:afolder).uuid
    assert_includes group_uuids, groups(:asubfolder).uuid
    assert_not_includes group_uuids, groups(:system_group).uuid
    assert_not_includes group_uuids, groups(:private).uuid
  end

  test "get list of groups that are not folders" do
    authorize_with :active
    get :index, filters: [['group_class', '=', nil]], format: :json
    assert_response :success
    group_uuids = []
    json_response['items'].each do |group|
      assert_equal nil, group['group_class']
      group_uuids << group['uuid']
    end
    assert_not_includes group_uuids, groups(:afolder).uuid
    assert_not_includes group_uuids, groups(:asubfolder).uuid
    assert_includes group_uuids, groups(:private).uuid
  end

  test "get list of groups with bogus group_class" do
    authorize_with :active
    get :index, {
      filters: [['group_class', '=', 'nogrouphasthislittleclass']],
      format: :json,
    }
    assert_response :success
    assert_equal [], json_response['items']
    assert_equal 0, json_response['items_available']
  end

  test 'get group-owned objects' do
    authorize_with :active
    get :owned_items, {
      id: groups(:afolder).uuid,
      format: :json,
    }
    assert_response :success
    assert_operator 2, :<=, json_response['items_available']
    assert_operator 2, :<=, json_response['items'].count
  end

  test 'get group-owned objects with limit' do
    authorize_with :active
    get :owned_items, {
      id: groups(:afolder).uuid,
      limit: 1,
      format: :json,
    }
    assert_response :success
    assert_operator 1, :<, json_response['items_available']
    assert_equal 1, json_response['items'].count
  end

  test 'get group-owned objects with limit and offset' do
    authorize_with :active
    get :owned_items, {
      id: groups(:afolder).uuid,
      limit: 1,
      offset: 12345,
      format: :json,
    }
    assert_response :success
    assert_operator 1, :<, json_response['items_available']
    assert_equal 0, json_response['items'].count
  end

  test 'get group-owned objects with additional filter matching nothing' do
    authorize_with :active
    get :owned_items, {
      id: groups(:afolder).uuid,
      filters: [['uuid', 'in', ['foo_not_a_uuid','bar_not_a_uuid']]],
      format: :json,
    }
    assert_response :success
    assert_equal [], json_response['items']
    assert_equal 0, json_response['items_available']
  end

  test 'get group-owned objects without include_linked' do
    unexpected_uuid = specimens(:in_afolder_linked_from_asubfolder).uuid
    authorize_with :active
    get :owned_items, {
      id: groups(:asubfolder).uuid,
      format: :json,
    }
    assert_response :success
    uuids = json_response['items'].collect { |i| i['uuid'] }
    assert_equal nil, uuids.index(unexpected_uuid)
  end

  test 'get group-owned objects with include_linked' do
    expected_uuid = specimens(:in_afolder_linked_from_asubfolder).uuid
    authorize_with :active
    get :owned_items, {
      id: groups(:asubfolder).uuid,
      include_linked: true,
      format: :json,
    }
    assert_response :success
    uuids = json_response['items'].collect { |i| i['uuid'] }
    assert_includes uuids, expected_uuid, "Did not get #{expected_uuid}"

    expected_name = links(:specimen_is_in_two_folders).name
    found_specimen_name = false
    assert(json_response['links'].any?,
           "Expected a non-empty array of links in response")
    json_response['links'].each do |link|
      if link['head_uuid'] == expected_uuid
        if link['name'] == expected_name
          found_specimen_name = true
        end
      end
    end
    assert(found_specimen_name,
           "Expected to find name '#{expected_name}' in response")
  end
end
