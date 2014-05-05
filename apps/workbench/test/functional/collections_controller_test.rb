require 'test_helper'

class CollectionsControllerTest < ActionController::TestCase
  def collection_params(collection_name, file_name=nil)
    uuid = api_fixture('collections')[collection_name.to_s]['uuid']
    params = {uuid: uuid, id: uuid}
    params[:file] = file_name if file_name
    params
  end

  def expected_contents(params, token)
    unless token.is_a? String
      token = params[:api_token] || token[:arvados_api_token]
    end
    [token, params[:uuid], params[:file]].join('/')
  end

  def assert_hash_includes(actual_hash, expected_hash, msg=nil)
    expected_hash.each do |key, value|
      assert_equal(value, actual_hash[key], msg)
    end
  end

  def assert_no_session
    assert_hash_includes(session, {arvados_api_token: nil},
                         "session includes unexpected API token")
  end

  def assert_session_for_auth(client_auth)
    api_token =
      api_fixture('api_client_authorizations')[client_auth.to_s]['api_token']
    assert_hash_includes(session, {arvados_api_token: api_token},
                         "session token does not belong to #{client_auth}")
  end

  # Mock the collection file reader to avoid external calls and return
  # a predictable string.
  CollectionsController.class_eval do
    def file_enumerator(opts)
      [[opts[:arvados_api_token], opts[:uuid], opts[:file]].join('/')]
    end
  end

  test "viewing a collection" do
    params = collection_params(:foo_file)
    sess = session_for(:active)
    get(:show, params, sess)
    assert_response :success
    assert_equal([['.', 'foo', 3]], assigns(:object).files)
  end

  test "viewing a collection with a reader token" do
    params = collection_params(:foo_file)
    params[:reader_tokens] =
      [api_fixture('api_client_authorizations')['active']['api_token']]
    get(:show, params)
    assert_response :success
    assert_equal([['.', 'foo', 3]], assigns(:object).files)
    assert_no_session
  end

  test "viewing the index with a reader token" do
    params = {reader_tokens:
      [api_fixture('api_client_authorizations')['spectator']['api_token']]
    }
    get(:index, params)
    assert_response :success
    assert_no_session
    listed_collections = assigns(:collections).map { |c| c.uuid }
    assert_includes(listed_collections,
                    api_fixture('collections')['bar_file']['uuid'],
                    "spectator reader token didn't list bar file")
    refute_includes(listed_collections,
                    api_fixture('collections')['foo_file']['uuid'],
                    "spectator reader token listed foo file")
  end

  test "getting a file from Keep" do
    params = collection_params(:foo_file, 'foo')
    sess = session_for(:active)
    get(:show_file, params, sess)
    assert_response :success
    assert_equal(expected_contents(params, sess), @response.body,
                 "failed to get a correct file from Keep")
  end

  test "can't get a file from Keep without permission" do
    params = collection_params(:foo_file, 'foo')
    sess = session_for(:spectator)
    get(:show_file, params, sess)
    assert_includes([403, 422], @response.code.to_i)
  end
end
