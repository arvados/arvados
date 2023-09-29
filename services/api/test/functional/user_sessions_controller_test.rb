# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'test_helper'

class UserSessionsControllerTest < ActionController::TestCase

  setup do
    @allowed_return_to = ",https://controller.api.client.invalid"
  end

  test "login route deleted" do
    @request.headers['Authorization'] = 'Bearer '+Rails.configuration.SystemRootToken
    get :login, params: {provider: 'controller', return_to: @allowed_return_to}
    assert_response 404
  end

  test "controller cannot create session without SystemRootToken" do
    get :create, params: {provider: 'controller', auth_info: {email: "foo@bar.com"}, return_to: @allowed_return_to}
    assert_response 401
  end

  test "controller cannot create session with wrong SystemRootToken" do
    @request.headers['Authorization'] = 'Bearer blah'
    get :create, params: {provider: 'controller', auth_info: {email: "foo@bar.com"}, return_to: @allowed_return_to}
    assert_response 401
  end

  test "controller can create session using SystemRootToken" do
    @request.headers['Authorization'] = 'Bearer '+Rails.configuration.SystemRootToken
    get :create, params: {provider: 'controller', auth_info: {email: "foo@bar.com"}, return_to: @allowed_return_to}
    assert_response :redirect
    api_client_auth = assigns(:api_client_auth)
    assert_not_nil api_client_auth
    assert_includes(@response.redirect_url, 'api_token='+api_client_auth.token)
  end
end
