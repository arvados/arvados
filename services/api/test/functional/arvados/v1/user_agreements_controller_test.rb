# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'test_helper'

class Arvados::V1::UserAgreementsControllerTest < ActionController::TestCase

  test "active user get user agreements" do
    authorize_with :active
    get :index
    assert_response :success
    assert_not_nil assigns(:objects)
    agreements_list = JSON.parse(@response.body)
    assert_not_nil agreements_list['items']
    assert_not_nil agreements_list['items'][0]
  end

  test "active user get user agreement signatures" do
    authorize_with :active
    get :signatures
    assert_response :success
    assert_not_nil assigns(:objects)
    agreements_list = JSON.parse(@response.body)
    assert_not_nil agreements_list['items']
    assert_not_nil agreements_list['items'][0]
    assert_equal 1, agreements_list['items'].count
  end

  test "inactive user get user agreements" do
    authorize_with :inactive
    get :index
    assert_response :success
    assert_not_nil assigns(:objects)
    agreements_list = JSON.parse(@response.body)
    assert_not_nil agreements_list['items']
    assert_not_nil agreements_list['items'][0]
  end

  test "uninvited user receives empty list of user agreements" do
    authorize_with :inactive_uninvited
    get :index
    assert_response :success
    assert_not_nil assigns(:objects)
    agreements_list = JSON.parse(@response.body)
    assert_not_nil agreements_list['items']
    assert_nil agreements_list['items'][0]
  end

end
