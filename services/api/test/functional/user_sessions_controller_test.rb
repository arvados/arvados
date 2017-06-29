# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'test_helper'

class UserSessionsControllerTest < ActionController::TestCase

  test "new user from new api client" do
    authorize_with :inactive
    api_client_page = 'http://client.example.com/home'
    get :login, return_to: api_client_page
    assert_response :redirect
    assert_equal(0, @response.redirect_url.index(api_client_page + '?'),
                 'Redirect url ' + @response.redirect_url +
                 ' should start with ' + api_client_page + '?')
    assert_not_nil assigns(:api_client)
  end

end
