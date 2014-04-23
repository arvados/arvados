class Arvados::V1::ApiClientAuthorizationsController < ApplicationController
  accept_attribute_as_json :scopes, Array
  before_filter :current_api_client_is_trusted
  before_filter :admin_required, :only => :create_system_auth
  skip_before_filter :render_404_if_no_object, :only => :create_system_auth

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
          scopes: Oj.load(params[:scopes] || '["all"]'))
    @object.save!
    show
  end

  def create
    if resource_attrs[:owner_uuid]
      # The model has an owner_id attribute instead of owner_uuid, but
      # we can't expect the client to know the local numeric ID. We
      # translate UUID to numeric ID here.
      resource_attrs[:user_id] =
        User.where(uuid: resource_attrs.delete(:owner_uuid)).first.andand.id
    end
    resource_attrs[:api_client_id] = Thread.current[:api_client].id
    super
  end

  protected

  def find_objects_for_index
    # Here we are deliberately less helpful about searching for client
    # authorizations. Rather than use the generic index/where/order
    # features, we look up tokens belonging to the current user and
    # filter by exact match on api_token (which we expect in the form
    # of a where[uuid] parameter to make things easier for API client
    # libraries).
    @objects = model_class.
      includes(:user, :api_client).
      where('user_id=? and (? or api_token=?)', current_user.id, !@where['uuid'], @where['uuid']).
      order('created_at desc')
    unless @where['scopes'].nil?
      @objects = @objects.select { |auth|
        (auth.scopes & @where['scopes']) == (auth.scopes | @where['scopes'])
      }
    end
  end

  def find_object_by_uuid
    # Again, to make things easier for the client and our own routing,
    # here we look for the api_token key in a "uuid" (POST) or "id"
    # (GET) parameter.
    @object = model_class.where('api_token=?', params[:uuid] || params[:id]).first
  end

  def current_api_client_is_trusted
    unless Thread.current[:api_client].andand.is_trusted
      render :json => { errors: ['Forbidden: this API client cannot manipulate other clients\' access tokens.'] }.to_json, status: 403
    end
  end
end
