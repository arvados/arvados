# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'test_helper'

class Arvados::V1::QueryTest < ActionController::TestCase
  test 'no fallback orders when order is unambiguous' do
    @controller = Arvados::V1::LogsController.new
    authorize_with :active
    get :index, params: {
      order: ['id asc'],
      controller: 'logs',
    }
    assert_response :success
    assert_equal ['logs.id asc'], assigns(:objects).order_values
  end

  test 'fallback orders when order is ambiguous' do
    @controller = Arvados::V1::LogsController.new
    authorize_with :active
    get :index, params: {
      order: ['event_type asc'],
      controller: 'logs',
    }
    assert_response :success
    assert_equal('logs.event_type asc, logs.modified_at desc, logs.uuid',
                 assigns(:objects).order_values.join(', '))
  end

  test 'skip fallback orders already given by client' do
    @controller = Arvados::V1::LogsController.new
    authorize_with :active
    get :index, params: {
      order: ['modified_at asc'],
      controller: 'logs',
    }
    assert_response :success
    assert_equal('logs.modified_at asc, logs.uuid',
                 assigns(:objects).order_values.join(', '))
  end

  test 'eliminate superfluous orders' do
    @controller = Arvados::V1::LogsController.new
    authorize_with :active
    get :index, params: {
      order: ['logs.modified_at asc',
              'modified_at desc',
              'event_type desc',
              'logs.event_type asc'],
      controller: 'logs',
    }
    assert_response :success
    assert_equal('logs.modified_at asc, logs.event_type desc, logs.uuid',
                 assigns(:objects).order_values.join(', '))
  end

  test 'eliminate orders after the first unique column' do
    @controller = Arvados::V1::LogsController.new
    authorize_with :active
    get :index, params: {
      order: ['event_type asc',
              'id asc',
              'uuid asc',
              'modified_at desc'],
      controller: 'logs',
    }
    assert_response :success
    assert_equal('logs.event_type asc, logs.id asc',
                 assigns(:objects).order_values.join(', '))
  end

  test 'do not count items_available if count=none' do
    @controller = Arvados::V1::LinksController.new
    authorize_with :active
    get :index, params: {
      count: 'none',
    }
    assert_response(:success)
    refute(json_response.has_key?('items_available'))
  end

  test 'do not count items_available if count=none for group contents endpoint' do
    @controller = Arvados::V1::GroupsController.new
    authorize_with :active
    get :contents, params: {
      count: 'none',
    }
    assert_response(:success)
    refute(json_response.has_key?('items_available'))
  end

  [{}, {count: nil}, {count: ''}, {count: 'exact'}].each do |params|
    test "count items_available if params=#{params.inspect}" do
      @controller = Arvados::V1::LinksController.new
      authorize_with :active
      get :index, params: params
      assert_response(:success)
      assert_operator(json_response['items_available'], :>, 0)
    end
  end

  test 'error if count=bogus' do
    @controller = Arvados::V1::LinksController.new
    authorize_with :active
    get :index, params: {
      count: 'bogus',
    }
    assert_response(422)
  end
end
