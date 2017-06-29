# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class WebsocketController < ApplicationController
  skip_before_filter :find_objects_for_index

  def index
  end

  def model_class
    "Websocket"
  end
end
