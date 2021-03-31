# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'test_helper'

class GroupsTest < ActionDispatch::IntegrationTest
  [[], ['replication_confirmed']].each do |orders|
    test "results are consistent when provided orders #{orders} is incomplete" do
      last = nil
      (0..20).each do
        get '/arvados/v1/groups/contents',
          params: {
            id: groups(:aproject).uuid,
            filters: [["uuid", "is_a", "arvados#collection"]].to_json,
            orders: orders.to_json,
            format: :json,
          },
          headers: auth(:active)
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
      get "/arvados/v1/groups/contents",
        params: {
          id: groups(:aproject).uuid,
          limit: limit,
          offset: offset,
          format: :json,
        },
        headers: auth(:active)

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
      get "/arvados/v1/groups/contents",
        params: {
          id: groups(:aproject).uuid,
          limit: 5,
          :filters => [['any', '@@', search_filter]].to_json
        },
        headers: auth(:active)
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
    get "/arvados/v1/groups/contents",
      params: {
        :filters => [['name', '@@', 'Private']].to_json
      },
      headers: auth(:active)
    assert_response 422
  end

  test "group contents with include trash collections" do
    get "/arvados/v1/groups/contents",
      params: {
        include_trash: "true",
        filters: [["uuid", "is_a", "arvados#collection"]].to_json,
        limit: 1000
      },
      headers: auth(:active)
    assert_response 200

    coll_uuids = []
    json_response['items'].each { |c| coll_uuids << c['uuid'] }
    assert_includes coll_uuids, collections(:foo_collection_in_aproject).uuid
    assert_includes coll_uuids, collections(:expired_collection).uuid
  end

  test "group contents without trash collections" do
    get "/arvados/v1/groups/contents",
      params: {
        filters: [["uuid", "is_a", "arvados#collection"]].to_json,
        limit: 1000
      },
      headers: auth(:active)
    assert_response 200

    coll_uuids = []
    json_response['items'].each { |c| coll_uuids << c['uuid'] }
    assert_includes coll_uuids, collections(:foo_collection_in_aproject).uuid
    assert_not_includes coll_uuids, collections(:expired_collection).uuid
  end

  test "unsharing a project results in hiding it from previously shared user" do
    # remove sharing link for project
    delete "/arvados/v1/links/#{links(:share_starred_project_with_project_viewer).uuid}", headers: auth(:admin)
    assert_response 200

    # verify that the user can no longer see the project
    get "/arvados/v1/groups",
      params: {
        filters: [['group_class', '=', 'project']].to_json,
        limit: 1000
      }, headers: auth(:project_viewer)
    assert_response 200
    found_projects = {}
    json_response['items'].each do |g|
      found_projects[g['uuid']] = g
    end
    assert_equal false, found_projects.include?(groups(:starred_and_shared_active_user_project).uuid)

    # share the project
    post "/arvados/v1/links", params: {
      link: {
        link_class: "permission",
        name: "can_read",
        head_uuid: groups(:starred_and_shared_active_user_project).uuid,
        tail_uuid: users(:project_viewer).uuid,
      }
    }, headers: auth(:system_user)
    assert_response 200
    assert_equal 'permission', json_response['link_class']

    # verify that project_viewer user can now see shared project again
    get "/arvados/v1/groups", params: {
      filters: [['group_class', '=', 'project']].to_json,
      limit: 1000
    }, headers: auth(:project_viewer)
    assert_response 200
    found_projects = {}
    json_response['items'].each do |g|
      found_projects[g['uuid']] = g
    end
    assert_equal true, found_projects.include?(groups(:starred_and_shared_active_user_project).uuid)
  end

  test 'count none works with offset' do
    first_results = nil
    (0..10).each do |offset|
      get "/arvados/v1/groups/contents", params: {
        id: groups(:aproject).uuid,
        offset: offset,
        format: :json,
        order: :uuid,
        count: :none,
      }, headers: auth(:active)
      assert_response :success
      assert_nil json_response['items_available']
      if first_results.nil?
        first_results = json_response['items']
      else
        assert_equal first_results[offset]['uuid'], json_response['items'][0]['uuid']
      end
    end
  end
end

class NonTransactionalGroupsTest < ActionDispatch::IntegrationTest
  # Transactional tests are disabled to be able to test the concurrent
  # asynchronous permissions update feature.
  # This is needed because nested transactions share the connection pool, so
  # one thread is locked while trying to talk to the database, until the other
  # one finishes.
  self.use_transactional_tests = false

  teardown do
    # Explicitly reset the database after each test.
    post '/database/reset', params: {}, headers: auth(:admin)
    assert_response :success
  end

  test "create request with async=true does not defer permissions update" do
    Rails.configuration.API.AsyncPermissionsUpdateInterval = 1 # second
    name = "Random group #{rand(1000)}"
    assert_equal nil, Group.find_by_name(name)

    # Following the implementation of incremental permission updates
    # (#16007) the async flag is now a no-op.  Permission changes are
    # visible immediately.

    # Trigger the asynchronous permission update by using async=true parameter.
    post "/arvados/v1/groups",
      params: {
        group: {
          name: name,
          group_class: "project"
        },
        async: true
      },
      headers: auth(:active)
    assert_response 202

    # The group exists in the database
    assert_not_nil Group.find_by_name(name)
    get "/arvados/v1/groups",
      params: {
        filters: [["name", "=", name]].to_json,
        limit: 10
      },
      headers: auth(:active)
    assert_response 200
    assert_equal 1, json_response['items_available']

    # Wait a bit and try again.
    sleep(1)
    get "/arvados/v1/groups",
      params: {
        filters: [["name", "=", name]].to_json,
        limit: 10
      },
      headers: auth(:active)
    assert_response 200
    assert_equal 1, json_response['items_available']
  end
end
