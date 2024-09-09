# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class Arvados::V1::UsersController < ApplicationController
  accept_attribute_as_json :prefs, Hash
  accept_param_as_json :updates

  skip_before_action :find_object_by_uuid, only:
    [:activate, :current, :system, :setup, :merge, :batch_update]
  skip_before_action :render_404_if_no_object, only:
    [:activate, :current, :system, :setup, :merge, :batch_update]
  before_action :admin_required, only: [:setup, :unsetup, :batch_update]

  # Internal API used by controller to update local cache of user
  # records from LoginCluster.
  def batch_update
    @objects = []
    # update_remote_user takes a row lock on the User record, so sort
    # the keys so we always lock them in the same order.
    sorted = params[:updates].keys.sort
    sorted.each do |uuid|
      attrs = params[:updates][uuid]
      attrs[:uuid] = uuid
      u = User.update_remote_user nullify_attrs(attrs)
      @objects << u
    end
    @offset = 0
    @limit = -1
    render_list
  end

  def self._current_method_description
    "Return the user record associated with the API token authorizing this request."
  end

  def current
    if current_user
      @object = current_user
      show
    else
      send_error("Not logged in", status: 401)
    end
  end

  def self._system_method_description
    "Return this cluster's system (\"root\") user record."
  end

  def system
    @object = system_user
    show
  end

  def self._activate_method_description
    "Set the `is_active` flag on a user record."
  end

  def activate
    if params[:id] and params[:id].match(/\D/)
      params[:uuid] = params.delete :id
    end
    if current_user.andand.is_admin && params[:uuid]
      @object = User.find_by_uuid params[:uuid]
    else
      @object = current_user
    end
    if not @object.is_active
      if @object.uuid[0..4] == Rails.configuration.Login.LoginCluster &&
         @object.uuid[0..4] != Rails.configuration.ClusterID
        logger.warn "Local user #{@object.uuid} called users#activate but only LoginCluster can do that"
        raise ArgumentError.new "cannot activate user #{@object.uuid} here, only the #{@object.uuid[0..4]} cluster can do that"
      elsif not (current_user.is_admin or @object.is_invited)
        logger.warn "User #{@object.uuid} called users.activate " +
          "but is not invited"
        raise ArgumentError.new "Cannot activate without being invited."
      end
      act_as_system_user do
        required_uuids = Link.where("owner_uuid = ? and link_class = ? and name = ? and tail_uuid = ? and head_uuid like ?",
                                    system_user_uuid,
                                    'signature',
                                    'require',
                                    system_user_uuid,
                                    Collection.uuid_like_pattern).
          collect(&:head_uuid)
        signed_uuids = Link.where(owner_uuid: system_user_uuid,
                                  link_class: 'signature',
                                  name: 'click',
                                  tail_uuid: @object.uuid,
                                  head_uuid: required_uuids).
          collect(&:head_uuid)
        todo_uuids = required_uuids - signed_uuids
        if todo_uuids.empty?
          @object.update is_active: true
          logger.info "User #{@object.uuid} activated"
        else
          logger.warn "User #{@object.uuid} called users.activate " +
            "before signing agreements #{todo_uuids.inspect}"
          raise ArvadosModel::PermissionDeniedError.new \
          "Cannot activate without user agreements #{todo_uuids.inspect}."
        end
      end
    end
    show
  end

  def self._setup_method_description
    "Convenience method to \"fully\" set up a user record with a virtual machine login and notification email."
  end

  # create user object and all the needed links
  def setup
    if params[:uuid]
      @object = User.find_by_uuid(params[:uuid])
      if !@object
        return render_404_if_no_object
      end
    elsif !params[:user] || params[:user].empty?
      raise ArgumentError.new "Required uuid or user"
    elsif !params[:user]['email']
      raise ArgumentError.new "Require user email"
    else
      @object = model_class.create! resource_attrs
    end

    @response = @object.setup(vm_uuid: params[:vm_uuid],
                              send_notification_email: params[:send_notification_email])

    send_json kind: "arvados#HashList", items: @response.as_api_response(nil)
  end

  def self._unsetup_method_description
    "Unset a user's active flag and delete associated records."
  end

  # delete user agreements, vm, repository, login links; set state to inactive
  def unsetup
    reload_object_before_update
    @object.unsetup
    show
  end

  def self._merge_method_description
    "Transfer ownership of one user's data to another."
  end

  def merge
    if (params[:old_user_uuid] || params[:new_user_uuid])
      if !current_user.andand.is_admin
        return send_error("Must be admin to use old_user_uuid/new_user_uuid", status: 403)
      end
      if !params[:old_user_uuid] || !params[:new_user_uuid]
        return send_error("Must supply both old_user_uuid and new_user_uuid", status: 422)
      end
      new_user = User.find_by_uuid(params[:new_user_uuid])
      if !new_user
        return send_error("User in new_user_uuid not found", status: 422)
      end
      @object = User.find_by_uuid(params[:old_user_uuid])
      if !@object
        return send_error("User in old_user_uuid not found", status: 422)
      end
    else
      if Thread.current[:api_client_authorization].scopes != ['all']
        return send_error("cannot merge with a scoped token", status: 403)
      end

      new_auth = ApiClientAuthorization.validate(token: params[:new_user_token])
      if !new_auth
        return send_error("invalid new_user_token", status: 401)
      end

      if new_auth.user.uuid[0..4] == Rails.configuration.ClusterID
        if new_auth.scopes != ['all']
          return send_error("supplied new_user_token has restricted scope", status: 403)
        end
      end
      new_user = new_auth.user
      @object = current_user
    end

    if @object.uuid == new_user.uuid
      return send_error("cannot merge user to self", status: 422)
    end

    if !params[:new_owner_uuid]
      return send_error("missing new_owner_uuid", status: 422)
    end

    if !new_user.can?(write: params[:new_owner_uuid])
      return send_error("cannot move objects into supplied new_owner_uuid: new user does not have write permission", status: 403)
    end

    act_as_system_user do
      @object.merge(new_owner_uuid: params[:new_owner_uuid],
                    new_user_uuid: new_user.uuid,
                    redirect_to_new_user: params[:redirect_to_new_user])
    end
    show
  end

  protected

  def self._merge_requires_parameters
    {
      new_owner_uuid: {
        type: 'string',
        required: true,
        description: "UUID of the user or group that will take ownership of data owned by the old user.",
      },
      new_user_token: {
        type: 'string',
        required: false,
        description: "Valid API token for the user receiving ownership. If you use this option, it takes ownership of data owned by the user making the request.",
      },
      redirect_to_new_user: {
        type: 'boolean',
        required: false,
        default: false,
        description: "If true, authorization attempts for the old user will be redirected to the new user.",
      },
      old_user_uuid: {
        type: 'string',
        required: false,
        description: "UUID of the user whose ownership is being transferred to `new_owner_uuid`. You must be an admin to use this option.",
      },
      new_user_uuid: {
        type: 'string',
        required: false,
        description: "UUID of the user receiving ownership. You must be an admin to use this option.",
      }
    }
  end

  def self._setup_requires_parameters
    {
      uuid: {
        type: 'string',
        required: false,
        description: "UUID of an existing user record to set up."
      },
      user: {
        type: 'object',
        required: false,
        description: "Attributes of a new user record to set up.",
      },
      repo_name: {
        type: 'string',
        required: false,
        description: "This parameter is obsolete and ignored.",
      },
      vm_uuid: {
        type: 'string',
        required: false,
        description: "If given, setup creates a login link to allow this user to access the Arvados virtual machine with this UUID.",
      },
      send_notification_email: {
        type: 'boolean',
        required: false,
        default: false,
        description: "If true, send an email to the user notifying them they can now access this Arvados cluster.",
      },
    }
  end

  def self._update_requires_parameters
    super.merge({
      bypass_federation: {
        type: 'boolean',
        required: false,
        default: false,
        description: "If true, do not try to update the user on any other clusters in the federation,
only the cluster that received the request.
You must be an administrator to use this flag.",
      },
    })
  end

  def apply_filters(model_class=nil)
    return super if @read_users.any?(&:is_admin)
    if params[:uuid] != current_user.andand.uuid
      # Non-admin index/show returns very basic information about readable users.
      safe_attrs = ["uuid", "is_active", "is_admin", "is_invited", "email", "first_name", "last_name", "username", "can_write", "can_manage", "kind"]
      if @select
        @select = @select & safe_attrs
      else
        @select = safe_attrs
      end
      @filters += [['is_active', '=', true]]
    end
    # This gets called from within find_object_by_uuid.
    # find_object_by_uuid stores the original value of @select in
    # @preserve_select, edits the value of @select, calls
    # find_objects_for_index, then restores @select from the value
    # of @preserve_select.  So if we want our updated value of
    # @select here to stick, we have to set @preserve_select.
    @preserve_select = @select
    super
  end

  def nullable_attributes
    super + [:email, :first_name, :last_name, :username]
  end
end
