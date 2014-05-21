require 'test_helper'

class Arvados::V1::KeepServicesControllerTest < ActionController::TestCase

  test "search keep_services by service_port with < query" do
    authorize_with :active
    get :index, {
      filters: [['service_port', '<', 25107]]
    }
    assert_response :success
    assert_equal false, assigns(:objects).any?
  end

  test "search keep_disks by service_port with >= query" do
    authorize_with :active
    get :index, {
      filters: [['service_port', '>=', 25107]]
    }
    assert_response :success
    assert_equal true, assigns(:objects).any?
  end

end
