# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'integration_helper'
require 'helpers/repository_stub_helper'
require 'helpers/share_object_helper'

class RepositoriesTest < ActionDispatch::IntegrationTest
  include RepositoryStubHelper
  include ShareObjectHelper

  reset_api_fixtures :after_each_test, false

  setup do
    need_javascript
  end

  test "browse using arv-git-http" do
    repo = api_fixture('repositories')['foo']
    commit_sha1 = '1de84a854e2b440dc53bf42f8548afa4c17da332'
    visit page_with_token('active', "/repositories/#{repo['uuid']}/commit/#{commit_sha1}")
    assert_text "Date:   Tue Mar 18 15:55:28 2014 -0400"
    visit page_with_token('active', "/repositories/#{repo['uuid']}/tree/#{commit_sha1}")
    assert_selector "tbody td a", "foo"
    assert_text "12 bytes"
  end
end
