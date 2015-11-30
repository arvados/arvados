require 'test_helper'

class Arvados::V1::KeepServicesControllerTest < ActionController::TestCase

  test "search by service_port with < query" do
    authorize_with :active
    get :index, {
      filters: [['service_port', '<', 25107]]
    }
    assert_response :success
    assert_equal false, assigns(:objects).any?
  end

  test "search by service_port with >= query" do
    authorize_with :active
    get :index, {
      filters: [['service_port', '>=', 25107]]
    }
    assert_response :success
    assert_equal true, assigns(:objects).any?
  end

  [:admin, :active, :inactive, :anonymous].each do |u|
    test "accessible to #{u} user" do
      authorize_with u
      get :accessible
      assert_response :success
      assert_not_empty json_response['items']
      json_response['items'].each do |ks|
        assert_not_equal ks['service_type'], 'proxy'
      end
    end
  end

end
