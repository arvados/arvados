# Copyright (C) The Arvados Authors. All rights reserved.
#
# SPDX-License-Identifier: AGPL-3.0

require 'safe_json'

class Arvados::V1::ApiClientAuthorizationsController < ApplicationController
  accept_attribute_as_json :scopes, Array
  before_action :current_api_client_is_trusted, :except => [:current]
  before_action :admin_required, :only => :create_system_auth
  skip_before_action :render_404_if_no_object, :only => [:create_system_auth, :current]
  skip_before_action :find_object_by_uuid, :only => [:create_system_auth, :current]

  def self._create_system_auth_requires_parameters
    {
      api_client_id: {type: 'integer', required: false},
      scopes: {type: 'array', required: false}
    }
  end

  def create_system_auth
    @object = ApiClientAuthorization.
      new(user_id: system_user.id,
          api_client_id: params[:api_client_id] || current_api_client.andand.id,
          created_by_ip_address: remote_ip,
          scopes: SafeJSON.load(params[:scopes] || '["all"]'))
    @object.save!
    show
  end

  def create
    # Note: the user could specify a owner_uuid for a different user, which on
    # the surface appears to be a security hole.  However, the record will be
    # rejected before being saved to the database by the ApiClientAuthorization
    # model which enforces that user_id == current user or the user is an
    # admin.

    if resource_attrs[:owner_uuid]
      # The model has an owner_id attribute instead of owner_uuid, but
      # we can't expect the client to know the local numeric ID. We
      # translate UUID to numeric ID here.
      resource_attrs[:user_id] =
        User.where(uuid: resource_attrs.delete(:owner_uuid)).first.andand.id
    elsif not resource_attrs[:user_id]
      resource_attrs[:user_id] = current_user.id
    end
    resource_attrs[:api_client_id] = Thread.current[:api_client].id
    super
  end

  def current
    @object = Thread.current[:api_client_authorization].dup
    if params[:remote]
      # Client is validating a salted token. Don't return the unsalted
      # secret!
      @object.api_token = nil
    end
    show
  end

  protected

  def default_orders
    ["#{table_name}.created_at desc"]
  end

  def find_objects_for_index
    # Here we are deliberately less helpful about searching for client
    # authorizations.  We look up tokens belonging to the current user
    # and filter by exact matches on uuid, api_token, and scopes.
    wanted_scopes = []
    if @filters
      wanted_scopes.concat(@filters.map { |attr, operator, operand|
        ((attr == 'scopes') and (operator == '=')) ? operand : nil
      })
      @filters.select! { |attr, operator, operand|
        operator == '=' && (attr == 'uuid' || attr == 'api_token')
      }
    end
    if @where
      wanted_scopes << @where['scopes']
      @where.select! { |attr, val|
        # "where":{"uuid":"zzzzz-zzzzz-zzzzzzzzzzzzzzz"} is OK but
        # "where":{"api_client_id":1} is not supported
        # "where":{"uuid":["contains","-"]} is not supported
        # "where":{"uuid":["uuid1","uuid2","uuid3"]} is not supported
        val.is_a?(String) && (attr == 'uuid' || attr == 'api_token')
      }
    end
    if current_api_client_authorization.andand.api_token != Rails.configuration.SystemRootToken
      @objects = model_class.where('user_id=?', current_user.id)
    end
    if wanted_scopes.compact.any?
      # We can't filter on scopes effectively using AR/postgres.
      # Instead we get the entire result set, do our own filtering on
      # scopes to get a list of UUIDs, then start a new query
      # (restricted to the selected UUIDs) so super can apply the
      # offset/limit/order params in the usual way.
      @request_limit = @limit
      @request_offset = @offset
      @limit = @objects.count
      @offset = 0
      super
      wanted_scopes.compact.each do |scope_list|
        if @objects.respond_to?(:where) && scope_list.length < 2
          @objects = @objects.
                     where('scopes in (?)',
                           [scope_list.to_yaml, SafeJSON.dump(scope_list)])
        else
          if @objects.respond_to?(:where)
            # Eliminate rows with scopes=['all'] before doing the
            # expensive filter. They are typically the majority of
            # rows, and they obviously won't match given
            # scope_list.length>=2, so loading them all into
            # ActiveRecord objects is a huge waste of time.
            @objects = @objects.
                       where('scopes not in (?)',
                             [['all'].to_yaml, SafeJSON.dump(['all'])])
          end
          sorted_scopes = scope_list.sort
          @objects = @objects.select { |auth| auth.scopes.sort == sorted_scopes }
        end
      end
      @limit = @request_limit
      @offset = @request_offset
      @objects = model_class.where('uuid in (?)', @objects.collect(&:uuid))
    end
    super
  end

  def find_object_by_uuid(with_lock: false)
    uuid_param = params[:uuid] || params[:id]
    if (uuid_param != current_api_client_authorization.andand.uuid &&
        !Thread.current[:api_client].andand.is_trusted)
      return forbidden
    end
    @limit = 1
    @offset = 0
    @orders = []
    @where = {}
    @filters = [['uuid', '=', uuid_param]]
    find_objects_for_index
    query = @objects
    if with_lock && Rails.configuration.API.LockBeforeUpdate
      query = query.lock
    end
    @object = query.first
  end

  def current_api_client_is_trusted
    if Thread.current[:api_client].andand.is_trusted
      return true
    end
    # A non-trusted client can do a search for its own token if it
    # explicitly restricts the search to its own UUID or api_token.
    # Any other kind of query must return 403, even if it matches only
    # the current token, because that's currently how Workbench knows
    # (after searching on scopes) the difference between "the token
    # I'm using now *is* the only sharing token for this collection"
    # (403) and "my token is trusted, and there is one sharing token
    # for this collection" (200).
    #
    # The @filters test here also prevents a non-trusted token from
    # filtering on its own scopes, and discovering whether any _other_
    # equally scoped tokens exist (403=yes, 200=no).
    return forbidden if !@objects
    full_set = @objects.except(:limit).except(:offset) if @objects
    if (full_set.count == 1 and
        full_set.first.uuid == current_api_client_authorization.andand.uuid and
        (@filters.map(&:first) & %w(uuid api_token)).any?)
      return true
    end
    forbidden
  end

  def forbidden
    send_error('Forbidden: this API client cannot manipulate other clients\' access tokens.',
               status: 403)
  end
end
