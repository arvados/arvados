# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'webrick'
require 'webrick/https'
require 'test_helper'
require 'helpers/users_test_helper'

class RemoteUsersTest < ActionDispatch::IntegrationTest
  def auth(remote:)
    token = salt_token(fixture: :active, remote: remote)
    token.sub!('/zzzzz-', '/'+remote+'-')
    {"HTTP_AUTHORIZATION" => "Bearer #{token}"}
  end

  setup do
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
    Rails.configuration.remote_hosts['zcccc'] = @remote_host
    Arvados::V1::SchemaController.any_instance.stubs(:root_url).returns "https://#{@remote_host}"
    @stub_status = 200
    @stub_content = {
      uuid: 'zbbbb-tpzed-000000000000000',
      is_admin: true,
      is_active: true,
    }
  end

  teardown do
    @remote_server.andand.stop
  end

  test 'authenticate with remote token' do
    get '/arvados/v1/users/current', {}, auth(remote: 'zbbbb')
    assert_response :success
    assert_equal 'zbbbb-tpzed-000000000000000', json_response['uuid']
    assert_equal false, json_response['is_admin']
  end

  test 'authenticate with remote token from wrong site' do
    @stub_content[:uuid] = 'zcccc-tpzed-000000000000000'
    get '/arvados/v1/users/current', {}, auth(remote: 'zbbbb')
    assert_response 401
  end

  test 'authenticate with remote token that fails validate' do
    @stub_status = 401
    @stub_content = {
      error: 'not authorized',
    }
    get '/arvados/v1/users/current', {}, auth(remote: 'zbbbb')
    assert_response 401
  end

  test 'remote api server is not an api server' do
    @stub_status = 200
    @stub_content = '<html>bad</html>'
    get '/arvados/v1/users/current', {}, auth(remote: 'zbbbb')
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
end
