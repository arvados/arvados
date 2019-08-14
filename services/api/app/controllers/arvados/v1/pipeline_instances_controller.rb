# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class Arvados::V1::PipelineInstancesController < ApplicationController
  accept_attribute_as_json :components, Hash
  accept_attribute_as_json :properties, Hash
  accept_attribute_as_json :components_summary, Hash

  def create
    return send_error("Unsupported legacy jobs API",
                      status: 400)
  end

  def cancel
    return send_error("Unsupported legacy jobs API",
                      status: 400)
  end
end
