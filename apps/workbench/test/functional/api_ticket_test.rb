# This test exercises behavior in ApplicationController.

require 'test_helper'

class ApiTicketTest < ActionController::TestCase
  test "api_ticket temporarily overrides api_token" do
    # ApiClientAuthorizationsController provides the easiest way to get
    # different results across different users.
    @controller = ApiClientAuthorizationsController.new
    def get_page_with(*get_args)
      get :index, *get_args
      assert_response(:success, "failed to get index with #{get_args}")
      JSON.parse(@response.body).map { |auth| auth['api_token'] }
    end
    auths = api_fixture('api_client_authorizations')
    json_param = {format: :json}
    ticket_params =
      json_param.merge(api_ticket: auths['active_trustedclient']['api_token'])
    token_params =
      json_param.merge(api_token: auths['admin_trustedclient']['api_token'])
    # Make sure api_ticket works with no state.
    ticket_results = get_page_with(ticket_params)
    # Set up a session by using api_token.
    token_results = get_page_with(token_params)
    assert_not_equal(ticket_results, token_results,
                     "different API tokens reported identical results")
    # Make sure api_ticket overrides the session.
    assert_equal(ticket_results, get_page_with(ticket_params),
                 "results using api_ticket are inconsistent")
    # Make sure using api_ticket didn't break the session.
    assert_equal(token_results, get_page_with(json_param),
                 "results relying on session token are inconsistent")
  end
end
