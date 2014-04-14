require 'test_helper'

class PermissionsTest < ActionDispatch::IntegrationTest
  fixtures :users, :groups, :api_client_authorizations, :collections

  test "adding and removing direct can_read links" do
    # try to read collection as spectator
    get "/arvados/v1/collections/#{collections(:foo_file).uuid}", {:format => :json}, auth(:spectator)
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
    }, auth(:spectator)
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
    }, auth(:admin)
    u = jresponse['uuid']
    assert_response :success

    # read collection as spectator
    get "/arvados/v1/collections/#{collections(:foo_file).uuid}", {:format => :json}, auth(:spectator)
    assert_response :success

    # try to delete permission as spectator
    delete "/arvados/v1/links/#{u}", {:format => :json}, auth(:spectator)
    assert_response 403

    # delete permission as admin
    delete "/arvados/v1/links/#{u}", {:format => :json}, auth(:admin)
    assert_response :success

    # try to read collection as spectator
    get "/arvados/v1/collections/#{collections(:foo_file).uuid}", {:format => :json}, auth(:spectator)
    assert_response 404
  end


  test "adding can_read links from user to group, group to collection" do
    # try to read collection as spectator
    get "/arvados/v1/collections/#{collections(:foo_file).uuid}", {:format => :json}, auth(:spectator)
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
    }, auth(:admin)
    assert_response :success

    # try to read collection as spectator
    get "/arvados/v1/collections/#{collections(:foo_file).uuid}", {:format => :json}, auth(:spectator)
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
    }, auth(:admin)
    u = jresponse['uuid']
    assert_response :success

    # try to read collection as spectator
    get "/arvados/v1/collections/#{collections(:foo_file).uuid}", {:format => :json}, auth(:spectator)
    assert_response :success

    # delete permission for group to read collection
    delete "/arvados/v1/links/#{u}", {:format => :json}, auth(:admin)
    assert_response :success

    # try to read collection as spectator
    get "/arvados/v1/collections/#{collections(:foo_file).uuid}", {:format => :json}, auth(:spectator)
    assert_response 404
    
  end


  test "adding can_read links from group to collection, user to group" do
    # try to read collection as spectator
    get "/arvados/v1/collections/#{collections(:foo_file).uuid}", {:format => :json}, auth(:spectator)
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
    }, auth(:admin)
    assert_response :success

    # try to read collection as spectator
    get "/arvados/v1/collections/#{collections(:foo_file).uuid}", {:format => :json}, auth(:spectator)
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
    }, auth(:admin)
    u = jresponse['uuid']
    assert_response :success

    # try to read collection as spectator
    get "/arvados/v1/collections/#{collections(:foo_file).uuid}", {:format => :json}, auth(:spectator)
    assert_response :success

    # delete permission for spectator to read group
    delete "/arvados/v1/links/#{u}", {:format => :json}, auth(:admin)
    assert_response :success

    # try to read collection as spectator
    get "/arvados/v1/collections/#{collections(:foo_file).uuid}", {:format => :json}, auth(:spectator)
    assert_response 404
    
  end

  test "adding can_read links from user to group, group to group, group to collection" do
    # try to read collection as spectator
    get "/arvados/v1/collections/#{collections(:foo_file).uuid}", {:format => :json}, auth(:spectator)
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
    }, auth(:admin)
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
    }, auth(:admin)
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
    }, auth(:admin)
    u = jresponse['uuid']
    assert_response :success

    # try to read collection as spectator
    get "/arvados/v1/collections/#{collections(:foo_file).uuid}", {:format => :json}, auth(:spectator)
    assert_response :success

    # delete permission for group to read collection
    delete "/arvados/v1/links/#{u}", {:format => :json}, auth(:admin)
    assert_response :success

    # try to read collection as spectator
    get "/arvados/v1/collections/#{collections(:foo_file).uuid}", {:format => :json}, auth(:spectator)
    assert_response 404
  end

  test "read-only group-admin sees correct subset of user list" do
    get "/arvados/v1/users", {:format => :json}, auth(:rominiadmin)
    assert_response :success
    resp_uuids = jresponse['items'].collect { |i| i['uuid'] }
    [[true, users(:rominiadmin).uuid],
     [true, users(:active).uuid],
     [false, users(:miniadmin).uuid],
     [false, users(:spectator).uuid]].each do |should_find, uuid|
      assert_equal should_find, !resp_uuids.index(uuid).nil?, "rominiadmin should #{'not ' if !should_find}see #{uuid} in user list"
    end
  end

  test "read-only group-admin cannot modify administered user" do
    put "/arvados/v1/users/#{users(:active).uuid}", {
      :user => {
        first_name: 'KilroyWasHere'
      },
      :format => :json
    }, auth(:rominiadmin)
    assert_response 403
  end

  test "read-only group-admin cannot read or update non-administered user" do
    get "/arvados/v1/users/#{users(:spectator).uuid}", {
      :format => :json
    }, auth(:rominiadmin)
    assert_response 404

    put "/arvados/v1/users/#{users(:spectator).uuid}", {
      :user => {
        first_name: 'KilroyWasHere'
      },
      :format => :json
    }, auth(:rominiadmin)
    assert_response 404
  end

  test "RO group-admin finds user's specimens, RW group-admin can update" do
    [[:rominiadmin, false],
     [:miniadmin, true]].each do |which_user, update_should_succeed|
      get "/arvados/v1/specimens", {:format => :json}, auth(which_user)
      assert_response :success
      resp_uuids = jresponse['items'].collect { |i| i['uuid'] }
      [[true, specimens(:owned_by_active_user).uuid],
       [true, specimens(:owned_by_private_group).uuid],
       [false, specimens(:owned_by_spectator).uuid],
      ].each do |should_find, uuid|
        assert_equal(should_find, !resp_uuids.index(uuid).nil?,
                     "%s should%s see %s in specimen list" %
                     [which_user.to_s,
                      should_find ? '' : 'not ',
                      uuid])
        put "/arvados/v1/specimens/#{uuid}", {
          :specimen => {
            properties: {
              miniadmin_was_here: true
            }
          },
          :format => :json
        }, auth(which_user)
        if !should_find
          assert_response 404
        elsif !update_should_succeed
          assert_response 403
        else
          assert_response :success
        end
      end
    end
  end

end
