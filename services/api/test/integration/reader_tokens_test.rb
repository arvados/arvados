# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'test_helper'

class ReaderTokensTest < ActionDispatch::IntegrationTest
  fixtures :all

  def owned_by_foo
    collections(:collection_owned_by_foo).uuid
  end

  def get_collections(main_auth, read_auth, formatter=:to_a)
    params = {}
    params[:reader_tokens] = [api_token(read_auth)].send(formatter) if read_auth
    headers = {}
    headers.merge!(auth(main_auth)) if main_auth
    get('/arvados/v1/collections', params: params, headers: headers)
  end

  def get_collection_uuids(main_auth, read_auth, formatter=:to_a)
    get_collections(main_auth, read_auth, formatter)
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
    post('/arvados/v1/collections.json',
      params: {collection: {}, reader_tokens: [api_token(read_auth)].send(formatter)},
      headers: headers)
    assert_response expected
  end

  test "active user can't see foo-owned collection" do
    # Other tests in this suite assume that the active user doesn't
    # have read permission to the owned_by_foo collection.
    # This test checks that this assumption still holds.
    refute_includes(get_collection_uuids(:active, nil), owned_by_foo,
                    ["active user can read the owned_by_foo collection",
                     "other tests will return false positives"].join(" - "))
  end

  [nil, :active_noscope].each do |main_auth|
    [:foo, :foo_collections].each do |read_auth|
      [:to_a, :to_json].each do |formatter|
        test "#{main_auth.inspect} auth with #{formatter} reader token #{read_auth} can#{"'t" if main_auth} read" do
          get_collections(main_auth, read_auth)
          assert_response(if main_auth then 403 else 200 end)
        end

        test "#{main_auth.inspect} auth with #{formatter} reader token #{read_auth} can't write" do
          assert_post_denied(main_auth, read_auth, formatter)
        end
      end
    end
  end

  test "scopes are still limited with reader tokens" do
    get('/arvados/v1/collections',
      params: {reader_tokens: [api_token(:foo_collections)]},
      headers: auth(:active_noscope))
    assert_response 403
  end

  test "reader tokens grant no permissions when expired" do
    get_collections(:active_noscope, :expired)
    assert_response 403
  end

  test "reader tokens grant no permissions outside their scope" do
    refute_includes(get_collection_uuids(:active, :admin_vm), owned_by_foo,
                    "scoped reader token granted permissions out of scope")
  end
end
