require 'test_helper'

class KeepProxyTest < ActionDispatch::IntegrationTest
  test "request keep disks" do
    get "/arvados/v1/keep_services/accessable", {:format => :json}, auth(:active)
    assert_response :success
    services = json_response['items']

    assert_equal 2, services.length
    assert_equal 'disk', services[0]['service_type']
    assert_equal 'disk', services[1]['service_type']

    get "/arvados/v1/keep_services/accessable", {:format => :json}, auth(:active).merge({'HTTP_X_KEEP_PROXY_REQUIRED' => true})
    assert_response :success
    services = json_response['items']

    assert_equal 1, services.length

    assert_equal "zzzzz-bi6l4-h0a0xwut9qa6g3a", services[0]['uuid']
    assert_equal "keep.qr1hi.arvadosapi.com", services[0]['service_host']
    assert_equal 25333, services[0]['service_port']
    assert_equal true, services[0]['service_ssl_flag']
    assert_equal 'proxy', services[0]['service_type']
  end
end
