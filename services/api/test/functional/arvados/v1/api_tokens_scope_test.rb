# The v1 API uses token scopes to control access to the REST API at the path
# level.  This is enforced in the base ApplicationController, making it a
# functional test that we can run against many different controllers.

require 'test_helper'

class Arvados::V1::ApiTokensScopeTest < ActionController::TestCase
  test "VM login scopes work" do
    # A system administration script makes an API token with limited scope
    # for virtual machines to let it see logins.
    @controller = Arvados::V1::VirtualMachinesController.new
    authorize_with :admin_vm
    get :logins, uuid: virtual_machines(:testvm).uuid
    assert_response :success
    get :logins, uuid: virtual_machines(:testvm2).uuid
    assert(@response.status >= 400, "getting testvm2 logins should have failed")
  end
end
