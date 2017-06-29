# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class LogsController < ApplicationController
  before_filter :ensure_current_user_is_admin
end
