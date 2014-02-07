require 'test_helper'

class Arvados::V1::KeepDisksControllerTest < ActionController::TestCase

  test "add keep disk with admin token" do
    authorize_with :admin
    post :ping, {
      ping_secret: '',          # required by discovery doc, but ignored
      service_host: '::1',
      service_port: 55555,
      service_ssl_flag: false,
      filesystem_uuid: 'eb1e77a1-db84-4193-b6e6-ca2894f67d5f'
    }
    assert_response :success
    assert_not_nil assigns(:object)
    new_keep_disk = JSON.parse(@response.body)
    assert_not_nil new_keep_disk['uuid']
    assert_not_nil new_keep_disk['ping_secret']
    assert_not_equal '', new_keep_disk['ping_secret']
  end

  test "add keep disk with no filesystem_uuid" do
    authorize_with :admin
    opts = {
      ping_secret: '',
      service_host: '::1',
      service_port: 55555,
      service_ssl_flag: false
    }
    post :ping, opts
    assert_response :success
    assert_not_nil JSON.parse(@response.body)['uuid']

    post :ping, opts.merge(filesystem_uuid: '')
    assert_response :success
    assert_not_nil JSON.parse(@response.body)['uuid']
  end

  test "refuse to add keep disk without admin token" do
    post :ping, {
      ping_secret: '',
      service_host: '::1',
      service_port: 55555,
      service_ssl_flag: false
    }
    assert_response 404
  end

  test "ping keep disk" do
    post :ping, {
      uuid: keep_disks(:nonfull).uuid,
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

end
