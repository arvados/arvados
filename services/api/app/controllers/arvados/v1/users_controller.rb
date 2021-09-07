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
    params[:updates].andand.each do |uuid, attrs|
      begin
        u = User.find_or_create_by(uuid: uuid)
      rescue ActiveRecord::RecordNotUnique
        retry
      end
      needupdate = {}
      nullify_attrs(attrs).each do |k,v|
        if !v.nil? && u.send(k) != v
          needupdate[k] = v
        end
      end
      if needupdate.length > 0
        begin
          u.update_attributes!(needupdate)
        rescue ActiveRecord::RecordInvalid
          loginCluster = Rails.configuration.Login.LoginCluster
          if u.uuid[0..4] == loginCluster && !needupdate[:username].nil?
            local_user = User.find_by_username(needupdate[:username])
            # A cached user record from the LoginCluster is stale, reset its username
            # and retry the update operation.
            if local_user.andand.uuid[0..4] == loginCluster && local_user.uuid != u.uuid
              new_username = "#{needupdate[:username]}conflict#{rand(99999999)}"
              Rails.logger.warn("cached username '#{needupdate[:username]}' collision with user '#{local_user.uuid}' - renaming to '#{new_username}' before retrying")
              local_user.update_attributes!({username: new_username})
              retry
            end
          end
          raise # Not the issue we're handling above
        end
      end
      @objects << u
    end
    @offset = 0
    @limit = -1
    render_list
  end

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
    elsif !params[:user] || params[:user].empty?
      raise ArgumentError.new "Required uuid or user"
    elsif !params[:user]['email']
      raise ArgumentError.new "Require user email"
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
                              send_notification_email: params[:send_notification_email])

    send_json kind: "arvados#HashList", items: @response.as_api_response(nil)
  end

  # delete user agreements, vm, repository, login links; set state to inactive
  def unsetup
    reload_object_before_update
    @object.unsetup
    show
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
      if !Thread.current[:api_client].andand.is_trusted
        return send_error("supplied API token is not from a trusted client", status: 403)
      elsif Thread.current[:api_client_authorization].scopes != ['all']
        return send_error("cannot merge with a scoped token", status: 403)
      end

      new_auth = ApiClientAuthorization.validate(token: params[:new_user_token])
      if !new_auth
        return send_error("invalid new_user_token", status: 401)
      end

      if new_auth.user.uuid[0..4] == Rails.configuration.ClusterID
        if !new_auth.api_client.andand.is_trusted
          return send_error("supplied new_user_token is not from a trusted client", status: 403)
        elsif new_auth.scopes != ['all']
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
        type: 'string', required: true,
      },
      new_user_token: {
        type: 'string', required: false,
      },
      redirect_to_new_user: {
        type: 'boolean', required: false, default: false,
      },
      old_user_uuid: {
        type: 'string', required: false,
      },
      new_user_uuid: {
        type: 'string', required: false,
      }
    }
  end

  def self._setup_requires_parameters
    {
      uuid: {
        type: 'string', required: false,
      },
      user: {
        type: 'object', required: false,
      },
      repo_name: {
        type: 'string', required: false,
      },
      vm_uuid: {
        type: 'string', required: false,
      },
      send_notification_email: {
        type: 'boolean', required: false, default: false,
      },
    }
  end

  def self._update_requires_parameters
    super.merge({
      bypass_federation: {
        type: 'boolean', required: false, default: false,
      },
    })
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

  def nullable_attributes
    super + [:email, :first_name, :last_name, :username]
  end
end
