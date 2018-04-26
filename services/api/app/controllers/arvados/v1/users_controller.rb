# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

class Arvados::V1::UsersController < ApplicationController
  accept_attribute_as_json :prefs, Hash

  skip_before_filter :find_object_by_uuid, only:
    [:activate, :current, :system, :setup]
  skip_before_filter :render_404_if_no_object, only:
    [:activate, :current, :system, :setup]
  before_filter :admin_required, only: [:setup, :unsetup, :update_uuid]

  def current
    if current_user
      @object = current_user
      show
    else
      send_error("Not logged in", status: 401)
    end
  end

  def system
    @object = system_user
    show
  end

  def activate
    if current_user.andand.is_admin && params[:uuid]
      @object = User.find params[:uuid]
    else
      @object = current_user
    end
    if not @object.is_active
      if not (current_user.is_admin or @object.is_invited)
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
          @object.update_attributes is_active: true
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

  # create user object and all the needed links
  def setup
    if params[:uuid]
      @object = User.find_by_uuid(params[:uuid])
      if !@object
        return render_404_if_no_object
      end
    elsif !params[:user]
      raise ArgumentError.new "Required uuid or user"
    elsif !params[:user]['email']
      raise ArgumentError.new "Require user email"
    elsif !params[:openid_prefix]
      raise ArgumentError.new "Required openid_prefix parameter is missing."
    else
      @object = model_class.create! resource_attrs
    end

    # It's not always possible for the client to know the user's
    # username when submitting this request: the username might have
    # been assigned automatically in create!() above. If client
    # provided a plain repository name, prefix it with the username
    # now that we know what it is.
    if params[:repo_name].nil?
      full_repo_name = nil
    elsif @object.username.nil?
      raise ArgumentError.
        new("cannot setup a repository because user has no username")
    elsif params[:repo_name].index("/")
      full_repo_name = params[:repo_name]
    else
      full_repo_name = "#{@object.username}/#{params[:repo_name]}"
    end

    @response = @object.setup(repo_name: full_repo_name,
                              vm_uuid: params[:vm_uuid],
                              openid_prefix: params[:openid_prefix])

    # setup succeeded. send email to user
    if params[:send_notification_email]
      UserNotifier.account_is_setup(@object).deliver_now
    end

    send_json kind: "arvados#HashList", items: @response.as_api_response(nil)
  end

  # delete user agreements, vm, repository, login links; set state to inactive
  def unsetup
    reload_object_before_update
    @object.unsetup
    show
  end

  # Change UUID to a new (unused) uuid and transfer all owned/linked
  # objects accordingly.
  def update_uuid
    @object.update_uuid(new_uuid: params[:new_uuid])
    show
  end

  protected

  def self._setup_requires_parameters
    {
      user: {
        type: 'object', required: false
      },
      openid_prefix: {
        type: 'string', required: false
      },
      repo_name: {
        type: 'string', required: false
      },
      vm_uuid: {
        type: 'string', required: false
      },
      send_notification_email: {
        type: 'boolean', required: false, default: false
      },
    }
  end

  def self._update_uuid_requires_parameters
    {
      new_uuid: {
        type: 'string', required: true,
      },
    }
  end

  def apply_filters(model_class=nil)
    return super if @read_users.any?(&:is_admin)
    if params[:uuid] != current_user.andand.uuid
      # Non-admin index/show returns very basic information about readable users.
      safe_attrs = ["uuid", "is_active", "email", "first_name", "last_name", "username"]
      if @select
        @select = @select & safe_attrs
      else
        @select = safe_attrs
      end
      @filters += [['is_active', '=', true]]
    end
    super
  end
end
