require 'test_helper'

class GroupTest < ActionDispatch::IntegrationTest

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

end
