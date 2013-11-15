require 'test_helper'

class Arvados::V1::KeepDisksControllerTest < ActionController::TestCase

  test "should get index with ping_secret" do
    authorize_with :admin
    get :index
    assert_response :success
    assert_not_nil assigns(:objects)
    items = JSON.parse(@response.body)['items']
    assert_not_equal 0, items.size
    assert_not_nil items[0]['ping_secret']
  end

  # inactive user does not see any keep disks
  test "inactive user should get empty index" do
    authorize_with :inactive
    get :index
    assert_response :success
    items = JSON.parse(@response.body)['items']
    assert_equal 0, items.size
  end

  # active user sees non-secret attributes of keep disks
  test "active user should get non-empty index with no ping_secret" do
    authorize_with :active
    get :index
    assert_response :success
    items = JSON.parse(@response.body)['items']
    assert_not_equal 0, items.size
    items.each do |item|
      assert_nil item['ping_secret']
      assert_not_nil item['is_readable']
      assert_not_nil item['is_writable']
      assert_not_nil item['service_host']
      assert_not_nil item['service_port']
    end
  end

end
