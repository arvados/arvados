require 'test_helper'

class KeepProxyTest < ActionDispatch::IntegrationTest
  test "request keep disks" do
    get "/arvados/v1/keep_services/accessible", {:format => :json}, auth(:active)
    assert_response :success
    services = json_response['items']

    assert_operator 2, :<=, services.length
    services.each do |service|
      assert_equal 'disk', service['service_type']
    end
  end

  test "request keep proxy" do
    get "/arvados/v1/keep_services/accessible", {:format => :json}, auth(:active).merge({'HTTP_X_EXTERNAL_CLIENT' => '1'})
    assert_response :success
    services = json_response['items']

    assert_equal 1, services.length

    assert_equal keep_services(:proxy).uuid, services[0]['uuid']
    assert_equal keep_services(:proxy).service_host, services[0]['service_host']
    assert_equal keep_services(:proxy).service_port, services[0]['service_port']
    assert_equal keep_services(:proxy).service_ssl_flag, services[0]['service_ssl_flag']
    assert_equal 'proxy', services[0]['service_type']
  end
end
