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

  test "logins without ssh keys are listed" do
    u, vm = nil
    act_as_system_user do
      u = create :active_user, first_name: 'Bob', last_name: 'Blogin'
      vm = VirtualMachine.create! hostname: 'foo.shell'
      Link.create!(tail_uuid: u.uuid,
                   head_uuid: vm.uuid,
                   link_class: 'permission',
                   name: 'can_login',
                   properties: {'username' => 'bobblogin'})
    end
    authorize_with :admin
    get :logins, id: vm.uuid
    assert_response :success
    assert_equal 1, json_response['items'].length
    assert_equal nil, json_response['items'][0]['public_key']
    assert_equal nil, json_response['items'][0]['authorized_key_uuid']
    assert_equal u.uuid, json_response['items'][0]['user_uuid']
    assert_equal 'bobblogin', json_response['items'][0]['username']
  end
end
