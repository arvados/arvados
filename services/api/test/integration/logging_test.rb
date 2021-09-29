# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'stringio'

class LoggingTest < ActionDispatch::IntegrationTest
  fixtures :collections

  test "request_id" do
    buf = StringIO.new
    logcopy = ActiveSupport::Logger.new(buf)
    logcopy.level = :info
    begin
      Rails.logger.extend(ActiveSupport::Logger.broadcast(logcopy))
      get "/arvados/v1/collections/#{collections(:foo_file).uuid}",
          params: {:format => :json},
          headers: auth(:active).merge({ 'X-Request-Id' => 'req-aaaaaaaaaaaaaaaaaaaa' })
      assert_response :success
      assert_match /^{.*"request_id":"req-aaaaaaaaaaaaaaaaaaaa"/, buf.string
    ensure
      # We don't seem to have an "unbroadcast" option, so this is how
      # we avoid filling buf with unlimited logs from subsequent
      # tests.
      logcopy.level = :fatal
    end
  end
end
