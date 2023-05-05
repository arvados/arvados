# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'update_priorities'

class Arvados::V1::ContainersController < ApplicationController
  accept_attribute_as_json :environment, Hash
  accept_attribute_as_json :mounts, Hash
  accept_attribute_as_json :runtime_constraints, Hash
  accept_attribute_as_json :runtime_status, Hash
  accept_attribute_as_json :command, Array
  accept_attribute_as_json :scheduling_parameters, Hash

  skip_before_action :find_object_by_uuid, only: [:current]
  skip_before_action :render_404_if_no_object, only: [:current]

  def auth
    if @object.locked_by_uuid != Thread.current[:api_client_authorization].uuid
      raise ArvadosModel::PermissionDeniedError.new("Not locked by your token")
    end
    if @object.runtime_token.nil?
      @object = @object.auth
    else
      @object = ApiClientAuthorization.validate(token: @object.runtime_token)
      if @object.nil?
        raise ArvadosModel::PermissionDeniedError.new("Invalid runtime_token")
      end
    end
    show
  end

  def update
    if (resource_attrs.keys - [:cost, :gateway_address, :output_properties, :progress, :runtime_status]).empty?
      # If no attributes are being updated besides these, there are no
      # cascading changes to other rows/tables, the only lock will the
      # single row lock on SQL UPDATE.
      super
    else
      Container.transaction do
        # Get locks ahead of time to avoid deadlock in cascading priority
        # update
        row_lock_for_priority_update @object.uuid
        super
      end
    end
  end

  def find_objects_for_index
    super
    if action_name == 'lock' || action_name == 'unlock'
      # Avoid loading more fields than we need
      @objects = @objects.select(:id, :uuid, :state, :priority, :auth_uuid, :locked_by_uuid, :lock_count)
      # This gets called from within find_object_by_uuid.
      # find_object_by_uuid stores the original value of @select in
      # @preserve_select, edits the value of @select, calls
      # find_objects_for_index, then restores @select from the value
      # of @preserve_select.  So if we want our updated value of
      # @select here to stick, we have to set @preserve_select.
      @select = @preserve_select = %w(uuid state priority auth_uuid locked_by_uuid)
    elsif action_name == 'update_priority'
      # We're going to reload in update_priority!, which will select
      # all attributes, but will fail if we don't select :id now.
      @objects = @objects.select(:id, :uuid)
    end
  end

  def lock
    @object.lock
    show
  end

  def unlock
    @object.unlock
    show
  end

  def update_priority
    @object.update_priority!
    show
  end

  def current
    if Thread.current[:api_client_authorization].nil?
      send_error("Not logged in", status: 401)
    else
      @object = Container.for_current_token
      if @object.nil?
        send_error("Token is not associated with a container.", status: 404)
      else
        show
      end
    end
  end

  def secret_mounts
    c = Container.for_current_token
    if @object && c && @object.uuid == c.uuid
      send_json({"secret_mounts" => @object.secret_mounts})
    else
      send_error("Token is not associated with this container.", status: 403)
    end
  end
end
