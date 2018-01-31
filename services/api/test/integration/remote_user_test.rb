# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'webrick'
require 'webrick/https'
require 'test_helper'
require 'helpers/users_test_helper'

class RemoteUsersTest < ActionDispatch::IntegrationTest
  include DbCurrentTime

  def salted_active_token(remote:)
    salt_token(fixture: :active, remote: remote).sub('/zzzzz-', '/'+remote+'-')
  end

  def auth(remote:)
    token = salted_active_token(remote: remote)
    {"HTTP_AUTHORIZATION" => "Bearer #{token}"}
  end

  # For remote authentication tests, we bring up a simple stub server
  # (on a port chosen by webrick) and configure the SUT so the stub is
  # responsible for clusters "zbbbb" (a well-behaved cluster) and
  # "zbork" (a misbehaving cluster).
  #
  # Test cases can override the stub's default response to
  # .../users/current by changing @stub_status and @stub_content.
  setup do
    clnt = HTTPClient.new
    clnt.ssl_config.verify_mode = OpenSSL::SSL::VERIFY_NONE
    HTTPClient.stubs(:new).returns clnt

    @controller = Arvados::V1::UsersController.new
    ready = Thread::Queue.new
    srv = WEBrick::HTTPServer.new(
      Port: 0,
      Logger: WEBrick::Log.new(
        Rails.root.join("log", "webrick.log").to_s,
        WEBrick::Log::INFO),
      AccessLog: [[File.open(Rails.root.join(
                              "log", "webrick_access.log").to_s, 'a+'),
                   WEBrick::AccessLog::COMBINED_LOG_FORMAT]],
      SSLEnable: true,
      SSLVerifyClient: OpenSSL::SSL::VERIFY_NONE,
      SSLPrivateKey: OpenSSL::PKey::RSA.new(
        File.open(Rails.root.join("tmp", "self-signed.key")).read),
      SSLCertificate: OpenSSL::X509::Certificate.new(
        File.open(Rails.root.join("tmp", "self-signed.pem")).read),
      SSLCertName: [["CN", WEBrick::Utils::getservername]],
      StartCallback: lambda { ready.push(true) })
    srv.mount_proc '/discovery/v1/apis/arvados/v1/rest' do |req, res|
      Rails.cache.delete 'arvados_v1_rest_discovery'
      res.body = Arvados::V1::SchemaController.new.send(:discovery_doc).to_json
    end
    srv.mount_proc '/arvados/v1/users/current' do |req, res|
      res.status = @stub_status
      res.body = @stub_content.is_a?(String) ? @stub_content : @stub_content.to_json
    end
    Thread.new do
      srv.start
    end
    ready.pop
    @remote_server = srv
    @remote_host = "127.0.0.1:#{srv.config[:Port]}"
    Rails.configuration.remote_hosts['zbbbb'] = @remote_host
    Rails.configuration.remote_hosts['zbork'] = @remote_host
    Arvados::V1::SchemaController.any_instance.stubs(:root_url).returns "https://#{@remote_host}"
    @stub_status = 200
    @stub_content = {
      uuid: 'zbbbb-tpzed-000000000000000',
      email: 'foo@example.com',
      username: 'barney',
      is_admin: true,
      is_active: true,
    }
  end

  teardown do
    @remote_server.andand.stop
  end

  test 'authenticate with remote token' do
    get '/arvados/v1/users/current', {format: 'json'}, auth(remote: 'zbbbb')
    assert_response :success
    assert_equal 'zbbbb-tpzed-000000000000000', json_response['uuid']
    assert_equal false, json_response['is_admin']
    assert_equal 'foo@example.com', json_response['email']
    assert_equal 'barney', json_response['username']

    # revoke original token
    @stub_status = 401

    # re-authorize before cache expires
    get '/arvados/v1/users/current', {format: 'json'}, auth(remote: 'zbbbb')
    assert_response :success

    # simulate cache expiry
    ApiClientAuthorization.where(
      uuid: salted_active_token(remote: 'zbbbb').split('/')[1]).
      update_all(expires_at: db_current_time - 1.minute)

    # re-authorize after cache expires
    get '/arvados/v1/users/current', {format: 'json'}, auth(remote: 'zbbbb')
    assert_response 401

    # simulate cached token indicating wrong user (e.g., local user
    # entry was migrated out of the way taking the cached token with
    # it, or authorizing cluster reassigned auth to a different user)
    ApiClientAuthorization.where(
      uuid: salted_active_token(remote: 'zbbbb').split('/')[1]).
      update_all(user_id: users(:active).id)

    # revive original token and re-authorize
    @stub_status = 200
    @stub_content[:username] = 'blarney'
    @stub_content[:email] = 'blarney@example.com'
    get '/arvados/v1/users/current', {format: 'json'}, auth(remote: 'zbbbb')
    assert_response :success
    assert_equal 'barney', json_response['username'], 'local username should not change once assigned'
    assert_equal 'blarney@example.com', json_response['email']
  end

  test 'authenticate with remote token, remote username conflicts with local' do
    @stub_content[:username] = 'active'
    get '/arvados/v1/users/current', {format: 'json'}, auth(remote: 'zbbbb')
    assert_response :success
    assert_equal 'active2', json_response['username']
  end

  test 'authenticate with remote token, remote username is nil' do
    @stub_content.delete :username
    get '/arvados/v1/users/current', {format: 'json'}, auth(remote: 'zbbbb')
    assert_response :success
    assert_equal 'foo', json_response['username']
  end

  test 'authenticate with remote token from misbhehaving remote cluster' do
    get '/arvados/v1/users/current', {format: 'json'}, auth(remote: 'zbork')
    assert_response 401
  end

  test 'authenticate with remote token that fails validate' do
    @stub_status = 401
    @stub_content = {
      error: 'not authorized',
    }
    get '/arvados/v1/users/current', {format: 'json'}, auth(remote: 'zbbbb')
    assert_response 401
  end

  ['v2',
   'v2/',
   'v2//',
   'v2///',
   "v2/'; delete from users where 1=1; commit; select '/lol",
   'v2/foo/bar',
   'v2/zzzzz-gj3su-077z32aux8dg2s1',
   'v2/zzzzz-gj3su-077z32aux8dg2s1/',
   'v2/3kg6k6lzmp9kj5cpkcoxie963cmvjahbt2fod9zru30k1jqdmi',
   'v2/3kg6k6lzmp9kj5cpkcoxie963cmvjahbt2fod9zru30k1jqdmi/zzzzz-gj3su-077z32aux8dg2s1',
   'v2//3kg6k6lzmp9kj5cpkcoxie963cmvjahbt2fod9zru30k1jqdmi',
   'v8/zzzzz-gj3su-077z32aux8dg2s1/3kg6k6lzmp9kj5cpkcoxie963cmvjahbt2fod9zru30k1jqdmi',
   '/zzzzz-gj3su-077z32aux8dg2s1/3kg6k6lzmp9kj5cpkcoxie963cmvjahbt2fod9zru30k1jqdmi',
   '"v2/zzzzz-gj3su-077z32aux8dg2s1/3kg6k6lzmp9kj5cpkcoxie963cmvjahbt2fod9zru30k1jqdmi"',
   '/',
   '//',
   '///',
  ].each do |token|
    test "authenticate with malformed remote token #{token}" do
      get '/arvados/v1/users/current', {format: 'json'}, {"HTTP_AUTHORIZATION" => "Bearer #{token}"}
      assert_response 401
    end
  end

  test "ignore extra fields in remote token" do
    token = salted_active_token(remote: 'zbbbb') + '/foo/bar/baz/*'
    get '/arvados/v1/users/current', {format: 'json'}, {"HTTP_AUTHORIZATION" => "Bearer #{token}"}
    assert_response :success
  end

  test 'remote api server is not an api server' do
    @stub_status = 200
    @stub_content = '<html>bad</html>'
    get '/arvados/v1/users/current', {format: 'json'}, auth(remote: 'zbbbb')
    assert_response 401
  end

  ['zbbbb', 'z0000'].each do |token_valid_for|
    test "validate #{token_valid_for}-salted token for remote cluster zbbbb" do
      salted_token = salt_token(fixture: :active, remote: token_valid_for)
      get '/arvados/v1/users/current', {format: 'json', remote: 'zbbbb'}, {
            "HTTP_AUTHORIZATION" => "Bearer #{salted_token}"
          }
      if token_valid_for == 'zbbbb'
        assert_response 200
        assert_equal(users(:active).uuid, json_response['uuid'])
      else
        assert_response 401
      end
    end
  end

  test "list readable groups with salted token" do
    salted_token = salt_token(fixture: :active, remote: 'zbbbb')
    get '/arvados/v1/groups', {
          format: 'json',
          remote: 'zbbbb',
          limit: 10000,
        }, {
          "HTTP_AUTHORIZATION" => "Bearer #{salted_token}"
        }
    assert_response 200
    group_uuids = json_response['items'].collect { |i| i['uuid'] }
    assert_includes(group_uuids, 'zzzzz-j7d0g-fffffffffffffff')
    refute_includes(group_uuids, 'zzzzz-j7d0g-000000000000000')
    assert_includes(group_uuids, groups(:aproject).uuid)
    refute_includes(group_uuids, groups(:trashed_project).uuid)
    refute_includes(group_uuids, groups(:testusergroup_admins).uuid)
  end
end
