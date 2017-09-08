# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'refresh_permission_view'

if defined?(Rails::Server)
  refresh_permission_view
end
