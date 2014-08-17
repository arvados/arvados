require 'test_helper'

class Arvados::V1::LinksControllerTest < ActionController::TestCase

  test "no symbol keys in serialized hash" do
    link = {
      properties: {username: 'testusername'},
      link_class: 'test',
      name: 'encoding',
      tail_uuid: users(:admin).uuid,
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

  %w(created_at modified_at).each do |attr|
    {nil: nil, bogus: 2.days.ago}.each do |bogustype, bogusvalue|
      test "cannot set #{bogustype} #{attr} in create" do
        authorize_with :active
        post :create, {
          link: {
            properties: {},
            link_class: 'test',
            name: 'test',
          }.merge(attr => bogusvalue)
        }
        assert_response :success
        resp = JSON.parse @response.body
        assert_in_delta Time.now, Time.parse(resp[attr]), 3.0
      end
      test "cannot set #{bogustype} #{attr} in update" do
        really_created_at = links(:test_timestamps).created_at
        authorize_with :active
        put :update, {
          id: links(:test_timestamps).uuid,
          link: {
            :properties => {test: 'test'},
            attr => bogusvalue
          }
        }
        assert_response :success
        resp = JSON.parse @response.body
        case attr
        when 'created_at'
          assert_in_delta really_created_at, Time.parse(resp[attr]), 0.001
        else
          assert_in_delta Time.now, Time.parse(resp[attr]), 3.0
        end
      end
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

  test "head and tail exist, head_kind and tail_kind are returned" do
    link = {
      link_class: 'test',
      name: 'stuff',
      head_uuid: users(:active).uuid,
      tail_uuid: users(:spectator).uuid,
    }
    authorize_with :admin
    post :create, link: link
    assert_response :success
    l = JSON.parse(@response.body)
    assert 'arvados#user', l['head_kind']
    assert 'arvados#user', l['tail_kind']
  end

  test "can supply head_kind and tail_kind without error" do
    link = {
      link_class: 'test',
      name: 'stuff',
      head_uuid: users(:active).uuid,
      tail_uuid: users(:spectator).uuid,
      head_kind: "arvados#user",
      tail_kind: "arvados#user",
    }
    authorize_with :admin
    post :create, link: link
    assert_response :success
    l = JSON.parse(@response.body)
    assert 'arvados#user', l['head_kind']
    assert 'arvados#user', l['tail_kind']
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

  test "filter links with 'is_a' operator" do
    authorize_with :admin
    get :index, {
      filters: [ ['tail_uuid', 'is_a', 'arvados#user'] ]
    }
    assert_response :success
    found = assigns(:objects)
    assert_not_equal 0, found.count
    assert_equal found.count, (found.select { |f| f.tail_uuid.match /[a-z0-9]{5}-tpzed-[a-z0-9]{15}/}).count
  end

  test "filter links with 'is_a' operator with more than one" do
    authorize_with :admin
    get :index, {
      filters: [ ['tail_uuid', 'is_a', ['arvados#user', 'arvados#group'] ] ],
    }
    assert_response :success
    found = assigns(:objects)
    assert_not_equal 0, found.count
    assert_equal found.count, (found.select { |f| f.tail_uuid.match /[a-z0-9]{5}-(tpzed|j7d0g)-[a-z0-9]{15}/}).count
  end

  test "filter links with 'is_a' operator with bogus type" do
    authorize_with :admin
    get :index, {
      filters: [ ['tail_uuid', 'is_a', ['arvados#bogus'] ] ],
    }
    assert_response :success
    found = assigns(:objects)
    assert_equal 0, found.count
  end

  test "filter links with 'is_a' operator with collection" do
    authorize_with :admin
    get :index, {
      filters: [ ['head_uuid', 'is_a', ['arvados#collection'] ] ],
    }
    assert_response :success
    found = assigns(:objects)
    assert_not_equal 0, found.count
    assert_equal found.count, (found.select { |f| f.head_uuid.match /.....-4zz18-.............../}).count
  end

  test "test can still use where tail_kind" do
    authorize_with :admin
    get :index, {
      where: { tail_kind: 'arvados#user' }
    }
    assert_response :success
    found = assigns(:objects)
    assert_not_equal 0, found.count
    assert_equal found.count, (found.select { |f| f.tail_uuid.match /[a-z0-9]{5}-tpzed-[a-z0-9]{15}/}).count
  end

  test "test can still use where head_kind" do
    authorize_with :admin
    get :index, {
      where: { head_kind: 'arvados#user' }
    }
    assert_response :success
    found = assigns(:objects)
    assert_not_equal 0, found.count
    assert_equal found.count, (found.select { |f| f.head_uuid.match /[a-z0-9]{5}-tpzed-[a-z0-9]{15}/}).count
  end

  test "test can still use filter tail_kind" do
    authorize_with :admin
    get :index, {
      filters: [ ['tail_kind', '=', 'arvados#user'] ]
    }
    assert_response :success
    found = assigns(:objects)
    assert_not_equal 0, found.count
    assert_equal found.count, (found.select { |f| f.tail_uuid.match /[a-z0-9]{5}-tpzed-[a-z0-9]{15}/}).count
  end

  test "test can still use filter head_kind" do
    authorize_with :admin
    get :index, {
      filters: [ ['head_kind', '=', 'arvados#user'] ]
    }
    assert_response :success
    found = assigns(:objects)
    assert_not_equal 0, found.count
    assert_equal found.count, (found.select { |f| f.head_uuid.match /[a-z0-9]{5}-tpzed-[a-z0-9]{15}/}).count
  end

  test "head_kind matches head_uuid" do
    link = {
      link_class: 'test',
      name: 'stuff',
      head_uuid: groups(:public).uuid,
      head_kind: "arvados#user",
      tail_uuid: users(:spectator).uuid,
      tail_kind: "arvados#user",
    }
    authorize_with :admin
    post :create, link: link
    assert_response 422
  end

  test "tail_kind matches tail_uuid" do
    link = {
      link_class: 'test',
      name: 'stuff',
      head_uuid: users(:active).uuid,
      head_kind: "arvados#user",
      tail_uuid: groups(:public).uuid,
      tail_kind: "arvados#user",
    }
    authorize_with :admin
    post :create, link: link
    assert_response 422
  end

  test "test with virtual_machine" do
    link = {
      tail_kind: "arvados#user",
      tail_uuid: users(:active).uuid,
      head_kind: "arvados#virtual_machine",
      head_uuid: virtual_machines(:testvm).uuid,
      link_class: "permission",
      name: "can_login",
      properties: {username: "repo_and_user_name"}
    }
    authorize_with :admin
    post :create, link: link
    assert_response 422
  end

  test "test with virtualMachine" do
    link = {
      tail_kind: "arvados#user",
      tail_uuid: users(:active).uuid,
      head_kind: "arvados#virtualMachine",
      head_uuid: virtual_machines(:testvm).uuid,
      link_class: "permission",
      name: "can_login",
      properties: {username: "repo_and_user_name"}
    }
    authorize_with :admin
    post :create, link: link
    assert_response :success
  end

  test "refuse duplicate name" do
    skip "Fix for uniqueness constraints"
    the_name = links(:job_name_in_aproject).name
    the_project = links(:job_name_in_aproject).tail_uuid
    authorize_with :active
    post :create, link: {
      tail_uuid: the_project,
      head_uuid: specimens(:owned_by_active_user).uuid,
      link_class: 'name',
      name: the_name,
      properties: {this_s: "a duplicate name"}
    }
    assert_response 422
  end

  test "project owner can show a project permission" do
    uuid = links(:project_viewer_can_read_project).uuid
    authorize_with :active
    get :show, id: uuid
    assert_response :success
    assert_equal(uuid, assigns(:object).andand.uuid)
  end

  test "admin can show a project permission" do
    uuid = links(:project_viewer_can_read_project).uuid
    authorize_with :admin
    get :show, id: uuid
    assert_response :success
    assert_equal(uuid, assigns(:object).andand.uuid)
  end

  test "project viewer can't show others' project permissions" do
    authorize_with :project_viewer
    get :show, id: links(:admin_can_write_aproject).uuid
    assert_response 404
  end

  test "requesting a nonexistent link returns 404" do
    authorize_with :active
    get :show, id: 'zzzzz-zzzzz-zzzzzzzzzzzzzzz'
    assert_response 404
  end

  test "project owner can index project permissions" do
    skip "Test tickles known bug"
    # readable_by only lets users see permission links that relate to them
    # directly.  It does not allow users to see permission links for groups
    # they manage.
    # We'd like to fix this general issue, but we haven't settled on a general
    # way to do it that doesn't involve making readable_by ridiculously hairy.
    # This test demonstrates the desired behavior once we're ready to tackle
    # it.  In the meantime, clients should use /permissions to get this
    # information.
    authorize_with :active
    get :index, filters: [['link_class', '=', 'permission'],
                          ['head_uuid', '=', groups(:aproject).uuid]]
    assert_response :success
    assert_not_nil assigns(:objects)
    assert_includes(assigns(:objects).map(&:uuid),
                    links(:project_viewer_can_read_project).uuid)
  end

  test "admin can index project permissions" do
    authorize_with :admin
    get :index, filters: [['link_class', '=', 'permission'],
                          ['head_uuid', '=', groups(:aproject).uuid]]
    assert_response :success
    assert_not_nil assigns(:objects)
    assert_includes(assigns(:objects).map(&:uuid),
                    links(:project_viewer_can_read_project).uuid)
  end

  test "project viewer can't index others' project permissions" do
    authorize_with :project_viewer
    get :index, filters: [['link_class', '=', 'permission'],
                          ['head_uuid', '=', groups(:aproject).uuid],
                          ['tail_uuid', '!=', users(:project_viewer).uuid]]
    assert_response :success
    assert_not_nil assigns(:objects)
    assert_empty assigns(:objects)
  end
end
