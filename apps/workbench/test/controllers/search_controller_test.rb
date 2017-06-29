# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'test_helper'

class SearchControllerTest < ActionController::TestCase
  # These tests don't do state-changing API calls. Save some time by
  # skipping the database reset.
  reset_api_fixtures :after_each_test, false
  reset_api_fixtures :after_suite, true

  include Rails.application.routes.url_helpers

  test 'Get search dialog' do
    xhr :get, :choose, {
      format: :js,
      title: 'Search',
      action_name: 'Show',
      action_href: url_for(host: 'localhost', controller: :actions, action: :show),
      action_data: {}.to_json,
    }, session_for(:active)
    assert_response :success
  end

  test 'Get search results for all projects' do
    xhr :get, :choose, {
      format: :json,
      partial: true,
    }, session_for(:active)
    assert_response :success
    assert_not_empty(json_response['content'],
                     'search results for all projects should not be empty')
  end

  test 'Get search results for empty project' do
    xhr :get, :choose, {
      format: :json,
      partial: true,
      project_uuid: api_fixture('groups')['empty_project']['uuid'],
    }, session_for(:active)
    assert_response :success
    assert_empty(json_response['content'],
                 'search results for empty project should be empty')
  end

  test 'search results for aproject and verify recursive contents' do
    xhr :get, :choose, {
      format: :json,
      partial: true,
      project_uuid: api_fixture('groups')['aproject']['uuid'],
    }, session_for(:active)
    assert_response :success
    assert_not_empty(json_response['content'],
                 'search results for aproject should not be empty')
    items = []
    json_response['content'].scan /<div[^>]+>/ do |div_tag|
      div_tag.scan(/\ data-object-uuid=\"(.*?)\"/).each do |uuid,|
        items << uuid
      end
    end

    assert_includes(items, api_fixture('collections')['collection_to_move_around_in_aproject']['uuid'])
    assert_includes(items, api_fixture('groups')['asubproject']['uuid'])
    assert_includes(items, api_fixture('collections')['baz_collection_name_in_asubproject']['uuid'])
    assert_includes(items,
      api_fixture('groups')['subproject_in_asubproject_with_same_name_as_one_in_active_user_home']['uuid'])
  end
end
