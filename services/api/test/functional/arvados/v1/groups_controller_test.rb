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

  test "get list of projects" do
    authorize_with :active
    get :index, filters: [['group_class', '=', 'project']], format: :json
    assert_response :success
    group_uuids = []
    json_response['items'].each do |group|
      assert_equal 'project', group['group_class']
      group_uuids << group['uuid']
    end
    assert_includes group_uuids, groups(:aproject).uuid
    assert_includes group_uuids, groups(:asubproject).uuid
    assert_not_includes group_uuids, groups(:system_group).uuid
    assert_not_includes group_uuids, groups(:private).uuid
  end

  test "get list of groups that are not projects" do
    authorize_with :active
    get :index, filters: [['group_class', '=', nil]], format: :json
    assert_response :success
    group_uuids = []
    json_response['items'].each do |group|
      assert_equal nil, group['group_class']
      group_uuids << group['uuid']
    end
    assert_not_includes group_uuids, groups(:aproject).uuid
    assert_not_includes group_uuids, groups(:asubproject).uuid
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

  def check_project_contents_response
    assert_response :success
    assert_operator 2, :<=, json_response['items_available']
    assert_operator 2, :<=, json_response['items'].count
    kinds = json_response['items'].collect { |i| i['kind'] }.uniq
    expect_kinds = %w'arvados#group arvados#specimen arvados#pipelineTemplate arvados#job'
    assert_equal expect_kinds, (expect_kinds & kinds)

    json_response['items'].each do |i|
      if i['kind'] == 'arvados#group'
        assert(i['group_class'] == 'project',
               "group#contents returned a non-project group")
      end
    end
  end

  test 'get group-owned objects' do
    authorize_with :active
    get :contents, {
      id: groups(:aproject).uuid,
      format: :json,
      include_linked: true,
    }
    check_project_contents_response
  end

  test "user with project read permission can see project objects" do
    authorize_with :project_viewer
    get :contents, {
      id: groups(:aproject).uuid,
      format: :json,
      include_linked: true,
    }
    check_project_contents_response
  end

  [false, true].each do |include_linked|
    test "list objects across projects, include_linked=#{include_linked}" do
      authorize_with :project_viewer
      get :contents, {
        format: :json,
        include_linked: include_linked,
        filters: [['uuid', 'is_a', 'arvados#specimen']]
      }
      assert_response :success
      found_uuids = json_response['items'].collect { |i| i['uuid'] }
      [[:in_aproject, true],
       [:in_asubproject, true],
       [:owned_by_private_group, false]].each do |specimen_fixture, should_find|
        if should_find
          assert_includes found_uuids, specimens(specimen_fixture).uuid, "did not find specimen fixture '#{specimen_fixture}'"
        else
          refute_includes found_uuids, specimens(specimen_fixture).uuid, "found specimen fixture '#{specimen_fixture}'"
        end
      end
    end
  end

  [false, true].each do |include_linked|
    test "list objects in home project, include_linked=#{include_linked}" do
      authorize_with :active
      get :contents, {
        format: :json,
        id: users(:active).uuid,
        include_linked: include_linked,
      }
      assert_response :success
      found_uuids = json_response['items'].collect { |i| i['uuid'] }
      if include_linked
        assert_includes found_uuids, collections(:empty).uuid, "empty collection did not appear in home project"
      end
      assert_includes found_uuids, specimens(:owned_by_active_user).uuid, "specimen did not appear in home project"
      refute_includes found_uuids, specimens(:in_asubproject).uuid, "specimen appeared unexpectedly in home project"
    end
  end

  test "user with project read permission can see project collections" do
    authorize_with :project_viewer
    get :contents, {
      id: groups(:asubproject).uuid,
      format: :json,
      include_linked: true,
    }
    ids = json_response['items'].map { |item| item["uuid"] }
    assert_includes ids, collections(:baz_file).uuid
  end

  test 'list objects across multiple projects' do
    authorize_with :project_viewer
    get :contents, {
      format: :json,
      include_linked: false,
      filters: [['uuid', 'is_a', 'arvados#specimen']]
    }
    assert_response :success
    found_uuids = json_response['items'].collect { |i| i['uuid'] }
    [[:in_aproject, true],
     [:in_asubproject, true],
     [:owned_by_private_group, false]].each do |specimen_fixture, should_find|
      if should_find
        assert_includes found_uuids, specimens(specimen_fixture).uuid, "did not find specimen fixture '#{specimen_fixture}'"
      else
        refute_includes found_uuids, specimens(specimen_fixture).uuid, "found specimen fixture '#{specimen_fixture}'"
      end
    end
  end

  # Even though the project_viewer tests go through other controllers,
  # I'm putting them here so they're easy to find alongside the other
  # project tests.
  def check_new_project_link_fails(link_attrs)
    @controller = Arvados::V1::LinksController.new
    post :create, link: {
      link_class: "permission",
      name: "can_read",
      head_uuid: groups(:aproject).uuid,
    }.merge(link_attrs)
    assert_includes(403..422, response.status)
  end

  test "user with project read permission can't add users to it" do
    authorize_with :project_viewer
    check_new_project_link_fails(tail_uuid: users(:spectator).uuid)
  end

  test "user with project read permission can't add items to it" do
    authorize_with :project_viewer
    check_new_project_link_fails(tail_uuid: collections(:baz_file).uuid)
  end

  test "user with project read permission can't rename items in it" do
    authorize_with :project_viewer
    @controller = Arvados::V1::LinksController.new
    post :update, {
      id: links(:job_name_in_aproject).uuid,
      link: {name: "Denied test name"},
    }
    assert_includes(403..404, response.status)
  end

  test "user with project read permission can't remove items from it" do
    @controller = Arvados::V1::PipelineTemplatesController.new
    authorize_with :project_viewer
    post :update, {
      id: links(:template_name_in_aproject).head_uuid,
      pipeline_template: {
        owner_uuid: users(:project_viewer).uuid,
      }
    }
    assert_response 403
  end

  test "user with project read permission can't delete it" do
    authorize_with :project_viewer
    post :destroy, {id: groups(:aproject).uuid}
    assert_response 403
  end

  test 'get group-owned objects with limit' do
    authorize_with :active
    get :contents, {
      id: groups(:aproject).uuid,
      limit: 1,
      format: :json,
    }
    assert_response :success
    assert_operator 1, :<, json_response['items_available']
    assert_equal 1, json_response['items'].count
  end

  test 'get group-owned objects with limit and offset' do
    authorize_with :active
    get :contents, {
      id: groups(:aproject).uuid,
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
    get :contents, {
      id: groups(:aproject).uuid,
      filters: [['uuid', 'in', ['foo_not_a_uuid','bar_not_a_uuid']]],
      format: :json,
    }
    assert_response :success
    assert_equal [], json_response['items']
    assert_equal 0, json_response['items_available']
  end

  test 'get group-owned objects without include_linked' do
    unexpected_uuid = specimens(:in_aproject_linked_from_asubproject).uuid
    authorize_with :active
    get :contents, {
      id: groups(:asubproject).uuid,
      format: :json,
    }
    assert_response :success
    uuids = json_response['items'].collect { |i| i['uuid'] }
    assert_equal nil, uuids.index(unexpected_uuid)
  end

  test 'get group-owned objects with include_linked' do
    expected_uuid = specimens(:in_asubproject).uuid
    authorize_with :active
    get :contents, {
      id: groups(:asubproject).uuid,
      include_linked: true,
      format: :json,
    }
    assert_response :success
    uuids = json_response['items'].collect { |i| i['uuid'] }
    assert_includes uuids, expected_uuid, "Did not get #{expected_uuid}"

    expected_name = specimens(:in_asubproject).name
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

  [false, true].each do |inc_ind|
    test "get all pages of group-owned #{'and -linked ' if inc_ind}objects" do
      authorize_with :active
      limit = 5
      offset = 0
      items_available = nil
      uuid_received = {}
      owner_received = {}
      while true
        # Behaving badly here, using the same controller multiple
        # times within a test.
        @json_response = nil
        get :contents, {
          id: groups(:aproject).uuid,
          include_linked: inc_ind,
          limit: limit,
          offset: offset,
          format: :json,
        }
        assert_response :success
        assert_operator(0, :<, json_response['items'].count,
                        "items_available=#{items_available} but received 0 "\
                        "items with offset=#{offset}")
        items_available ||= json_response['items_available']
        assert_equal(items_available, json_response['items_available'],
                     "items_available changed between page #{offset/limit} "\
                     "and page #{1+offset/limit}")
        json_response['items'].each do |item|
          uuid = item['uuid']
          assert_equal(nil, uuid_received[uuid],
                       "Received '#{uuid}' again on page #{1+offset/limit}")
          uuid_received[uuid] = true
          owner_received[item['owner_uuid']] = true
          offset += 1
          if not inc_ind
            assert_equal groups(:aproject).uuid, item['owner_uuid']
          end
        end
        break if offset >= items_available
      end
      if inc_ind
        assert_operator 0, :<, (json_response.keys - [users(:active).uuid]).count,
        "Set include_linked=true but did not receive any non-owned items"
      end
    end
  end

  %w(offset limit).each do |arg|
    ['foo', '', '1234five', '0x10', '-8'].each do |val|
      test "Raise error on bogus #{arg} parameter #{val.inspect}" do
        authorize_with :active
        get :contents, {
          :id => groups(:aproject).uuid,
          :format => :json,
          arg => val,
        }
        assert_response 422
      end
    end
  end

  test 'get writable_by list for owned group' do
    authorize_with :active
    get :show, {
      id: groups(:aproject).uuid,
      format: :json
    }
    assert_response :success
    assert_not_nil(json_response['writable_by'],
                   "Should receive uuid list in 'writable_by' field")
    assert_includes(json_response['writable_by'], users(:active).uuid,
                    "owner should be included in writable_by list")
  end

  test 'no writable_by list for group with read-only access' do
    authorize_with :rominiadmin
    get :show, {
      id: groups(:testusergroup_admins).uuid,
      format: :json
    }
    assert_response :success
    assert_nil(json_response['writable_by'],
               "Should not receive uuid list in 'writable_by' field")
  end

  test 'get writable_by list by admin user' do
    authorize_with :admin
    get :show, {
      id: groups(:testusergroup_admins).uuid,
      format: :json
    }
    assert_response :success
    assert_not_nil(json_response['writable_by'],
                   "Should receive uuid list in 'writable_by' field")
    assert_includes(json_response['writable_by'],
                    users(:admin).uuid,
                    "Current user should be included in 'writable_by' field")
  end
end
