# The v1 API uses token scopes to control access to the REST API at the path
# level.  This is enforced in the base ApplicationController, making it a
# functional test that we can run against many different controllers.

require 'test_helper'

class Arvados::V1::ApiTokensScopeTest < ActionController::IntegrationTest
  fixtures :all

  def setup
    @token = {}
  end

  def auth_with(name)
    @token = {api_token: api_client_authorizations(name).api_token}
  end

  def v1_url(*parts)
    (['arvados', 'v1'] + parts).join('/')
  end

  def request_with_auth(method, path, params={})
    https!
    send(method, path, @token.merge(params))
  end

  def get_with_auth(*args)
    request_with_auth(:get_via_redirect, *args)
  end

  def post_with_auth(*args)
    request_with_auth(:post_via_redirect, *args)
  end

  test "token without scope has no access" do
    # Logs are good for this test, because logs have relatively
    # few access controls enforced at the model level.
    auth_with :admin_noscope
    get_with_auth v1_url('logs')
    assert_response 403
    get_with_auth v1_url('logs', logs(:log1).uuid)
    assert_response 403
    post_with_auth(v1_url('logs'), log: {})
    assert_response 403
  end

  test "VM login scopes work" do
    # A system administration script makes an API token with limited scope
    # for virtual machines to let it see logins.
    def vm_logins_url(name)
      v1_url('virtual_machines', virtual_machines(name).uuid, 'logins')
    end
    auth_with :admin_vm
    get_with_auth vm_logins_url(:testvm)
    assert_response :success
    get_with_auth vm_logins_url(:testvm2)
    assert(@response.status >= 400, "getting testvm2 logins should have failed")
  end
end
