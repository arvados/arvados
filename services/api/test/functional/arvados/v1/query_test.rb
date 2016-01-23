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
end
