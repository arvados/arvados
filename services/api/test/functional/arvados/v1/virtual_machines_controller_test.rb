require 'test_helper'

class Arvados::V1::VirtualMachinesControllerTest < ActionController::TestCase
  def get_logins_for(vm_sym)
    authorize_with :admin
    get(:logins, id: virtual_machines(vm_sym).uuid)
  end

  def find_login(sshkey_sym)
    assert_response :success
    want_key = authorized_keys(sshkey_sym).public_key
    logins = json_response["items"].select do |login|
      login["public_key"] == want_key
    end
    assert_equal(1, logins.size, "failed to find #{sshkey_sym} login")
    logins.first
  end

  test "username propagated from permission" do
    get_logins_for(:testvm2)
    admin_login = find_login(:admin)
    perm = links(:admin_can_login_to_testvm2)
    assert_equal(perm.properties["username"], admin_login["username"])
  end

  test "groups propagated from permission" do
    get_logins_for(:testvm2)
    admin_login = find_login(:admin)
    perm = links(:admin_can_login_to_testvm2)
    assert_equal(perm.properties["groups"], admin_login["groups"])
  end

  test "groups is an empty list by default" do
    get_logins_for(:testvm2)
    active_login = find_login(:active)
    perm = links(:active_can_login_to_testvm2)
    assert_equal([], active_login["groups"])
  end

  test "logins without usernames not listed" do
    get_logins_for(:testvm2)
    assert_response :success
    spectator_uuid = users(:spectator).uuid
    assert_empty(json_response.
                 select { |login| login["user_uuid"] == spectator_uuid })
  end
end
