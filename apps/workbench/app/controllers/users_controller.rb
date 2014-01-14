class UsersController < ApplicationController
  skip_before_filter :find_object_by_uuid, :only => :welcome
  skip_around_filter :thread_with_api_token, :only => :welcome
  around_filter :thread_with_optional_api_token, :only => :welcome

  def welcome
    if current_user
      redirect_to home_user_path(current_user.uuid)
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
  end
end
