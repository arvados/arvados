class Arvados::V1::UsersController < ApplicationController
  skip_before_filter :find_object_by_uuid, only:
    [:activate, :event_stream, :current, :system, :setup]
  skip_before_filter :render_404_if_no_object, only:
    [:activate, :event_stream, :current, :system, :setup]
  before_filter :admin_required, only: [:setup, :unsetup]

  def current
    @object = current_user
    show
  end
  def system
    @object = system_user
    show
  end

  class ChannelStreamer
    Q_UPDATE_INTERVAL = 12
    def initialize(opts={})
      @opts = opts
    end
    def each
      return unless @opts[:channel]
      @redis = Redis.new(:timeout => 0)
      @redis.subscribe(@opts[:channel]) do |event|
        event.message do |channel, msg|
          yield msg + "\n"
        end
      end
    end
  end

  def event_stream
    channel = current_user.andand.uuid
    if current_user.andand.is_admin
      channel = params[:uuid] || channel
    end
    if client_accepts_plain_text_stream
      self.response.headers['Last-Modified'] = Time.now.ctime.to_s
      self.response_body = ChannelStreamer.new(channel: channel)
    else
      render json: {
        href: url_for(uuid: channel),
        comment: ('To retrieve the event stream as plain text, ' +
                  'use a request header like "Accept: text/plain"')
      }
    end
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
        if todo_uuids == []
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
    @object = nil
    if params[:uuid]
      @object = User.find_by_uuid params[:uuid]
      if !@object
        return render_404_if_no_object
      end
      object_found = true
    else
      if !params[:user]
        raise ArgumentError.new "Required uuid or user"
      else
        if params[:user]['uuid']
          @object = User.find_by_uuid params[:user]['uuid']
          if @object
            object_found = true
          end
        end

        if !@object
          if !params[:user]['email']
            raise ArgumentError.new "Require user email"
          end

          if !params[:openid_prefix]
            raise ArgumentError.new "Required openid_prefix parameter is missing."
          end

          @object = model_class.create! resource_attrs
        end
      end
    end

    if object_found
      @response = @object.setup_repo_vm_links params[:repo_name],
                    params[:vm_uuid], params[:openid_prefix]
    else
      @response = User.setup @object, params[:openid_prefix],
                    params[:repo_name], params[:vm_uuid]
    end

    # setup succeeded. send email to user
    if params[:send_notification_email] == true || params[:send_notification_email] == 'true'
      UserNotifier.account_is_setup(@object).deliver
    end

    render json: { kind: "arvados#HashList", items: @response.as_api_response(nil) }
  end

  # delete user agreements, vm, repository, login links; set state to inactive
  def unsetup
    reload_object_before_update
    @object.unsetup
    show
  end

  protected

  def self._setup_requires_parameters
    {
      send_notification_email: { type: 'boolean', required: true },
    }
  end

end
