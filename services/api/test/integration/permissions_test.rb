require 'test_helper'

class PermissionsTest < ActionDispatch::IntegrationTest
  fixtures :users, :groups, :api_client_authorizations, :collections

  test "adding and removing direct can_read links" do
    auth = {'HTTP_AUTHORIZATION' => "OAuth2 #{api_client_authorizations(:spectator).api_token}"}
    admin_auth = {'HTTP_AUTHORIZATION' => "OAuth2 #{api_client_authorizations(:admin).api_token}"}

    # try to read collection as spectator
    get "/arvados/v1/collections/#{collections(:foo_file).uuid}", {:format => :json}, auth
    assert_response 404

    # try to add permission as spectator
    post "/arvados/v1/links", {
      :format => :json,
      :link => {
        tail_uuid: users(:spectator).uuid,
        link_class: 'permission',
        name: 'can_read',
        head_uuid: collections(:foo_file).uuid,
        properties: {}
      }
    }, auth
    assert_response 422

    # add permission as admin
    post "/arvados/v1/links", {
      :format => :json,
      :link => {
        tail_uuid: users(:spectator).uuid,
        link_class: 'permission',
        name: 'can_read',
        head_uuid: collections(:foo_file).uuid,
        properties: {}
      }
    }, admin_auth
    u = jresponse['uuid']
    assert_response :success

    # read collection as spectator
    get "/arvados/v1/collections/#{collections(:foo_file).uuid}", {:format => :json}, auth
    assert_response :success

    # try to delete permission as spectator
    delete "/arvados/v1/links/#{u}", {:format => :json}, auth
    assert_response 403

    # delete permission as admin
    delete "/arvados/v1/links/#{u}", {:format => :json}, admin_auth
    assert_response :success

    # try to read collection as spectator
    get "/arvados/v1/collections/#{collections(:foo_file).uuid}", {:format => :json}, auth
    assert_response 404
  end


  test "adding can_read links from user to group, group to collection" do
    auth = {'HTTP_AUTHORIZATION' => "OAuth2 #{api_client_authorizations(:spectator).api_token}"}
    admin_auth = {'HTTP_AUTHORIZATION' => "OAuth2 #{api_client_authorizations(:admin).api_token}"}
    
    # try to read collection as spectator
    get "/arvados/v1/collections/#{collections(:foo_file).uuid}", {:format => :json}, auth
    assert_response 404

    # add permission for spectator to read group
    post "/arvados/v1/links", {
      :format => :json,
      :link => {
        tail_uuid: users(:spectator).uuid,
        link_class: 'permission',
        name: 'can_read',
        head_uuid: groups(:private).uuid,
        properties: {}
      }
    }, admin_auth
    assert_response :success

    # try to read collection as spectator
    get "/arvados/v1/collections/#{collections(:foo_file).uuid}", {:format => :json}, auth
    assert_response 404

    # add permission for group to read collection
    post "/arvados/v1/links", {
      :format => :json,
      :link => {
        tail_uuid: groups(:private).uuid,
        link_class: 'permission',
        name: 'can_read',
        head_uuid: collections(:foo_file).uuid,
        properties: {}
      }
    }, admin_auth
    u = jresponse['uuid']
    assert_response :success

    # try to read collection as spectator
    get "/arvados/v1/collections/#{collections(:foo_file).uuid}", {:format => :json}, auth
    assert_response :success

    # delete permission for group to read collection
    delete "/arvados/v1/links/#{u}", {:format => :json}, admin_auth
    assert_response :success

    # try to read collection as spectator
    get "/arvados/v1/collections/#{collections(:foo_file).uuid}", {:format => :json}, auth
    assert_response 404
    
  end


  test "adding can_read links from group to collection, user to group" do
    auth = {'HTTP_AUTHORIZATION' => "OAuth2 #{api_client_authorizations(:spectator).api_token}"}
    admin_auth = {'HTTP_AUTHORIZATION' => "OAuth2 #{api_client_authorizations(:admin).api_token}"}
    
    # try to read collection as spectator
    get "/arvados/v1/collections/#{collections(:foo_file).uuid}", {:format => :json}, auth
    assert_response 404

    # add permission for group to read collection
    post "/arvados/v1/links", {
      :format => :json,
      :link => {
        tail_uuid: groups(:private).uuid,
        link_class: 'permission',
        name: 'can_read',
        head_uuid: collections(:foo_file).uuid,
        properties: {}
      }
    }, admin_auth
    assert_response :success

    # try to read collection as spectator
    get "/arvados/v1/collections/#{collections(:foo_file).uuid}", {:format => :json}, auth
    assert_response 404

    # add permission for spectator to read group
    post "/arvados/v1/links", {
      :format => :json,
      :link => {
        tail_uuid: users(:spectator).uuid,
        link_class: 'permission',
        name: 'can_read',
        head_uuid: groups(:private).uuid,
        properties: {}
      }
    }, admin_auth
    u = jresponse['uuid']
    assert_response :success

    # try to read collection as spectator
    get "/arvados/v1/collections/#{collections(:foo_file).uuid}", {:format => :json}, auth
    assert_response :success

    # delete permission for spectator to read group
    delete "/arvados/v1/links/#{u}", {:format => :json}, admin_auth
    assert_response :success

    # try to read collection as spectator
    get "/arvados/v1/collections/#{collections(:foo_file).uuid}", {:format => :json}, auth
    assert_response 404
    
  end

  test "adding can_read links from user to group, group to group, group to collection" do
    auth = {'HTTP_AUTHORIZATION' => "OAuth2 #{api_client_authorizations(:spectator).api_token}"}
    admin_auth = {'HTTP_AUTHORIZATION' => "OAuth2 #{api_client_authorizations(:admin).api_token}"}
    
    # try to read collection as spectator
    get "/arvados/v1/collections/#{collections(:foo_file).uuid}", {:format => :json}, auth
    assert_response 404

    # add permission for user to read group
    post "/arvados/v1/links", {
      :format => :json,
      :link => {
        tail_uuid: users(:spectator).uuid,
        link_class: 'permission',
        name: 'can_read',
        head_uuid: groups(:private).uuid,
        properties: {}
      }
    }, admin_auth
    assert_response :success

    # add permission for group to read group
    post "/arvados/v1/links", {
      :format => :json,
      :link => {
        tail_uuid: groups(:private).uuid,
        link_class: 'permission',
        name: 'can_read',
        head_uuid: groups(:empty_lonely_group).uuid,
        properties: {}
      }
    }, admin_auth
    assert_response :success

    # add permission for group to read collection
    post "/arvados/v1/links", {
      :format => :json,
      :link => {
        tail_uuid: groups(:empty_lonely_group).uuid,
        link_class: 'permission',
        name: 'can_read',
        head_uuid: collections(:foo_file).uuid,
        properties: {}
      }
    }, admin_auth
    u = jresponse['uuid']
    assert_response :success

    # try to read collection as spectator
    get "/arvados/v1/collections/#{collections(:foo_file).uuid}", {:format => :json}, auth
    assert_response :success

    # delete permission for group to read collection
    delete "/arvados/v1/links/#{u}", {:format => :json}, admin_auth
    assert_response :success

    # try to read collection as spectator
    get "/arvados/v1/collections/#{collections(:foo_file).uuid}", {:format => :json}, auth
    assert_response 404
  end
end
