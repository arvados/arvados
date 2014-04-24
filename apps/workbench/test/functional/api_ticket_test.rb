# This test exercises behavior in ApplicationController.

require 'test_helper'

class ApiTicketTest < ActionController::TestCase
  def setup
    # ApiClientAuthorizationsController provides the easiest way to get
    # different results across different users.
    @controller = ApiClientAuthorizationsController.new
  end

  def sorted_tokens(auth_list)
    auth_list.map { |auth| auth['api_token'] }.sort
  end

  def tokens_owned_by(user)
    res = api_fixture('api_client_authorizations').each_value.select { |auth|
      (auth['user'] == user.to_s) and (Time.now < auth['expires_at'])
    }
    sorted_tokens(res)
  end

  def token_for(auth_name)
    api_fixture('api_client_authorizations')[auth_name.to_s]['api_token']
  end

  def build_params(params)
    params = params.dup
    params[:format] ||= :json
    [:api_token, :api_ticket].each do |key|
      if auth_name = params.delete(key)
        params[key] = token_for(auth_name)
      end
    end
    params
  end

  def get_tokens_with(*get_args)
    get :index, *get_args
    assert_response(:success, "failed to get tokens with #{get_args}")
    sorted_tokens(JSON.parse(@response.body))
  end

  test "api_ticket works with no state" do
    assert_equal(tokens_owned_by(:active),
                 get_tokens_with(build_params(api_ticket:
                                              :active_trustedclient)),
                 "bad results with stateless ticket")
  end

  test "api_ticket temporarily overrides session token" do
    orig_session = session_for :admin_trustedclient
    assert_equal(tokens_owned_by(:active),
                 get_tokens_with(build_params(api_ticket:
                                              :active_trustedclient),
                                 orig_session.dup),
                 "bad results when overriding session token")
    orig_session.each_pair do |key, value|
      assert_equal(value, session[key], "session #{key} changed")
    end
  end
end
