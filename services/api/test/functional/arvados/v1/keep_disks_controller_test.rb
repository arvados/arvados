require 'test_helper'

class Arvados::V1::KeepDisksControllerTest < ActionController::TestCase

  test "add keep disk with admin token" do
    authorize_with :admin
    post :ping, {
      ping_secret: '',          # required by discovery doc, but ignored
      filesystem_uuid: 'eb1e77a1-db84-4193-b6e6-ca2894f67d5f'
    }
    assert_response :success
    assert_not_nil assigns(:object)
    new_keep_disk = JSON.parse(@response.body)
    assert_not_nil new_keep_disk['uuid']
    assert_not_nil new_keep_disk['ping_secret']
    assert_not_equal '', new_keep_disk['ping_secret']
  end

  [
    {ping_secret: ''},
    {ping_secret: '', filesystem_uuid: ''},
  ].each do |opts|
    test "add keep disk with no filesystem_uuid #{opts}" do
      authorize_with :admin
      post :ping, opts
      assert_response :success
      assert_not_nil JSON.parse(@response.body)['uuid']
    end
  end

  test "refuse to add keep disk without admin token" do
    post :ping, {
      ping_secret: '',
    }
    assert_response 404
  end

  test "ping keep disk" do
    post :ping, {
      id: keep_disks(:nonfull).uuid,
      ping_secret: keep_disks(:nonfull).ping_secret,
      filesystem_uuid: keep_disks(:nonfull).filesystem_uuid
    }
    assert_response :success
    assert_not_nil assigns(:object)
    keep_disk = JSON.parse(@response.body)
    assert_not_nil keep_disk['uuid']
    assert_not_nil keep_disk['ping_secret']
  end

  test "admin should get index with ping_secret" do
    authorize_with :admin
    get :index
    assert_response :success
    assert_not_nil assigns(:objects)
    items = JSON.parse(@response.body)['items']
    assert_not_equal 0, items.size
    assert_not_nil items[0]['ping_secret']
  end

  # inactive user sees keep disks
  test "inactive user should get index" do
    authorize_with :inactive
    get :index
    assert_response :success
    items = JSON.parse(@response.body)['items']
    assert_not_equal 0, items.size

    # Check these are still included
    assert items[0]['service_host']
    assert items[0]['service_port']
  end

  # active user sees non-secret attributes of keep disks
  test "active user should get non-empty index with no ping_secret" do
    authorize_with :active
    get :index
    assert_response :success
    items = JSON.parse(@response.body)['items']
    assert_not_equal 0, items.size
    items.each do |item|
      assert_nil item['ping_secret']
      assert_not_nil item['is_readable']
      assert_not_nil item['is_writable']
      assert_not_nil item['service_host']
      assert_not_nil item['service_port']
    end
  end

  test "search keep_services with 'any' operator" do
    authorize_with :active
    get :index, {
      where: { any: ['contains', 'o2t1q5w'] }
    }
    assert_response :success
    found = assigns(:objects).collect(&:uuid)
    assert_equal true, !!found.index('zzzzz-penuu-5w2o2t1q5wy7fhn')
  end


end
