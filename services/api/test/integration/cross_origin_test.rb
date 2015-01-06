require 'test_helper'

class CrossOriginTest < ActionDispatch::IntegrationTest
  def options *args
    # Rails doesn't support OPTIONS the same way as GET, POST, etc.
    reset! unless integration_session
    integration_session.__send__(:process, :options, *args).tap do
      copy_session_variables!
    end
  end

  %w(/login /logout /auth/example/callback /auth/joshid).each do |path|
    test "OPTIONS requests are refused at #{path}" do
      options path, {}, {}
      assert_no_cors_headers
    end

    test "CORS headers do not exist at GET #{path}" do
      get path, {}, {}
      assert_no_cors_headers
    end
  end

  %w(/discovery/v1/apis/arvados/v1/rest).each do |path|
    test "CORS headers are set at GET #{path}" do
      get path, {}, {}
      assert_response :success
      assert_cors_headers
    end
  end

  ['/arvados/v1/collections',
   '/arvados/v1/users',
   '/arvados/v1/api_client_authorizations'].each do |path|
    test "CORS headers are set and body is stub at OPTIONS #{path}" do
      options path, {}, {}
      assert_response :success
      assert_cors_headers
      assert_equal '-', response.body
    end

    test "CORS headers are set at authenticated GET #{path}" do
      get path, {}, auth(:active_trustedclient)
      assert_response :success
      assert_cors_headers
    end

    # CORS headers are OK only if cookies are *not* used to determine
    # whether a transaction is allowed. The following is a (far from
    # perfect) test that the usual Rails cookie->session mechanism
    # does not grant access to any resources.
    ['GET', 'POST'].each do |method|
      test "Session does not work at #{method} #{path}" do
        send method.downcase, path, {format: 'json'}, {user_id: 1}
        assert_response 401
        assert_cors_headers
      end
    end
  end

  protected
  def assert_cors_headers
    assert_equal '*', response.headers['Access-Control-Allow-Origin']
    allowed = response.headers['Access-Control-Allow-Methods'].split(', ')
    %w(GET HEAD POST PUT DELETE).each do |m|
      assert_includes allowed, m, "A-C-A-Methods should include #{m}"
    end
    assert_equal 'Authorization', response.headers['Access-Control-Allow-Headers']
  end

  def assert_no_cors_headers
    response.headers.keys.each do |h|
      assert_no_match /^Access-Control-/i, h
    end
  end
end
