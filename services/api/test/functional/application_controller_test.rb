require 'test_helper'

class ApplicationControllerTest < ActionController::TestCase
  BAD_UUID = "zzzzz-zzzzz-zzzzzzzzzzzzzzz"

  def now_timestamp
    Time.now.utc.to_i
  end

  setup do
    # These tests are meant to check behavior in ApplicationController.
    # We instantiate a small concrete controller for convenience.
    @controller = Arvados::V1::SpecimensController.new
    @start_stamp = now_timestamp
  end

  def check_error_token
    token = json_response['error_token']
    assert_not_nil token
    token_time = token.split('+', 2).first.to_i
    assert_operator(token_time, :>=, @start_stamp, "error token too old")
    assert_operator(token_time, :<=, now_timestamp, "error token too new")
  end

  def check_404(errmsg="Path not found")
    assert_response 404
    assert_equal([errmsg], json_response['errors'])
    check_error_token
  end

  test "requesting nonexistent object returns 404 error" do
    authorize_with :admin
    get(:show, id: BAD_UUID)
    check_404
  end

  test "requesting object without read permission returns 404 error" do
    authorize_with :spectator
    get(:show, id: specimens(:owned_by_active_user).uuid)
    check_404
  end

  test "submitting bad object returns error" do
    authorize_with :spectator
    post(:create, specimen: {badattr: "badvalue"})
    assert_response 422
    check_error_token
  end

  ['foo', '', 'FALSE', 'TRUE', nil, [true], {a:true}, '"true"'].each do |bogus|
    test "bogus boolean parameter #{bogus.inspect} returns error" do
      authorize_with :active
      post :create, {
        specimen: {},
        ensure_unique_name: bogus
      }
      assert_response 422
    end
  end
end
