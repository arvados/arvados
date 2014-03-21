class Arvados::V1::UsersController < ApplicationController
  skip_before_filter :find_object_by_uuid, only:
    [:activate, :event_stream, :current, :system, :setup]
  skip_before_filter :render_404_if_no_object, only:
    [:activate, :event_stream, :current, :system, :setup]

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
        required_uuids = Link.where(owner_uuid: system_user_uuid,
                                    link_class: 'signature',
                                    name: 'require',
                                    tail_uuid: system_user_uuid,
                                    head_kind: 'arvados#collection').
          collect(&:head_uuid)
        signed_uuids = Link.where(owner_uuid: system_user_uuid,
                                  link_class: 'signature',
                                  name: 'click',
                                  tail_kind: 'arvados#user',
                                  tail_uuid: @object.uuid,
                                  head_kind: 'arvados#collection',
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
    # check if default openid_prefix needs to be overridden
    if params[:openid_prefix]
      openid_prefix = params[:openid_prefix]
    else 
      openid_prefix = Rails.configuration.openid_prefix
    end
    login_perm_props = {identity_url_prefix: openid_prefix}

    @object = model_class.new resource_attrs

    # Lookup for user. If exists, only create any missing links
    @object_found = find_user_from_input 

    if !@object_found
      if !@object[:email]
        raise "No email found in the input. Aborting user creation."
      end

      if @object.save
        oid_login_perm = Link.where(tail_uuid: @object[:email],
                                    head_kind: 'arvados#user',
                                    link_class: 'permission',
                                    name: 'can_login')

        if [] == oid_login_perm
          # create openid login permission
          oid_login_perm = Link.create(link_class: 'permission',
                                       name: 'can_login',
                                       tail_kind: 'email',
                                       tail_uuid: @object[:email],
                                       head_kind: 'arvados#user',
                                       head_uuid: @object[:uuid],
                                       properties: login_perm_props
                                      )
          logger.info { "openid login permission: " + oid_login_perm[:uuid] }
        end
      else
        raise "Save failed"
      end
    else
      @object = @object_found
    end
    
    # create links
    create_user_repo_link params[:repo_name]
    create_vm_login_permission_link params[:vm_uuid], params[:repo_name]
    create_user_group_link 

    show  
  end

  protected 

  # find the user from the given user parameters
  def find_user_from_input
    if @object[:uuid]
      found_object = User.find_by_uuid @object[:uuid]
    end

    if !found_object
      if !@object[:email]
        return
      end

      found_objects = User.where('email=?', @object[:email])  
      found_object = found_objects.first
    end

    return found_object
  end
  
  # link the repo_name passed
  def create_user_repo_link(repo_name)
    if not repo_name
      logger.warn ("Repository name not given for #{@object[:uuid]}.")
      return
    end

    # Check for an existing repository with the same name we're about to use.
    repo = (repos = Repository.where(name: repo_name)) != nil ? repos.first : nil
    if repo
      logger.warn "Repository exists for #{repo_name}: #{repo[:uuid]}."

      # Look for existing repository access for this repo
      repo_perms = Link.where(tail_uuid: @object[:uuid],
                              head_kind: 'arvados#repository',
                              head_uuid: repo[:uuid],
                              link_class: 'permission',
                              name: 'can_write')
      if [] != repo_perms
        logger.warn "User already has repository access " + 
            repo_perms.collect { |p| p[:uuid] }.inspect
        return
      end
    end

    # create repo, if does not already exist
    repo ||= Repository.create(name: repo_name)
    logger.info { "repo uuid: " + repo[:uuid] }

    repo_perm = Link.create(tail_kind: 'arvados#user',
                            tail_uuid: @object[:uuid],
                            head_kind: 'arvados#repository',
                            head_uuid: repo[:uuid],
                            link_class: 'permission',
                            name: 'can_write')
    logger.info { "repo permission: " + repo_perm[:uuid] }
  end

  # create login permission for the given vm_uuid, if it does not already exist
  def create_vm_login_permission_link(vm_uuid, repo_name)
    # Look up the given virtual machine just to make sure it really exists.
    begin
      vm = (vms = VirtualMachine.where(uuid: vm_uuid)) != nil ? vms.first : nil
      if not vm
        logger.warn "Could not find virtual machine for #{vm_uuid.inspect}"
        return
      end

      logger.info { "vm uuid: " + vm[:uuid] }

      login_perm = Link.where(tail_uuid: @object[:uuid],
                              head_uuid: vm[:uuid],
                              head_kind: 'arvados#virtualMachine',
                              link_class: 'permission',
                              name: 'can_login')
      if [] == login_perm
        login_perm = Link.create(tail_kind: 'arvados#user',
                                 tail_uuid: @object[:uuid],
                                 head_kind: 'arvados#virtualMachine',
                                 head_uuid: vm[:uuid],
                                 link_class: 'permission',
                                 name: 'can_login',
                                 properties: {username: repo_name})
        logger.info { "login permission: " + login_perm[:uuid] }
      end
    end
  end

  # add the user to the 'All users' group
  def create_user_group_link
    # Look up the "All users" group (we expect uuid *-*-fffffffffffffff).
    group = Group.where(name: 'All users').select do |g|
      g[:uuid].match /-f+$/
    end.first

    if not group
      logger.warn "No 'All users' group with uuid '*-*-fffffffffffffff'."
      return
    else
      logger.info { "\"All users\" group uuid: " + group[:uuid] }

      group_perm = Link.where(tail_uuid: @object[:uuid],
                              head_uuid: group[:uuid],
                              head_kind: 'arvados#group',
                              link_class: 'permission',
                              name: 'can_read')

      if [] == group_perm
        group_perm = Link.create(tail_kind: 'arvados#user',
                                 tail_uuid: @object[:uuid],
                                 head_kind: 'arvados#group',
                                 head_uuid: group[:uuid],
                                 link_class: 'permission',
                                 name: 'can_read')
        logger.info { "group permission: " + group_perm[:uuid] }
      end
    end
  end

end
