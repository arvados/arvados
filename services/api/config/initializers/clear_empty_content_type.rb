# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

# Rails handler stack crashes if the request Content-Type header value
# is "", which is sometimes the case in GET requests from
# ruby-google-api-client (which have no body content anyway).
#
# This middleware deletes such headers, so a request with an empty
# Content-Type value is equivalent to a missing Content-Type header.
class ClearEmptyContentType
  def initialize(app=nil, options=nil)
    @app = app
  end

  def call(env)
    if env["CONTENT_TYPE"] == ""
      env.delete("CONTENT_TYPE")
    end
    @app.call(env) if @app.respond_to?(:call)
  end
end

Server::Application.configure do
  config.middleware.use ClearEmptyContentType
end
