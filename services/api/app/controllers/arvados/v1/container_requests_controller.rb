# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'update_priorities'

class Arvados::V1::ContainerRequestsController < ApplicationController
  accept_attribute_as_json :environment, Hash
  accept_attribute_as_json :mounts, Hash
  accept_attribute_as_json :runtime_constraints, Hash
  accept_attribute_as_json :command, Array
  accept_attribute_as_json :filters, Array
  accept_attribute_as_json :scheduling_parameters, Hash
  accept_attribute_as_json :secret_mounts, Hash

  def self._index_requires_parameters
    (super rescue {}).
      merge({
        include_trash: {
          type: 'boolean', required: false, default: false, description: "Include container requests whose owner project is trashed.",
        },
      })
  end

  def self._show_requires_parameters
    (super rescue {}).
      merge({
        include_trash: {
          type: 'boolean', required: false, default: false, description: "Show container request even if its owner project is trashed.",
        },
      })
  end

  def update
    if (resource_attrs.keys - [:owner_uuid, :name, :description, :properties]).empty? or @object.container_uuid.nil?
      # If no attributes are being updated besides these, there are no
      # cascading changes to other rows/tables, the only lock will be
      # the single row lock on SQL UPDATE.
      super
    else
      # Get locks ahead of time to avoid deadlock in cascading priority
      # update
      Container.transaction do
        row_lock_for_priority_update @object.container_uuid
        super
      end
    end
  end
end
