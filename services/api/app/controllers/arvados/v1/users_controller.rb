class Arvados::V1::UsersController < ApplicationController
  skip_before_filter :find_object_by_uuid, only:
    [:activate, :event_stream, :current, :system]
  skip_before_filter :render_404_if_no_object, only:
    [:activate, :event_stream, :current, :system]

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
  def create
		if params[:openid_prefix]		# check if default openid_prefix needs to be overridden
			openid_prefix = params[:openid_prefix]
		else 
			openid_prefix = 'https://www.google.com/accounts/o8/id'		# default openid prefix
		end
		login_perm_props = {identity_url_prefix: openid_prefix}

		# check if only to probe the given user parameter
		just_probe = (params[:just_probe] == 'true') ? true : false;

 		@object = model_class.new resource_attrs

		# If user_param is passed, lookup for user. If exists, skip create and create any missing links. 
		if params[:user_param]
			begin 
	 			@object_found = find_user_from_user_param params[:user_param]
		  end

			if !@object_found
				@object = User.new		# when user_param is used, it will be used as user object
				@object[:email] = params[:user_param]				
  	 		need_to_create = true
			else
				@object = @object_found
			end
		else		# need to create user for the given :user data
			need_to_create = true
		end

		# if just probing, return any object found	
		if just_probe 
			@object[:email] = nil	
			show
		  return
		end

		# create if need be, and then create or update the links as needed 
		if need_to_create
			if @object.save		# save succeeded
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
		end

		# create links
		create_user_repo_link params[:repo_name]
		create_vm_login_permission_link params[:vm_uuid], params[:repo_name]
		create_user_group_link 

		show
  end

	protected 

	# find the user from the given user parameter
	def find_user_from_user_param(user_param)
		found_object = User.find_by_uuid user_param

		if !found_object
			begin
				if !user_param.match(/\w\@\w+\.\w+/)
					logger.warn ("Given user param is not valid email format: #{user_param}")
					raise ArgumentError.new "User param is not of valid email format. Stop"
				else
          found_objects = User.where('email=?', user_param)  
       
				 	if found_objects.size > 1
						logger.warn ("Found #{found_objects.size} users with email #{user_param}. Stop.")
						raise ArgumentError.new "Found #{found_objects.size} users with email #{user_param}. Stop."
					elsif found_objects.size == 1
						found_object = found_objects.first
					end
        end
   		end
		end

		return found_object
	end
	
	# link the repo_name passed
	def create_user_repo_link(repo_name)
		if not repo_name
			logger.warn ("Repository name not given for #{@object[:uuid]}. Skip creating the link")
			return
		end

		# Check for an existing repository with the same name we're about to use.
		repo = (repositories = Repository.where(name: repo_name)) != nil ? repositories.first : nil
		if repo
  		logger.warn "Repository already exists with name #{repo_name}: #{repo[:uuid]}. Will link to user."

			# Look for existing repository access (perhaps using a different repository/user name).
			repo_perms = Link.where(tail_uuid: @object[:uuid],
    	                        head_kind: 'arvados#repository',
    	                        head_uuid: repo[:uuid],
    	                        link_class: 'permission',
    	                        name: 'can_write')
			if [] != repo_perms
  			logger.warn "User already has repository access " + repo_perms.collect { |p| p[:uuid] }.inspect
				return
			end
		end

		repo ||= Repository.create(name: repo_name)		# create repo, if does not already exist
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
			  logger.warn "Could not look up virtual machine with uuid #{vm_uuid.inspect}"
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
  		logger.warn "Could not look up the 'All users' group with uuid '*-*-fffffffffffffff'. Skip."
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
