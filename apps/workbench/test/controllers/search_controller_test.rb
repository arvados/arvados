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
end
