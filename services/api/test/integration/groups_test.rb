require 'test_helper'

class GroupsTest < ActionDispatch::IntegrationTest

  test "get all pages of group-owned objects" do
    limit = 5
    offset = 0
    items_available = nil
    uuid_received = {}
    owner_received = {}
    while true
      @json_response = nil

      get "/arvados/v1/groups/contents", {
        id: groups(:aproject).uuid,
        limit: limit,
        offset: offset,
        format: :json,
      }, auth(:active)

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
        assert_equal groups(:aproject).uuid, item['owner_uuid']
      end
      break if offset >= items_available
    end
  end

  [
    ['Collection_', true],            # collections and pipelines templates
    ['hash', true],                   # pipeline templates
    ['fa7aeb5140e2848d39b', false],   # script_parameter of pipeline instances
    ['fa7aeb5140e2848d39b:*', true],  # script_parameter of pipeline instances
    ['project pipeline', true],       # finds "Completed pipeline in A Project"
    ['project pipeli:*', true],       # finds "Completed pipeline in A Project"
    ['proje pipeli:*', false],        # first word is incomplete, so no prefix match
    ['no-such-thing', false],         # script_parameter of pipeline instances
  ].each do |search_filter, expect_results|
    test "full text search of group-owned objects for #{search_filter}" do
      get "/arvados/v1/groups/contents", {
        id: groups(:aproject).uuid,
        limit: 5,
        :filters => [['any', '@@', search_filter]].to_json
      }, auth(:active)
      assert_response :success
      if expect_results
        refute_empty json_response['items']
        json_response['items'].each do |item|
          assert item['uuid']
          assert_equal groups(:aproject).uuid, item['owner_uuid']
        end
      else
        assert_empty json_response['items']
      end
    end
  end

  test "full text search is not supported for individual columns" do
    get "/arvados/v1/groups/contents", {
      :filters => [['name', '@@', 'Private']].to_json
    }, auth(:active)
    assert_response 422
  end
end
