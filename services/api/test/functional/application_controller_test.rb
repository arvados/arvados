# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

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
    json_response['errors'].each do |err|
      assert(err.include?(errmsg), "error message '#{err}' expected to include '#{errmsg}'")
    end
    check_error_token
  end

  test "requesting nonexistent object returns 404 error" do
    authorize_with :admin
    get(:show, params: {id: BAD_UUID})
    check_404
  end

  test "requesting object without read permission returns 404 error" do
    authorize_with :spectator
    get(:show, params: {id: specimens(:owned_by_active_user).uuid})
    check_404
  end

  test "submitting bad object returns error" do
    authorize_with :spectator
    post(:create, params: {specimen: {badattr: "badvalue"}})
    assert_response 422
    check_error_token
  end

  ['foo', '', 'FALSE', 'TRUE', nil, [true], {a:true}, '"true"'].each do |bogus|
    test "bogus boolean parameter #{bogus.inspect} returns error" do
      @controller = Arvados::V1::GroupsController.new
      authorize_with :active
      post :create, params: {
        group: {},
        ensure_unique_name: bogus
      }
      assert_response 422
      assert_match(/parameter must be a boolean/, json_response['errors'].first,
                   'Helpful error message not found')
    end
  end

  [[true, [true, 'true', 1, '1']],
   [false, [false, 'false', 0, '0']]].each do |bool, boolparams|
    boolparams.each do |boolparam|
      # Ensure boolparam is acceptable as a boolean
      test "boolean parameter #{boolparam.inspect} acceptable" do
        @controller = Arvados::V1::GroupsController.new
        authorize_with :active
        post :create, params: {
          group: {group_class: "project"},
          ensure_unique_name: boolparam
        }
        assert_response :success
      end

      # Ensure boolparam is acceptable as the _intended_ boolean
      test "boolean parameter #{boolparam.inspect} accepted as #{bool.inspect}" do
        @controller = Arvados::V1::GroupsController.new
        authorize_with :active
        post :create, params: {
          group: {
            name: groups(:aproject).name,
            owner_uuid: groups(:aproject).owner_uuid,
            group_class: "project"
          },
          ensure_unique_name: boolparam
        }
        assert_response (bool ? :success : 422)
      end
    end
  end

  test "exceptions with backtraces get logged at exception_backtrace key" do
    Group.stubs(:new).raises(Exception, 'Whoops')
    Rails.logger.expects(:info).with(any_parameters) do |param|
      param.include?('Whoops') and param.include?('"exception_backtrace":')
    end
    @controller = Arvados::V1::GroupsController.new
    authorize_with :active
    post :create, params: {
      group: {},
    }
  end
end
