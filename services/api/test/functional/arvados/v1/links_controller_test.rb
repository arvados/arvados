require 'test_helper'

class Arvados::V1::LinksControllerTest < ActionController::TestCase

  test "no symbol keys in serialized hash" do
    link = {
      properties: {username: 'testusername'},
      link_class: 'test',
      name: 'encoding',
      tail_kind: 'arvados#user',
      tail_uuid: users(:admin).uuid,
      head_kind: 'arvados#virtualMachine',
      head_uuid: virtual_machines(:testvm).uuid
    }
    authorize_with :admin
    [link, link.to_json].each do |formatted_link|
      post :create, link: formatted_link
      assert_response :success
      assert_not_nil assigns(:object)
      assert_equal 'testusername', assigns(:object).properties['username']
      assert_equal false, assigns(:object).properties.has_key?(:username)
    end
  end
  
  test "head must exist" do
    link = {
      link_class: 'test',
      name: 'stuff',
      tail_uuid: users(:active).uuid,
      head_uuid: 'zzzzz-tpzed-xyzxyzxerrrorxx'
    }
    authorize_with :admin
    post :create, link: link
    assert_response 422
  end

  test "tail must exist" do
    link = {
      link_class: 'test',
      name: 'stuff',
      head_uuid: users(:active).uuid,
      tail_uuid: 'zzzzz-tpzed-xyzxyzxerrrorxx'
    }
    authorize_with :admin
    post :create, link: link
    assert_response 422
  end

  test "tail must be visible by user" do
    link = {
      link_class: 'test',
      name: 'stuff',
      head_uuid: users(:active).uuid,
      tail_uuid: virtual_machines(:testvm).uuid
    }
    authorize_with :active
    post :create, link: link
    assert_response 422
  end

end
