class UsersController < ApplicationController
  skip_around_filter :require_thread_api_token, only: :welcome
  skip_before_filter :check_user_agreements, only: [:welcome, :inactive]
  skip_before_filter :check_user_profile, only: [:welcome, :inactive, :profile]
  skip_before_filter :find_object_by_uuid, only: [:welcome, :activity, :storage]
  before_filter :ensure_current_user_is_admin, only: [:sudo, :unsetup, :setup]

  def show
    if params[:uuid] == current_user.uuid
      respond_to do |f|
        f.html do
          redirect_to(params[:return_to] || project_path(params[:uuid]))
        end
      end
    else
      super
    end
  end

  def welcome
    if current_user
      redirect_to (params[:return_to] || '/')
    end
  end

  def inactive
    if current_user.andand.is_invited
      redirect_to (params[:return_to] || '/')
    end
  end

  def profile
    params[:offer_return_to] ||= params[:return_to]
  end

  def activity
    @breadcrumb_page_name = nil
    @users = User.limit(params[:limit])
    @user_activity = {}
    @activity = {
      logins: {},
      jobs: {},
      pipeline_instances: {}
    }
    @total_activity = {}
    @spans = [['This week', Time.now.beginning_of_week, Time.now],
              ['Last week',
               Time.now.beginning_of_week.advance(weeks:-1),
               Time.now.beginning_of_week],
              ['This month', Time.now.beginning_of_month, Time.now],
              ['Last month',
               1.month.ago.beginning_of_month,
               Time.now.beginning_of_month]]
    @spans.each do |span, threshold_start, threshold_end|
      @activity[:logins][span] = Log.select(%w(uuid modified_by_user_uuid)).
        filter([[:event_type, '=', 'login'],
                [:object_kind, '=', 'arvados#user'],
                [:created_at, '>=', threshold_start],
                [:created_at, '<', threshold_end]])
      @activity[:jobs][span] = Job.select(%w(uuid modified_by_user_uuid)).
        filter([[:created_at, '>=', threshold_start],
                [:created_at, '<', threshold_end]])
      @activity[:pipeline_instances][span] = PipelineInstance.select(%w(uuid modified_by_user_uuid)).
        filter([[:created_at, '>=', threshold_start],
                [:created_at, '<', threshold_end]])
      @activity.each do |type, act|
        records = act[span]
        @users.each do |u|
          @user_activity[u.uuid] ||= {}
          @user_activity[u.uuid][span + ' ' + type.to_s] ||= 0
        end
        records.each do |record|
          @user_activity[record.modified_by_user_uuid] ||= {}
          @user_activity[record.modified_by_user_uuid][span + ' ' + type.to_s] ||= 0
          @user_activity[record.modified_by_user_uuid][span + ' ' + type.to_s] += 1
          @total_activity[span + ' ' + type.to_s] ||= 0
          @total_activity[span + ' ' + type.to_s] += 1
        end
      end
    end
    @users = @users.sort_by do |a|
      [-@user_activity[a.uuid].values.inject(:+), a.full_name]
    end
    # Prepend a "Total" pseudo-user to the sorted list
    @user_activity[nil] = @total_activity
    @users = [OpenStruct.new(uuid: nil)] + @users
  end

  def storage
    @breadcrumb_page_name = nil
    @users = User.limit(params[:limit])
    @user_storage = {}
    total_storage = {}
    @log_date = {}
    @users.each do |u|
      @user_storage[u.uuid] ||= {}
      storage_log = Log.
        filter([[:object_uuid, '=', u.uuid],
                [:event_type, '=', 'user-storage-report']]).
        order(:created_at => :desc).
        limit(1)
      storage_log.each do |log_entry|
        # We expect this block to only execute once since we specified limit(1)
        @user_storage[u.uuid] = log_entry['properties']
        @log_date[u.uuid] = log_entry['event_at']
      end
      total_storage.merge!(@user_storage[u.uuid]) { |k,v1,v2| v1 + v2 }
    end
    @users = @users.sort_by { |u|
      [-@user_storage[u.uuid].values.push(0).inject(:+), u.full_name]}
    # Prepend a "Total" pseudo-user to the sorted list
    @users = [OpenStruct.new(uuid: nil)] + @users
    @user_storage[nil] = total_storage
  end

  def show_pane_list
    if current_user.andand.is_admin
      super | %w(Admin)
    else
      super
    end
  end

  def index_pane_list
    if current_user.andand.is_admin
      super | %w(Activity)
    else
      super
    end
  end

  def sudo
    resp = arvados_api_client.api(ApiClientAuthorization, '', {
                                    api_client_authorization: {
                                      owner_uuid: @object.uuid
                                    }
                                  })
    redirect_to root_url(api_token: resp[:api_token])
  end

  def home
    @my_ssh_keys = AuthorizedKey.where(authorized_user_uuid: current_user.uuid)
    @my_tag_links = {}

    @my_jobs = Job.
      limit(10).
      order('created_at desc').
      where(created_by: current_user.uuid)

    @my_collections = Collection.
      limit(10).
      order('created_at desc').
      where(created_by: current_user.uuid)
    collection_uuids = @my_collections.collect &:uuid

    @persist_state = {}
    collection_uuids.each do |uuid|
      @persist_state[uuid] = 'cache'
    end

    Link.filter([['head_uuid', 'in', collection_uuids],
                             ['link_class', 'in', ['tag', 'resources']]]).
      each do |link|
      case link.link_class
      when 'tag'
        (@my_tag_links[link.head_uuid] ||= []) << link
      when 'resources'
        if link.name == 'wants'
          @persist_state[link.head_uuid] = 'persistent'
        end
      end
    end

    @my_pipelines = PipelineInstance.
      limit(10).
      order('created_at desc').
      where(created_by: current_user.uuid)

    respond_to do |f|
      f.js { render template: 'users/home.js' }
      f.html { render template: 'users/home' }
    end
  end

  def unsetup
    if current_user.andand.is_admin
      @object.unsetup
    end
    show
  end

  def setup
    respond_to do |format|
      if current_user.andand.is_admin
        setup_params = {}
        setup_params[:send_notification_email] = "#{Rails.configuration.send_user_setup_notification_email}"
        if params['user_uuid'] && params['user_uuid'].size>0
          setup_params[:uuid] = params['user_uuid']
        end
        if params['email'] && params['email'].size>0
          user = {email: params['email']}
          setup_params[:user] = user
        end
        if params['openid_prefix'] && params['openid_prefix'].size>0
          setup_params[:openid_prefix] = params['openid_prefix']
        end
        if params['repo_name'] && params['repo_name'].size>0
          setup_params[:repo_name] = params['repo_name']
        end
        if params['vm_uuid'] && params['vm_uuid'].size>0
          setup_params[:vm_uuid] = params['vm_uuid']
        end

        setup_resp = User.setup setup_params
        if setup_resp
          prev_groups = nil
          setup_resp[:items].each do |item|
            if item[:head_kind] == "arvados#virtualMachine"
              prev_groups = item[:properties][:groups]
              break
            end
          end
          if params[:groups]
            new_groups = params[:groups].split(',').map(&:strip).select{|i| !i.empty?}
            if new_groups != prev_groups
              vm_login_perms = Link.where(tail_uuid: params['user_uuid'],
                                          head_kind: 'arvados#virtualMachine',
                                          link_class: 'permission',
                                          name: 'can_login')
              if vm_login_perms.any?
                perm = vm_login_perms.first
                props = perm.properties
                props[:groups] = new_groups
                perm.save!
              end
            end
          end

          format.js
        else
          self.render_error status: 422
        end
      else
        self.render_error status: 422
      end
    end
  end

  def setup_popup
    @vms = VirtualMachine.all.results

    @current_selections = find_current_links @object

    respond_to do |format|
      format.html
      format.js
    end
  end

  def manage_account
    # repositories current user can read / write
    repo_links = Link.
      filter([['head_uuid', 'is_a', 'arvados#repository'],
              ['tail_uuid', '=', current_user.uuid],
              ['link_class', '=', 'permission'],
             ])

    owned_repositories = Repository.where(owner_uuid: current_user.uuid)

    @my_repositories = (Repository.where(uuid: repo_links.collect(&:head_uuid)) |
                        owned_repositories).
                       uniq { |repo| repo.uuid }


    @repo_writable = {}
    repo_links.each do |link|
      if link.name.in? ['can_write', 'can_manage']
        @repo_writable[link.head_uuid] = link.name
      end
    end

    owned_repositories.each do |repo|
      @repo_writable[repo.uuid] = 'can_manage'
    end

    # virtual machines the current user can login into
    @my_vm_logins = {}
    Link.where(tail_uuid: current_user.uuid,
               link_class: 'permission',
               name: 'can_login').
          each do |perm_link|
            if perm_link.properties.andand[:username]
              @my_vm_logins[perm_link.head_uuid] ||= []
              @my_vm_logins[perm_link.head_uuid] << perm_link.properties[:username]
            end
          end
    @my_virtual_machines = VirtualMachine.where(uuid: @my_vm_logins.keys)

    # current user's ssh keys
    @my_ssh_keys = AuthorizedKey.where(key_type: 'SSH', owner_uuid: current_user.uuid)

    respond_to do |f|
      f.html { render template: 'users/manage_account' }
    end
  end

  def add_ssh_key_popup
    respond_to do |format|
      format.html
      format.js
    end
  end

  def add_ssh_key
    respond_to do |format|
      key_params = {'key_type' => 'SSH'}
      key_params['authorized_user_uuid'] = current_user.uuid

      if params['name'] && params['name'].size>0
        key_params['name'] = params['name'].strip
      end
      if params['public_key'] && params['public_key'].size>0
        key_params['public_key'] = params['public_key'].strip
      end

      if !key_params['name'] && params['public_key'].andand.size>0
        split_key = key_params['public_key'].split
        key_params['name'] = split_key[-1] if (split_key.size == 3)
      end

      new_key = AuthorizedKey.create! key_params
      if new_key
        format.js
      else
        self.render_error status: 422
      end
    end
  end

  def request_shell_access
    logger.warn "request_access: #{params.inspect}"
    params['request_url'] = request.url
    RequestShellAccessReporter.send_request(current_user, params).deliver
  end

  protected

  def find_current_links user
    current_selections = {}

    if !user
      return current_selections
    end

    # oid login perm
    oid_login_perms = Link.where(tail_uuid: user.email,
                                   head_kind: 'arvados#user',
                                   link_class: 'permission',
                                   name: 'can_login')

    if oid_login_perms.any?
      prefix_properties = oid_login_perms.first.properties
      current_selections[:identity_url_prefix] = prefix_properties[:identity_url_prefix]
    end

    # repo perm
    repo_perms = Link.where(tail_uuid: user.uuid,
                            head_kind: 'arvados#repository',
                            link_class: 'permission',
                            name: 'can_write')
    if repo_perms.any?
      repo_uuid = repo_perms.first.head_uuid
      repos = Repository.where(head_uuid: repo_uuid)
      if repos.any?
        repo_name = repos.first.name
        current_selections[:repo_name] = repo_name
      end
    end

    # vm login perm
    vm_login_perms = Link.where(tail_uuid: user.uuid,
                              head_kind: 'arvados#virtualMachine',
                              link_class: 'permission',
                              name: 'can_login')
    if vm_login_perms.any?
      vm_perm = vm_login_perms.first
      vm_uuid = vm_perm.head_uuid
      current_selections[:vm_uuid] = vm_uuid
      current_selections[:groups] = vm_perm.properties[:groups].andand.join(', ')
    end

    return current_selections
  end

end
