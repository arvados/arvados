require 'test_helper'

class Arvados::V1::QueryTest < ActionController::TestCase
  test 'no fallback orders when order is unambiguous' do
    @controller = Arvados::V1::LogsController.new
    authorize_with :active
    get :index, {
      order: ['id asc'],
      controller: 'logs',
    }
    assert_response :success
    assert_equal ['logs.id asc'], assigns(:objects).order_values
  end

  test 'fallback orders when order is ambiguous' do
    @controller = Arvados::V1::LogsController.new
    authorize_with :active
    get :index, {
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
    get :index, {
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
    get :index, {
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
    get :index, {
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
end
