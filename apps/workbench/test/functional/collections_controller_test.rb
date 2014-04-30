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
