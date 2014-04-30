require 'test_helper'

class Arvados::V1::ReaderTokensTest < ActionController::IntegrationTest
  fixtures :all

  def spectator_specimen
    specimens(:owned_by_spectator).uuid
  end

  def get_specimens(main_auth, read_auth, formatter=:to_a)
    params = {}
    params[:reader_tokens] = [api_token(read_auth)].send(formatter) if read_auth
    headers = {}
    headers.merge!(auth(main_auth)) if main_auth
    get('/arvados/v1/specimens', params, headers)
  end

  def get_specimen_uuids(main_auth, read_auth, formatter=:to_a)
    get_specimens(main_auth, read_auth, formatter)
    assert_response :success
    json_response['items'].map { |spec| spec['uuid'] }
  end

  def assert_post_denied(main_auth, read_auth, formatter=:to_a)
    if main_auth
      headers = auth(main_auth)
      expected = 403
    else
      headers = {}
      expected = 401
    end
    post('/arvados/v1/specimens.json',
         {specimen: {}, reader_tokens: [api_token(read_auth)].send(formatter)},
         headers)
    assert_response expected
  end

  test "active user can't see spectator specimen" do
    # Other tests in this suite assume that the active user doesn't
    # have read permission to the owned_by_spectator specimen.
    # This test checks that this assumption still holds.
    refute_includes(get_specimen_uuids(:active, nil), spectator_specimen,
                    ["active user can read the owned_by_spectator specimen",
                     "other tests will return false positives"].join(" - "))
  end

  [nil, :active_noscope].each do |main_auth|
    [:spectator, :spectator_specimens].each do |read_auth|
      test "#{main_auth} auth with reader token #{read_auth} can read" do
        assert_includes(get_specimen_uuids(main_auth, read_auth),
                        spectator_specimen, "did not find spectator specimen")
      end

      test "#{main_auth} auth with JSON read token #{read_auth} can read" do
        assert_includes(get_specimen_uuids(main_auth, read_auth, :to_json),
                        spectator_specimen, "did not find spectator specimen")
      end

      test "#{main_auth} auth with reader token #{read_auth} can't write" do
        assert_post_denied(main_auth, read_auth)
      end

      test "#{main_auth} auth with JSON read token #{read_auth} can't write" do
        assert_post_denied(main_auth, read_auth, :to_json)
      end
    end
  end

  test "scopes are still limited with reader tokens" do
    get('/arvados/v1/collections',
        {reader_tokens: [api_token(:spectator_specimens)]},
        auth(:active_noscope))
    assert_response 403
  end

  test "reader tokens grant no permissions when expired" do
    get_specimens(:active_noscope, :expired)
    assert_response 403
  end

  test "reader tokens grant no permissions outside their scope" do
    refute_includes(get_specimen_uuids(:active, :admin_vm), spectator_specimen,
                    "scoped reader token granted permissions out of scope")
  end
end
