class Arvados::V1::UserAgreementsController < ApplicationController
  before_filter :admin_required, except: [:index, :sign, :signatures]

  def model_class
    Link
  end

  def index
    current_user_uuid = current_user.uuid
    act_as_system_user do
      uuids = Link.where(owner_uuid: system_user_uuid,
                         link_class: 'signature',
                         name: 'require',
                         tail_kind: 'arvados#user',
                         tail_uuid: system_user_uuid,
                         head_kind: 'arvados#collection').
        collect &:head_uuid
      @objects = Collection.where('uuid in (?)', uuids)
    end
    @response_resource_name = 'collection'
    super
  end

  def signatures
    current_user_uuid = (current_user.andand.is_admin && params[:uuid]) ||
      current_user.uuid
    act_as_system_user do
      @objects = Link.where(owner_uuid: system_user_uuid,
                            link_class: 'signature',
                            name: 'click',
                            tail_kind: 'arvados#user',
                            tail_uuid: current_user_uuid,
                            head_kind: 'arvados#collection')
    end
    @response_resource_name = 'link'
    render_list
  end

  def sign
    current_user_uuid = current_user.uuid
    act_as_system_user do
      @object = Link.create(link_class: 'signature',
                            name: 'click',
                            tail_kind: 'arvados#user',
                            tail_uuid: current_user_uuid,
                            head_kind: 'arvados#collection',
                            head_uuid: params[:id])
    end
    show
  end

  def create
    usage_error
  end
  
  def new
    usage_error
  end

  def update
    usage_error
  end

  def destroy
    usage_error
  end

  protected
  def usage_error
    raise ArgumentError.new \
    "Manage user agreements via Collections and Links instead."
  end
  
end
