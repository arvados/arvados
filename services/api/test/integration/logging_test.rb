# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'stringio'
require 'test_helper'

class LoggingTest < ActionDispatch::IntegrationTest
  fixtures :collections

  test "request_id" do
    buf = StringIO.new
    logcopy = ActiveSupport::Logger.new(buf)
    logcopy.level = :info
    begin
      Rails.logger.broadcast_to(logcopy)
      get "/arvados/v1/collections/#{collections(:foo_file).uuid}",
          params: {:format => :json},
          headers: auth(:active).merge({ 'X-Request-Id' => 'req-aaaaaaaaaaaaaaaaaaaa' })
      assert_response :success
      assert_match /^{.*"request_id":"req-aaaaaaaaaaaaaaaaaaaa"/, buf.string
    ensure
      Rails.logger.broadcasts.delete(logcopy)
    end
  end
end
