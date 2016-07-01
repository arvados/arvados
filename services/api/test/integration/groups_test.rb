require 'test_helper'

class GroupsTest < ActionDispatch::IntegrationTest
  [[], ['replication_confirmed']].each do |orders|
    test "results are consistent when provided orders #{orders} is incomplete" do
      last = nil
      (0..20).each do
        get '/arvados/v1/groups/contents', {
          id: groups(:aproject).uuid,
          filters: [["uuid", "is_a", "arvados#collection"]].to_json,
          orders: orders.to_json,
          format: :json,
        }, auth(:active)
        assert_response :success
        if last.nil?
          last = json_response['items']
        else
          assert_equal last, json_response['items']
        end
      end
    end
  end

  test "get all pages of group-owned objects" do
    limit = 5
    offset = 0
    items_available = nil
    uuid_received = {}
    owner_received = {}
    while true
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

  [
    [['owner_uuid', '!=', 'zzzzz-tpzed-xurymjxw79nv3jz'], 200,
        'zzzzz-d1hrv-subprojpipeline', 'zzzzz-d1hrv-1xfj6xkicf2muk2'],
    [["pipeline_instances.state", "not in", ["Complete", "Failed"]], 200,
        'zzzzz-d1hrv-1xfj6xkicf2muk2', 'zzzzz-d1hrv-i3e77t9z5y8j9cc'],
    [['container_requests.requesting_container_uuid', '=', nil], 200,
        'zzzzz-xvhdp-cr4queuedcontnr', 'zzzzz-xvhdp-cr4requestercn2'],
    [['container_requests.no_such_column', '=', nil], 422],
    [['container_requests.', '=', nil], 422],
    [['.requesting_container_uuid', '=', nil], 422],
    [['no_such_table.uuid', '!=', 'zzzzz-tpzed-xurymjxw79nv3jz'], 422],
  ].each do |filter, expect_code, expect_uuid, not_expect_uuid|
    test "get contents with '#{filter}' filter" do
      get "/arvados/v1/groups/contents", {
        :filters => [filter].to_json
      }, auth(:active)
      assert_response expect_code
      if expect_code == 200
        assert_not_empty json_response['items']
        item_uuids = json_response['items'].collect {|item| item['uuid']}
        assert_includes(item_uuids, expect_uuid)
        assert_not_includes(item_uuids, not_expect_uuid)
      end
    end
  end
end
