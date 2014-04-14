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
    assert_equal found.count, (found.select { |f| f.head_uuid.match /[a-f0-9]{32}\+\d+/}).count
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

end
