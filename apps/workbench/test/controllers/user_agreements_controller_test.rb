# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'test_helper'

class UserAgreementsControllerTest < ActionController::TestCase
  test 'User agreements page shows form if some user agreements are not signed' do
    get :index, params: {}, session: session_for(:inactive)
    assert_response 200
  end

  test 'User agreements page redirects if all user agreements signed' do
    get :index, params: {return_to: root_path}, session: session_for(:active)
    assert_response :redirect
    assert_equal(root_url,
                 @response.redirect_url,
                 "Active user was not redirected to :return_to param")
  end
end