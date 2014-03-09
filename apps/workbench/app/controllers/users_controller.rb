class UsersController < ApplicationController
  skip_before_filter :find_object_by_uuid, :only => [:welcome, :activity]
  skip_around_filter :thread_with_mandatory_api_token, :only => :welcome

  def welcome
    if current_user
      params[:action] = 'home'
      home
    end
  end

  def activity
    @breadcrumb_page_name = nil
    @users = User.all
    @user_activity = {}
    @activity = {
      logins: {},
      jobs: {},
      pipeline_instances: {}
    }
    @spans = [['This week', Time.now.beginning_of_week, Time.now],
              ['Last week', 1.week.ago.beginning_of_week, nil],
              ['This month', Time.now.beginning_of_month, Time.now],
              ['Last month', 1.month.ago.beginning_of_month, nil]].
      collect do |span|
      span[2] ||= span[1].advance(months: 1) if span[0].match /month/
      span[2] ||= span[1].advance(weeks: 1) if span[0].match /week/
      span
    end
    @spans.each do |span, threshold_start, threshold_end|
      @activity[:logins][span] = Log.
        filter([[:event_type, '=', 'login'],
                [:object_kind, '=', 'arvados#user'],
                [:created_at, '>=', threshold_start],
                [:created_at, '<', threshold_end]])
      @activity[:jobs][span] = Job.
        filter([[:created_at, '>=', threshold_start],
                [:created_at, '<', threshold_end]])
      @activity[:pipeline_instances][span] = PipelineInstance.
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
        end
      end
    end
    @users = @users.sort_by do |a|
      [-@user_activity[a.uuid].values.inject(:+), a.full_name]
    end
  end

  def home
    @showallalerts = false
    @my_ssh_keys = AuthorizedKey.where(authorized_user_uuid: current_user.uuid)
    # @my_vm_perms = Link.where(tail_uuid: current_user.uuid, head_kind: 'arvados#virtual_machine', link_class: 'permission', name: 'can_login')
    # @my_repo_perms = Link.where(tail_uuid: current_user.uuid, head_kind: 'arvados#repository', link_class: 'permission', name: 'can_write')

    @my_tag_links = {}

    @my_jobs = Job.
      limit(10).
      order('created_at desc').
      where(created_by: current_user.uuid)

    @my_collections = Collection.
      limit(10).
      order('created_at desc').
      where(created_by: current_user.uuid)

    Link.limit(1000).where(head_uuid: @my_collections.collect(&:uuid),
                           link_class: 'tag').each do |link|
      (@my_tag_links[link.head_uuid] ||= []) << link
    end

    @my_pipelines = PipelineInstance.
      limit(10).
      order('created_at desc').
      where(created_by: current_user.uuid)


    # A Tutorial is a Link which has link_class "resources" and name
    # "wants", and is owned by the Tutorials Group (i.e., named
    # "Arvados Tutorials" and owned by the system user).
    @tutorial_group = Group.where(owner_uuid: User.system.uuid,
                                  name: 'Arvados Tutorials').first
    if @tutorial_group
      @tutorial_links = Link.where(tail_uuid: @tutorial_group.uuid,
                                   link_class: 'resources',
                                   name: 'wants')
    else
      @tutorial_links = []
    end
    @tutorial_complete = {
      'Run a job' => @my_last_job
    }
    respond_to do |f|
      f.js { render template: 'users/home.js' }
      f.html { render template: 'users/home' }
    end
  end
end
