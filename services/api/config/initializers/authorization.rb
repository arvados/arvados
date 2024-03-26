# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require_relative "../../app/middlewares/arvados_api_token"

Server::Application.configure do
  config.middleware.delete ActionDispatch::RemoteIp
  config.middleware.insert 0, ActionDispatch::RemoteIp
  config.middleware.insert 1, ArvadosApiToken
end
