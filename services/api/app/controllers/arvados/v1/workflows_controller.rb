# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class Arvados::V1::WorkflowsController < ApplicationController
  def update
    if @object.collection_uuid.nil?
      # Only allowed to update directly when collection_uuid is nil (legacy behavior)
      super
    else
      raise ArvadosModel::PermissionDeniedError.new("Cannot directly update Workflow records that have collection_uuid set, must update the linked collection (#{@object.collection_uuid})")
    end
  end
end
