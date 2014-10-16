require 'test_helper'

class CollectionsControllerTest < ActionController::TestCase
  NONEXISTENT_COLLECTION = "ffffffffffffffffffffffffffffffff+0"

  def stub_file_content
    # For the duration of the current test case, stub file download
    # content with a randomized (but recognizable) string. Return the
    # string, the test case can use it in assertions.
    txt = 'the quick brown fox ' + rand(2**32).to_s
    @controller.stubs(:file_enumerator).returns([txt])
    txt
  end

  def collection_params(collection_name, file_name=nil)
    uuid = api_fixture('collections')[collection_name.to_s]['uuid']
    params = {uuid: uuid, id: uuid}
    params[:file] = file_name if file_name
    params
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

  def show_collection(params, session={}, response=:success)
    params = collection_params(params) if not params.is_a? Hash
    session = session_for(session) if not session.is_a? Hash
    get(:show, params, session)
    assert_response response
  end

  test "viewing a collection" do
    show_collection(:foo_file, :active)
    assert_equal([['.', 'foo', 3]], assigns(:object).files)
  end

  test "viewing a collection fetches related projects" do
    show_collection({id: api_fixture('collections')["foo_file"]['portable_data_hash']}, :active)
    assert_includes(assigns(:same_pdh).map(&:owner_uuid),
                    api_fixture('groups')['aproject']['uuid'],
                    "controller did not find linked project")
  end

  test "viewing a collection fetches related permissions" do
    show_collection(:bar_file, :active)
    assert_includes(assigns(:permissions).map(&:uuid),
                    api_fixture('links')['bar_file_readable_by_active']['uuid'],
                    "controller did not find permission link")
  end

  test "viewing a collection fetches jobs that output it" do
    show_collection(:bar_file, :active)
    assert_includes(assigns(:output_of).map(&:uuid),
                    api_fixture('jobs')['foobar']['uuid'],
                    "controller did not find output job")
  end

  test "viewing a collection fetches jobs that logged it" do
    show_collection(:baz_file, :active)
    assert_includes(assigns(:log_of).map(&:uuid),
                    api_fixture('jobs')['foobar']['uuid'],
                    "controller did not find logger job")
  end

  test "viewing a collection fetches logs about it" do
    show_collection(:foo_file, :active)
    assert_includes(assigns(:logs).map(&:uuid),
                    api_fixture('logs')['log4']['uuid'],
                    "controller did not find related log")
  end

  test "viewing collection files with a reader token" do
    params = collection_params(:foo_file)
    params[:reader_token] =
      api_fixture('api_client_authorizations')['active']['api_token']
    get(:show_file_links, params)
    assert_response :success
    assert_equal([['.', 'foo', 3]], assigns(:object).files)
    assert_no_session
  end

  test "reader token Collection links end with trailing slash" do
    # Testing the fix for #2937.
    session = session_for(:active_trustedclient)
    post(:share, collection_params(:foo_file), session)
    assert(@controller.download_link.ends_with? '/',
           "Collection share link does not end with slash for wget")
  end

  test "getting a file from Keep" do
    params = collection_params(:foo_file, 'foo')
    sess = session_for(:active)
    expect_content = stub_file_content
    get(:show_file, params, sess)
    assert_response :success
    assert_equal(expect_content, @response.body,
                 "failed to get a correct file from Keep")
  end

  test "can't get a file from Keep without permission" do
    params = collection_params(:foo_file, 'foo')
    sess = session_for(:spectator)
    get(:show_file, params, sess)
    assert_response 404
  end

  test "trying to get a nonexistent file from Keep returns a 404" do
    params = collection_params(:foo_file, 'gone')
    sess = session_for(:admin)
    get(:show_file, params, sess)
    assert_response 404
  end

  test "getting a file from Keep with a good reader token" do
    params = collection_params(:foo_file, 'foo')
    read_token = api_fixture('api_client_authorizations')['active']['api_token']
    params[:reader_token] = read_token
    expect_content = stub_file_content
    get(:show_file, params)
    assert_response :success
    assert_equal(expect_content, @response.body,
                 "failed to get a correct file from Keep using a reader token")
    assert_not_equal(read_token, session[:arvados_api_token],
                     "using a reader token set the session's API token")
  end

  test "trying to get from Keep with an unscoped reader token prompts login" do
    params = collection_params(:foo_file, 'foo')
    params[:reader_token] =
      api_fixture('api_client_authorizations')['active_noscope']['api_token']
    get(:show_file, params)
    assert_response :redirect
  end

  test "can get a file with an unpermissioned auth but in-scope reader token" do
    params = collection_params(:foo_file, 'foo')
    sess = session_for(:expired)
    read_token = api_fixture('api_client_authorizations')['active']['api_token']
    params[:reader_token] = read_token
    expect_content = stub_file_content
    get(:show_file, params, sess)
    assert_response :success
    assert_equal(expect_content, @response.body,
                 "failed to get a correct file from Keep using a reader token")
    assert_not_equal(read_token, session[:arvados_api_token],
                     "using a reader token set the session's API token")
  end

  test "inactive user can retrieve user agreement" do
    ua_collection = api_fixture('collections')['user_agreement']
    # Here we don't test whether the agreement can be retrieved from
    # Keep. We only test that show_file decides to send file content,
    # so we use the file content stub.
    stub_file_content
    get :show_file, {
      uuid: ua_collection['uuid'],
      file: ua_collection['manifest_text'].match(/ \d+:\d+:(\S+)/)[1]
    }, session_for(:inactive)
    assert_nil(assigns(:unsigned_user_agreements),
               "Did not skip check_user_agreements filter " +
               "when showing the user agreement.")
    assert_response :success
  end

  test "requesting nonexistent Collection returns 404" do
    show_collection({uuid: NONEXISTENT_COLLECTION, id: NONEXISTENT_COLLECTION},
                    :active, 404)
  end

  test "use a reasonable read buffer even if client requests a huge range" do
    fakefiledata = mock
    IO.expects(:popen).returns(fakefiledata)
    fakefiledata.expects(:read).twice.with() do |length|
      # Fail the test if read() is called with length>1MiB:
      length < 2**20
      ## Force the ActionController::Live thread to lose the race to
      ## verify that @response.body.length actually waits for the
      ## response (see below):
      # sleep 3
    end.returns("foo\n", nil)
    fakefiledata.expects(:close)
    foo_file = api_fixture('collections')['foo_file']
    @request.headers['Range'] = 'bytes=0-4294967296/*'
    get :show_file, {
      uuid: foo_file['uuid'],
      file: foo_file['manifest_text'].match(/ \d+:\d+:(\S+)/)[1]
    }, session_for(:active)
    # Wait for the whole response to arrive before deciding whether
    # mocks' expectations were met. Otherwise, Mocha will fail the
    # test depending on how slowly the ActionController::Live thread
    # runs.
    @response.body.length
  end
end
